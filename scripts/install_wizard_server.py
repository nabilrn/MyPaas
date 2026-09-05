#!/usr/bin/env python3
"""Hardened HTTP entrypoint for the standalone MyPaas install wizard."""

from __future__ import annotations

import http.cookies
import importlib.util
import json
import os
import pathlib
import secrets
import sys
import tempfile
import threading
import time
from http.server import HTTPServer
from types import ModuleType
from urllib.parse import parse_qs, urlparse

from install_wizard_preflight import (
    check_domain_dns,
    extract_cloudflare_tunnel_token,
    probe_cloudflare_tunnel,
    probe_github_oauth,
    validate_domain,
    validate_owner_email,
)
from install_wizard_security import (
    BackupUploadError,
    DEFAULT_MAX_BACKUP_BYTES,
    DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
    store_backup_upload,
)

SESSION_COOKIE = "mypaas_wizard_session"
DEFAULT_RESTORE_PATH = "/tmp/mypaas-restore.tar.gz"
MAX_JSON_BODY_BYTES = 64 * 1024


def positive_int_env(name: str, fallback: int) -> int:
    raw = os.environ.get(name, str(fallback)).strip()
    try:
        value = int(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be a positive integer") from exc
    if value <= 0:
        raise ValueError(f"{name} must be a positive integer")
    return value


def load_wizard(path: str) -> ModuleType:
    spec = importlib.util.spec_from_file_location("mypaas_install_wizard", path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load install wizard: {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def public_result(result) -> dict:
    data = result.to_dict()
    data.pop("value", None)
    return data


def stage_restore_host_config(wizard: ModuleType) -> None:
    """Persist only destination-host placement before restored config is merged."""
    env_path = pathlib.Path(wizard.ENV_FILE)
    env_path.parent.mkdir(parents=True, exist_ok=True)
    values = {
        "DOCKER_SOCKET": "/var/run/docker.sock",
        "DOCKER_HOST": "",
        "DOCKER_BIND_HOST": str(wizard.DEFAULTS.get("DOCKER_BIND_HOST", "127.0.0.1")),
        "PROJECT_NETWORK": str(wizard.DEFAULTS.get("PROJECT_NETWORK", "mypaas-projects")),
        "STATIC_ROOT": "/var/lib/mypaas/static",
        "CADDY_STATIC_ROOT": "/var/lib/mypaas/static",
        "STATD_SOCKET": "/run/mypaas/statd.sock",
    }
    fd, temporary_name = tempfile.mkstemp(prefix=env_path.name + ".restore-host-", dir=str(env_path.parent))
    temporary = pathlib.Path(temporary_name)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\n") as handle:
            for key, value in values.items():
                if "'" in value or "\n" in value or "\r" in value or "\x00" in value:
                    raise ValueError(f"unsafe destination-host value for {key}")
                handle.write(f"{key}='{value}'\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temporary, 0o600)
        os.replace(temporary, env_path)
        os.chmod(env_path, 0o600)
    except Exception:
        try:
            temporary.unlink()
        except FileNotFoundError:
            pass
        raise


def make_handler(
    wizard: ModuleType,
    *,
    session_token: str,
    restore_path: str = DEFAULT_RESTORE_PATH,
    max_backup_bytes: int = DEFAULT_MAX_BACKUP_BYTES,
    max_expanded_backup_bytes: int = DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
    shutdown_delay_seconds: float = 3.0,
):
    class SecureWizardHandler(wizard.Handler):
        def send_text(self, message: str, status: int) -> None:
            body = message.encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.send_header("Cache-Control", "no-store")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def send_json(self, payload: dict, status: int = 200) -> None:
            body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Cache-Control", "no-store")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def read_json(self) -> dict | None:
            raw_length = self.headers.get("Content-Length", "").strip()
            try:
                length = int(raw_length)
            except ValueError:
                self.send_json({"error": "A valid Content-Length is required."}, 400)
                return None
            if length <= 0 or length > MAX_JSON_BODY_BYTES:
                self.send_json({"error": "Preflight request body is empty or too large."}, 400)
                return None
            try:
                payload = json.loads(self.rfile.read(length).decode("utf-8"))
            except (UnicodeDecodeError, json.JSONDecodeError):
                self.send_json({"error": "Preflight request must be valid JSON."}, 400)
                return None
            if not isinstance(payload, dict):
                self.send_json({"error": "Preflight request must be a JSON object."}, 400)
                return None
            return payload

        def send_wizard_with_session(self) -> None:
            body = wizard.form_html()
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Cache-Control", "no-store")
            self.send_header(
                "Set-Cookie",
                f"{SESSION_COOKIE}={session_token}; Path=/; HttpOnly; SameSite=Strict; Max-Age=1800",
            )
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def has_session(self) -> bool:
            raw_cookie = self.headers.get("Cookie", "")
            if not raw_cookie:
                return False
            parsed = http.cookies.SimpleCookie()
            try:
                parsed.load(raw_cookie)
            except http.cookies.CookieError:
                return False
            morsel = parsed.get(SESSION_COOKIE)
            return morsel is not None and secrets.compare_digest(morsel.value, session_token)

        def do_GET(self) -> None:
            parsed = urlparse(self.path)
            if parsed.path in {"/brand/logo.svg", "/health"}:
                super().do_GET()
                return
            query = parse_qs(parsed.query)
            supplied = query.get("token", [""])[0]
            if not secrets.compare_digest(supplied, wizard.TOKEN):
                self.send_html(
                    wizard.form_html("Invalid or missing wizard token. Use the URL printed by install-vm.sh."),
                    403,
                )
                return
            self.send_wizard_with_session()

        def handle_preflight(self, path: str) -> None:
            if not self.has_session():
                self.send_json({"error": "Invalid or expired wizard session."}, 403)
                return
            payload = self.read_json()
            if payload is None:
                return

            if path == "/preflight/domain":
                raw_domain = str(payload.get("domain", ""))
                format_result = validate_domain(raw_domain)
                dns_result = check_domain_dns(raw_domain) if format_result.ok else format_result
                self.send_json(
                    {
                        "format": public_result(format_result),
                        "dns": public_result(dns_result),
                    }
                )
                return

            if path == "/preflight/github":
                owner_result = validate_owner_email(str(payload.get("ownerEmail", "")))
                oauth_result = probe_github_oauth(
                    str(payload.get("clientId", "")),
                    str(payload.get("clientSecret", "")),
                    str(payload.get("callbackUrl", "")),
                )
                self.send_json(
                    {
                        "oauth": public_result(oauth_result),
                        "ownerEmail": public_result(owner_result),
                    }
                )
                return

            if path == "/preflight/cloudflare":
                raw_token = str(payload.get("token", ""))
                parsed_token = extract_cloudflare_tunnel_token(raw_token)
                probe_result = probe_cloudflare_tunnel(parsed_token.value) if parsed_token.ok else parsed_token
                self.send_json(
                    {
                        "format": public_result(parsed_token),
                        "connection": public_result(probe_result),
                    }
                )
                return

            self.send_json({"error": "Unknown preflight check."}, 404)

        def do_POST(self) -> None:
            parsed = urlparse(self.path)
            if parsed.path.startswith("/preflight/"):
                self.handle_preflight(parsed.path)
                return
            if parsed.path != "/upload-backup":
                super().do_POST()
                return

            if not self.has_session():
                self.send_text("Invalid or expired wizard session.", 403)
                return

            raw_length = self.headers.get("Content-Length", "").strip()
            try:
                content_length = int(raw_length)
            except ValueError:
                self.send_text("Backup upload requires a valid Content-Length.", 400)
                return

            try:
                store_backup_upload(
                    self.rfile,
                    content_length,
                    restore_path,
                    max_upload_bytes=max_backup_bytes,
                    max_expanded_bytes=max_expanded_backup_bytes,
                )
                stage_restore_host_config(wizard)
            except BackupUploadError as exc:
                self.send_text(str(exc), 400 if content_length <= max_backup_bytes else 413)
                return
            except (OSError, ValueError):
                self.send_text("Could not stage the validated backup for restore.", 500)
                return

            self.send_html(
                wizard.success_html(
                    title="Backup uploaded",
                    message="The MyPaas backup was validated and staged for restore.",
                )
            )

            def delayed_shutdown() -> None:
                if shutdown_delay_seconds > 0:
                    time.sleep(shutdown_delay_seconds)
                self.server.shutdown()

            threading.Thread(target=delayed_shutdown, daemon=True).start()

    return SecureWizardHandler


def main(argv: list[str] | None = None) -> None:
    argv = list(sys.argv[1:] if argv is None else argv)
    if len(argv) != 1:
        raise SystemExit("usage: install_wizard_server.py <install-wizard.py>")

    wizard = load_wizard(argv[0])
    max_backup_bytes = positive_int_env("WIZARD_MAX_BACKUP_BYTES", DEFAULT_MAX_BACKUP_BYTES)
    max_expanded_backup_bytes = positive_int_env(
        "WIZARD_MAX_EXPANDED_BACKUP_BYTES",
        DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
    )
    restore_path = os.environ.get("WIZARD_RESTORE_PATH", DEFAULT_RESTORE_PATH)
    session_token = secrets.token_urlsafe(32)
    handler = make_handler(
        wizard,
        session_token=session_token,
        restore_path=restore_path,
        max_backup_bytes=max_backup_bytes,
        max_expanded_backup_bytes=max_expanded_backup_bytes,
    )
    server = HTTPServer((wizard.HOST, wizard.PORT), handler)
    server.serve_forever()


if __name__ == "__main__":
    main()
