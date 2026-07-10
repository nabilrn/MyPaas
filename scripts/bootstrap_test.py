import os
import subprocess
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
BOOTSTRAP_PATH = ROOT_DIR / "scripts" / "bootstrap.sh"


class BootstrapTest(unittest.TestCase):
    def test_help_documents_public_overrides(self) -> None:
        bash = os.environ.get("BASH_EXECUTABLE", "bash")

        result = subprocess.run(
            [bash, "scripts/bootstrap.sh", "--help"],
            check=True,
            capture_output=True,
            cwd=ROOT_DIR,
            text=True,
        )

        self.assertIn("MYPAAS_REPO_URL", result.stdout)
        self.assertIn("MYPAAS_REF", result.stdout)
        self.assertIn("MYPAAS_INSTALL_DIR", result.stdout)
        self.assertIn("INSTALL_WIZARD", result.stdout)

    def test_defaults_to_official_main_repository_and_wizard(self) -> None:
        content = BOOTSTRAP_PATH.read_text(encoding="utf-8")

        self.assertIn("https://github.com/nabilrn/MyPaas.git", content)
        self.assertIn('REF="${MYPAAS_REF:-main}"', content)
        self.assertIn('INSTALL_WIZARD="${INSTALL_WIZARD:-true}"', content)
        self.assertIn('INSTALL_WIZARD="$INSTALL_WIZARD" bash scripts/install-vm.sh', content)

    def test_existing_checkout_requires_clean_matching_origin(self) -> None:
        content = BOOTSTRAP_PATH.read_text(encoding="utf-8")

        self.assertIn("status --porcelain", content)
        self.assertIn("remote get-url origin", content)
        self.assertIn("fetch --depth 1 origin", content)
        self.assertIn("merge --ff-only FETCH_HEAD", content)


if __name__ == "__main__":
    unittest.main()
