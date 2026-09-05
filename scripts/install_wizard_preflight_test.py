import io
import json
import unittest
from pathlib import Path

import install_wizard_preflight as preflight


class FakeResponse:
    def __init__(self, payload: dict):
        self.body = io.BytesIO(json.dumps(payload).encode("utf-8"))

    def read(self):
        return self.body.read()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False


class InstallWizardPreflightTest(unittest.TestCase):
    def test_domain_normalizes_url_and_validates_hostname_labels(self) -> None:
        result = preflight.validate_domain("https://Example.COM/path")
        self.assertTrue(result.ok)
        self.assertEqual(result.value, "example.com")
        self.assertFalse(preflight.validate_domain("bad..example.com").ok)
        self.assertFalse(preflight.validate_domain("localhost").ok)

    def test_domain_dns_reports_resolution_separately_from_format(self) -> None:
        result = preflight.check_domain_dns(
            "example.com",
            resolver=lambda *_args, **_kwargs: [(2, 1, 6, "", ("93.184.216.34", 443))],
        )
        self.assertTrue(result.ok)
        self.assertEqual(result.code, "domain_dns_resolved")

        def fail_resolver(*_args, **_kwargs):
            raise OSError("not found")

        result = preflight.check_domain_dns("example.com", resolver=fail_resolver)
        self.assertFalse(result.ok)
        self.assertEqual(result.code, "domain_dns_unresolved")

    def test_owner_email_is_not_claimed_verified_before_oauth(self) -> None:
        result = preflight.validate_owner_email("Owner@Example.com")
        self.assertTrue(result.ok)
        self.assertEqual(result.code, "owner_email_unverified")
        self.assertEqual(result.value, "owner@example.com")
        self.assertIn("only after", result.message)

    def test_github_oauth_bad_code_confirms_credentials_without_claiming_callback_match(self) -> None:
        captured = {}

        def open_request(request, timeout):
            captured["request"] = request
            captured["timeout"] = timeout
            return FakeResponse({"error": "bad_verification_code"})

        result = preflight.probe_github_oauth(
            "client-id",
            "client-secret",
            "https://example.com/api/auth/github/callback",
            open_request=open_request,
        )
        self.assertTrue(result.ok)
        self.assertEqual(result.code, "github_credentials_valid")
        self.assertIn("first sign-in", result.message)
        body = captured["request"].data.decode("utf-8")
        self.assertIn("client_id=client-id", body)
        self.assertIn("redirect_uri=https%3A%2F%2Fexample.com%2Fapi%2Fauth%2Fgithub%2Fcallback", body)

    def test_github_oauth_classifies_bad_credentials_without_echoing_secret(self) -> None:
        def open_request(_request, timeout):
            return FakeResponse({"error": "incorrect_client_credentials"})

        result = preflight.probe_github_oauth(
            "client-id",
            "super-secret-value",
            "https://example.com/api/auth/github/callback",
            open_request=open_request,
        )
        self.assertFalse(result.ok)
        self.assertEqual(result.code, "incorrect_client_credentials")
        self.assertNotIn("super-secret-value", result.message)

    def test_github_oauth_rejects_wrong_callback_shape_before_network(self) -> None:
        def fail_if_called(*_args, **_kwargs):
            raise AssertionError("network should not run")

        result = preflight.probe_github_oauth(
            "client-id",
            "client-secret",
            "https://example.com/wrong",
            open_request=fail_if_called,
        )
        self.assertFalse(result.ok)
        self.assertEqual(result.code, "github_callback_invalid")

    def test_cloudflare_token_parser_accepts_raw_token_or_full_command(self) -> None:
        token = "eyJ" + "A" * 64
        raw = preflight.extract_cloudflare_tunnel_token(token)
        command = preflight.extract_cloudflare_tunnel_token(
            f"docker run cloudflare/cloudflared:latest tunnel run --token {token}"
        )
        self.assertTrue(raw.ok)
        self.assertEqual(raw.value, token)
        self.assertTrue(command.ok)
        self.assertEqual(command.value, token)

    def test_cloudflare_probe_classification_requires_registered_connection(self) -> None:
        ok = preflight.classify_cloudflared_probe(
            "INF Registered tunnel connection connIndex=0",
            None,
            True,
        )
        self.assertTrue(ok.ok)
        rejected = preflight.classify_cloudflared_probe("ERR invalid tunnel token", 1, False)
        self.assertFalse(rejected.ok)
        self.assertEqual(rejected.code, "cloudflare_tunnel_rejected")
        timeout = preflight.classify_cloudflared_probe("", None, True)
        self.assertFalse(timeout.ok)
        self.assertEqual(timeout.code, "cloudflare_tunnel_unconfirmed")

    def test_cloudflare_probe_keeps_token_out_of_process_arguments(self) -> None:
        token = "eyJ" + "B" * 64
        observed = {}

        class FakeProcess:
            returncode = None

            def communicate(self, timeout=None):
                self.returncode = 0
                return "INF Registered tunnel connection connIndex=0", None

            def terminate(self):
                self.returncode = 0

            def kill(self):
                self.returncode = -9

        def popen(command, **_kwargs):
            observed["command"] = command
            env_file = command[command.index("--env-file") + 1]
            observed["env"] = Path(env_file).read_text(encoding="utf-8")
            return FakeProcess()

        result = preflight.probe_cloudflare_tunnel(
            token,
            docker_prefix=["docker"],
            popen=popen,
        )
        self.assertTrue(result.ok)
        self.assertNotIn(token, " ".join(observed["command"]))
        self.assertEqual(observed["env"], f"TUNNEL_TOKEN={token}\n")


if __name__ == "__main__":
    unittest.main()
