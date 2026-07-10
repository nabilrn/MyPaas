import importlib.util
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
WIZARD_PATH = ROOT_DIR / "scripts" / "install-wizard.py"
SPEC = importlib.util.spec_from_file_location("install_wizard", WIZARD_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("unable to load install wizard")
WIZARD = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(WIZARD)


class InstallConfigTest(unittest.TestCase):
    def test_wizard_uses_docker_bind_host_for_caddy_upstream(self) -> None:
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

        self.assertIn("DOCKER_BIND_HOST=172.18.0.1", content)
        self.assertIn("CADDY_UPSTREAM_HOST=172.18.0.1", content)

    def test_terminal_installer_uses_detected_bind_host_for_caddy(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("CADDY_UPSTREAM_HOST=$docker_bind_host", installer)

    def test_production_compose_falls_back_to_docker_bind_host(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")

        self.assertIn(
            "CADDY_UPSTREAM_HOST: ${CADDY_UPSTREAM_HOST:-${DOCKER_BIND_HOST:-host.docker.internal}}",
            compose,
        )


if __name__ == "__main__":
    unittest.main()
