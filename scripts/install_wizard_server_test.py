import http.client
import io
import json
import tarfile
import tempfile
import threading
import unittest
from http.server import HTTPServer
from pathlib import Path
from unittest.mock import patch

import install_wizard_server as server
from install_wizard_preflight import PreflightResult


ROOT_DIR = Path(__file__).resolve().parents[1]
WIZARD_PATH = ROOT_DIR / "scripts" / "install-wizard.py"


def backup_bytes() -> bytes:
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w:gz") as archive:
        for name, body in (
            ("database.sql", b"select 1;\n"),
            (".env", b"PUBLIC_DOMAIN=example.com\n"),
        ):
            info = tarfile.TarInfo(name)
            info.size = len(body)
            info.mode = 0o600
            archive.addfile(info, io.BytesIO(body))
    return buffer.getvalue()


class InstallWizardServerTest(unittest.TestCase):
    def start_server(self, restore_path: Path, *, max_backup_bytes: int = 1024 * 1024):
        wizard = server.load_wizard(str(WIZARD_PATH))
        wizard.TOKEN = "master-token"
        handler = server.make_handler(
            wizard,
            session_token="session-token",
            restore_path=str(restore_path),
            max_backup_bytes=max_backup_bytes,
            shutdown_delay_seconds=0,
        )
        httpd = HTTPServer(("127.0.0.1", 0), handler)
        thread = threading.Thread(target=httpd.serve_forever, daemon=True)
        thread.start()
        return httpd, thread

    def authenticated_cookie(self, host: str, port: int) -> str:
        conn = http.client.HTTPConnection(host, port, timeout=5)
        conn.request("GET", "/?token=master-token")
        response = conn.getresponse()
        response.read()
        self.assertEqual(response.status, 200)
        cookie = response.getheader("Set-Cookie").split(";", 1)[0]
        conn.close()
        return cookie

    def test_backup_upload_requires_session_from_authenticated_wizard_get(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            httpd, thread = self.start_server(destination)
            host, port = httpd.server_address
            payload = backup_bytes()
            try:
                conn = http.client.HTTPConnection(host, port, timeout=5)
                conn.request(
                    "POST",
                    "/upload-backup",
                    body=payload,
                    headers={"Content-Type": "application/octet-stream"},
                )
                response = conn.getresponse()
                response.read()
                self.assertEqual(response.status, 403)
                self.assertFalse(destination.exists())
                conn.close()

                cookie = self.authenticated_cookie(host, port)
                self.assertEqual(cookie, f"{server.SESSION_COOKIE}=session-token")

                conn = http.client.HTTPConnection(host, port, timeout=5)
                conn.request(
                    "POST",
                    "/upload-backup",
                    body=payload,
                    headers={
                        "Content-Type": "application/octet-stream",
                        "Cookie": cookie,
                    },
                )
                response = conn.getresponse()
                response.read()
                self.assertEqual(response.status, 200)
                conn.close()
                self.assertEqual(destination.read_bytes(), payload)
            finally:
                httpd.shutdown()
                httpd.server_close()
                thread.join(timeout=5)

    def test_backup_upload_returns_413_before_accepting_oversized_body(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            httpd, thread = self.start_server(destination, max_backup_bytes=10)
            host, port = httpd.server_address
            try:
                cookie = self.authenticated_cookie(host, port)
                conn = http.client.HTTPConnection(host, port, timeout=5)
                conn.request(
                    "POST",
                    "/upload-backup",
                    body=b"01234567890",
                    headers={"Cookie": cookie},
                )
                response = conn.getresponse()
                response.read()
                self.assertEqual(response.status, 413)
                self.assertFalse(destination.exists())
                conn.close()
            finally:
                httpd.shutdown()
                httpd.server_close()
                thread.join(timeout=5)

    def test_backup_upload_rejects_invalid_archive(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            httpd, thread = self.start_server(destination)
            host, port = httpd.server_address
            try:
                cookie = self.authenticated_cookie(host, port)
                conn = http.client.HTTPConnection(host, port, timeout=5)
                conn.request(
                    "POST",
                    "/upload-backup",
                    body=b"not-a-tarball",
                    headers={"Cookie": cookie},
                )
                response = conn.getresponse()
                response.read()
                self.assertEqual(response.status, 400)
                self.assertFalse(destination.exists())
                conn.close()
            finally:
                httpd.shutdown()
                httpd.server_close()
                thread.join(timeout=5)

    def test_preflight_endpoints_require_authenticated_wizard_session(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            httpd, thread = self.start_server(destination)
            host, port = httpd.server_address
            try:
                conn = http.client.HTTPConnection(host, port, timeout=5)
                conn.request(
                    "POST",
                    "/preflight/domain",
                    body=json.dumps({"domain": "example.com"}),
                    headers={"Content-Type": "application/json"},
                )
                response = conn.getresponse()
                response.read()
                self.assertEqual(response.status, 403)
                conn.close()
            finally:
                httpd.shutdown()
                httpd.server_close()
                thread.join(timeout=5)

    def test_github_preflight_response_never_echoes_submitted_secret(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            httpd, thread = self.start_server(destination)
            host, port = httpd.server_address
            try:
                cookie = self.authenticated_cookie(host, port)
                with patch.object(
                    server,
                    "probe_github_oauth",
                    return_value=PreflightResult(True, "github_oauth_valid", "GitHub OAuth configuration is valid."),
                ):
                    conn = http.client.HTTPConnection(host, port, timeout=5)
                    body = json.dumps(
                        {
                            "clientId": "client-id",
                            "clientSecret": "never-echo-this-secret",
                            "callbackUrl": "https://example.com/api/auth/github/callback",
                            "ownerEmail": "owner@example.com",
                        }
                    )
                    conn.request(
                        "POST",
                        "/preflight/github",
                        body=body,
                        headers={"Content-Type": "application/json", "Cookie": cookie},
                    )
                    response = conn.getresponse()
                    response_body = response.read().decode("utf-8")
                    self.assertEqual(response.status, 200)
                    self.assertNotIn("never-echo-this-secret", response_body)
                    decoded = json.loads(response_body)
                    self.assertEqual(decoded["oauth"]["code"], "github_oauth_valid")
                    self.assertEqual(decoded["ownerEmail"]["code"], "owner_email_unverified")
                    conn.close()
            finally:
                httpd.shutdown()
                httpd.server_close()
                thread.join(timeout=5)

    def test_cloudflare_preflight_response_does_not_echo_tunnel_token(self) -> None:
        token = "eyJ" + "C" * 64
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            httpd, thread = self.start_server(destination)
            host, port = httpd.server_address
            try:
                cookie = self.authenticated_cookie(host, port)
                with patch.object(
                    server,
                    "probe_cloudflare_tunnel",
                    return_value=PreflightResult(True, "cloudflare_tunnel_valid", "Cloudflare accepted the Tunnel token."),
                ):
                    conn = http.client.HTTPConnection(host, port, timeout=5)
                    conn.request(
                        "POST",
                        "/preflight/cloudflare",
                        body=json.dumps({"token": token}),
                        headers={"Content-Type": "application/json", "Cookie": cookie},
                    )
                    response = conn.getresponse()
                    response_body = response.read().decode("utf-8")
                    self.assertEqual(response.status, 200)
                    self.assertNotIn(token, response_body)
                    decoded = json.loads(response_body)
                    self.assertEqual(decoded["format"]["code"], "cloudflare_token_detected")
                    self.assertEqual(decoded["connection"]["code"], "cloudflare_tunnel_valid")
                    conn.close()
            finally:
                httpd.shutdown()
                httpd.server_close()
                thread.join(timeout=5)


if __name__ == "__main__":
    unittest.main()
