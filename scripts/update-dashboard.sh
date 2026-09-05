#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
DASHBOARD_IMAGE_REPO="${MYPAAS_DASHBOARD_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-dashboard}"
TARGET_TAG="${MYPAAS_DASHBOARD_IMAGE_TAG:-${1:-}}"
VERIFY_ATTEMPTS="${DASHBOARD_UPDATE_VERIFY_ATTEMPTS:-12}"
VERIFY_DELAY_SECONDS="${DASHBOARD_UPDATE_VERIFY_DELAY_SECONDS:-2}"
STATUS_HELPER="$ROOT_DIR/scripts/update-status.sh"

if [[ -r "$STATUS_HELPER" ]]; then
  # shellcheck source=scripts/update-status.sh
  source "$STATUS_HELPER"
fi

status_phase() {
  if declare -F mypaas_update_status_write >/dev/null 2>&1; then
    mypaas_update_status_write "$1" "$2" "$3"
  fi
}

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
  [[ "$value" =~ ^[1-9][0-9]*$ ]] || die "$name must be a positive integer"
}

verify_dashboard() {
  local docker_cmd="$1"
  local expected_tag="$2"
  local attempt image

  for ((attempt = 1; attempt <= VERIFY_ATTEMPTS; attempt++)); do
    if [[ "$($docker_cmd inspect --format '{{.State.Running}}' mypaas-dashboard 2>/dev/null || true)" == "true" ]]; then
      image="$($docker_cmd inspect --format '{{.Config.Image}}' mypaas-dashboard 2>/dev/null || true)"
      if [[ "$image" == *":$expected_tag" ]] && curl -fsSL --max-redirs 5 http://127.0.0.1:3000/ >/dev/null; then
        return 0
      fi
    fi
    if (( attempt < VERIFY_ATTEMPTS )); then
      sleep "$VERIFY_DELAY_SECONDS"
    fi
  done
  return 1
}

main() {
  [[ -n "$TARGET_TAG" ]] || die "dashboard image tag is required"
  [[ "$TARGET_TAG" =~ ^[A-Za-z0-9._-]+$ ]] || die "dashboard image tag contains unsupported characters"
  validate_integer DASHBOARD_UPDATE_VERIFY_ATTEMPTS "$VERIFY_ATTEMPTS"
  validate_integer DASHBOARD_UPDATE_VERIFY_DELAY_SECONDS "$VERIFY_DELAY_SECONDS"
  [[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"

  local docker_cmd compose_cmd target_image old_image_id rollback_tag rollback_ready=false
  docker_cmd="${DOCKER_BIN:-$(docker_prefix)}"
  compose_cmd="${COMPOSE_BIN:-$docker_cmd compose}"
  target_image="$DASHBOARD_IMAGE_REPO:$TARGET_TAG"
  rollback_tag="rollback-dashboard-$(date +%s)-$$"
  old_image_id="$($docker_cmd inspect --format '{{.Image}}' mypaas-dashboard 2>/dev/null || true)"

  if [[ -n "$old_image_id" ]]; then
    $docker_cmd tag "$old_image_id" "$DASHBOARD_IMAGE_REPO:$rollback_tag"
    rollback_ready=true
  fi

  log "Pulling dashboard image $target_image"
  $docker_cmd pull "$target_image"

  status_phase updating applying "Recreating the MyPaas dashboard"
  log "Recreating dashboard only"
  if MYPAAS_IMAGE_TAG="$TARGET_TAG" $compose_cmd -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-deps dashboard; then
    status_phase updating verifying "Verifying the updated MyPaas dashboard"
    if verify_dashboard "$docker_cmd" "$TARGET_TAG"; then
      log "Dashboard updated successfully to ${TARGET_TAG:0:12}"
      if [[ "$rollback_ready" == "true" ]]; then
        $docker_cmd image rm "$DASHBOARD_IMAGE_REPO:$rollback_tag" >/dev/null 2>&1 || true
      fi
      return 0
    fi
  fi

  if [[ "$rollback_ready" == "true" ]]; then
    status_phase updating rolling_back "Dashboard verification failed; restoring the previous dashboard image"
    log "Dashboard update failed; restoring previous dashboard image"
    if MYPAAS_IMAGE_TAG="$rollback_tag" $compose_cmd -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-deps dashboard \
      && verify_dashboard "$docker_cmd" "$rollback_tag"; then
      die "dashboard update to ${TARGET_TAG:0:12} failed; previous dashboard runtime was restored"
    fi
  fi

  die "dashboard update to ${TARGET_TAG:0:12} failed and rollback could not be verified"
}

main "$@"
