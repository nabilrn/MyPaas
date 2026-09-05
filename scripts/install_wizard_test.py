import http.client
import importlib.util
import io
import os
import stat
import sys
import tarfile
import tempfile
import threading
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
WIZARD_PATH = ROOT_DIR / "scripts" / "install-wizard.py"
RUNNER_PATH = ROOT_DIR / "scripts" / "run-install-wizard.sh"
SCRIPTS_DIR = str(WIZARD_PATH.parent)
if SCRIPTS_DIR not in sys.path:
    sys.path.insert(0, SCRIPTS_DIR)
import install_wizard_security as WIZARD_SECURITY

SPEC = importlib.util.spec_from_file_location("install_wizard", WIZARD_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("unable to load install wizard")
WIZARD = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(WIZARD)


def backup_bytes(extra_member: str | None = None, symlink_member: str | None = None) -> bytes:
    output = io.BytesIO()
    with tarfile.open(fileobj=output, mode="w:gz") as archive:
        for name, data in (("database.sql", b"select 1;\n"), (".env", b"PUBLIC_DOMAIN=example.com\n")):
            info = tarfile.TarInfo(name)
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
        if extra_member is not None:
            data = b"unexpected"
            info = tarfile.TarInfo(extra_member)
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
        if symlink_member is not None:
            info = tarfile.TarInfo(symlink_member)
            info.type = tarfile.SYMTYPE
            info.linkname = "/etc/passwd"
            archive.addfile(info)
    return output.getvalue()


class InstallConfigTest(unittest.TestCase):
    def test_raw_wizard_output_is_normalized_by_terminal_installer(self) -> None:
        values = dict(WIZARD.DEFAULTS)
        values.update(
            {
                "PUBLIC_DOMAIN": "mypaas.example.com",
                "OWNER_EMAIL": "owner@example.com",
                "GITHUB_CLIENT_ID": "client-id",
                "GITHUB_CLIENT_SECRET": "client-secret",
                "CLOUDFLARE_TUNNEL_TOKEN": "tunnel-token",
                "DOCKER_BIND_HOST": "172.18.0.1",
            }
        )

        content = WIZARD.build_env(values)
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("DOCKER_BIND_HOST=172.18.0.1", content)
        self.assertIn("CADDY_UPSTREAM_HOST=172.18.0.1", content)
        self.assertIn("sed -i 's#^CADDY_UPSTREAM_HOST=.*#CADDY_UPSTREAM_HOST=runtime#'", installer)

    def test_terminal_installer_provisions_and_persists_three_networks(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn('control_network="${CONTROL_NETWORK:-mypaas-control}"', installer)
        self.assertIn('project_network="${PROJECT_NETWORK:-mypaas-projects}"', installer)
        self.assertIn('routing_network="${ROUTING_NETWORK:-mypaas-routing}"', installer)
        self.assertIn('validate_network_names "$control_network" "$project_network" "$routing_network"', installer)
        self.assertIn('ensure_docker_network "$control_network"', installer)
        self.assertIn('ensure_docker_network "$project_network"', installer)
        self.assertIn('ensure_docker_network "$routing_network"', installer)
        self.assertIn('docker_network_gateway "$project_network"', installer)
        self.assertNotIn('docker_network_gateway "$control_network"', installer)
        self.assertIn("CONTROL_NETWORK=$control_network", installer)
        self.assertIn("PROJECT_NETWORK=$project_network", installer)
        self.assertIn("ROUTING_NETWORK=$routing_network", installer)
        self.assertIn("CADDY_UPSTREAM_HOST=runtime", installer)

    def test_installer_pins_caddy_admin_to_unix_socket(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("CADDY_ADMIN=unix//run/mypaas/caddy-admin.sock", installer)
        self.assertIn("sed -i 's#^CADDY_ADMIN=.*#CADDY_ADMIN=unix//run/mypaas/caddy-admin.sock#'", installer)

    def test_podman_installer_installs_catatonit_for_buildkit(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("! command_exists catatonit", installer)
        self.assertIn("podman catatonit docker-ce-cli", installer)

    def test_installer_uses_verified_statd_release_by_default(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn('INSTALL_STATD="${INSTALL_STATD:-true}"', installer)
        self.assertIn('STATD_INSTALL_MODE="${STATD_INSTALL_MODE:-release}"', installer)
        self.assertIn('STATD_VERSION="${STATD_VERSION:-v0.2.0}"', installer)
        self.assertIn('STATD_ONLY="${STATD_ONLY:-false}"', installer)
        self.assertIn("--statd-only)", installer)
        self.assertIn("mypaas-statd-linux-${arch}.tar.gz", installer)
        self.assertIn("SHA256SUMS.selected", installer)
        self.assertIn("sha256sum -c", installer)
        self.assertIn("mypaas-statd --version", installer)
        self.assertIn("systemctl enable mypaas-statd", installer)
        self.assertIn("systemctl restart mypaas-statd", installer)
        self.assertIn("STATD_SOCKET=/run/mypaas/statd.sock", installer)
        self.assertNotIn('pull --ff-only origin "$STATD_REF" || true', installer)

        values = dict(WIZARD.DEFAULTS)
        values.update(
            {
                "PUBLIC_DOMAIN": "mypaas.example.com",
                "OWNER_EMAIL": "owner@example.com",
                "GITHUB_CLIENT_ID": "client-id",
                "GITHUB_CLIENT_SECRET": "client-secret",
                "CLOUDFLARE_TUNNEL_TOKEN": "tunnel-token",
            }
        )
        content = WIZARD.build_env(values)
        self.assertIn("STATD_SOCKET=/run/mypaas/statd.sock", content)

    def test_source_statd_install_is_explicit_and_fail_closed(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("STATD_INSTALL_MODE must be release or source", installer)
        self.assertIn("checkout --detach", installer)
        self.assertIn("STATD_REF does not resolve", installer)

    def test_production_compose_pins_runtime_upstream_mode(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")

        self.assertIn("CADDY_UPSTREAM_HOST: runtime", compose)
        self.assertIn("ROUTING_NETWORK: ${ROUTING_NETWORK:-mypaas-routing}", compose)
        self.assertNotIn("${CADDY_UPSTREAM_HOST:-${", compose)
        self.assertNotIn("host.docker.internal:host-gateway", compose)

    def test_production_compose_passes_metrics_credentials_to_api(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")
        api = compose.split("  api:\n", 1)[1].split("\n  dashboard:\n", 1)[0]

        self.assertIn("ENABLE_METRICS: ${ENABLE_METRICS:-true}", api)
        self.assertIn("METRICS_USERNAME: ${METRICS_USERNAME:-}", api)
        self.assertIn("METRICS_PASSWORD: ${METRICS_PASSWORD:-}", api)

    def test_production_compose_passes_quota_config_to_api(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")
        api = compose.split("  api:\n", 1)[1].split("\n  dashboard:\n", 1)[0]

        self.assertIn("USER_RAM_QUOTA_GB: ${USER_RAM_QUOTA_GB:-6}", api)
        self.assertIn("USER_CPU_QUOTA: ${USER_CPU_QUOTA:-3}", api)
        self.assertIn("MAX_PROJECTS: ${MAX_PROJECTS:-20}", api)

    def test_production_compose_separates_control_project_and_routing_networks(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")
        postgres = compose.split("  postgres:\n", 1)[1].split("\n  api:\n", 1)[0]
        api = compose.split("  api:\n", 1)[1].split("\n  dashboard:\n", 1)[0]
        caddy = compose.split("  caddy:\n", 1)[1].split("\n  cloudflared:\n", 1)[0]

        self.assertIn("- control", postgres)
        self.assertIn("- projects", postgres)
        self.assertNotIn("- routing", postgres)
        self.assertIn("- control", api)
        self.assertNotIn("- projects", api)
        self.assertNotIn("- routing", api)
        self.assertIn("- control", caddy)
        self.assertIn("- routing", caddy)
        self.assertNotIn("- projects", caddy)
        self.assertIn("name: ${CONTROL_NETWORK:-mypaas-control}", compose)
        self.assertIn("name: ${PROJECT_NETWORK:-mypaas-projects}", compose)
        self.assertIn("name: ${ROUTING_NETWORK:-mypaas-routing}", compose)

    def test_caddy_admin_is_unix_only_in_production_compose(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")
        caddy = compose.split("  caddy:\n", 1)[1].split("\n  cloudflared:\n", 1)[0]
        api = compose.split("  api:\n", 1)[1].split("\n  dashboard:\n", 1)[0]

        self.assertIn("CADDY_ADMIN: unix//run/mypaas/caddy-admin.sock", api)
        self.assertIn('CADDY_ADMIN: "unix//run/mypaas/caddy-admin.sock"', caddy)
        self.assertIn("/run/mypaas:/run/mypaas", caddy)
        self.assertNotIn("2019:2019", caddy)

    def test_api_container_drops_capabilities_and_blocks_privilege_gain(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")
        api = compose.split("  api:\n", 1)[1].split("\n  dashboard:\n", 1)[0]

        self.assertIn("security_opt:\n      - no-new-privileges:true", api)
        self.assertIn("cap_drop:\n      - ALL", api)
        self.assertIn("${DOCKER_SOCKET}:/var/run/docker.sock", api)

    def test_caddy_access_logs_are_persistent(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")
        caddy = compose.split("  caddy:\n", 1)[1].split("\n  cloudflared:\n", 1)[0]

        self.assertIn("caddy_logs:/var/log/caddy", caddy)
        self.assertIn("  caddy_logs:\n", compose)

    def test_wizard_uses_real_brand_asset_inter_and_short_setup_copy(self) -> None:
        html = WIZARD.form_html().decode("utf-8")

        self.assertTrue(Path(WIZARD.BRAND_LOGO_PATH).is_file())
        self.assertTrue(WIZARD.BRAND_LOGO_PATH.endswith("mypaas-logo.svg"))
        self.assertIn('/brand/logo.svg', html)
        self.assertNotIn('/brand/logo.png', html)
        self.assertIn('Inter Variable', html)
        self.assertIn('Set up MyPaas', html)
        self.assertIn('Nothing is saved until the final step.', html)
        self.assertIn('1. Get the tunnel token', html)
        self.assertIn('2. Add two public hostname routes', html)
        self.assertIn('3. Confirm DNS', html)
        self.assertNotIn('mark-dot', html)

    def test_wizard_masks_secrets_and_authenticates_backup_fetch(self) -> None:
        html = WIZARD.form_html().decode("utf-8")

        self.assertRegex(html, r'name="GITHUB_CLIENT_SECRET"[^>]*type="password"')
        self.assertRegex(html, r'name="CLOUDFLARE_TUNNEL_TOKEN"[^>]*type="password"')
        self.assertRegex(html, r'name="POSTGRES_PASSWORD"[^>]*type="password"')
        self.assertRegex(html, r'name="METRICS_PASSWORD"[^>]*type="password"')
        self.assertRegex(html, r'name="JWT_SECRET"[^>]*type="password"')
        self.assertRegex(html, r'name="ENCRYPTION_KEY"[^>]*type="password"')
        self.assertIn("'X-Wizard-Token': wizardToken", html)

    def test_backup_receiver_accepts_only_current_mypaas_backup_shape(self) -> None:
        payload = backup_bytes()
        with tempfile.TemporaryDirectory() as directory:
            destination = os.path.join(directory, "restore.tar.gz")
            WIZARD_SECURITY.receive_backup(
                io.BytesIO(payload),
                str(len(payload)),
                destination,
                max_bytes=len(payload) + 1,
                max_expanded_bytes=1024 * 1024,
            )
            self.assertTrue(os.path.isfile(destination))
            self.assertEqual(stat.S_IMODE(os.stat(destination).st_mode), 0o600)
            WIZARD_SECURITY.validate_backup_archive(destination, max_expanded_bytes=1024 * 1024)

    def test_backup_receiver_rejects_path_traversal_and_extra_members(self) -> None:
        for extra in ("../escape", "nested/database.sql", "other.txt"):
            with self.subTest(extra=extra), tempfile.TemporaryDirectory() as directory:
                payload = backup_bytes(extra_member=extra)
                destination = os.path.join(directory, "restore.tar.gz")
                with self.assertRaises(WIZARD_SECURITY.BackupUploadError):
                    WIZARD_SECURITY.receive_backup(
                        io.BytesIO(payload),
                        str(len(payload)),
                        destination,
                        max_bytes=len(payload) + 1,
                        max_expanded_bytes=1024 * 1024,
                    )
                self.assertFalse(os.path.exists(destination))

    def test_backup_receiver_rejects_links_truncation_and_size_limits(self) -> None:
        linked = backup_bytes(symlink_member="extra-link")
        with tempfile.TemporaryDirectory() as directory:
            destination = os.path.join(directory, "restore.tar.gz")
            with self.assertRaises(WIZARD_SECURITY.BackupUploadError):
                WIZARD_SECURITY.receive_backup(
                    io.BytesIO(linked),
                    str(len(linked)),
                    destination,
                    max_bytes=len(linked) + 1,
                    max_expanded_bytes=1024 * 1024,
                )

        valid = backup_bytes()
        with tempfile.TemporaryDirectory() as directory:
            destination = os.path.join(directory, "restore.tar.gz")
            with self.assertRaises(WIZARD_SECURITY.BackupUploadError):
                WIZARD_SECURITY.receive_backup(
                    io.BytesIO(valid[:-4]),
                    str(len(valid)),
                    destination,
                    max_bytes=len(valid) + 1,
                    max_expanded_bytes=1024 * 1024,
                )
            with self.assertRaises(WIZARD_SECURITY.BackupTooLargeError):
                WIZARD_SECURITY.receive_backup(
                    io.BytesIO(valid),
                    str(len(valid)),
                    destination,
                    max_bytes=len(valid) - 1,
                    max_expanded_bytes=1024 * 1024,
                )

    def test_backup_upload_endpoint_rejects_missing_wizard_token(self) -> None:
        old_token = WIZARD.TOKEN
        old_path = WIZARD.BACKUP_PATH
        try:
            WIZARD.TOKEN = "test-wizard-token"
            with tempfile.TemporaryDirectory() as directory:
                WIZARD.BACKUP_PATH = os.path.join(directory, "restore.tar.gz")
                server = WIZARD.HTTPServer(("127.0.0.1", 0), WIZARD.Handler)
                thread = threading.Thread(target=server.serve_forever, daemon=True)
                thread.start()
                try:
                    payload = backup_bytes()
                    connection = http.client.HTTPConnection("127.0.0.1", server.server_port, timeout=3)
                    connection.request(
                        "POST",
                        "/upload-backup",
                        body=payload,
                        headers={"Content-Type": "application/octet-stream", "Content-Length": str(len(payload))},
                    )
                    response = connection.getresponse()
                    response.read()
                    connection.close()
                    self.assertEqual(response.status, 403)
                    self.assertFalse(os.path.exists(WIZARD.BACKUP_PATH))
                finally:
                    server.shutdown()
                    server.server_close()
                    thread.join(timeout=3)
        finally:
            WIZARD.TOKEN = old_token
            WIZARD.BACKUP_PATH = old_path

    def test_success_html_renders_auto_close_script(self) -> None:
        html = WIZARD.success_html(title="Done", message="Saved.").decode("utf-8")

        self.assertIn('/brand/logo.svg', html)
        self.assertIn("setTimeout(() => {", html)
        self.assertIn("window.close();", html)
        self.assertIn("}, 4000);", html)

    def test_installer_enables_temporary_public_wizard_by_default(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn('WIZARD_PUBLIC_TUNNEL="${WIZARD_PUBLIC_TUNNEL:-true}"', installer)
        self.assertIn('bash "$ROOT_DIR/scripts/run-install-wizard.sh"', installer)

    def test_terminal_installer_accepts_legacy_bootstrap_aliases(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("env_alias OWNER_EMAIL GITHUB_EMAIL_ACCOUNT", installer)
        self.assertIn("env_alias GITHUB_CLIENT_ID GITHUB_OAUTH_CLIENT_ID", installer)
        self.assertIn("env_alias GITHUB_CLIENT_SECRET GITHUB_OAUTH_CLIENT_SECRET GITHUB_OAUTH_CLIENT_SCREET", installer)
        self.assertIn("env_alias GITHUB_CALLBACK_URL REDIRECT_URI RECIRECT_URI", installer)
        self.assertIn("env_alias CLOUDFLARE_TUNNEL_TOKEN CLOUDFLARE_TUNNELS_TOKEN", installer)

    def test_terminal_installer_normalizes_public_domain_and_deploy_rejects_malformed_hosts(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")
        deployer = (ROOT_DIR / "scripts" / "deploy-to-vm.sh").read_text(encoding="utf-8")

        self.assertIn("normalize_public_domain", installer)
        self.assertIn("validate_public_domain", installer)
        self.assertIn("PUBLIC_DOMAIN must be a hostname, not a URL", installer)
        self.assertIn("is_valid_public_domain", deployer)
        self.assertIn('[[ "$value" == *.* ]] || return 1', deployer)
        self.assertIn("PUBLIC_DOMAIN must be a bare DNS hostname like mypaas.example.com", deployer)

    def test_wizard_runner_uses_ephemeral_cloudflare_tunnel_and_cleanup(self) -> None:
        runner = RUNNER_PATH.read_text(encoding="utf-8")

        self.assertIn("cloudflare/cloudflared:latest", runner)
        self.assertIn("--network host", runner)
        self.assertIn("trycloudflare\\.com", runner)
        self.assertIn("trap cleanup EXIT INT TERM", runner)
        self.assertIn("SSH fallback", runner)


if __name__ == "__main__":
    unittest.main()
