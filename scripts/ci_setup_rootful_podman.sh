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

# Use the distro package's native socket activation instead of launching
# `podman system service` in the background. During GitHub's Ubuntu 24.04
# Podman rollback, the service process can stay alive without binding its
# socket, while systemd can create the listening socket deterministically.
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
