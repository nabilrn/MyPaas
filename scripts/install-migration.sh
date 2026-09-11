#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_VM_SCRIPT="${MYPAAS_INSTALL_VM_SCRIPT:-$ROOT_DIR/scripts/install-vm.sh}"
MIGRATION_TMPDIR=""

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "$MIGRATION_TMPDIR" ]]; then
    rm -rf -- "$MIGRATION_TMPDIR"
  fi
}

read_migration_url() {
  local value=""
  if [[ -t 0 ]]; then
    read -r -s -p "Migration download URL: " value
    printf '\n' >&2
  else
    IFS= read -r value || true
  fi
  [[ -n "$value" ]] || die "migration download URL is required"
  case "$value" in
    https://*|http://localhost/*|http://127.0.0.1/*) ;;
    *) die "migration download URL must use HTTPS (localhost HTTP is allowed for local testing)" ;;
  esac
  printf '%s' "$value"
}

curl_config_url() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  [[ "$value" != *$'\n'* && "$value" != *$'\r'* ]] || die "migration download URL contains a newline"
  printf 'url = "%s"\n' "$value"
}

main() {
  [[ $# -eq 0 ]] || die "install-migration.sh does not accept the migration URL as an argument; provide it through stdin or the hidden prompt"
  command -v curl >/dev/null 2>&1 || die "curl is required"
  [[ -f "$INSTALL_VM_SCRIPT" ]] || die "install-vm script not found: $INSTALL_VM_SCRIPT"

  local migration_url archive
  migration_url="$(read_migration_url)"
  umask 077
  MIGRATION_TMPDIR="$(mktemp -d /tmp/mypaas-migration.XXXXXX)"
  trap cleanup EXIT
  archive="$MIGRATION_TMPDIR/mypaas-export.tar.gz"

  log "Downloading migration package through protected curl configuration"
  curl_config_url "$migration_url" \
    | curl --fail --location --silent --show-error --config - --output "$archive"
  chmod 0600 "$archive"
  unset migration_url

  log "Starting MyPaas migration installer"
  MIGRATE_URL="file://$archive" bash "$INSTALL_VM_SCRIPT"
}

main "$@"
