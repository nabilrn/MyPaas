import os
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
SCRIPT = ROOT_DIR / "scripts" / "install-migration.sh"
MIGRATION_PAGE = ROOT_DIR / "frontend" / "src" / "routes" / "admin" / "migration" / "+page.svelte"


class MigrationInstallTest(unittest.TestCase):
    def test_helper_keeps_remote_url_out_of_installer_argv(self) -> None:
        secret_url = "https://example.test/api/admin/migrate/abc/download?token=super-secret-token"
        with tempfile.TemporaryDirectory() as temp_dir:
            temp = Path(temp_dir)
            fake_bin = temp / "bin"
            fake_bin.mkdir()
            curl_args = temp / "curl-args.txt"
            curl_config = temp / "curl-config.txt"
            install_args = temp / "install-args.txt"
            install_env = temp / "install-env.txt"

            fake_curl = fake_bin / "curl"
            fake_curl.write_text(
                "#!/usr/bin/env bash\n"
                "set -euo pipefail\n"
                "printf '%s\\n' \"$@\" > \"$FAKE_CURL_ARGS\"\n"
                "cat > \"$FAKE_CURL_CONFIG\"\n"
                "out=''\n"
                "while [[ $# -gt 0 ]]; do\n"
                "  if [[ \"$1\" == '--output' ]]; then out=\"$2\"; shift 2; else shift; fi\n"
                "done\n"
                "[[ -n \"$out\" ]]\n"
                "printf 'archive' > \"$out\"\n",
                encoding="utf-8",
            )
            fake_curl.chmod(0o755)

            fake_install = temp / "install-vm.sh"
            fake_install.write_text(
                "#!/usr/bin/env bash\n"
                "set -euo pipefail\n"
                "printf '%s\\n' \"$@\" > \"$FAKE_INSTALL_ARGS\"\n"
                "printf '%s\\n' \"${MIGRATE_URL:-}\" > \"$FAKE_INSTALL_ENV\"\n",
                encoding="utf-8",
            )
            fake_install.chmod(0o755)

            env = os.environ.copy()
            env["PATH"] = f"{fake_bin}{os.pathsep}{env['PATH']}"
            env["MYPAAS_INSTALL_VM_SCRIPT"] = str(fake_install)
            env["FAKE_CURL_ARGS"] = str(curl_args)
            env["FAKE_CURL_CONFIG"] = str(curl_config)
            env["FAKE_INSTALL_ARGS"] = str(install_args)
            env["FAKE_INSTALL_ENV"] = str(install_env)

            result = subprocess.run(
                [os.environ.get("BASH_EXECUTABLE", "bash"), str(SCRIPT)],
                input=secret_url + "\n",
                check=True,
                capture_output=True,
                cwd=ROOT_DIR,
                env=env,
                text=True,
            )

            self.assertNotIn(secret_url, curl_args.read_text(encoding="utf-8"))
            self.assertIn(secret_url, curl_config.read_text(encoding="utf-8"))
            self.assertEqual("", install_args.read_text(encoding="utf-8").strip())
            installer_url = install_env.read_text(encoding="utf-8").strip()
            self.assertTrue(installer_url.startswith("file:///tmp/mypaas-migration."))
            self.assertNotIn("super-secret-token", installer_url)
            self.assertNotIn("super-secret-token", result.stdout)
            self.assertNotIn("super-secret-token", result.stderr)

    def test_helper_refuses_remote_url_as_argument(self) -> None:
        result = subprocess.run(
            [os.environ.get("BASH_EXECUTABLE", "bash"), str(SCRIPT), "https://example.test/?token=secret"],
            capture_output=True,
            cwd=ROOT_DIR,
            text=True,
        )
        self.assertNotEqual(0, result.returncode)
        self.assertIn("does not accept the migration URL as an argument", result.stderr)

    def test_helper_can_bootstrap_curl_on_supported_apt_host(self) -> None:
        content = SCRIPT.read_text(encoding="utf-8")

        self.assertIn("ensure_curl()", content)
        self.assertIn("command -v apt-get", content)
        self.assertIn("apt-get install -y ca-certificates curl", content)
        self.assertIn("sudo is required to install curl", content)

    def test_dashboard_separates_secret_url_from_generated_command(self) -> None:
        page = MIGRATION_PAGE.read_text(encoding="utf-8")
        command_block = page.split("$: migrationCommand =", 1)[1].split("\n\t\t: '';", 1)[0]

        self.assertIn("sourceBuildSha", command_block)
        self.assertIn("git -C \"$install_dir\" fetch --depth 1 origin ${sourceBuildSha}", command_block)
        self.assertIn("bash scripts/install-migration.sh", command_block)
        self.assertNotIn("migrationUrl", command_block)
        self.assertNotIn("downloadToken", command_block)
        self.assertNotIn("--migrate-url", command_block)
        self.assertIn("Copy migration URL", page)
        self.assertIn("Current installed revision is unavailable", page)

    def test_dashboard_uses_host_installed_revision_not_api_build_directly(self) -> None:
        page = MIGRATION_PAGE.read_text(encoding="utf-8")

        self.assertIn("fetch('/internal/system-update', { cache: 'no-store' })", page)
        self.assertIn("snapshot.status.currentSha", page)
        self.assertIn("type { UpdateSnapshot }", page)
        self.assertNotIn("api.admin.getSettings()", page)

    def test_dashboard_refreshes_revision_for_prepare_and_ready_state(self) -> None:
        page = MIGRATION_PAGE.read_text(encoding="utf-8")

        self.assertIn("async function refreshInstalledRevision()", page)
        start_block = page.split("async function startMigration()", 1)[1].split("\n\tfunction startPolling()", 1)[0]
        poll_block = page.split("function startPolling()", 1)[1].split("\n\tasync function copyToClipboard", 1)[0]
        self.assertIn("await refreshInstalledRevision()", start_block)
        self.assertIn("status === 'ready'", start_block)
        self.assertIn("await refreshInstalledRevision()", poll_block)
        self.assertIn("status.status === 'ready'", poll_block)


if __name__ == "__main__":
    unittest.main()
