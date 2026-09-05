from __future__ import annotations

import json
import os
import re
import secrets
import socket
import subprocess
import time
from typing import Callable
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlparse
from urllib.request import Request, urlopen


GITHUB_TOKEN_URL = "https://github.com/login/oauth/access_token"
CLOUDFLARE_IMAGE = "cloudflare/cloudflared:latest"
CLOUDFLARE_TOKEN_RE = re.compile(r"(?<![A-Za-z0-9_-])(eyJ[A-Za-z0-9._=-]{20,})(?![A-Za-z0-9_-])")
HOST_LABEL_RE = re.compile(r"^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$")


class PreflightError(ValueError):
    """Raised when supplied setup data is invalid or a live check fails."""


def validate_hostname(value: str) -> str:
    host = value.strip().lower().rstrip(".")
    if not host or len(host) > 253 or "." not in host:
        raise PreflightError("Enter a DNS hostname such as example.com")
    if "://" in host or "/" in host or ":" in host or "@" in host:
        raise PreflightError("Enter the hostname only, without a URL, path, port, or credentials")
    labels = host.split(".")
    if any(not HOST_LABEL_RE.fullmatch(label) for label in labels):
        raise PreflightError("The domain contains an invalid DNS label")
    return host


def check_domain(hostname: str, resolver: Callable[..., list] = socket.getaddrinfo) -> dict[str, object]:
    host = validate_hostname(hostname)
    try:
        records = resolver(host, 443, type=socket.SOCK_STREAM)
    except OSError as exc:
        raise PreflightError("The domain does not resolve from this VM yet") from exc

    addresses = sorted({record[4][0] for record in records if record and len(record) > 4 and record[4]})
    if not addresses:
        raise PreflightError("The domain resolved without an address")

    wildcard_host = f"mypaas-preflight-{secrets.token_hex(4)}.{host}"
    wildcard_resolved = False
    try:
        wildcard_records = resolver(wildcard_host, 443, type=socket.SOCK_STREAM)
        wildcard_resolved = any(record and len(record) > 4 and record[4] for record in wildcard_records)
    except OSError:
        wildcard_resolved = False

    return {
        "ok": True,
        "hostname": host,
        "addresses": addresses[:8],
        "wildcardResolved": wildcard_resolved,
        "message": (
            "Domain and wildcard DNS resolve from this VM."
            if wildcard_resolved
            else "Domain resolves. Wildcard DNS is not visible yet; configure it before deploying projects."
        ),
    }


def validate_https_callback(value: str) -> str:
    callback = value.strip()
    parsed = urlparse(callback)
    if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password or parsed.fragment:
        raise PreflightError("GitHub callback must be an HTTPS URL")
    return callback


def check_github_oauth(
    client_id: str,
    client_secret: str,
    callback_url: str,
    opener: Callable[..., object] = urlopen,
) -> dict[str, object]:
    client_id = client_id.strip()
    client_secret = client_secret.strip()
    callback = validate_https_callback(callback_url)
    if not client_id or not client_secret:
        raise PreflightError("GitHub Client ID and Client Secret are required")
    if any(char.isspace() for char in client_id + client_secret):
        raise PreflightError("GitHub credentials cannot contain whitespace")

    body = urlencode(
        {
            "client_id": client_id,
            "client_secret": client_secret,
            "code": "mypaas-preflight-invalid-code",
            "redirect_uri": callback,
        }
    ).encode("utf-8")
    request = Request(
        GITHUB_TOKEN_URL,
        data=body,
        method="POST",
        headers={
            "Accept": "application/json",
            "Content-Type": "application/x-www-form-urlencoded",
            "User-Agent": "MyPaas-Install-Wizard",
        },
    )

    try:
        with opener(request, timeout=8) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except HTTPError as exc:
        try:
            payload = json.loads(exc.read().decode("utf-8"))
        except Exception as decode_exc:
            raise PreflightError(f"GitHub returned HTTP {exc.code}") from decode_exc
    except (URLError, TimeoutError, OSError) as exc:
        raise PreflightError("Could not reach GitHub from this VM") from exc
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise PreflightError("GitHub returned an unreadable OAuth response") from exc

    error = str(payload.get("error", "")).strip()
    if error == "bad_verification_code":
        return {
            "ok": True,
            "credentialsAccepted": True,
            "callbackAccepted": True,
            "message": "GitHub accepted the OAuth app credentials and callback. Owner identity is verified during sign-in.",
        }
    if error == "incorrect_client_credentials":
        raise PreflightError("GitHub rejected the Client ID or Client Secret")
    if error == "redirect_uri_mismatch":
        raise PreflightError("The callback URL does not match the GitHub OAuth App")
    if not error and payload.get("access_token"):
        raise PreflightError("GitHub unexpectedly returned an access token during preflight")
    raise PreflightError("GitHub OAuth configuration could not be verified")


