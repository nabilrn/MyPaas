import os
import subprocess
import tempfile
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
        self.assertIn("USE_PODMAN", result.stdout)
        self.assertIn("AUTO_UPDATE_ENABLED", result.stdout)
        self.assertIn("AUTO_UPDATE_INTERVAL_MINUTES", result.stdout)
        self.assertIn("AUTO_UPDATE_REF", result.stdout)

    def test_defaults_to_official_main_repository_and_wizard(self) -> None:
        content = BOOTSTRAP_PATH.read_text(encoding="utf-8")

        self.assertIn("https://github.com/nabilrn/MyPaas.git", content)
        self.assertIn('REF="${MYPAAS_REF:-main}"', content)
        self.assertIn('INSTALL_WIZARD="${INSTALL_WIZARD:-true}"', content)
        self.assertIn('USE_PODMAN="${USE_PODMAN:-true}"', content)
        self.assertIn(
            'INSTALL_WIZARD="$INSTALL_WIZARD" USE_PODMAN="$USE_PODMAN" bash scripts/install-vm.sh',
            content,
        )

    def test_existing_checkout_requires_clean_matching_origin_and_resets_to_fetched_ref(self) -> None:
        content = BOOTSTRAP_PATH.read_text(encoding="utf-8")

        self.assertIn("status --porcelain", content)
        self.assertIn("remote get-url origin", content)
        self.assertIn("fetch --depth 1 origin", content)
        self.assertIn("reset --hard FETCH_HEAD", content)
        self.assertNotIn("merge --ff-only FETCH_HEAD", content)

    def test_existing_install_preserves_detected_container_engine(self) -> None:
        content = BOOTSTRAP_PATH.read_text(encoding="utf-8")

        self.assertIn("detect_existing_runtime", content)
        self.assertIn("socket_has_mypaas_containers", content)
        self.assertIn("Preserving existing Docker Engine runtime", content)
        self.assertIn("Preserving existing Podman runtime", content)
        self.assertIn("refusing a split-runtime update", content)
        self.assertIn("Refusing an in-place Docker/Podman engine switch", content)

    def test_fresh_install_runtime_detection_is_successful_noop(self) -> None:
        bash = os.environ.get("BASH_EXECUTABLE", "bash")
        content = BOOTSTRAP_PATH.read_text(encoding="utf-8")
        harness = content.rsplit('main "$@"', 1)[0] + 'detect_existing_runtime\nprintf "ok\\n"\n'

        with tempfile.TemporaryDirectory() as temp_dir:
            env = os.environ.copy()
            env["MYPAAS_INSTALL_DIR"] = str(Path(temp_dir) / "MyPaas")
            result = subprocess.run(
                [bash],
                input=harness,
                check=True,
                capture_output=True,
                cwd=ROOT_DIR,
                env=env,
                text=True,
            )

        self.assertEqual("ok", result.stdout.strip())


if __name__ == "__main__":
    unittest.main()
