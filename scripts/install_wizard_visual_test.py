import importlib.util
import os
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

    def test_installer_uses_flat_compact_workspace_geometry(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn("Installer visual contract mirrors", html)
        self.assertIn("grid-template-columns: 12rem minmax(0, 1fr)", html)
        self.assertIn("form.panel { min-width: 0; border-left: 1px solid var(--workspace-divider); }", html)
        self.assertIn(".panel { border: 0; border-radius: 0;", html)
        self.assertIn("min-height: 36px", html)
        self.assertIn("border-bottom: 1px solid var(--workspace-divider)", html)
        self.assertNotIn("decorative-gradient", html)

    def test_installer_copy_matches_durable_owner_identity_contract(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn("Set up MyPaaS", html)
        self.assertIn("Owner verified primary GitHub email", html)
        self.assertIn("After binding, MyPaaS identifies this account by GitHub numeric user ID.", html)
        self.assertIn("paste the token or the full Add-a-replica command", html)
        self.assertNotIn("Only this whitelisted email can log in as the first owner.", html)

    def test_visual_transform_is_idempotent(self) -> None:
        source = APP.ORIGINAL_FORM_HTML().decode("utf-8")
        once = VISUAL.apply_visual_contract(source)
        twice = VISUAL.apply_visual_contract(once)

        self.assertEqual(once, twice)
        self.assertEqual(once.count("Installer visual contract mirrors"), 1)


if __name__ == "__main__":
    unittest.main()