def extract_cloudflare_tunnel_token(value: str) -> str:
    raw = value.strip()
    match = CLOUDFLARE_TOKEN_RE.search(raw)
    if not match:
        raise PreflightError("Paste the Cloudflare Tunnel token or the Add a replica command containing the eyJ... token")
    return match.group(1)


def _container_cli() -> list[str]:
    candidates = (["docker"], ["sudo", "-n", "docker"])
    for prefix in candidates:
        try:
            result = subprocess.run(
                [*prefix, "version"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                timeout=5,
                check=False,
            )
        except (OSError, subprocess.SubprocessError):
            continue
        if result.returncode == 0:
            return list(prefix)
    raise PreflightError("Docker-compatible container access is unavailable to the install wizard")


def _free_loopback_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
        probe.bind(("127.0.0.1", 0))
        return int(probe.getsockname()[1])


def check_cloudflare_tunnel(
    token_or_command: str,
    *,
    timeout_seconds: float = 20.0,
    cli_resolver: Callable[[], list[str]] = _container_cli,
    readiness_opener: Callable[..., object] = urlopen,
) -> dict[str, object]:
    token = extract_cloudflare_tunnel_token(token_or_command)
    prefix = cli_resolver()
    metrics_port = _free_loopback_port()
    container_name = f"mypaas-wizard-tunnel-check-{os.getpid()}-{secrets.token_hex(3)}"
    command = [
        *prefix,
        "run",
        "--rm",
        "--name",
        container_name,
        "--network",
        "host",
        "-e",
        "TUNNEL_TOKEN",
        CLOUDFLARE_IMAGE,
        "tunnel",
        "--no-autoupdate",
        "--loglevel",
        "info",
        "--metrics",
        f"127.0.0.1:{metrics_port}",
        "run",
    ]
    environment = os.environ.copy()
    environment["TUNNEL_TOKEN"] = token
    process = None
    output = ""

    try:
        process = subprocess.Popen(
            command,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            env=environment,
        )
        deadline = time.monotonic() + timeout_seconds
        ready_url = f"http://127.0.0.1:{metrics_port}/ready"
        while time.monotonic() < deadline:
            if process.poll() is not None:
                break
            try:
                with readiness_opener(ready_url, timeout=1) as response:
                    if getattr(response, "status", 0) == 200:
                        return {
                            "ok": True,
                            "tokenDetected": True,
                            "message": "Cloudflare accepted the Tunnel token and the connector became ready.",
                        }
            except Exception:
                pass
            time.sleep(0.4)

        if process.poll() is not None and process.stdout is not None:
            output = process.stdout.read()[-4000:]
        sanitized = output.replace(token, "[redacted]").lower()
        if "token" in sanitized and any(word in sanitized for word in ("invalid", "parse", "failed", "error")):
            raise PreflightError("Cloudflare rejected the Tunnel token")
        if process.poll() is not None:
            raise PreflightError("cloudflared exited before the tunnel became ready")
        raise PreflightError("Cloudflare Tunnel did not become ready before the preflight timeout")
    except OSError as exc:
        raise PreflightError("Could not start the cloudflared validation container") from exc
    finally:
        if process is not None and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=4)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=2)
        try:
            subprocess.run(
                [*prefix, "rm", "-f", container_name],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                timeout=5,
                check=False,
            )
        except (OSError, subprocess.SubprocessError):
            pass
