import importlib.util
import io
import json
import os
import socket
import threading
import unittest
from pathlib import Path
from unittest import mock
from urllib.error import HTTPError
from urllib.request import Request, urlopen


ROOT_DIR = Path(__file__).resolve().parents[1]
PREFLIGHT_PATH = ROOT_DIR / "scripts" / "install_wizard_preflight.py"
APP_PATH = ROOT_DIR / "scripts" / "install-wizard-preflight-app.py"
RUNNER_PATH = ROOT_DIR / "scripts" / "run-install-wizard.sh"


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


PREFLIGHT = load_module("install_wizard_preflight_test_module", PREFLIGHT_PATH)
APP = load_module("install_wizard_preflight_app_test_module", APP_PATH)


class FakeResponse:
    def __init__(self, payload: dict, status: int = 200):
        self.payload = payload
        self.status = status

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False

    def read(self):
        return json.dumps(self.payload).encode("utf-8")


class InstallWizardPreflightTest(unittest.TestCase):
    def test_domain_preflight_checks_root_and_random_wildcard(self) -> None:
        calls = []

        def resolver(host, port, type):
            calls.append((host, port, type))
            return [
                (socket.AF_INET, socket.SOCK_STREAM, 6, "", ("203.0.113.10", port)),
            ]

        result = PREFLIGHT.check_domain("Example.COM.", resolver=resolver)

        self.assertTrue(result["ok"])
        self.assertTrue(result["wildcardResolved"])
        self.assertEqual(result["hostname"], "example.com")
        self.assertEqual(result["addresses"], ["203.0.113.10"])
        self.assertEqual(calls[0][0], "example.com")
        self.assertRegex(calls[1][0], r"^mypaas-preflight-[0-9a-f]{8}\.example\.com$")

    def test_domain_preflight_rejects_url_instead_of_hostname(self) -> None:
        with self.assertRaisesRegex(PREFLIGHT.PreflightError, "hostname only"):
            PREFLIGHT.validate_hostname("https://example.com/path")

    def test_github_preflight_accepts_expected_invalid_code_response(self) -> None:
        captured = {}

        def opener(request, timeout):
            captured["request"] = request
            captured["timeout"] = timeout
            return FakeResponse({"error": "bad_verification_code"})

        result = PREFLIGHT.check_github_oauth(
            "client-id",
            "client-secret",
            "https://example.com/api/auth/github/callback",
            opener=opener,
        )

        self.assertTrue(result["credentialsAccepted"])
        self.assertTrue(result["callbackFormatAccepted"])
        self.assertIn("Owner identity is verified during sign-in", result["message"])
        self.assertEqual(captured["timeout"], 8)
        self.assertEqual(captured["request"].full_url, PREFLIGHT.GITHUB_TOKEN_URL)

    def test_github_preflight_requires_mypaas_callback_path(self) -> None:
        with self.assertRaisesRegex(PREFLIGHT.PreflightError, "/api/auth/github/callback"):
            PREFLIGHT.validate_https_callback("https://example.com/wrong-callback")

    def test_github_preflight_rejects_bad_credentials(self) -> None:
        def opener(request, timeout):
            return FakeResponse({"error": "incorrect_client_credentials"})

        with self.assertRaisesRegex(PREFLIGHT.PreflightError, "rejected the Client ID or Client Secret"):
            PREFLIGHT.check_github_oauth(
                "wrong-id",
                "wrong-secret",
                "https://example.com/api/auth/github/callback",
                opener=opener,
            )

    def test_github_preflight_rejects_callback_mismatch_when_github_reports_it(self) -> None:
        def opener(request, timeout):
            return FakeResponse({"error": "redirect_uri_mismatch"})

        with self.assertRaisesRegex(PREFLIGHT.PreflightError, "callback URL does not match"):
            PREFLIGHT.check_github_oauth(
                "client-id",
                "client-secret",
                "https://wrong.example.com/api/auth/github/callback",
                opener=opener,
            )

    def test_cloudflare_token_is_extracted_from_add_replica_command(self) -> None:
        token = "eyJhbGciOiJIUzI1NiJ9.eyJ0dW5uZWwiOiIxMjM0NTYifQ.signature-value"
        command = f"docker run cloudflare/cloudflared:latest tunnel --no-autoupdate run --token {token}"

        self.assertEqual(PREFLIGHT.extract_cloudflare_tunnel_token(command), token)
        self.assertEqual(PREFLIGHT.extract_cloudflare_tunnel_token(token), token)

    def test_cloudflare_probe_keeps_token_out_of_process_arguments(self) -> None:
        token = "eyJhbGciOiJIUzI1NiJ9.eyJ0dW5uZWwiOiIxMjM0NTYifQ.signature-value"
        captured = {}

        class Process:
            stdout = io.StringIO("")

            def poll(self):
                return None

            def terminate(self):
                captured["terminated"] = True

            def wait(self, timeout=None):
                return 0

            def kill(self):
                captured["killed"] = True

        def popen(command, **kwargs):
            captured["command"] = command
            captured["env"] = kwargs["env"]
            return Process()

        class ReadyResponse:
            status = 200

            def __enter__(self):
                return self

            def __exit__(self, exc_type, exc, tb):
                return False

        with mock.patch.object(PREFLIGHT, "_free_loopback_port", return_value=49123), \
             mock.patch.object(PREFLIGHT.subprocess, "Popen", side_effect=popen), \
             mock.patch.object(PREFLIGHT.subprocess, "run"):
            result = PREFLIGHT.check_cloudflare_tunnel(
                token,
                cli_resolver=lambda: ["docker"],
                readiness_opener=lambda url, timeout: ReadyResponse(),
            )

        self.assertTrue(result["ok"])
        self.assertNotIn(token, captured["command"])
        self.assertNotEqual(captured["env"].get("TUNNEL_TOKEN"), token)
        self.assertIn("--env-file", captured["command"])
        env_file = captured["command"][captured["command"].index("--env-file") + 1]
        self.assertFalse(os.path.exists(env_file))
        self.assertTrue(captured["terminated"])

    def test_preflight_app_injects_checks_without_redesigning_wizard(self) -> None:
        html = APP.form_html().decode("utf-8")

        self.assertIn('id="check-domain-button"', html)
        self.assertIn('id="check-github-button"', html)
        self.assertIn('id="check-cloudflare-button"', html)
        self.assertIn("postPreflight('domain'", html)
        self.assertIn("postPreflight('github'", html)
        self.assertIn("postPreflight('cloudflare'", html)
        self.assertIn('class="panel stepper"', html)

    def test_preflight_app_normalizes_full_cloudflare_command_before_save(self) -> None:
        token = "eyJhbGciOiJIUzI1NiJ9.eyJ0dW5uZWwiOiIxMjM0NTYifQ.signature-value"
        values = dict(APP.BASE.DEFAULTS)
        values.update(
            {
                "PUBLIC_DOMAIN": "example.com",
                "OWNER_EMAIL": "owner@example.com",
                "GITHUB_CLIENT_ID": "client-id",
                "GITHUB_CLIENT_SECRET": "client-secret",
                "GITHUB_CALLBACK_URL": "https://example.com/api/auth/github/callback",
                "CLOUDFLARE_TUNNEL_TOKEN": f"cloudflared tunnel run --token {token}",
            }
        )

        content = APP.build_env(values)

        self.assertIn(f"CLOUDFLARE_TUNNEL_TOKEN={token}", content)
        self.assertNotIn("cloudflared tunnel run", content)

    def test_preflight_endpoints_require_wizard_token(self) -> None:
        server = APP.BASE.HTTPServer(("127.0.0.1", 0), APP.Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            request = Request(
                f"http://127.0.0.1:{server.server_port}/preflight/domain",
                data=b'{"hostname":"example.com"}',
                method="POST",
                headers={"Content-Type": "application/json"},
            )
            with self.assertRaises(HTTPError) as error:
                urlopen(request, timeout=2)
            self.assertEqual(error.exception.code, 403)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)

    def test_runner_prefers_preflight_app_and_preserves_base_script(self) -> None:
        runner = RUNNER_PATH.read_text(encoding="utf-8")

        self.assertIn("install-wizard-preflight-app.py", runner)
        self.assertIn('WIZARD_BASE_SCRIPT="$WIZARD_SCRIPT" python3 "$WIZARD_APP_SCRIPT"', runner)
        self.assertIn('python3 "$WIZARD_SCRIPT"', runner)


if __name__ == "__main__":
    unittest.main()
