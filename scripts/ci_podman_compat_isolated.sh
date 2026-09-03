#!/usr/bin/env bash
set -euo pipefail

if [[ "${MYPAAS_PODMAN_INNER:-}" == "1" ]]; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y podman catatonit conmon crun docker.io docker-compose-v2 ca-certificates

  mkdir -p /run/podman /run/user/0 /etc/containers /run/containers/storage /var/lib/containers/storage
  chmod 700 /run/user/0

  cat >/etc/containers/storage.conf <<'EOF'
[storage]
driver = "vfs"
runroot = "/run/containers/storage"
graphroot = "/var/lib/containers/storage"
EOF

  cat >/tmp/mypaas-containers.conf <<'EOF'
[engine]
cgroup_manager = "cgroupfs"
conmon_path = ["/usr/bin/conmon"]
runtime = "crun"

[engine.runtimes]
crun = ["/usr/bin/crun"]
EOF

  /usr/bin/podman system service --time=0 unix:///run/podman/podman.sock \
    >/tmp/mypaas-podman-service.log 2>&1 &
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
    echo "isolated Podman socket did not become ready" >&2
    cat /tmp/mypaas-podman-service.log >&2 || true
    exit 1
  fi

  export DOCKER_HOST=unix:///run/podman/podman.sock
  expected="$(/usr/bin/podman version --format '{{.Client.Version}}')"
  actual="$(docker version --format '{{.Server.Version}}')"
  test "$actual" = "$expected"
  test "$(/usr/bin/podman info --format '{{.Host.Conmon.Path}}')" = "/usr/bin/conmon"
  test "$(/usr/bin/podman info --format '{{.Host.OCIRuntime.Path}}')" = "/usr/bin/crun"
  docker version
  docker compose version

  bash scripts/podman_compat_smoke.sh
  exit 0
fi

# GitHub's Ubuntu 24.04 fleet is currently in a staged Podman rollback. Run the
# compatibility qualification inside a clean privileged Noble container so it
# always exercises the distro Podman 4.9.3 stack instead of the runner image's
# temporary mixed 4.x/5.x host installation.
docker run --rm \
  --privileged \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v "$PWD:/workspace" \
  -w /workspace \
  -e MYPAAS_PODMAN_INNER=1 \
  ubuntu:24.04 \
  bash scripts/ci_podman_compat_isolated.sh
