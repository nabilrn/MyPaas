#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
REF="${MYPAAS_REF:-main}"
REMOTE="${MYPAAS_REMOTE:-origin}"
DASHBOARD_IMAGE_REPO="${MYPAAS_DASHBOARD_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-dashboard}"
IMAGE_WAIT_SECONDS="${AUTO_UPDATE_IMAGE_WAIT_SECONDS:-300}"
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

cleanup_lock() {
  rmdir "$LOCK_DIR" >/dev/null 2>&1 || true
}

is_frontend_only() {
  local current_sha="$1"
  local target_sha="$2"
  local found=false path

  while IFS= read -r path; do
    [[ -n "$path" ]] || continue
    found=true
    if [[ "$path" != frontend/* ]]; then
      return 1
    fi
  done < <(git_repo diff --name-only "$current_sha" "$target_sha")

  [[ "$found" == "true" ]]
}

main() {
  [[ "$IMAGE_WAIT_SECONDS" =~ ^[0-9]+$ ]] || die "AUTO_UPDATE_IMAGE_WAIT_SECONDS must be an integer"
  [[ "$REF" =~ ^[A-Za-z0-9._/-]+$ ]] || die "MYPAAS_REF contains unsupported characters"
  [[ -d "$ROOT_DIR/.git" ]] || die "$ROOT_DIR is not a Git checkout"
  [[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"

  log "Checking $REMOTE/$REF for MyPaas updates"
  git_repo fetch --depth 1 "$REMOTE" "$REF"

  local current_sha target_sha docker_cmd target_dashboard
  current_sha="$(git_repo rev-parse HEAD)"
  target_sha="$(git_repo rev-parse FETCH_HEAD)"

  if [[ "$current_sha" == "$target_sha" ]]; then
    exec env MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/update-vm.sh"
  fi

  if ! is_frontend_only "$current_sha" "$target_sha"; then
    log "Update touches platform/backend scope; using full updater"
    exec env MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/update-vm.sh"
  fi

  log "Frontend-only update detected; using dashboard fast path"
  if [[ -n "$(git_repo status --porcelain)" ]]; then
    die "$ROOT_DIR has local changes; automatic update is disabled until the checkout is clean"
  fi
  if ! mkdir "$LOCK_DIR" >/dev/null 2>&1; then
    log "Another MyPaas update is already running; skipping"
    return 0
  fi
  trap cleanup_lock EXIT

  docker_cmd="$(docker_prefix)"
  target_dashboard="$DASHBOARD_IMAGE_REPO:$target_sha"
  log "Waiting for dashboard image for ${target_sha:0:12}"
  if ! wait_for_image "$docker_cmd" "$target_dashboard"; then
    log "Dashboard image is not published yet; leaving the running installation unchanged"
    return 0
  fi

  log "Updating checkout ${current_sha:0:12} -> ${target_sha:0:12}"
  git_repo reset --hard "$target_sha"
  restore_checkout_owner

  if DOCKER_BIN="$docker_cmd" MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" \
    MYPAAS_DASHBOARD_IMAGE_TAG="$target_sha" bash "$ROOT_DIR/scripts/update-dashboard.sh"; then
    log "Frontend-only MyPaas update completed at ${target_sha:0:12}"
    return 0
  fi

  log "Frontend fast path failed; restoring checkout ${current_sha:0:12}"
  git_repo reset --hard "$current_sha"
  restore_checkout_owner
  die "frontend-only update to ${target_sha:0:12} failed; previous checkout was restored"
}

main "$@"
