import os
import subprocess
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
PODMAN_SCRIPT = ROOT_DIR / "scripts" / "migrate-to-podman.sh"
EXPORT_SCRIPT = ROOT_DIR / "scripts" / "migrate-export.sh"


class MigrationSafetyTest(unittest.TestCase):
    def run_script(self, path: Path, *args: str) -> subprocess.CompletedProcess[str]:
        bash = os.environ.get("BASH_EXECUTABLE", "bash")
        return subprocess.run(
            [bash, str(path.relative_to(ROOT_DIR)), *args],
            cwd=ROOT_DIR,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_podman_migration_help_is_safe_and_non_destructive(self) -> None:
        result = self.run_script(PODMAN_SCRIPT, "--help")
        self.assertEqual(result.returncode, 0)
        self.assertIn("retired", result.stdout.lower())
        self.assertIn("USE_PODMAN=true", result.stdout)

        content = PODMAN_SCRIPT.read_text(encoding="utf-8")
        self.assertNotIn("apt-get remove", content)
        self.assertNotIn("systemctl stop docker", content)
        self.assertNotIn("ln -sf /run/podman/podman.sock", content)

    def test_podman_migration_refuses_normal_execution(self) -> None:
        result = self.run_script(PODMAN_SCRIPT)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing destructive", result.stderr.lower())

    def test_standalone_exporter_is_retired(self) -> None:
        help_result = self.run_script(EXPORT_SCRIPT, "--help")
        self.assertEqual(help_result.returncode, 0)
        self.assertIn("single supported exporter", help_result.stdout.lower())
        self.assertIn("engine-managed compose", help_result.stdout.lower())

        run_result = self.run_script(EXPORT_SCRIPT)
        self.assertNotEqual(run_result.returncode, 0)
        self.assertIn("refusing the retired", run_result.stderr.lower())

        content = EXPORT_SCRIPT.read_text(encoding="utf-8")
        self.assertNotIn("xargs docker stop", content)
        self.assertNotIn("tar czf", content)


if __name__ == "__main__":
    unittest.main()
