import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import install_wizard_restore_env as restore_env


ROOT_DIR = Path(__file__).resolve().parents[1]
EXAMPLE = ROOT_DIR / ".env.example"


class InstallWizardRestoreEnvTest(unittest.TestCase):
    def valid_restored_env(self) -> str:
        return """ENVIRONMENT=production
POSTGRES_USER=mypaas
POSTGRES_PASSWORD=backup-password
POSTGRES_DB=mypaas
PUBLIC_DOMAIN=example.com
OWNER_EMAIL=owner@example.com
GITHUB_CLIENT_ID=client-id
GITHUB_CLIENT_SECRET=client-secret
GITHUB_CALLBACK_URL=https://example.com/api/auth/github/callback
CLOUDFLARE_TUNNEL_TOKEN=eyJbackup-token
JWT_SECRET=jwt-secret
ENCRYPTION_KEY=encryption-key
PROJECT_NETWORK=old-project-network
CONTROL_NETWORK=old-control-network
ROUTING_NETWORK=old-routing-network
DOCKER_SOCKET=/tmp/old.sock
CADDY_ADMIN=127.0.0.1:2019
CADDY_UPSTREAM_HOST=old-host
"""

    def test_merge_preserves_fresh_host_runtime_placement(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            current = root / ".env"
            restored = root / "restored.env"
            current.write_text(
                "CONTROL_NETWORK=mypaas-control\n"
                "PROJECT_NETWORK=mypaas-projects\n"
                "ROUTING_NETWORK=mypaas-routing\n"
                "DOCKER_SOCKET=/var/run/docker.sock\n"
                "CADDY_ADMIN=unix//run/mypaas/caddy-admin.sock\n"
                "CADDY_UPSTREAM_HOST=runtime\n",
                encoding="utf-8",
            )
            restored.write_text(self.valid_restored_env(), encoding="utf-8")

            restore_env.merge_env_files(current, restored, EXAMPLE)
            merged = restore_env.parse_env(current)

            self.assertEqual(merged["PUBLIC_DOMAIN"], "example.com")
            self.assertEqual(merged["POSTGRES_PASSWORD"], "backup-password")
            self.assertEqual(merged["CONTROL_NETWORK"], "mypaas-control")
            self.assertEqual(merged["PROJECT_NETWORK"], "mypaas-projects")
            self.assertEqual(merged["ROUTING_NETWORK"], "mypaas-routing")
            self.assertEqual(merged["DOCKER_SOCKET"], "/var/run/docker.sock")
            self.assertEqual(merged["CADDY_ADMIN"], "unix//run/mypaas/caddy-admin.sock")
            self.assertEqual(merged["CADDY_UPSTREAM_HOST"], "runtime")
            self.assertEqual(os.stat(current).st_mode & 0o777, 0o600)

    def test_restore_rejects_shell_environment_keys(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            current = root / ".env"
            restored = root / "restored.env"
            restored.write_text(self.valid_restored_env() + "BASH_ENV=/tmp/pwn\n", encoding="utf-8")
            with self.assertRaisesRegex(restore_env.RestoreEnvError, "unsupported restored setting BASH_ENV"):
                restore_env.merge_env_files(current, restored, EXAMPLE)
            self.assertFalse(current.exists())

    def test_restore_rejects_duplicate_keys(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            restored = Path(tmp) / "restored.env"
            restored.write_text(self.valid_restored_env() + "PUBLIC_DOMAIN=evil.example\n", encoding="utf-8")
            with self.assertRaisesRegex(restore_env.RestoreEnvError, "duplicate environment key PUBLIC_DOMAIN"):
                restore_env.parse_env(restored, allowed_keys=restore_env.allowed_keys_from_example(EXAMPLE))

    def test_merged_file_does_not_execute_command_substitution_when_sourced(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            current = root / ".env"
            restored = root / "restored.env"
            marker = root / "executed"
            restored.write_text(
                self.valid_restored_env().replace(
                    "GITHUB_CLIENT_SECRET=client-secret",
                    f"GITHUB_CLIENT_SECRET=$(touch {marker})",
                ),
                encoding="utf-8",
            )

            restore_env.merge_env_files(current, restored, EXAMPLE)
            completed = subprocess.run(
                ["bash", "-c", 'set -a; source "$1"; set +a; printf "%s" "$GITHUB_CLIENT_SECRET"', "bash", str(current)],
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )
            self.assertFalse(marker.exists())
            self.assertIn("$(touch", completed.stdout)

    def test_restore_rejects_single_quote_before_replacing_current_file(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            current = root / ".env"
            restored = root / "restored.env"
            current.write_text("CONTROL_NETWORK=mypaas-control\n", encoding="utf-8")
            restored.write_text(
                self.valid_restored_env().replace("GITHUB_CLIENT_SECRET=client-secret", "GITHUB_CLIENT_SECRET=bad'value"),
                encoding="utf-8",
            )
            before = current.read_text(encoding="utf-8")
            with self.assertRaisesRegex(restore_env.RestoreEnvError, "single quote"):
                restore_env.merge_env_files(current, restored, EXAMPLE)
            self.assertEqual(current.read_text(encoding="utf-8"), before)

    def test_restore_requires_complete_control_plane_credentials(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            current = root / ".env"
            restored = root / "restored.env"
            restored.write_text(
                self.valid_restored_env().replace("GITHUB_CLIENT_SECRET=client-secret\n", ""),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(restore_env.RestoreEnvError, "GITHUB_CLIENT_SECRET"):
                restore_env.merge_env_files(current, restored, EXAMPLE)


if __name__ == "__main__":
    unittest.main()
