#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
UPDATE_CONFIG_FILE="${MYPAAS_UPDATE_CONFIG:-/etc/mypaas/update.env}"
EXPLICIT_REF_SET="${MYPAAS_REF+x}"
EXPLICIT_WAIT_SET="${AUTO_UPDATE_IMAGE_WAIT_SECONDS+x}"

read_update_setting() {
  local key="$1"
  local line value
  [[ -r "$UPDATE_CONFIG_FILE" ]] || return 1
  line="$(grep -E "^${key}=" "$UPDATE_CONFIG_FILE" | tail -n 1 || true)"
  [[ -n "$line" ]] || return 1
  value="${line#*=}"
  if [[ ${#value} -ge 2 ]]; then
    if [[ "${value:0:1}" == '"' && "${value: -1}" == '"' ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "${value:0:1}" == "'" && "${value: -1}" == "'" ]]; then
      value="${value:1:${#value}-2}"
    fi
  fi
  printf '%s' "$value"
}

if [[ -z "$EXPLICIT_REF_SET" ]]; then
  persisted_ref="$(read_update_setting MYPAAS_REF 2>/dev/null || true)"
  if [[ -n "$persisted_ref" ]]; then
    MYPAAS_REF="$persisted_ref"
  fi
fi
if [[ -z "$EXPLICIT_WAIT_SET" ]]; then
  persisted_wait="$(read_update_setting AUTO_UPDATE_IMAGE_WAIT_SECONDS 2>/dev/null || true)"
  if [[ -n "$persisted_wait" ]]; then
    AUTO_UPDATE_IMAGE_WAIT_SECONDS="$persisted_wait"
  fi
fi

REF="${MYPAAS_REF:-main}"
REMOTE="${MYPAAS_REMOTE:-origin}"
API_IMAGE_REPO="${MYPAAS_API_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-api}"
DASHBOARD_IMAGE_REPO="${MYPAAS_DASHBOARD_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-dashboard}"
IMAGE_WAIT_SECONDS="${AUTO_UPDATE_IMAGE_WAIT_SECONDS:-300}"
VERIFY_ATTEMPTS="${AUTO_UPDATE_VERIFY_ATTEMPTS:-12}"
VERIFY_DELAY_SECONDS="${AUTO_UPDATE_VERIFY_DELAY_SECONDS:-5}"
LOCK_DIR="${AUTO_UPDATE_LOCK_DIR:-$ROOT_DIR/.git/mypaas-update.lock}"
CHECKOUT_OWNER="$(stat -c '%u:%g' "$ROOT_DIR" 2>/dev/null || true)"

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

git_repo() {
  # systemd runs the updater as root by default. Explicitly trusting only the
  # configured MyPaas checkout avoids Git's dubious-ownership rejection for
  # installs owned by a non-root user without changing global Git config.
  git -c safe.directory="$ROOT_DIR" -C "$ROOT_DIR" "$@"
}

restore_checkout_owner() {
  if [[ "${EUID:-$(id -u)}" -eq 0 && -n "$CHECKOUT_OWNER" && "$CHECKOUT_OWNER" != "0:0" ]]; then
    chown -R "$CHECKOUT_OWNER" "$ROOT_DIR"
  fi
}

docker_prefix() {
  if docker ps >/dev/null 2>&1; then
    printf 'docker'
    return
  fi
  if command_exists sudo && sudo docker ps >/dev/null 2>&1; then
    printf 'sudo docker'
    return
  fi
  die "current user cannot access Docker"
}

validate_integer() {
  local name="$1"
  local value="$2"
  [[ "$value" =~ ^[0-9]+$ ]] || die "$name must be an integer"
}

validate_ref() {
  [[ "$REF" =~ ^[A-Za-z0-9._/-]+$ ]] || die "MYPAAS_REF contains unsupported characters"
}

cleanup_lock() {
  rmdir "$LOCK_DIR" >/dev/null 2>&1 || true
}

wait_for_image() {
  local docker_cmd="$1"
  local image="$2"
  local started now
  started="$(date +%s)"
  while true; do
    if $docker_cmd pull "$image" >/dev/null 2>&1; then
      return 0
    fi
    now="$(date +%s)"
    if (( now - started >= IMAGE_WAIT_SECONDS )); then
      return 1
    fi
    sleep 10
  done
}

running_image_id() {
  local docker_cmd="$1"
  local container="$2"
  $docker_cmd inspect --format '{{.Image}}' "$container" 2>/dev/null || true
}

container_env_value() {
  local docker_cmd="$1"
  local container="$2"
  local key="$3"
  $docker_cmd inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$container" 2>/dev/null \
    | sed -n "s/^${key}=//p" | tail -n 1
}

env_file_value() {
  local key="$1"
  grep -E "^${key}=" "$ENV_FILE" | tail -n 1 | cut -d= -f2- || true
}

reconcile_statd() {
  ENV_FILE="$ENV_FILE" MYPAAS_INSTALL_DIR="$ROOT_DIR" bash "$ROOT_DIR/scripts/reconcile-statd.sh"
}

verify_stack() {
  local docker_cmd="$1"
  local attempt
  for ((attempt = 1; attempt <= VERIFY_ATTEMPTS; attempt++)); do
    if DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" \
      bash "$ROOT_DIR/scripts/verify-production.sh" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$VERIFY_DELAY_SECONDS"
  done
  return 1
}

redeploy_current_for_env_drift() {
  local docker_cmd="$1"
  local current_sha="$2"
  local desired_socket api_socket
  desired_socket="$(env_file_value STATD_SOCKET)"
  api_socket="$(container_env_value "$docker_cmd" mypaas-api STATD_SOCKET)"
  if [[ -n "$desired_socket" && "$api_socket" != "$desired_socket" ]]; then
    log "API runtime environment is missing the reconciled STATD_SOCKET; recreating the current stack"
    MYPAAS_IMAGE_TAG="$current_sha" DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" \
      ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" bash "$ROOT_DIR/scripts/deploy-to-vm.sh"
    verify_stack "$docker_cmd" || die "current MyPaas stack failed verification after STATD_SOCKET reconciliation"
  fi
}

main() {
  validate_integer AUTO_UPDATE_IMAGE_WAIT_SECONDS "$IMAGE_WAIT_SECONDS"
  validate_integer AUTO_UPDATE_VERIFY_ATTEMPTS "$VERIFY_ATTEMPTS"
  validate_integer AUTO_UPDATE_VERIFY_DELAY_SECONDS "$VERIFY_DELAY_SECONDS"
  validate_ref

  [[ -d "$ROOT_DIR/.git" ]] || die "$ROOT_DIR is not a Git checkout"
  [[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"

  if ! mkdir "$LOCK_DIR" >/dev/null 2>&1; then
    log "Another MyPaas update is already running; skipping"
    return 0
  fi
  trap cleanup_lock EXIT

  if [[ -n "$(git_repo status --porcelain)" ]]; then
    die "$ROOT_DIR has local changes; automatic update is disabled until the checkout is clean"
  fi

  log "Checking $REMOTE/$REF for MyPaas updates"
  git_repo fetch --depth 1 "$REMOTE" "$REF"

  local current_sha target_sha docker_cmd
  current_sha="$(git_repo rev-parse HEAD)"
  target_sha="$(git_repo rev-parse FETCH_HEAD)"
  docker_cmd="$(docker_prefix)"

  if [[ "$current_sha" == "$target_sha" ]]; then
    # Host-native dependencies and ignored production env files can drift even
    # when the Git checkout is already current. Reconcile them before returning.
    reconcile_statd
    redeploy_current_for_env_drift "$docker_cmd" "$current_sha"
    log "MyPaas is already up to date (${current_sha:0:12})"
    return 0
  fi

  local target_api target_dashboard
  target_api="$API_IMAGE_REPO:$target_sha"
  target_dashboard="$DASHBOARD_IMAGE_REPO:$target_sha"

  log "Waiting for release images for ${target_sha:0:12}"
  if ! wait_for_image "$docker_cmd" "$target_api"; then
    log "API image is not published yet; leaving the running installation unchanged"
    return 0
  fi
  if ! wait_for_image "$docker_cmd" "$target_dashboard"; then
    log "Dashboard image is not published yet; leaving the running installation unchanged"
    return 0
  fi

  local rollback_tag api_image_id dashboard_image_id rollback_ready=false
  rollback_tag="rollback-${current_sha:0:12}"
  api_image_id="$(running_image_id "$docker_cmd" mypaas-api)"
  dashboard_image_id="$(running_image_id "$docker_cmd" mypaas-dashboard)"
  if [[ -n "$api_image_id" && -n "$dashboard_image_id" ]]; then
    $docker_cmd tag "$api_image_id" "$API_IMAGE_REPO:$rollback_tag"
    $docker_cmd tag "$dashboard_image_id" "$DASHBOARD_IMAGE_REPO:$rollback_tag"
    rollback_ready=true
  fi

  log "Updating checkout ${current_sha:0:12} -> ${target_sha:0:12}"
  git_repo reset --hard "$target_sha"
  restore_checkout_owner

  local deploy_ok=true
  if ! reconcile_statd; then
    deploy_ok=false
  elif ! MYPAAS_IMAGE_TAG="$target_sha" DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" \
    ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" bash "$ROOT_DIR/scripts/deploy-to-vm.sh"; then
    deploy_ok=false
  elif ! verify_stack "$docker_cmd"; then
    deploy_ok=false
  fi

  if [[ "$deploy_ok" != "true" ]]; then
    log "Update failed; restoring checkout ${current_sha:0:12}"
    git_repo reset --hard "$current_sha"
    restore_checkout_owner
    if [[ "$rollback_ready" == "true" ]]; then
      log "Attempting best-effort runtime rollback"
      MYPAAS_IMAGE_TAG="$rollback_tag" DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" \
        ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" bash "$ROOT_DIR/scripts/deploy-to-vm.sh" || true
    else
      printf 'WARNING: previous runtime images were not available for automatic rollback.\n' >&2
    fi
    die "MyPaas update to ${target_sha:0:12} failed"
  fi

  log "MyPaas updated successfully to ${target_sha:0:12}"
  if [[ "$rollback_ready" == "true" ]]; then
    $docker_cmd image rm "$API_IMAGE_REPO:$rollback_tag" "$DASHBOARD_IMAGE_REPO:$rollback_tag" >/dev/null 2>&1 || true
  fi
}

main "$@"
