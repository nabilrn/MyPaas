import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
VERIFY_SCRIPT = ROOT_DIR / "scripts" / "verify-production.sh"


class VerifyProductionTest(unittest.TestCase):
    def test_cloudflared_check_uses_docker_inspect_not_compose_ps_service(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn("inspect --format '{{.State.Running}}' mypaas-cloudflared", content)
        self.assertNotIn("ps cloudflared", content)


if __name__ == "__main__":
    unittest.main()
