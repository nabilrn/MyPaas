import subprocess
import sys
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
VERIFY_SCRIPT = ROOT_DIR / "scripts" / "verify-production.sh"
COMPOSE_FILE = ROOT_DIR / "docker-compose.prod.yml"
ASSET_EXTRACTOR = ROOT_DIR / "scripts" / "extract_dashboard_assets.py"
CADDY_FILE = ROOT_DIR / "Caddyfile.prod"
SVELTE_CONFIG = ROOT_DIR / "frontend" / "svelte.config.js"
ROOT_LAYOUT = ROOT_DIR / "frontend" / "src" / "routes" / "+layout.svelte"

# Keep the control-plane ingress contract coupled to production Compose aliases.


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
        self.assertIn(
            "for container in mypaas-api mypaas-dashboard mypaas-cloudflared",
            content,
        )
        self.assertIn('require_network mypaas-postgres-prod "$PROJECT_NETWORK"', content)
        self.assertIn('forbid_network mypaas-postgres-prod "$ROUTING_NETWORK"', content)
        self.assertIn('require_network mypaas-caddy-prod "$ROUTING_NETWORK"', content)
        self.assertIn('forbid_network mypaas-caddy-prod "$PROJECT_NETWORK"', content)

    def test_control_plane_service_aliases_are_explicit_and_caddy_ingress_is_verified(self) -> None:
        compose = COMPOSE_FILE.read_text(encoding="utf-8")
        verify = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn("      control:\n        aliases:\n          - api\n", compose)
        self.assertIn("      control:\n        aliases:\n          - dashboard\n", compose)
        self.assertIn("      control:\n        aliases:\n          - caddy\n", compose)
        self.assertIn("      control:\n        aliases:\n          - cloudflared\n", compose)
        self.assertIn("Checking API ingress through local Caddy", verify)
        self.assertIn('-H "Host: $PUBLIC_DOMAIN" http://127.0.0.1/api/health', verify)
        self.assertIn('-H "Host: $PUBLIC_DOMAIN" http://127.0.0.1/api/ready', verify)

    def test_caddy_admin_is_verified_only_through_unix_socket(self) -> None:
        content = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('CADDY_ADMIN_SOCKET="${CADDY_ADMIN_SOCKET:-/run/mypaas/caddy-admin.sock}"', content)
        self.assertIn('curl -fsS --unix-socket "$CADDY_ADMIN_SOCKET"', content)
        self.assertIn('exec -T api test -S "$CADDY_ADMIN_SOCKET"', content)
        self.assertIn('port mypaas-caddy-prod 2019/tcp', content)
        self.assertNotIn("http://127.0.0.1:2019", content)

    def test_dashboard_asset_parser_accepts_link_and_inline_import_references(self) -> None:
        html = """
        <link rel="stylesheet" href="/_app/immutable/assets/0.ABC123.css">
        <link rel="icon" href="data:image/svg+xml,%3Csvg%3E">
        <script>Promise.all([import("/_app/immutable/entry/start.REVZG4IO.js"), import("/_app/immutable/entry/app.D7lxbEp-.js")])</script>
        """
        result = subprocess.run(
            [sys.executable, str(ASSET_EXTRACTOR)],
            input=html,
            text=True,
            capture_output=True,
            check=True,
        )

        self.assertEqual(
            result.stdout.splitlines(),
            [
                "/_app/immutable/assets/0.ABC123.css",
                "/_app/immutable/entry/app.D7lxbEp-.js",
                "/_app/immutable/entry/start.REVZG4IO.js",
            ],
        )

    def test_dashboard_release_assets_cannot_drift_from_html_shell(self) -> None:
        verify = VERIFY_SCRIPT.read_text(encoding="utf-8")
        caddy = CADDY_FILE.read_text(encoding="utf-8")
        config = SVELTE_CONFIG.read_text(encoding="utf-8")
        layout = ROOT_LAYOUT.read_text(encoding="utf-8")

        self.assertIn('handle /_app/immutable/* {', caddy)
        self.assertIn('header >Cache-Control "no-store"', caddy)
        self.assertIn('header_down Cache-Control "public, max-age=31536000, immutable"', caddy)
        self.assertIn("@asset_error status 4xx 5xx", caddy)
        self.assertIn('header Cache-Control "no-store"', caddy)
        self.assertIn("dashboard_asset_paths", verify)
        self.assertIn('python3 "$ROOT_DIR/scripts/extract_dashboard_assets.py"', verify)
        self.assertIn("curl -fsSL --max-redirs 5", verify)
        self.assertIn("Dashboard HTML must be served with Cache-Control: no-store.", verify)
        self.assertIn("Dashboard HTML references an unavailable release asset", verify)
        self.assertIn("pollInterval: 60_000", config)
        self.assertIn("beforeNavigate", layout)
        self.assertIn("updated.current", layout)
        self.assertIn("location.href = to.url.href", layout)

    def test_optional_project_route_health_does_not_block_control_plane_verification(self) -> None:
        verify = VERIFY_SCRIPT.read_text(encoding="utf-8")

        self.assertIn('elif [[ "$REQUIRE_PROJECT_ROUTE" == "true" ]]; then', verify)
        self.assertIn(
            "ignoring workload health for control-plane verification",
            verify,
        )
        self.assertIn(
            "Existing project route $project_host is not healthy while REQUIRE_PROJECT_ROUTE=true.",
            verify,
        )


if __name__ == "__main__":
    unittest.main()
