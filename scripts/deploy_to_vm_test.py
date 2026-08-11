import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
DEPLOY_SCRIPT = ROOT_DIR / "scripts" / "deploy-to-vm.sh"


class DeployToVmTest(unittest.TestCase):
    def test_provisions_separate_control_and_project_networks(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"', content)
        self.assertIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"', content)
        self.assertIn('network inspect "$CONTROL_NETWORK"', content)
        self.assertIn('network inspect "$PROJECT_NETWORK"', content)

    def test_migration_container_uses_control_network(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('--network "$CONTROL_NETWORK"', content)
        self.assertNotIn('--network "$PROJECT_NETWORK"', content)
        self.assertNotIn("--network mypaas-prod", content)

    def test_legacy_project_gateway_bind_is_overridden_at_runtime(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('control_gateway="$(network_gateway "$CONTROL_NETWORK")"', content)
        self.assertIn('project_gateway="$(network_gateway "$PROJECT_NETWORK")"', content)
        self.assertIn('"${DOCKER_BIND_HOST:-}" == "$project_gateway"', content)
        self.assertIn('export DOCKER_BIND_HOST="$control_gateway"', content)
        self.assertIn('export CADDY_UPSTREAM_HOST="$DOCKER_BIND_HOST"', content)


if __name__ == "__main__":
    unittest.main()
