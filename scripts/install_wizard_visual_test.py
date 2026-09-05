import importlib.util
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
APP_PATH = ROOT_DIR / "scripts" / "install-wizard-preflight-app.py"
VISUAL_PATH = ROOT_DIR / "scripts" / "install_wizard_visual.py"
APP_CSS_PATH = ROOT_DIR / "frontend" / "src" / "app.css"


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


VISUAL = load_module("install_wizard_visual_test_module", VISUAL_PATH)
APP = load_module("install_wizard_visual_app_test_module", APP_PATH)


class InstallWizardVisualContractTest(unittest.TestCase):
    def test_installer_uses_canonical_workspace_tokens(self) -> None:
        css = APP_CSS_PATH.read_text(encoding="utf-8")
        visual = VISUAL.VISUAL_CONTRACT_CSS

        for token in (
            "--app-bg: #fafafa;",
            "--app-surface: #ffffff;",
            "--app-border: #dedede;",
            "--app-border-strong: #c9c9c9;",
            "--app-ink: #171717;",
            "--app-muted: #525252;",
            "--app-subtle: #737373;",
            "--app-bg: #0a0a0a;",
            "--app-surface: #141414;",
            "--app-border: #303030;",
            "--app-border-strong: #4a4a4a;",
            "--app-ink: #fafafa;",
            "--app-muted: #a3a3a3;",
            "--workspace-divider: color-mix(in oklch, var(--app-border) 82%, transparent);",
        ):
            self.assertIn(token, css)
            self.assertIn(token, visual)

    def test_installer_is_full_workspace_not_centered_panel(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn("Standalone installer implementation of frontend/DESIGN.md", html)
        self.assertIn("grid-template-columns: 12rem minmax(0, 1fr)", html)
        self.assertIn("min-height: calc(100vh - 56px)", html)
        self.assertIn('class="setup-form"', html)
        self.assertIn("min-height: 36px", html)
        self.assertNotIn("1180px", html)
        self.assertNotIn("meta-chip", html)
        self.assertNotIn("guide-card", html)
        self.assertNotIn("form.panel", html)

    def test_shell_uses_one_authenticated_workspace_surface(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn("Authenticated shell rule: chrome and workspace use one surface.", html)
        self.assertIn("html, body { margin: 0; min-height: 100%; background: var(--app-surface)", html)
        for selector in (
            ".app-shell,",
            ".app-header,",
            ".workspace,",
            ".setup-rail,",
            ".setup-main,",
            ".setup-form,",
        ):
            self.assertIn(selector, html)
        self.assertNotIn("background: var(--app-bg);", html)

    def test_sidebar_matches_frontend_secondary_navigation_grammar(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn('class="rail-group-label">Setup</p>', html)
        self.assertIn("border-left: 2px solid transparent", html)
        self.assertIn("padding: 8px 10px", html)
        self.assertIn("font-size: 13px", html)
        self.assertEqual(html.count('class="step-icon"'), 4)
        self.assertNotIn("step-number", html)
        self.assertNotIn("Fresh installation", html)

    def test_hidden_navigation_actions_cannot_be_resurrected_by_button_display(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn("[hidden] { display: none !important; }", html)
        self.assertIn('id="back-button" hidden', html)
        self.assertIn('id="submit-button" data-default-label="Install MyPaaS" hidden', html)

    def test_theme_control_uses_lucide_style_sun_and_moon_geometry(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn('id="theme-icon-sun"', html)
        self.assertIn('<circle cx="12" cy="12" r="4"/>', html)
        self.assertIn('id="theme-icon-moon"', html)
        self.assertIn('M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z', html)
        self.assertIn(".icon-button svg { display: block; width: 18px; height: 18px; }", html)

    def test_fresh_install_has_four_task_steps_and_restore_is_secondary(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertEqual(html.count("data-progress="), 4)
        self.assertIn("Step 1 of 4", html)
        self.assertIn('id="restore-toggle"', html)
        self.assertIn('id="restore-panel"', html)
        self.assertNotIn('<span class="step-title">Restore</span>', html)
        self.assertIn('<span class="step-title">Domain</span>', html)
        self.assertIn('<span class="step-title">GitHub</span>', html)
        self.assertIn('<span class="step-title">Cloudflare</span>', html)
        self.assertIn('<span class="step-title">Review</span>', html)

    def test_provider_sections_have_one_separator_owner(self) -> None:
        visual = VISUAL.VISUAL_CONTRACT_CSS
        provider_block = visual.split(".provider-header {", 1)[1].split("}", 1)[0]
        restore_block = visual.split(".restore-panel {", 1)[1].split("}", 1)[0]

        self.assertIn("One separator owner per section", visual)
        self.assertNotIn("border-bottom", provider_block)
        self.assertIn("border-top: 1px solid var(--workspace-divider)", visual.split(".value-list {", 1)[1].split("}", 1)[0])
        self.assertNotIn("border-block", restore_block)

    def test_restore_uses_tokenized_drag_and_drop_file_control(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn('id="backup-dropzone" class="backup-dropzone"', html)
        self.assertIn('class="backup-file-input" type="file"', html)
        self.assertIn("Drop backup here or choose a file", html)
        self.assertIn("MyPaaS .tar.gz backups only", html)
        self.assertIn("dragenter", html)
        self.assertIn("dragover", html)
        self.assertIn("addEventListener('drop'", html)
        self.assertIn("var(--control-border)", html)
        self.assertIn("var(--app-border-strong)", html)
        self.assertIn('id="upload-backup-btn" class="secondary-button" disabled', html)
        self.assertNotIn('class="field-input" type="file" id="backup-file"', html)

    def test_installer_removes_subdomain_onboarding_slop(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertNotIn("panel.example.com", html)
        self.assertNotIn("project.panel.example.com", html)
        self.assertNotIn("Confirm DNS", html)
        self.assertNotIn("Fresh Linux VM", html)
        self.assertNotIn("Environment <code>", html)
        self.assertIn("Use the root domain.", html)
        self.assertIn("*.example.com", html)

    def test_owner_and_provider_controls_follow_admin_patterns(self) -> None:
        html = APP.form_html().decode("utf-8")
        github_step = html.index('data-step="1"')
        cloudflare_step = html.index('data-step="2"')
        owner_position = html.index('id="OWNER_EMAIL"')

        self.assertGreater(owner_position, github_step)
        self.assertLess(owner_position, cloudflare_step)
        self.assertIn("After binding, MyPaaS identifies the owner by GitHub numeric user ID.", html)
        self.assertIn("GitHub docs", html)
        self.assertIn("Cloudflare Tunnel docs", html)
        self.assertIn('data-secret-target="GITHUB_CLIENT_SECRET"', html)
        self.assertIn('data-secret-target="CLOUDFLARE_TUNNEL_TOKEN"', html)
        self.assertIn('data-copy-target="github-homepage-example"', html)
        self.assertIn('data-copy-target="github-callback-example"', html)
        self.assertNotIn("⌁", html)


if __name__ == "__main__":
    unittest.main()
