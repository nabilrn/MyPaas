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


if __name__ == "__main__":
    unittest.main()
