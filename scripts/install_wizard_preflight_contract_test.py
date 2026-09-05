import ast
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]


class InstallWizardPreflightContractTest(unittest.TestCase):
    def test_preflight_python_sources_parse(self) -> None:
        for path in (
            ROOT_DIR / "scripts" / "install_wizard_preflight.py",
            ROOT_DIR / "scripts" / "install-wizard-preflight-app.py",
        ):
            ast.parse(path.read_text(encoding="utf-8"), filename=str(path))

    def test_preflight_app_does_not_log_secret_values(self) -> None:
        app = (ROOT_DIR / "scripts" / "install-wizard-preflight-app.py").read_text(encoding="utf-8")
        helper = (ROOT_DIR / "scripts" / "install_wizard_preflight.py").read_text(encoding="utf-8")

        self.assertNotIn('print(token', app)
        self.assertNotIn('print(token', helper)
        self.assertIn('output.replace(token, "[redacted]")', helper)
        self.assertIn('TUNNEL_TOKEN=', helper)
        self.assertIn('"--env-file"', helper)
        self.assertNotIn('environment["TUNNEL_TOKEN"] = token', helper)


if __name__ == "__main__":
    unittest.main()
