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
        self.assertIn('up -d --force-recreate api', content)


if __name__ == "__main__":
    unittest.main()
