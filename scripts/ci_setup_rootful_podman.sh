#!/usr/bin/env bash
set -euo pipefail

sudo apt-get update
sudo apt-get install -y podman catatonit conmon crun
sudo systemctl stop docker.service docker.socket podman.service podman.socket || true
sudo mkdir -p /run/podman /run/user/0
sudo chmod 700 /run/user/0
sudo rm -f /run/podman/podman.sock /var/run/docker.sock

cat >/tmp/mypaas-containers.conf <<'EOF'
[engine]
conmon_path = ["/usr/bin/conmon"]
runtime = "crun"

[engine.runtimes]
crun = ["/usr/bin/crun"]
EOF

# GitHub's staged Ubuntu 24.04 Podman rollback can leave bundled Podman 5.x
# systemd units and helper binaries under /usr/local while APT installs Podman
# 4.9.3 under /usr. Force the distro units and helper paths so the smoke tests
# one coherent Podman installation instead of a mixed runner image.
sudo rm -f /etc/systemd/system/podman.service /etc/systemd/system/podman.socket
sudo ln -s /usr/lib/systemd/system/podman.service /etc/systemd/system/podman.service
sudo ln -s /usr/lib/systemd/system/podman.socket /etc/systemd/system/podman.socket
sudo mkdir -p /etc/systemd/system/podman.service.d
sudo tee /etc/systemd/system/podman.service.d/mypaas-ci.conf >/dev/null <<'EOF'
[Service]
Environment="PATH=/usr/bin:/usr/sbin:/bin:/sbin"
Environment="HOME=/root"
Environment="XDG_RUNTIME_DIR=/run/user/0"
Environment="CONTAINERS_CONF_OVERRIDE=/tmp/mypaas-containers.conf"
EOF

sudo systemctl daemon-reload
sudo systemctl start podman.socket

ready=false
for _ in $(seq 1 40); do
  if [[ -S /run/podman/podman.sock ]]; then
    ready=true
    break
  fi
  sleep 0.25
done

if [[ "$ready" != "true" ]]; then
  echo "Rootful Podman socket did not become ready" >&2
  sudo systemctl status podman.socket podman.service --no-pager >&2 || true
  sudo journalctl -u podman.socket -u podman.service --no-pager -n 100 >&2 || true
  exit 1
fi

sudo ln -sfn /run/podman/podman.sock /var/run/docker.sock
