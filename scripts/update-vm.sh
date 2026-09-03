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

preflight_existing_runtime() {
  local docker_cmd="$1"
  local control_network networks network

  # A missing API is handled by the normal deployment path. When it exists,
  # prove that it is in a safe state before any checkout reset or container
  # recreation. This specifically protects Podman/Netavark from stale project
  # network references left by older DB Studio Compose access.
  if ! $docker_cmd inspect mypaas-api >/dev/null 2>&1; then
    return 0
  fi

  control_network="$(env_file_value CONTROL_NETWORK)"
  control_network="${control_network:-mypaas-control}"
  networks="$($docker_cmd inspect --format '{{range $name, $_ := .NetworkSettings.Networks}}{{println $name}}{{end}}' mypaas-api 2>/dev/null || true)"

  while IFS= read -r network; do
    network="$(printf '%s' "$network" | xargs)"
    [[ -n "$network" ]] || continue
    [[ "$network" == "$control_network" ]] && continue

    if ! $docker_cmd network inspect "$network" >/dev/null 2>&1; then
      die "mypaas-api references missing network $network; refusing to update before runtime state is repaired"
    fi

    log "Detaching mypaas-api from unexpected network $network before update"
    if ! $docker_cmd network disconnect -f "$network" mypaas-api >/dev/null; then
      die "failed to detach mypaas-api from unexpected network $network"
    fi
  done <<< "$networks"

  networks="$($docker_cmd inspect --format '{{range $name, $_ := .NetworkSettings.Networks}}{{println $name}}{{end}}' mypaas-api 2>/dev/null || true)"
  if [[ "$(printf '%s\n' "$networks" | sed '/^[[:space:]]*$/d' | wc -l | tr -d ' ')" != "1" ]] \
    || ! printf '%s\n' "$networks" | grep -Fxq "$control_network"; then
    die "mypaas-api network membership is not isolated to $control_network after preflight"
  fi

  if ! curl -fsS --max-time 5 http://127.0.0.1:8080/health >/dev/null; then
    die "existing MyPaas API is not healthy through 127.0.0.1:8080; refusing to update before host-port networking is repaired"
  fi
}

reconcile_statd() {
  ENV_FILE="$ENV_FILE" MYPAAS_INSTALL_DIR="$ROOT_DIR" bash "$ROOT_DIR/scripts/reconcile-statd.sh"
}

verify_stack() {
  local docker_cmd="$1"
  local expected_build_sha="${2:-}"
  local expected_image_tag="${3:-$expected_build_sha}"
  local attempt verify_log
  verify_log="$(mktemp)"
  for ((attempt = 1; attempt <= VERIFY_ATTEMPTS; attempt++)); do
    if MYPAAS_IMAGE_TAG="$expected_image_tag" MYPAAS_BUILD_SHA="$expected_build_sha" \
      DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" \
      EXPECTED_BUILD_SHA="$expected_build_sha" EXPECTED_IMAGE_TAG="$expected_image_tag" \
      REQUIRE_PROJECT_ROUTE="${AUTO_UPDATE_REQUIRE_PROJECT_ROUTE:-false}" \
      bash "$ROOT_DIR/scripts/verify-production.sh" >"$verify_log" 2>&1; then
      rm -f "$verify_log"
      return 0
    fi
    if (( attempt < VERIFY_ATTEMPTS )); then
      sleep "$VERIFY_DELAY_SECONDS"
    fi
  done
  printf 'Production verification failed after %s attempts. Last verifier output:\n' "$VERIFY_ATTEMPTS" >&2
  cat "$verify_log" >&2 || true
  rm -f "$verify_log"
  return 1
}

redeploy_current_for_env_drift() {
  local docker_cmd="$1"
  local current_sha="$2"
  local desired_socket api_socket
  desired_socket="$(env_file_value STATD_SOCKET)"
  api_socket="$(container_env_value "$docker_cmd" mypaas-api STATD_SOCKET)"
  if [[ -n "$desired_socket" && "$api_socket" != "$desired_socket" ]]; then
    preflight_existing_runtime "$docker_cmd"
    log "API runtime environment is missing the reconciled STATD_SOCKET; recreating the current stack"
    MYPAAS_IMAGE_TAG="$current_sha" MYPAAS_BUILD_SHA="$current_sha" DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" \
      ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" bash "$ROOT_DIR/scripts/deploy-to-vm.sh"
    verify_stack "$docker_cmd" "$current_sha" "$current_sha" || die "current MyPaas stack failed verification after STATD_SOCKET reconciliation"
  fi
}

migration_runner_ready() {
  local docker_cmd="$1"
  $docker_cmd run --rm migrate/migrate:latest -version >/dev/null 2>&1
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

  local target_skip_migrations=false
  if git_repo diff --quiet "$current_sha" "$target_sha" -- backend/migrations; then
    target_skip_migrations=true
    log "No control-plane migration changes detected; database migration helper will be skipped"
  else
    log "Control-plane migrations changed; preflighting migration helper before touching the checkout"
    if ! migration_runner_ready "$docker_cmd"; then
      log "Migration helper could not start; leaving the running installation and checkout unchanged"
      return 0
    fi
  fi

  # From this point onward the update is actually ready to mutate runtime state.
  # Clean any live DB Studio network attachment and prove the host-port path is
  # healthy before tagging rollback images or resetting the checkout.
  preflight_existing_runtime "$docker_cmd"

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
  elif ! MYPAAS_IMAGE_TAG="$target_sha" MYPAAS_BUILD_SHA="$target_sha" MYPAAS_SKIP_MIGRATIONS="$target_skip_migrations" \
    DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" \
    bash "$ROOT_DIR/scripts/deploy-to-vm.sh"; then
    deploy_ok=false
  elif ! verify_stack "$docker_cmd" "$target_sha" "$target_sha"; then
    deploy_ok=false
  fi

  if [[ "$deploy_ok" != "true" ]]; then
    log "Update failed; restoring checkout ${current_sha:0:12}"
    git_repo reset --hard "$current_sha"
    restore_checkout_owner
    local rollback_verified=false
    if [[ "$rollback_ready" == "true" ]]; then
      log "Attempting runtime rollback from verified local images"
      if MYPAAS_IMAGE_TAG="$rollback_tag" MYPAAS_BUILD_SHA="$current_sha" MYPAAS_SKIP_IMAGE_PULL=true MYPAAS_SKIP_MIGRATIONS=true \
        DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" \
        bash "$ROOT_DIR/scripts/deploy-to-vm.sh" \
        && verify_stack "$docker_cmd" "$current_sha" "$rollback_tag"; then
        rollback_verified=true
      fi
    else
      printf 'WARNING: previous runtime images were not available for automatic rollback.\n' >&2
    fi
    if [[ "$rollback_verified" == "true" ]]; then
      die "MyPaas update to ${target_sha:0:12} failed; previous runtime ${current_sha:0:12} was restored and verified"
    fi
    die "MyPaas update to ${target_sha:0:12} failed and the previous runtime could not be verified after rollback"
  fi

  log "MyPaas updated successfully to ${target_sha:0:12}"
  if [[ "$rollback_ready" == "true" ]]; then
    $docker_cmd image rm "$API_IMAGE_REPO:$rollback_tag" "$DASHBOARD_IMAGE_REPO:$rollback_tag" >/dev/null 2>&1 || true
  fi
}

main "$@"
