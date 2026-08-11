import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
VERIFY_SCRIPT = ROOT_DIR / "scripts" / "verify-production.sh"


class VerifyProductionTest(unittest.TestCase):
    def test_cloudflared_check_uses_docker_inspect_not_compose_ps_service(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn("inspect --format '{{.State.Running}}' mypaas-cloudflared", content)
        self.assertNotIn("ps cloudflared", content)

    def test_configured_statd_is_verified_on_host_and_inside_api(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('if [[ -n "${STATD_SOCKET:-}" ]]; then', content)
        self.assertIn("systemctl is-active --quiet mypaas-statd", content)
        self.assertIn('[[ ! -S "$STATD_SOCKET" ]]', content)
        self.assertIn('exec -T api test -S "$STATD_SOCKET"', content)
        self.assertIn("Skipping mypaas-statd verification because STATD_SOCKET is empty.", content)

    def test_control_plane_network_isolation_is_verified(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"', content)
        self.assertIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"', content)
        self.assertIn("require_network", content)
        self.assertIn("forbid_network", content)
        self.assertIn(
            "for container in mypaas-api mypaas-dashboard mypaas-caddy-prod mypaas-cloudflared",
            content,
        )
        self.assertIn('require_network mypaas-postgres-prod "$CONTROL_NETWORK"', content)
        self.assertIn('require_network mypaas-postgres-prod "$PROJECT_NETWORK"', content)


if __name__ == "__main__":
    unittest.main()
