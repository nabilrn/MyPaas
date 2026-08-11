#!/usr/bin/env bash
set -euo pipefail

project_network="${PROJECT_NETWORK:-mypaas-ci-projects}"
control_network="${CONTROL_NETWORK:-mypaas-ci-control}"
container_name="mypaas-ci-runtime-smoke"
shared_name="mypaas-ci-shared-smoke"
admin_name="mypaas-ci-admin-smoke"
route_name="mypaas-ci-route-smoke"
caddy_name="mypaas-ci-caddy-smoke"
compose_project="mypaas-ci-compose-smoke"
tmpdir="$(mktemp -d)"

cleanup() {
  docker rm -f "$container_name" "$shared_name" "$admin_name" "$route_name" "$caddy_name" >/dev/null 2>&1 || true
  docker compose -p "$compose_project" -f "$tmpdir/compose.yml" down -v --remove-orphans >/dev/null 2>&1 || true
  docker network rm "$project_network" >/dev/null 2>&1 || true
  docker network rm "$control_network" >/dev/null 2>&1 || true
  rm -rf "$tmpdir"
}
trap cleanup EXIT

docker version >/dev/null

docker network create "$project_network" >/dev/null
docker network create "$control_network" >/dev/null
project_gateway="$(docker network inspect "$project_network" --format '{{(index .IPAM.Config 0).Gateway}}')"
test -n "$project_gateway"

# Runtime command contract used by MyPaaS and the bounded static builder.
docker run -d \
  --name "$container_name" \
  --network "$project_network" \
  --label mypaas.smoke=true \
  --memory 128m \
  --cpus 0.50 \
  --pids-limit 64 \
  --restart unless-stopped \
  alpine:3.20 sleep 300 >/dev/null

docker ps -q --filter label=mypaas.smoke=true | grep -q .
docker inspect "$container_name" >/dev/null
docker stop --time 5 "$container_name" >/dev/null
docker start "$container_name" >/dev/null
docker restart "$container_name" >/dev/null

# Model shared PostgreSQL dual-homing while control-only services stay off the
# project network.
docker run -d \
  --name "$shared_name" \
  --network "$control_network" \
  alpine:3.20 sleep 300 >/dev/null
docker network connect --alias shared-db "$project_network" "$shared_name"

docker run -d \
  --name "$admin_name" \
  --network "$control_network" \
  --network-alias control-admin \
  alpine:3.20 sleep 300 >/dev/null

docker run --rm --network "$project_network" alpine:3.20 \
  ping -c 1 -W 1 shared-db >/dev/null
if docker run --rm --network "$project_network" alpine:3.20 \
  ping -c 1 -W 1 control-admin >/dev/null 2>&1; then
  echo "project network unexpectedly reached a control-only service" >&2
  exit 1
fi

# Exercise the final production routing model with real Caddy:
# - app lives on the project network and publishes on the project gateway;
# - Caddy is dual-homed for control-plane + data-plane traffic;
# - Caddy Admin is Unix-only, never TCP :2019.
mkdir -p "$tmpdir/caddy-run"
docker run -d \
  --name "$route_name" \
  --network "$project_network" \
  -p "${project_gateway}:18080:8080" \
  alpine:3.20 sh -c \
  'mkdir -p /www && printf "mypaas-route-ok\n" >/www/index.html && httpd -f -p 8080 -h /www' >/dev/null

cat > "$tmpdir/Caddyfile" <<EOF
{
  admin unix//run/mypaas/caddy-admin.sock
}
:8081 {
  reverse_proxy ${project_gateway}:18080
}
EOF

docker run -d \
  --name "$caddy_name" \
  --network "$control_network" \
  --network-alias caddy-data \
  -v "$tmpdir/Caddyfile:/etc/caddy/Caddyfile:ro" \
  -v "$tmpdir/caddy-run:/run/mypaas" \
  caddy:2-alpine >/dev/null
docker network connect --alias caddy-data "$project_network" "$caddy_name"

for _ in $(seq 1 30); do
  if [[ -S "$tmpdir/caddy-run/caddy-admin.sock" ]] && \
    docker run --rm --network "$project_network" alpine:3.20 \
      wget -qO- http://caddy-data:8081 2>/dev/null | grep -q '^mypaas-route-ok$'; then
    caddy_ready=true
    break
  fi
  sleep 0.25
done
if [[ "${caddy_ready:-false}" != "true" ]]; then
  echo "Caddy could not route from the project data plane to the managed project-gateway port" >&2
  docker logs "$caddy_name" >&2 || true
  exit 1
fi
if docker run --rm --network "$project_network" alpine:3.20 \
  wget -qO- http://caddy-data:2019 >/dev/null 2>&1; then
  echo "Caddy Admin unexpectedly accepted TCP :2019 from the project network" >&2
  exit 1
fi

cat > "$tmpdir/compose.yml" <<'EOF'
services:
  app:
    image: alpine:3.20
    command: ["sleep", "300"]
EOF

docker compose -p "$compose_project" -f "$tmpdir/compose.yml" up -d >/dev/null
compose_ids="$(docker ps -aq --filter "label=com.docker.compose.project=$compose_project")"
test -n "$compose_ids"
# MyPaaS uses batched inspect for Compose runtime discovery.
# shellcheck disable=SC2086
docker inspect $compose_ids >/dev/null

docker compose -p "$compose_project" -f "$tmpdir/compose.yml" stop >/dev/null
docker compose -p "$compose_project" -f "$tmpdir/compose.yml" start >/dev/null

echo "Podman Docker-compatibility smoke passed"
