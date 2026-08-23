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

    def test_three_network_boundaries_are_verified(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"', content)
        self.assertIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"', content)
        self.assertIn('ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"', content)
        self.assertIn("CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct.", content)
        self.assertIn("for container in mypaas-api mypaas-dashboard", content)
        self.assertIn('require_network mypaas-postgres-prod "$PROJECT_NETWORK"', content)
        self.assertIn('forbid_network mypaas-postgres-prod "$ROUTING_NETWORK"', content)
        self.assertIn('require_network mypaas-caddy-prod "$ROUTING_NETWORK"', content)
        self.assertIn('forbid_network mypaas-caddy-prod "$PROJECT_NETWORK"', content)
        self.assertIn('require_network mypaas-cloudflared "$CONTROL_NETWORK"', content)
        self.assertIn('forbid_network mypaas-cloudflared "$ROUTING_NETWORK"', content)

    def test_delivery_mode_controls_tunnel_verification(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('PUBLIC_DELIVERY_MODE="${PUBLIC_DELIVERY_MODE:-tunnel}"', content)
        self.assertIn('if [[ "$PUBLIC_DELIVERY_MODE" == "tunnel" ]]; then', content)
        self.assertIn('cloudflare-origin|direct)', content)
        self.assertIn('Caddy HTTPS port 443 is not published.', content)
        self.assertIn('must not be running when PUBLIC_DELIVERY_MODE=', content)

    def test_caddy_admin_is_verified_only_through_unix_socket(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('CADDY_ADMIN_SOCKET="${CADDY_ADMIN_SOCKET:-/run/mypaas/caddy-admin.sock}"', content)
        self.assertIn('curl -fsS --unix-socket "$CADDY_ADMIN_SOCKET"', content)
        self.assertIn('exec -T api test -S "$CADDY_ADMIN_SOCKET"', content)
        self.assertIn('port mypaas-caddy-prod 2019/tcp', content)
        self.assertNotIn("http://127.0.0.1:2019", content)


if __name__ == "__main__":
    unittest.main()
