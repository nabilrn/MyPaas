#!/usr/bin/env bash
set -euo pipefail

sudo apt-get update
# ubuntu-24.04 hosted runners are currently rolling back from a bundled Podman
# build to the distro build. Some runner revisions have Podman preinstalled
# without the CNI plugins and can retain incompatible rootful storage state.
# The runner is disposable, so make this smoke deterministic by explicitly
# installing the distro networking plugins and starting from a clean store.
sudo apt-get install -y podman catatonit conmon crun containernetworking-plugins
sudo systemctl stop docker.service docker.socket podman.service podman.socket || true
sudo pkill -9 -x podman || true
sudo rm -rf /var/lib/containers/storage /run/containers/storage
sudo mkdir -p /var/lib/containers/storage
sudo mkdir -p /run/podman /run/user/0
sudo chmod 700 /run/user/0

cat >/tmp/mypaas-containers.conf <<'EOF'
[engine]
conmon_path = ["/usr/bin/conmon"]
runtime = "crun"

[engine.runtimes]
crun = ["/usr/bin/crun"]
EOF

ready=false
for attempt in 1 2 3; do
  echo "Starting rootful Podman API service (attempt ${attempt}/3)"
  sudo pkill -f '^/usr/bin/podman system service --time=0 unix:///run/podman/podman.sock$' || true
  sudo rm -f /run/podman/podman.sock /var/run/docker.sock
  sudo rm -f /tmp/mypaas-podman-service.log

  sudo sh -c 'export PATH=/usr/bin:/usr/sbin:/bin:/sbin HOME=/root XDG_RUNTIME_DIR=/run/user/0 CONTAINERS_CONF_OVERRIDE=/tmp/mypaas-containers.conf; nohup /usr/bin/podman system service --time=0 unix:///run/podman/podman.sock >/tmp/mypaas-podman-service.log 2>&1 &'

  for _ in $(seq 1 120); do
    if [[ -S /run/podman/podman.sock ]]; then
      ready=true
      break
    fi
    sleep 0.25
  done

  if [[ "$ready" == "true" ]]; then
    break
  fi

  echo "Podman socket did not become ready on attempt ${attempt}" >&2
  cat /tmp/mypaas-podman-service.log >&2 || true
  ps aux | grep '[p]odman system service' >&2 || true
done

if [[ "$ready" != "true" ]]; then
  echo "Rootful Podman socket failed to become ready after 3 attempts" >&2
  cat /tmp/mypaas-podman-service.log >&2 || true
  exit 1
fi

sudo ln -sfn /run/podman/podman.sock /var/run/docker.sock
