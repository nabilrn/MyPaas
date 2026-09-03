#!/usr/bin/env bash
set -euo pipefail

sudo systemctl stop docker.service docker.socket podman.service podman.socket || true
sudo pkill -9 -x podman || true
sudo mkdir -p /run/podman /run/user/0
sudo chmod 700 /run/user/0
sudo rm -f /run/podman/podman.sock /var/run/docker.sock

# During GitHub's staged Ubuntu 24.04 Podman rollback, some runners still carry
# the bundled Podman 5.x toolchain under /usr/local while others already expose
# the distro Podman 4.9.3 toolchain under /usr. Never mix the two installations.
if [[ -x /usr/local/bin/podman && -x /usr/local/bin/conmon && -x /usr/local/bin/crun ]]; then
  podman_bin=/usr/local/bin/podman
  conmon_bin=/usr/local/bin/conmon
  crun_bin=/usr/local/bin/crun
else
  sudo apt-get update
  sudo apt-get install -y podman catatonit conmon crun
  podman_bin=/usr/bin/podman
  conmon_bin=/usr/bin/conmon
  crun_bin=/usr/bin/crun
fi

cat >/tmp/mypaas-containers.conf <<EOF
[engine]
conmon_path = ["$conmon_bin"]
runtime = "crun"

[engine.runtimes]
crun = ["$crun_bin"]
EOF

printf '%s\n' "$podman_bin" >/tmp/mypaas-podman-bin
printf '%s\n' "$conmon_bin" >/tmp/mypaas-conmon-bin
printf '%s\n' "$crun_bin" >/tmp/mypaas-crun-bin

service_log=/tmp/mypaas-podman-service.log
sudo rm -f "$service_log"
sudo env \
  PATH="$(dirname "$podman_bin"):$(dirname "$conmon_bin"):$(dirname "$crun_bin"):/usr/bin:/usr/sbin:/bin:/sbin" \
  HOME=/root \
  XDG_RUNTIME_DIR=/run/user/0 \
  CONTAINERS_CONF_OVERRIDE=/tmp/mypaas-containers.conf \
  "$podman_bin" system service --time=0 unix:///run/podman/podman.sock \
  >"$service_log" 2>&1 &
service_pid=$!

ready=false
for _ in $(seq 1 120); do
  if [[ -S /run/podman/podman.sock ]]; then
    ready=true
    break
  fi
  if ! kill -0 "$service_pid" 2>/dev/null; then
    break
  fi
  sleep 0.25
done

if [[ "$ready" != "true" ]]; then
  echo "Rootful Podman socket did not become ready" >&2
  cat "$service_log" >&2 || true
  ps aux | grep '[p]odman system service' >&2 || true
  exit 1
fi

sudo ln -sfn /run/podman/podman.sock /var/run/docker.sock
