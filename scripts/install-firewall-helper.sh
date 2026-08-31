#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SUDO=""
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
fi

if ! command -v python3 >/dev/null 2>&1; then
  command -v apt-get >/dev/null 2>&1 || { echo "python3 is required for the MyPaaS firewall helper" >&2; exit 1; }
  $SUDO apt-get update
  $SUDO apt-get install -y python3
fi

if ! command -v ufw >/dev/null 2>&1 && [[ ! -x /usr/sbin/ufw ]]; then
  command -v apt-get >/dev/null 2>&1 || { echo "ufw is required for the MyPaaS firewall helper" >&2; exit 1; }
  $SUDO apt-get update
  $SUDO apt-get install -y ufw
fi

$SUDO install -d -m 0755 /usr/local/lib/mypaas
$SUDO install -m 0755 "$ROOT_DIR/scripts/firewall-helper.py" /usr/local/lib/mypaas/firewall-helper.py
$SUDO install -m 0644 "$ROOT_DIR/scripts/mypaas-firewall.service" /etc/systemd/system/mypaas-firewall.service
$SUDO systemctl daemon-reload
$SUDO systemctl enable mypaas-firewall.service >/dev/null
$SUDO systemctl restart mypaas-firewall.service

for _ in {1..30}; do
  [[ -S /run/mypaas/firewall.sock ]] && break
  sleep 0.1
done

if [[ ! -S /run/mypaas/firewall.sock ]]; then
  $SUDO systemctl status mypaas-firewall.service --no-pager >&2 || true
  echo "mypaas-firewall started without creating /run/mypaas/firewall.sock" >&2
  exit 1
fi

echo "MyPaaS firewall helper ready. UFW is not enabled or disabled by this installer."
