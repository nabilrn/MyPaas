#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
REMOVE_STATD="${REMOVE_STATD:-true}"
STATD_DIR="${STATD_DIR:-/opt/mypaas-statd}"

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

sudo_cmd() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

echo "WARNING: This will completely destroy the MyPaas installation on this VM."
echo "All projects, databases, configurations, and backups will be PERMANENTLY DELETED."
read -r -p "Are you sure you want to continue? Type 'DESTROY' to confirm: " confirm

if [[ "$confirm" != "DESTROY" ]]; then
  echo "Aborted."
  exit 0
fi

cd "$ROOT_DIR"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"
PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"
ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"

docker_prefix() {
  if docker ps >/dev/null 2>&1; then
    printf 'docker'
    return
  fi
  if command -v sudo >/dev/null 2>&1 && sudo docker ps >/dev/null 2>&1; then
    printf 'sudo docker'
    return
  fi
  die "current user cannot access Docker."
}

DOCKER_BIN="$(docker_prefix)"
COMPOSE_BIN="$DOCKER_BIN compose"

log "Stopping and removing MyPaas core services and images..."
if [[ -f "$COMPOSE_FILE" ]]; then
  if [[ -f "$ENV_FILE" ]]; then
    $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" down -v --rmi all --remove-orphans || true
  else
    $COMPOSE_BIN -f "$COMPOSE_FILE" down -v --rmi all --remove-orphans || true
  fi
fi

remove_owned_network() {
  local network="$1"
  if ! $DOCKER_BIN network inspect "$network" >/dev/null 2>&1; then
    return
  fi

  local containers
  containers="$($DOCKER_BIN ps -aq --filter "network=$network" | xargs)"
  if [[ -n "$containers" ]]; then
    log "Removing containers attached to $network: $containers"
    # shellcheck disable=SC2086
    $DOCKER_BIN rm -f $containers || true
  fi

  log "Removing Docker-compatible network $network..."
  $DOCKER_BIN network rm "$network" || true
}

log "Removing MyPaas workload and platform networks..."
for network in "$ROUTING_NETWORK" "$PROJECT_NETWORK" "$CONTROL_NETWORK"; do
  remove_owned_network "$network"
done

if [[ -f /usr/local/lib/mypaas/firewall-helper.py ]]; then
  log "Removing MyPaaS-managed UFW rules..."
  sudo_cmd python3 - <<'PY' || true
import importlib.util

path = "/usr/local/lib/mypaas/firewall-helper.py"
spec = importlib.util.spec_from_file_location("mypaas_firewall_helper", path)
if spec is None or spec.loader is None:
    raise RuntimeError("cannot load MyPaaS firewall helper")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
for rule in list(module.current_managed_rules()):
    module.delete_rule(int(rule["port"]), str(rule["protocol"]))
PY
fi

log "Removing MyPaaS firewall helper..."
if command -v systemctl >/dev/null 2>&1; then
  sudo_cmd systemctl disable --now mypaas-firewall.service >/dev/null 2>&1 || true
fi
sudo_cmd rm -f /etc/systemd/system/mypaas-firewall.service
sudo_cmd rm -f /usr/local/lib/mypaas/firewall-helper.py
sudo_cmd rm -f /run/mypaas/firewall.sock
if command -v systemctl >/dev/null 2>&1; then
  sudo_cmd systemctl daemon-reload >/dev/null 2>&1 || true
fi

if [[ "$REMOVE_STATD" == "true" ]]; then
  log "Removing mypaas-statd host service..."
  if command -v systemctl >/dev/null 2>&1; then
    sudo_cmd systemctl disable --now mypaas-statd >/dev/null 2>&1 || true
  fi
  sudo_cmd rm -f /etc/systemd/system/mypaas-statd.service
  sudo_cmd rm -f /usr/local/bin/mypaas-statd
  if command -v systemctl >/dev/null 2>&1; then
    sudo_cmd systemctl daemon-reload >/dev/null 2>&1 || true
  fi
  if [[ -d "$STATD_DIR" ]]; then
    sudo_cmd rm -rf "$STATD_DIR"
  fi
  sudo_cmd rm -rf /run/mypaas
else
  log "Keeping mypaas-statd because REMOVE_STATD=false"
fi

log "Removing host directories..."
for dir in \
  /var/lib/mypaas \
  /tmp/mypaas/builds
do
  if [[ -d "$dir" ]]; then
    sudo_cmd rm -rf "$dir"
  fi
done

log "Removing .env file..."
if [[ -f "$ENV_FILE" ]]; then
  rm -f "$ENV_FILE"
fi

log "Uninstall complete. MyPaas has been totally destroyed."

cd /
if [[ -d "$ROOT_DIR" && "$ROOT_DIR" != "/" ]]; then
  echo "==> Removing MyPaas source folder ($ROOT_DIR)..."
  sudo_cmd rm -rf "$ROOT_DIR"
fi
