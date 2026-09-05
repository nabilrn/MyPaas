import importlib.util
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
APP_PATH = ROOT_DIR / "scripts" / "install-wizard-preflight-app.py"


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


APP = load_module("install_wizard_preflight_gate_test_module", APP_PATH)


class InstallWizardPreflightGateTest(unittest.TestCase):
    def setUp(self) -> None:
        APP.PREFLIGHT_STATE.clear()
        self.token = "eyJ" + "a" * 40
        self.values = {
            "PUBLIC_DOMAIN": "mypaas.example.com",
            "GITHUB_CLIENT_ID": "Iv1.1234567890",
            "GITHUB_CLIENT_SECRET": "github-secret-value",
            "GITHUB_CALLBACK_URL": "https://mypaas.example.com/api/auth/github/callback",
            "CLOUDFLARE_TUNNEL_TOKEN": self.token,
        }

    def qualify_all(self, *, wildcard_resolved: bool = True) -> None:
        APP.remember_domain_preflight(self.values["PUBLIC_DOMAIN"], wildcard_resolved=wildcard_resolved)
        APP.remember_github_preflight(
            self.values["GITHUB_CLIENT_ID"],
            self.values["GITHUB_CLIENT_SECRET"],
            self.values["GITHUB_CALLBACK_URL"],
        )
        APP.remember_cloudflare_preflight(self.values["CLOUDFLARE_TUNNEL_TOKEN"])

    def test_final_save_requires_all_live_preflights(self) -> None:
        with self.assertRaisesRegex(ValueError, "Domain settings changed after preflight"):
            APP.validate_save_preflights(self.values)

        self.qualify_all()
        APP.validate_save_preflights(self.values)

    def test_project_wildcard_must_be_verified_before_save(self) -> None:
        self.qualify_all(wildcard_resolved=False)

        with self.assertRaisesRegex(ValueError, "Project wildcard DNS is not verified yet"):
            APP.validate_save_preflights(self.values)

    def test_changed_github_secret_invalidates_previous_qualification(self) -> None:
        self.qualify_all()
        changed = dict(self.values)
        changed["GITHUB_CLIENT_SECRET"] = "rotated-secret-value"

        with self.assertRaisesRegex(ValueError, "GitHub OAuth settings changed after preflight"):
            APP.validate_save_preflights(changed)

    def test_changed_domain_invalidates_previous_qualification(self) -> None:
        self.qualify_all()
        changed = dict(self.values)
        changed["PUBLIC_DOMAIN"] = "new.example.com"
        changed["GITHUB_CALLBACK_URL"] = "https://new.example.com/api/auth/github/callback"

        with self.assertRaisesRegex(ValueError, "Domain settings changed after preflight"):
            APP.validate_save_preflights(changed)

    def test_full_cloudflare_command_and_token_share_the_same_fingerprint(self) -> None:
        command = f"docker run cloudflare/cloudflared:latest tunnel run --token {self.token}"
        self.values["CLOUDFLARE_TUNNEL_TOKEN"] = command
        self.qualify_all()

        saved = dict(self.values)
        saved["CLOUDFLARE_TUNNEL_TOKEN"] = self.token
        APP.validate_save_preflights(saved)

    def test_qualification_state_does_not_retain_plaintext_secrets(self) -> None:
        self.qualify_all()
        state = repr(APP.PREFLIGHT_STATE)

        self.assertNotIn(self.values["GITHUB_CLIENT_SECRET"], state)
        self.assertNotIn(self.token, state)
        self.assertIn("fingerprint", state)

    def test_html_continue_flow_is_preflight_gated(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn("Installer preflight gate contract", html)
        self.assertIn("event.stopImmediatePropagation()", html)
        self.assertIn("if (![0, 1, 2].includes(stepIndex)) return;", html)
        self.assertIn("if (stepIndex === 0) passed = await runDomainCheck();", html)
        self.assertIn("if (stepIndex === 1) passed = await runGithubCheck();", html)
        self.assertIn("if (stepIndex === 2) passed = await runCloudflareCheck({ verifyRouting: true });", html)
        self.assertIn("project wildcard DNS is not visible yet", html)
        self.assertIn("Checking…", html)


if __name__ == "__main__":
    unittest.main()
