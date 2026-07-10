#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${MYPAAS_REPO_URL:-https://github.com/nabilrn/MyPaas.git}"
REF="${MYPAAS_REF:-main}"
INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
INSTALL_WIZARD="${INSTALL_WIZARD:-true}"

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
    '  MYPAAS_REPO_URL       Git repository URL' \
    '  MYPAAS_REF            Branch or tag to install (default: main)' \
    '  MYPAAS_INSTALL_DIR    Checkout directory (default: $HOME/MyPaas)' \
    '  INSTALL_WIZARD        Start browser setup wizard (default: true)' \
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

checkout_repo() {
  if [[ -e "$INSTALL_DIR" && ! -d "$INSTALL_DIR" ]]; then
    die "$INSTALL_DIR exists and is not a directory"
  fi

  if [[ -d "$INSTALL_DIR/.git" ]]; then
    [[ -z "$(git -C "$INSTALL_DIR" status --porcelain)" ]] || die "$INSTALL_DIR has local changes; preserve or remove them before rerunning"
    [[ "$(git -C "$INSTALL_DIR" remote get-url origin)" == "$REPO_URL" ]] || die "$INSTALL_DIR points to a different Git origin"
    log "Updating existing MyPaas checkout"
    git -C "$INSTALL_DIR" fetch --depth 1 origin "$REF"
    git -C "$INSTALL_DIR" merge --ff-only FETCH_HEAD
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
  checkout_repo

  log "Starting MyPaas installer"
  cd "$INSTALL_DIR"
  INSTALL_WIZARD="$INSTALL_WIZARD" bash scripts/install-vm.sh
}

main "$@"
