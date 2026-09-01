#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
UPDATE_CONFIG_DIR="${MYPAAS_UPDATE_CONFIG_DIR:-/etc/mypaas}"
UPDATE_CONFIG_FILE="${MYPAAS_UPDATE_CONFIG:-$UPDATE_CONFIG_DIR/update.env}"
SERVICE_FILE="/etc/systemd/system/mypaas-update.service"
TIMER_FILE="/etc/systemd/system/mypaas-update.timer"
PATH_FILE="/etc/systemd/system/mypaas-update.path"
REQUEST_FILE="/run/mypaas/update.request"

EXPLICIT_ENABLED_SET="${AUTO_UPDATE_ENABLED+x}"
EXPLICIT_ENABLED="${AUTO_UPDATE_ENABLED-}"
EXPLICIT_INTERVAL_SET="${AUTO_UPDATE_INTERVAL_MINUTES+x}"
EXPLICIT_INTERVAL="${AUTO_UPDATE_INTERVAL_MINUTES-}"
EXPLICIT_REF_SET="${AUTO_UPDATE_REF+x}"
EXPLICIT_REF="${AUTO_UPDATE_REF-}"
EXPLICIT_WAIT_SET="${AUTO_UPDATE_IMAGE_WAIT_SECONDS+x}"
EXPLICIT_WAIT="${AUTO_UPDATE_IMAGE_WAIT_SECONDS-}"

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
  command_exists sudo || die "sudo is required to configure automatic updates"
  sudo "$@"
}

lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

read_setting() {
  local file="$1"
  local key="$2"
  local line value
  [[ -r "$file" ]] || return 1
  line="$(grep -E "^${key}=" "$file" | tail -n 1 || true)"
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

resolve_setting() {
  local explicit_set="$1"
  local explicit_value="$2"
  local key="$3"
  local fallback="$4"
  local value

  if [[ -n "$explicit_set" ]]; then
    printf '%s' "$explicit_value"
    return
  fi
  if value="$(read_setting "$UPDATE_CONFIG_FILE" "$key" 2>/dev/null)"; then
    printf '%s' "$value"
    return
  fi
  if value="$(read_setting "$ENV_FILE" "$key" 2>/dev/null)"; then
    printf '%s' "$value"
    return
  fi
  printf '%s' "$fallback"
}

validate_ref() {
  local ref="$1"
  [[ "$ref" =~ ^[A-Za-z0-9._/-]+$ ]] || die "AUTO_UPDATE_REF contains unsupported characters"
}

validate_interval() {
  local value="$1"
  [[ "$value" =~ ^[0-9]+$ ]] || die "AUTO_UPDATE_INTERVAL_MINUTES must be an integer"
  (( value >= 5 && value <= 10080 )) || die "AUTO_UPDATE_INTERVAL_MINUTES must be between 5 and 10080"
}

validate_wait() {
  local value="$1"
  [[ "$value" =~ ^[0-9]+$ ]] || die "AUTO_UPDATE_IMAGE_WAIT_SECONDS must be an integer"
  (( value <= 3600 )) || die "AUTO_UPDATE_IMAGE_WAIT_SECONDS must be between 0 and 3600"
}

quote_systemd_value() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '%s' "$value"
}

persist_policy() {
  local enabled="$1"
  local interval="$2"
  local ref="$3"
  local image_wait="$4"

  run_root install -d -m 0755 "$UPDATE_CONFIG_DIR"
  run_root tee "$UPDATE_CONFIG_FILE" >/dev/null <<EOF
# Managed by scripts/configure-auto-update.sh. No secrets are stored here.
AUTO_UPDATE_ENABLED=$enabled
AUTO_UPDATE_INTERVAL_MINUTES=$interval
AUTO_UPDATE_REF=$ref
MYPAAS_REF=$ref
AUTO_UPDATE_IMAGE_WAIT_SECONDS=$image_wait
EOF
  run_root chmod 0644 "$UPDATE_CONFIG_FILE"
}

