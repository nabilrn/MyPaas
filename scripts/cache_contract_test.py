import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CADDY = ROOT / "Caddyfile.prod"
APP_HTML = ROOT / "frontend" / "src" / "app.html"


class DashboardCacheContractTest(unittest.TestCase):
    def test_dashboard_documents_are_never_cached_across_releases(self) -> None:
        caddy = CADDY.read_text(encoding="utf-8")

        self.assertIn('handle /_app/immutable/* {', caddy)
        self.assertIn('header >Cache-Control "no-store"', caddy)
        self.assertNotIn("match header Content-Type text/html*", caddy)

    def test_immutable_assets_cache_only_successful_responses(self) -> None:
        caddy = CADDY.read_text(encoding="utf-8")

        self.assertIn(
            'header_down Cache-Control "public, max-age=31536000, immutable"',
            caddy,
        )
        self.assertIn("@asset_error status 4xx 5xx", caddy)
        self.assertIn("handle_response @asset_error", caddy)
        self.assertIn("exclude Cache-Control", caddy)
        self.assertIn('header Cache-Control "no-store"', caddy)
        self.assertIn("copy_response", caddy)

    def test_bootstrap_recovers_once_from_stale_release_assets(self) -> None:
        app = APP_HTML.read_text(encoding="utf-8")

        self.assertIn("mypaas:asset-recovery-at", app)
        self.assertIn("vite:preloadError", app)
        self.assertIn("unhandledrejection", app)
        self.assertIn("/_app/immutable/", app)
        self.assertIn("location.reload()", app)
        self.assertIn("event.preventDefault()", app)
        self.assertIn("recoveryCooldownMs = 30_000", app)


if __name__ == "__main__":
    unittest.main()
