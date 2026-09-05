#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
CHANNEL="${AUTO_UPDATE_CHANNEL:-release}"
INCLUDE_PRERELEASES="${AUTO_UPDATE_INCLUDE_PRERELEASES:-false}"
REF="${MYPAAS_REF:-${AUTO_UPDATE_REF:-main}}"
REMOTE="${MYPAAS_REMOTE:-origin}"
RELEASE_REPOSITORY="${MYPAAS_RELEASE_REPOSITORY:-nabilrn/MyPaas}"
DASHBOARD_IMAGE_REPO="${MYPAAS_DASHBOARD_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-dashboard}"
IMAGE_WAIT_SECONDS="${AUTO_UPDATE_IMAGE_WAIT_SECONDS:-300}"
LOCK_DIR="${AUTO_UPDATE_LOCK_DIR:-$ROOT_DIR/.git/mypaas-update.lock}"
STATUS_DIR="${MYPAAS_UPDATE_STATUS_DIR:-/run/mypaas/update}"
STATUS_FILE="${MYPAAS_UPDATE_STATUS_FILE:-$STATUS_DIR/status}"
CHECKOUT_OWNER="$(stat -c '%u:%g' "$ROOT_DIR" 2>/dev/null || true)"
LOCK_HELD=false
TERMINAL_STATUS_WRITTEN=false
CURRENT_SHA=""
TARGET_SHA=""
TARGET_VERSION=""
TARGET_REF=""

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

status_value() {
  printf '%s' "$1" | tr '\r\n=' '   '
}

write_status() {
  local state="$1"
  local message="$2"
  local tmp
  mkdir -p "$STATUS_DIR"
  tmp="$(mktemp "$STATUS_DIR/.status.XXXXXX")"
  {
    printf 'state=%s\n' "$(status_value "$state")"
    printf 'channel=%s\n' "$(status_value "$CHANNEL")"
    printf 'current_sha=%s\n' "$(status_value "$CURRENT_SHA")"
    printf 'target_sha=%s\n' "$(status_value "$TARGET_SHA")"
    printf 'target_version=%s\n' "$(status_value "$TARGET_VERSION")"
    printf 'message=%s\n' "$(status_value "$message")"
    printf 'updated_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } > "$tmp"
  chmod 0644 "$tmp"
  mv -f "$tmp" "$STATUS_FILE"
  case "$state" in
    idle|succeeded|failed|rolled_back|blocked) TERMINAL_STATUS_WRITTEN=true ;;
  esac
}

cleanup_lock() {
  if [[ "$LOCK_HELD" == "true" ]]; then
    rmdir "$LOCK_DIR" >/dev/null 2>&1 || true
    LOCK_HELD=false
  fi
}

on_exit() {
  local code=$?
  cleanup_lock
  if (( code != 0 )) && [[ "$TERMINAL_STATUS_WRITTEN" != "true" ]]; then
    write_status failed "Updater exited before completing the requested revision"
  fi
  trap - EXIT
  exit "$code"
}

normalize_bool() {
  local name="$1"
  local value
  value="$(printf '%s' "$2" | tr '[:upper:]' '[:lower:]')"
  case "$value" in
    true|1|yes|on) printf 'true' ;;
    false|0|no|off) printf 'false' ;;
    *) die "$name must be true or false" ;;
  esac
}

resolve_stable_release_tag() {
  local latest_url effective tag
  latest_url="https://github.com/$RELEASE_REPOSITORY/releases/latest"
  effective="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$latest_url")"
  tag="${effective##*/}"
  [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] || die "could not resolve a valid stable release tag"
  printf '%s' "$tag"
}

resolve_latest_published_tag() {
  command_exists python3 || die "python3 is required when AUTO_UPDATE_INCLUDE_PRERELEASES=true"
  curl -fsSL \
    -H 'Accept: application/vnd.github+json' \
    -H 'X-GitHub-Api-Version: 2022-11-28' \
    "https://api.github.com/repos/$RELEASE_REPOSITORY/releases?per_page=20" \
    | python3 -c 'import json,sys; releases=json.load(sys.stdin); release=next((r for r in releases if not r.get("draft")), None); print(release.get("tag_name", "") if release else "")'
}

resolve_target() {
  local tag
  case "$CHANNEL" in
    release)
      if [[ "$INCLUDE_PRERELEASES" == "true" ]]; then
        tag="$(resolve_latest_published_tag)"
      else
        tag="$(resolve_stable_release_tag)"
      fi
      [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] || die "release channel returned an invalid tag"
      TARGET_VERSION="$tag"
      TARGET_REF="refs/tags/$tag"
      log "Checking published release $tag"
      git_repo fetch --force "$REMOTE" "$TARGET_REF"
      ;;
    main)
      TARGET_VERSION="main"
      TARGET_REF="$REF"
      log "Checking development ref $REMOTE/$REF"
      git_repo fetch "$REMOTE" "$REF"
      ;;
    *)
      die "AUTO_UPDATE_CHANNEL must be release or main"
      ;;
  esac
  TARGET_SHA="$(git_repo rev-parse 'FETCH_HEAD^{commit}')"
}

