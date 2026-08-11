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

# Shared PostgreSQL is intentionally dual-homed; ordinary control services are
# not reachable from the project network.
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

# Final routing model:
# - allocated host ports remain stable runtime lookup keys;
# - Caddy does not hairpin through those published ports;
# - the API resolves the owner container by inspect and uses its stable short
#   container ID as the PROJECT_NETWORK DNS identity;
# - short IDs survive Dockerfile/image container rename during rolling deploys;
# - Caddy Admin remains Unix-only.
mkdir -p "$tmpdir/caddy-run"
route_id="$(docker run -d \
  --name "$route_name" \
  --network "$project_network" \
  -p "127.0.0.1:18080:8080" \
  alpine:3.20 sh -c \
  'mkdir -p /www && printf "mypaas-route-ok\n" >/www/index.html && httpd -f -p 8080 -h /www')"
route_short_id="${route_id:0:12}"
test "${#route_short_id}" -eq 12
docker inspect "$route_name" | grep -q '"HostPort": "18080"'

# Podman/netavark guarantees the short ID as a DNS alias on DNS-enabled custom
# networks. Verify that contract directly instead of trusting compatibility
# inspect IP fields, which can be misleading under the Docker API shim.
docker run --rm --network "$project_network" alpine:3.20 \
  wget -qO- "http://${route_short_id}:8080" | grep -q '^mypaas-route-ok$'

cat > "$tmpdir/Caddyfile" <<EOF
{
  admin unix//run/mypaas/caddy-admin.sock
}
:18081 {
  reverse_proxy ${route_short_id}:8080
}
EOF

# Production Compose attaches Caddy to both external networks before the Caddy
# process starts. Model that lifecycle exactly.
docker create \
  --name "$caddy_name" \
  --network "$control_network" \
  --network-alias caddy-edge \
  -v "$tmpdir/Caddyfile:/etc/caddy/Caddyfile:ro" \
  -v "$tmpdir/caddy-run:/run/mypaas" \
  caddy:2-alpine >/dev/null
docker network connect --alias caddy-edge "$project_network" "$caddy_name"
docker start "$caddy_name" >/dev/null

for _ in $(seq 1 20); do
  if docker exec "$caddy_name" wget -qO- "http://${route_short_id}:8080" 2>/dev/null | grep -q '^mypaas-route-ok$'; then
    caddy_project_reachability=true
    break
  fi
  sleep 0.25
done
if [[ "${caddy_project_reachability:-false}" != "true" ]]; then
  echo "dual-homed Caddy cannot resolve/reach the project runtime short ID" >&2
  docker inspect "$caddy_name" >&2 || true
  exit 1
fi

for _ in $(seq 1 30); do
  if [[ -S "$tmpdir/caddy-run/caddy-admin.sock" ]] && \
    docker run --rm --network "$control_network" alpine:3.20 \
      wget -qO- http://caddy-edge:18081 2>/dev/null | grep -q '^mypaas-route-ok$'; then
    caddy_ready=true
    break
  fi
  sleep 0.25
done
if [[ "${caddy_ready:-false}" != "true" ]]; then
  echo "dual-homed Caddy could not route to the project runtime DNS identity" >&2
  docker logs "$caddy_name" >&2 || true
  exit 1
fi

if docker run --rm --network "$project_network" alpine:3.20 \
  wget -qO- http://caddy-edge:2019 >/dev/null 2>&1; then
  echo "Caddy Admin unexpectedly accepted TCP :2019 from the project network" >&2
  exit 1
fi

# Compose contract: the main service keeps a MyPaaS-managed published port and
# joins the external project network. The resolver can select it by HostPort and
# use the short container ID as a stable network DNS identity.
cat > "$tmpdir/compose.yml" <<EOF
services:
  app:
    image: alpine:3.20
    command: ["sh", "-c", "mkdir -p /www && printf 'compose-route-ok\\n' >/www/index.html && httpd -f -p 8080 -h /www"]
    ports:
      - "127.0.0.1:18082:8080"
    networks:
      - default
      - mypaas_platform
networks:
  mypaas_platform:
    external: true
    name: "$project_network"
EOF

docker compose -p "$compose_project" -f "$tmpdir/compose.yml" up -d >/dev/null
compose_ids="$(docker ps -aq --filter "label=com.docker.compose.project=$compose_project")"
test -n "$compose_ids"
# shellcheck disable=SC2086
docker inspect $compose_ids >/dev/null
compose_id="$(printf '%s\n' "$compose_ids" | head -n1)"
compose_short_id="${compose_id:0:12}"
test "${#compose_short_id}" -eq 12
docker inspect "$compose_id" | grep -q '"HostPort": "18082"'
docker run --rm --network "$project_network" alpine:3.20 \
  wget -qO- "http://${compose_short_id}:8080" | grep -q '^compose-route-ok$'

docker compose -p "$compose_project" -f "$tmpdir/compose.yml" stop >/dev/null
docker compose -p "$compose_project" -f "$tmpdir/compose.yml" start >/dev/null

echo "Podman Docker-compatibility smoke passed"
