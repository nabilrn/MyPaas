import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CADDY = ROOT / "Caddyfile.prod"
APP_HTML = ROOT / "frontend" / "src" / "app.html"


class DashboardCacheContractTest(unittest.TestCase):
    def test_dashboard_documents_are_never_cached_across_releases(self) -> None:
        """Dashboard documents must never outlive the release that generated them."""
        caddy = CADDY.read_text(encoding="utf-8")

        self.assertIn('handle /_app/immutable/* {', caddy)
        self.assertIn('header >Cache-Control "no-store"', caddy)
        self.assertNotIn("match header Content-Type text/html*", caddy)

    def test_immutable_assets_cache_only_successful_responses(self) -> None:
        """Successful hashed assets are immutable while missing assets remain retryable."""
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
        """Automatic stale-asset recovery must be bounded across document reloads."""
        app = APP_HTML.read_text(encoding="utf-8")

        self.assertIn("mypaas:asset-recovery-at", app)
        self.assertIn("mypaas_asset_recovery_at", app)
        self.assertIn("vite:preloadError", app)
        self.assertIn("unhandledrejection", app)
        self.assertIn("/_app/immutable/", app)
        self.assertIn("location.reload()", app)
        self.assertIn("event.preventDefault()", app)
        self.assertIn("recoveryCooldownMs = 30_000", app)
        self.assertIn("document.cookie", app)
        self.assertIn("if (!persistRecovery(now)) return false", app)
        self.assertIn("return readRecoveryCookie() === timestamp", app)
        self.assertIn("setTimeout(clearRecovery, recoveryCooldownMs)", app)


if __name__ == "__main__":
    unittest.main()