ensure_complete_history() {
  if [[ "$(git_repo rev-parse --is-shallow-repository)" != "true" ]]; then
    return
  fi

  log "Expanding shallow checkout history for ancestry validation"
  git_repo fetch --unshallow "$REMOTE"
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

run_logged() {
  local log_file="$1"
  shift
  set +e
  "$@" 2>&1 | tee "$log_file"
  local code=${PIPESTATUS[0]}
  set -e
  return "$code"
}

main() {
  [[ "$IMAGE_WAIT_SECONDS" =~ ^[0-9]+$ ]] || die "AUTO_UPDATE_IMAGE_WAIT_SECONDS must be an integer"
  [[ "$REF" =~ ^[A-Za-z0-9._/-]+$ ]] || die "MYPAAS_REF contains unsupported characters"
  [[ "$RELEASE_REPOSITORY" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "MYPAAS_RELEASE_REPOSITORY is invalid"
  INCLUDE_PRERELEASES="$(normalize_bool AUTO_UPDATE_INCLUDE_PRERELEASES "$INCLUDE_PRERELEASES")"
  [[ -d "$ROOT_DIR/.git" ]] || die "$ROOT_DIR is not a Git checkout"
  [[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"
  command_exists curl || die "curl is required for release discovery"

  trap on_exit EXIT
  CURRENT_SHA="$(git_repo rev-parse HEAD)"
  write_status checking "Checking for a published MyPaas update"

  resolve_target
  ensure_complete_history
  write_status checking "Resolved $TARGET_VERSION at ${TARGET_SHA:0:12}"

  if [[ "$CURRENT_SHA" != "$TARGET_SHA" ]] && ! git_repo merge-base --is-ancestor "$CURRENT_SHA" "$TARGET_SHA"; then
    write_status blocked "Refusing to move to a target that is not a descendant of the current checkout"
    log "Target ${TARGET_SHA:0:12} is not a descendant of ${CURRENT_SHA:0:12}; leaving the installation unchanged"
    return 0
  fi

  if [[ "$CURRENT_SHA" == "$TARGET_SHA" ]]; then
    write_status updating "Reconciling host dependencies for the current revision"
    if env MYPAAS_REF="$TARGET_REF" MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" \
      bash "$ROOT_DIR/scripts/update-vm.sh"; then
      write_status idle "MyPaas is already up to date"
      return 0
    fi
    write_status failed "Current revision reconciliation failed"
    return 1
  fi

  if ! is_frontend_only "$CURRENT_SHA" "$TARGET_SHA"; then
    local full_log
    full_log="$(mktemp)"
    write_status updating "Applying MyPaas release $TARGET_VERSION"
    if run_logged "$full_log" env MYPAAS_REF="$TARGET_REF" MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" \
      bash "$ROOT_DIR/scripts/update-vm.sh"; then
      rm -f "$full_log"
      CURRENT_SHA="$TARGET_SHA"
      write_status succeeded "MyPaas updated successfully to $TARGET_VERSION"
      return 0
    fi
    if grep -q 'was restored and verified' "$full_log"; then
      write_status rolled_back "Update failed; the previous runtime was restored and verified"
    else
      write_status failed "Update failed; inspect the host updater logs"
    fi
    rm -f "$full_log"
    return 1
  fi

  log "Frontend-only update detected; using dashboard fast path"
  if [[ -n "$(git_repo status --porcelain)" ]]; then
    write_status blocked "$ROOT_DIR has local changes; automatic update is disabled until the checkout is clean"
    return 0
  fi
  if ! mkdir "$LOCK_DIR" >/dev/null 2>&1; then
    log "Another MyPaas update is already running; skipping"
    return 0
  fi
  LOCK_HELD=true

  local docker_cmd target_dashboard dashboard_log
  docker_cmd="$(docker_prefix)"
  target_dashboard="$DASHBOARD_IMAGE_REPO:$TARGET_SHA"
  log "Waiting for dashboard image for ${TARGET_SHA:0:12}"
  if ! wait_for_image "$docker_cmd" "$target_dashboard"; then
    write_status blocked "Dashboard image for $TARGET_VERSION is not published yet"
    log "Dashboard image is not published yet; leaving the running installation unchanged"
    return 0
  fi

  write_status updating "Applying dashboard release $TARGET_VERSION"
  log "Updating checkout ${CURRENT_SHA:0:12} -> ${TARGET_SHA:0:12}"
  git_repo reset --hard "$TARGET_SHA"
  restore_checkout_owner

  dashboard_log="$(mktemp)"
  if run_logged "$dashboard_log" env DOCKER_BIN="$docker_cmd" MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" \
    MYPAAS_DASHBOARD_IMAGE_TAG="$TARGET_SHA" bash "$ROOT_DIR/scripts/update-dashboard.sh"; then
    rm -f "$dashboard_log"
    CURRENT_SHA="$TARGET_SHA"
    write_status succeeded "Dashboard updated successfully to $TARGET_VERSION"
    log "Frontend-only MyPaas update completed at ${TARGET_SHA:0:12}"
    return 0
  fi

  log "Frontend fast path failed; restoring checkout ${CURRENT_SHA:0:12}"
  git_repo reset --hard "$CURRENT_SHA"
  restore_checkout_owner
  if grep -q 'previous dashboard runtime was restored' "$dashboard_log"; then
    write_status rolled_back "Dashboard update failed; the previous dashboard runtime was restored"
  else
    write_status failed "Dashboard update failed and rollback could not be verified"
  fi
  rm -f "$dashboard_log"
  return 1
}

main "$@"