disable_timer() {
  if ! command_exists systemctl; then
    return
  fi
  run_root systemctl disable --now mypaas-update.timer >/dev/null 2>&1 || true
  run_root rm -f "$TIMER_FILE"
}

main() {
  local enabled interval ref image_wait
  enabled="$(lower "$(resolve_setting "$EXPLICIT_ENABLED_SET" "$EXPLICIT_ENABLED" AUTO_UPDATE_ENABLED false)")"
  interval="$(resolve_setting "$EXPLICIT_INTERVAL_SET" "$EXPLICIT_INTERVAL" AUTO_UPDATE_INTERVAL_MINUTES 30)"
  ref="$(resolve_setting "$EXPLICIT_REF_SET" "$EXPLICIT_REF" AUTO_UPDATE_REF main)"
  image_wait="$(resolve_setting "$EXPLICIT_WAIT_SET" "$EXPLICIT_WAIT" AUTO_UPDATE_IMAGE_WAIT_SECONDS 300)"

  case "$enabled" in
    true|1|yes|on) enabled=true ;;
    false|0|no|off) enabled=false ;;
    *) die "AUTO_UPDATE_ENABLED must be true or false" ;;
  esac
  validate_interval "$interval"
  validate_ref "$ref"
  validate_wait "$image_wait"
  persist_policy "$enabled" "$interval" "$ref" "$image_wait"

  command_exists systemctl || die "MyPaas updates require systemd"

  local root_q env_q config_q
  root_q="$(quote_systemd_value "$ROOT_DIR")"
  env_q="$(quote_systemd_value "$ENV_FILE")"
  config_q="$(quote_systemd_value "$UPDATE_CONFIG_FILE")"

  log "Installing host update service (ref $ref)"

  run_root install -d -m 0755 /run/mypaas

  run_root tee "$SERVICE_FILE" >/dev/null <<EOF
[Unit]
Description=Update MyPaas when a published upstream revision is available
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
WorkingDirectory="$root_q"
EnvironmentFile=-$config_q
Environment="ENV_FILE=$env_q"
Environment="MYPAAS_INSTALL_DIR=$root_q"
ExecStartPre=-/usr/bin/rm -f $REQUEST_FILE
ExecStart=/usr/bin/env bash "$root_q/scripts/update-vm.sh"

[Install]
WantedBy=multi-user.target
EOF

  run_root tee "$PATH_FILE" >/dev/null <<EOF
[Unit]
Description=Watch for MyPaas update requests from the control plane

[Path]
PathExists=$REQUEST_FILE
Unit=mypaas-update.service

[Install]
WantedBy=multi-user.target
EOF

  if [[ "$enabled" == "true" ]]; then
    log "Installing automatic update timer (${interval} minute interval)"

    run_root tee "$TIMER_FILE" >/dev/null <<EOF
[Unit]
Description=Periodically check for MyPaas updates

[Timer]
OnBootSec=10min
OnUnitActiveSec=${interval}min
RandomizedDelaySec=2min
Persistent=true
Unit=mypaas-update.service

[Install]
WantedBy=timers.target
EOF
  else
    disable_timer
  fi

  run_root systemctl daemon-reload
  run_root systemctl enable --now mypaas-update.path >/dev/null
  if [[ "$enabled" == "true" ]]; then
    run_root systemctl enable --now mypaas-update.timer >/dev/null
    log "Automatic updates enabled"
  else
    log "Automatic updates are disabled; dashboard-triggered updates remain available"
  fi

  printf 'Policy: %s\n' "$UPDATE_CONFIG_FILE"
  printf 'Manual trigger: systemctl status mypaas-update.path\n'
  if [[ "$enabled" == "true" ]]; then
    printf 'Check timer: systemctl status mypaas-update.timer\n'
  fi
  printf 'View updater logs: journalctl -u mypaas-update.service\n'
}

main "$@"
