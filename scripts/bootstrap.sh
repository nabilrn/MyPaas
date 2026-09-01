#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${MYPAAS_REPO_URL:-https://github.com/nabilrn/MyPaas.git}"
REF="${MYPAAS_REF:-main}"
INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
INSTALL_WIZARD="${INSTALL_WIZARD:-true}"
EXPLICIT_USE_PODMAN_SET="${USE_PODMAN+x}"
USE_PODMAN="${USE_PODMAN:-true}"

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

run_root() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
    return
  fi
  command_exists sudo || die "sudo is required when not running as root"
  sudo "$@"
}

usage() {
  printf '%s\n' \
    'MyPaas bootstrap installer' \
    '' \
    'Environment overrides:' \
    '  MYPAAS_REPO_URL               Git repository URL' \
    '  MYPAAS_REF                    Branch or tag to install (default: main)' \
    '  MYPAAS_INSTALL_DIR            Checkout directory (default: $HOME/MyPaas)' \
    '  INSTALL_WIZARD                Start browser setup wizard (default: true)' \
    '  USE_PODMAN                    Fresh-install runtime choice (default: true; existing installs preserve their detected engine)' \
    '  AUTO_UPDATE_ENABLED           Enable systemd self-updates (default: false)' \
    '  AUTO_UPDATE_INTERVAL_MINUTES  Update check interval (default: 30)' \
    '  AUTO_UPDATE_REF               Ref watched by the updater (default: main)' \
    '' \
    'All install-vm.sh environment flags are forwarded to the installer.'
}

ensure_git() {
  if command_exists git; then
    return
  fi
  command_exists apt-get || die "git is required; automatic installation supports Ubuntu/Debian"

  log "Installing Git"
  run_root apt-get update
  run_root env DEBIAN_FRONTEND=noninteractive apt-get install -y git ca-certificates
}

socket_has_mypaas_containers() {
  local socket="$1"
  [[ -S "$socket" ]] || return 1
  DOCKER_HOST="unix://$socket" docker ps -a --format '{{.Names}}' 2>/dev/null \
    | grep -Eq '^mypaas-(postgres-prod|api|dashboard|caddy-prod|cloudflared)$'
}

detect_existing_runtime() {
  if [[ ! -d "$INSTALL_DIR/.git" || ! -f "$INSTALL_DIR/.env" ]]; then
    return 0
  fi
  command_exists docker || return 0

  local docker_socket="/var/run/docker.sock"
  local podman_socket="/run/podman/podman.sock"
  local docker_socket_target=""
  local docker_has_state=false
  local podman_has_state=false
  local detected_use_podman=""

  if [[ -e "$docker_socket" || -L "$docker_socket" ]]; then
    docker_socket_target="$(readlink -f "$docker_socket" 2>/dev/null || true)"
  fi

  if [[ -S "$docker_socket" && "$docker_socket_target" != "$podman_socket" ]] \
    && socket_has_mypaas_containers "$docker_socket"; then
    docker_has_state=true
  fi
  if socket_has_mypaas_containers "$podman_socket"; then
    podman_has_state=true
  fi

  if [[ "$docker_has_state" == "true" && "$podman_has_state" == "true" ]]; then
    die "MyPaas state exists in both Docker Engine and Podman; refusing a split-runtime update. Keep the current installation on one engine or migrate to a fresh VM."
  fi

  if [[ "$docker_has_state" == "true" ]]; then
    detected_use_podman=false
  elif [[ "$podman_has_state" == "true" ]]; then
    detected_use_podman=true
  else
    return 0
  fi

  if [[ -n "$EXPLICIT_USE_PODMAN_SET" && "$USE_PODMAN" != "$detected_use_podman" ]]; then
    die "Refusing an in-place Docker/Podman engine switch for an existing MyPaas installation. Use the VM migration workflow for engine changes."
  fi

  USE_PODMAN="$detected_use_podman"
  export USE_PODMAN
  if [[ "$USE_PODMAN" == "true" ]]; then
    log "Preserving existing Podman runtime"
  else
    log "Preserving existing Docker Engine runtime"
  fi
}

checkout_repo() {
  if [[ -e "$INSTALL_DIR" && ! -d "$INSTALL_DIR" ]]; then
    die "$INSTALL_DIR exists and is not a directory"
  fi

  if [[ -d "$INSTALL_DIR/.git" ]]; then
    [[ -z "$(git -C "$INSTALL_DIR" status --porcelain)" ]] || die "$INSTALL_DIR has local changes; preserve or remove them before rerunning"
    [[ "$(git -C "$INSTALL_DIR" remote get-url origin)" == "$REPO_URL" ]] || die "$INSTALL_DIR points to a different Git origin"
    log "Updating existing MyPaas checkout"
    git -C "$INSTALL_DIR" fetch --depth 1 origin "$REF"
    # This checkout is installer-managed and local changes were rejected above.
    # Resetting to FETCH_HEAD is intentional: upstream may rewrite/squash main,
    # where an ff-only merge can fail with unrelated histories.
    git -C "$INSTALL_DIR" reset --hard FETCH_HEAD
    return
  fi

  if [[ -d "$INSTALL_DIR" && -n "$(find "$INSTALL_DIR" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
    die "$INSTALL_DIR exists and is not empty"
  fi

  mkdir -p "$(dirname "$INSTALL_DIR")"
  log "Downloading MyPaas $REF"
  git clone --depth 1 --branch "$REF" "$REPO_URL" "$INSTALL_DIR"
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    return
  fi
  [[ $# -eq 0 ]] || die "unknown argument: $1"
  [[ "$(uname -s)" == "Linux" ]] || die "bootstrap.sh must run on a Linux VM"

  ensure_git
  detect_existing_runtime
  checkout_repo

  log "Starting MyPaas installer"
  cd "$INSTALL_DIR"
  INSTALL_WIZARD="$INSTALL_WIZARD" USE_PODMAN="$USE_PODMAN" bash scripts/install-vm.sh
}

main "$@"
