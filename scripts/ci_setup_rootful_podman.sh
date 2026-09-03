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

# Some hosted-runner revisions still carry stale Podman 5.8.4 systemd units
# under /usr/local even after the distro 4.9.3 package is installed. Remove
# those overrides so systemd resolves the units shipped by the apt package.
sudo rm -f /usr/local/lib/systemd/system/podman.service \
  /usr/local/lib/systemd/system/podman.socket
sudo systemctl daemon-reload

# Let systemd own creation of /run/podman/podman.sock. This avoids the runner
# transition failure where a manually launched `podman system service` stays
# alive but never binds the socket.
sudo systemctl start podman.socket

socket_unit="$(systemctl show -p FragmentPath --value podman.socket)"
service_unit="$(systemctl show -p FragmentPath --value podman.service)"
if [[ "$socket_unit" != "/usr/lib/systemd/system/podman.socket" || "$service_unit" != "/usr/lib/systemd/system/podman.service" ]]; then
  echo "Podman systemd units are not the distro package units" >&2
  echo "socket unit: $socket_unit" >&2
  echo "service unit: $service_unit" >&2
  exit 1
fi

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
