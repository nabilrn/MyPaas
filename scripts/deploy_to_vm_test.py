import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
DEPLOY_SCRIPT = ROOT_DIR / "scripts" / "deploy-to-vm.sh"
PROD_COMPOSE = ROOT_DIR / "docker-compose.prod.yml"


class DeployToVmTest(unittest.TestCase):
    def test_provisions_distinct_control_project_and_routing_networks(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"', content)
        self.assertIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"', content)
        self.assertIn('ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"', content)
        self.assertIn(
            'for network in "$CONTROL_NETWORK" "$PROJECT_NETWORK" "$ROUTING_NETWORK"',
            content,
        )
        self.assertIn(
            "CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct.",
            content,
        )

    def test_migration_container_uses_control_network(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('--network "$CONTROL_NETWORK"', content)
        self.assertNotIn('--network "$PROJECT_NETWORK"', content)
        self.assertNotIn('--network "$ROUTING_NETWORK"', content)
        self.assertNotIn("--network mypaas-prod", content)

    def test_migration_container_has_explicit_pid_budget(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('MIGRATION_PIDS_LIMIT="${MIGRATION_PIDS_LIMIT:-256}"', content)
        self.assertIn('--pids-limit "$MIGRATION_PIDS_LIMIT"', content)
        self.assertIn("MIGRATION_PIDS_LIMIT must be a positive integer.", content)

    def test_control_plane_recreation_is_serialized(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-1}"', content)
        self.assertIn("COMPOSE_PARALLEL_LIMIT must be a positive integer.", content)
        self.assertIn("export COMPOSE_PARALLEL_LIMIT", content)
        dashboard = content.index('echo "Starting dashboard..."')
        caddy = content.index(
            'echo "Reloading caddy configuration from the current checkout without restarting the proxy..."'
        )
        api = content.index('echo "Starting api..."')
        cloudflared = content.index('echo "Starting cloudflared..."')
        self.assertLess(dashboard, caddy)
        self.assertLess(caddy, api)
        self.assertLess(api, cloudflared)
        self.assertIn('up -d --no-deps dashboard', content)
        self.assertIn('up -d --no-deps api', content)
        self.assertIn('up -d --no-deps cloudflared', content)
        self.assertIn('up -d --force-recreate --no-deps api', content)

    def test_caddy_config_is_hot_reloaded_from_current_checkout_before_api_reconciliation(self) -> None:
        script = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('if managed_container_running mypaas-caddy-prod; then', script)
        self.assertIn("sh -c 'cat > /tmp/mypaas-Caddyfile.next'", script)
        self.assertIn('< "$ROOT_DIR/Caddyfile.prod"', script)
        self.assertIn(
            'caddy reload --config /tmp/mypaas-Caddyfile.next --adapter caddyfile',
            script,
        )
        self.assertIn('rm -f /tmp/mypaas-Caddyfile.next', script)
        self.assertNotIn(
            'caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile',
            script,
        )
        self.assertIn('--address unix//run/mypaas/caddy-admin.sock', script)
        self.assertNotIn("for service in caddy api dashboard cloudflared; do", script)

    def test_deploy_preflights_control_plane_port_ownership(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn("preflight_control_plane_ports", content)
        self.assertIn("preflight_managed_port 5432 mypaas-postgres-prod", content)
        self.assertIn("preflight_managed_port 8080 mypaas-api", content)
        self.assertIn("preflight_managed_port 3000 mypaas-dashboard", content)
        self.assertIn("preflight_managed_port 80 mypaas-caddy-prod", content)
        self.assertIn("stale container proxy", content)
        self.assertIn("another Docker-compatible engine", content)

    def test_failure_advice_does_not_recommend_destructive_uninstall(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertNotIn("To clean up the failed deployment and start fresh", content)
        self.assertIn("was not intentionally removed", content)
        self.assertIn("Do not run scripts/uninstall-vm.sh unless", content)

    def test_deploy_skips_redundant_migrations_for_unchanged_runtime(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('SKIP_MIGRATIONS="${MYPAAS_SKIP_MIGRATIONS:-auto}"', content)
        self.assertIn("current_runtime_build_sha()", content)
        self.assertIn("MYPAAS_BUILD_SHA=", content)
        self.assertIn('diff --quiet "$runtime_sha" HEAD -- backend/migrations', content)
        self.assertIn("if should_skip_migrations; then", content)
        self.assertIn("Skipping migrations: control-plane migration tree is unchanged", content)

    def test_runtime_rollback_does_not_rerun_up_migrations(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn(
            'if [[ "$SKIP_IMAGE_PULL" == "true" && "${MYPAAS_IMAGE_TAG:-}" == rollback-* ]]; then',
            content,
        )
        self.assertIn("Running an older", content)
        self.assertIn("cannot downgrade a schema", content)

    def test_database_restore_forces_migration_path(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        restore_guard = (
            'if [[ "$RESTORED_CONTROL_PLANE_DB" == "true" ]]; then\n'
            "    return 1\n"
            "  fi"
        )
        self.assertIn(restore_guard, content)

    def test_deploy_does_not_rewrite_managed_app_bind_host(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertNotIn('export DOCKER_BIND_HOST="$control_gateway"', content)
        self.assertNotIn('export CADDY_UPSTREAM_HOST="$DOCKER_BIND_HOST"', content)

    def test_deploy_recovers_live_podman_socket(self) -> None:
        script = DEPLOY_SCRIPT.read_text(encoding="utf-8")
        compose = PROD_COMPOSE.read_text(encoding="utf-8")

        self.assertIn('if [[ -S /run/podman/podman.sock ]]; then', script)
        self.assertIn('DOCKER_HOST="unix://$DOCKER_SOCKET"', script)
        self.assertIn('export DOCKER_SOCKET DOCKER_HOST', script)
        self.assertIn('${DOCKER_SOCKET}:/var/run/docker.sock', compose)
        self.assertIn('DOCKER_SOCKET: /var/run/docker.sock', compose)

    def test_deploy_uses_checkout_sha_image_tag_by_default(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('MYPAAS_IMAGE_TAG="$(git -c safe.directory="$ROOT_DIR" rev-parse HEAD)"', content)
        self.assertIn('$DOCKER_BIN pull "$API_IMAGE_REPO:$MYPAAS_IMAGE_TAG"', content)
        self.assertIn('$DOCKER_BIN pull "$DASHBOARD_IMAGE_REPO:$MYPAAS_IMAGE_TAG"', content)
        self.assertIn("Wait for the Docker publish workflow to finish", content)

    def test_database_restore_recreates_api_for_runtime_reconciliation(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn("RESTORED_CONTROL_PLANE_DB=false", content)
        self.assertIn("RESTORED_CONTROL_PLANE_DB=true", content)
        self.assertIn('if [[ "$RESTORED_CONTROL_PLANE_DB" == "true" ]]; then', content)
        self.assertIn('up -d --force-recreate --no-deps api', content)


if __name__ == "__main__":
    unittest.main()
