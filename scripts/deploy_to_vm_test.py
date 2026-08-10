import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
DEPLOY_SCRIPT = ROOT_DIR / "scripts" / "deploy-to-vm.sh"


class DeployToVmTest(unittest.TestCase):
    def test_migration_container_uses_configured_project_network(self) -> None:
        content = DEPLOY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('--network "${PROJECT_NETWORK:-mypaas-prod}"', content)
        self.assertNotIn("--network mypaas-prod", content)


if __name__ == "__main__":
    unittest.main()
