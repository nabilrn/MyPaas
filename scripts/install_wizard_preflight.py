#!/usr/bin/env python3
"""Non-destructive preflight checks for the standalone MyPaas installer."""

from __future__ import annotations

import json
import os
import re
import secrets
import socket
import subprocess
import tempfile
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import asdict, dataclass
from typing import Any, Callable


GITHUB_TOKEN_ENDPOINT = "https://github.com/login/oauth/access_token"
GITHUB_DOCS_URL = "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps"
CLOUDFLARE_TUNNEL_DOCS_URL = "https://developers.cloudflare.com/tunnel/advanced/tunnel-tokens/"
DOMAIN_LABEL = re.compile(r"^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$")
CLOUDFLARE_TOKEN = re.compile(r"^eyJ[A-Za-z0-9._~=-]{20,}$")
CLOUDFLARE_TOKEN_SEARCH = re.compile(r"(?<![A-Za-z0-9._~=-])(eyJ[A-Za-z0-9._~=-]{20,})(?![A-Za-z0-9._~=-])")


@dataclass(frozen=True)
class PreflightResult:
    ok: bool
    code: str
    message: str
    value: str = ""

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


def normalize_domain(value: str) -> str:
    value = value.strip()
    value = re.sub(r"^https?://", "", value, flags=re.IGNORECASE)
    value = value.split("/", 1)[0].strip().rstrip(".").lower()
    return value


def validate_domain(value: str) -> PreflightResult:
    domain = normalize_domain(value)
    if not domain:
        return PreflightResult(False, "domain_required", "Enter the public MyPaas domain.")
    if len(domain) > 253:
        return PreflightResult(False, "domain_too_long", "The domain is too long.")
    labels = domain.split(".")
    if len(labels) < 2 or any(not DOMAIN_LABEL.fullmatch(label) for label in labels):
        return PreflightResult(False, "domain_invalid", "Enter a valid hostname such as example.com.")
    return PreflightResult(True, "domain_valid", "Domain format is valid.", domain)


def check_domain_dns(value: str, resolver: Callable[..., Any] = socket.getaddrinfo) -> PreflightResult:
    validated = validate_domain(value)
    if not validated.ok:
        return validated
    try:
        records = resolver(validated.value, 443, type=socket.SOCK_STREAM)
    except OSError:
        return PreflightResult(
            False,
            "domain_dns_unresolved",
            "The domain does not resolve in public DNS yet.",
            validated.value,
        )
    if not records:
        return PreflightResult(
            False,
            "domain_dns_unresolved",
            "The domain does not resolve in public DNS yet.",
            validated.value,
        )
    return PreflightResult(True, "domain_dns_resolved", "Domain resolves in public DNS.", validated.value)


def validate_owner_email(value: str) -> PreflightResult:
    email = value.strip().lower()
    if not re.fullmatch(r"[^@\s]+@[^@\s]+\.[^@\s]+", email):
        return PreflightResult(False, "owner_email_invalid", "Enter a valid email address.")
    return PreflightResult(
        True,
        "owner_email_unverified",
        "Email format is valid. GitHub ownership is verified only after the first OAuth sign-in.",
        email,
    )


def _read_json_response(response: Any) -> dict[str, Any]:
    raw = response.read()
    if not raw:
        return {}
    decoded = json.loads(raw.decode("utf-8"))
    if not isinstance(decoded, dict):
        raise ValueError("response is not a JSON object")
    return decoded


def probe_github_oauth(
    client_id: str,
    client_secret: str,
    callback_url: str,
    *,
    open_request: Callable[..., Any] = urllib.request.urlopen,
    timeout_seconds: float = 10.0,
) -> PreflightResult:
    client_id = client_id.strip()
    client_secret = client_secret.strip()
    callback_url = callback_url.strip()
    if not client_id or not client_secret:
        return PreflightResult(False, "github_credentials_required", "Enter the GitHub OAuth Client ID and Client Secret.")
    try:
        callback = urllib.parse.urlparse(callback_url)
    except ValueError:
        return PreflightResult(False, "github_callback_invalid", "Enter a valid GitHub callback URL.")
    if callback.scheme != "https" or not callback.netloc or callback.path != "/api/auth/github/callback":
        return PreflightResult(
            False,
            "github_callback_invalid",
            "The callback URL must be https://<domain>/api/auth/github/callback.",
        )

    payload = urllib.parse.urlencode(
        {
            "client_id": client_id,
            "client_secret": client_secret,
            "code": "mypaas-preflight-" + secrets.token_urlsafe(12),
            "redirect_uri": callback_url,
        }
    ).encode("utf-8")
    request = urllib.request.Request(
        GITHUB_TOKEN_ENDPOINT,
        data=payload,
        headers={
            "Accept": "application/json",
            "Content-Type": "application/x-www-form-urlencoded",
            "User-Agent": "MyPaas-Install-Wizard",
        },
        method="POST",
    )

    try:
        with open_request(request, timeout=timeout_seconds) as response:
            result = _read_json_response(response)
    except urllib.error.HTTPError as exc:
        try:
            result = _read_json_response(exc)
        except (ValueError, json.JSONDecodeError, UnicodeDecodeError):
            return PreflightResult(False, "github_unreachable", "GitHub rejected the preflight request.")
    except (urllib.error.URLError, TimeoutError, OSError):
        return PreflightResult(False, "github_unreachable", "Could not reach GitHub from this VM.")
    except (ValueError, json.JSONDecodeError, UnicodeDecodeError):
        return PreflightResult(False, "github_unexpected_response", "GitHub returned an unexpected response.")

    error = str(result.get("error", ""))
    if error == "bad_verification_code":
        return PreflightResult(
            True,
            "github_oauth_valid",
            "GitHub accepted the OAuth app credentials and callback URL.",
        )
    if error == "incorrect_client_credentials":
        return PreflightResult(False, error, "GitHub rejected the Client ID or Client Secret.")
    if error == "redirect_uri_mismatch":
        return PreflightResult(False, error, "The callback URL does not match the GitHub OAuth App.")
    if error == "application_suspended":
        return PreflightResult(False, error, "The GitHub OAuth App is suspended.")
    if not error and result.get("access_token"):
        # A random one-use code should never produce a token. Do not expose it if
        # GitHub ever changes behavior unexpectedly.
        return PreflightResult(False, "github_unexpected_token", "GitHub returned an unexpected token response.")
    return PreflightResult(False, "github_unexpected_response", "GitHub could not confirm the OAuth configuration.")


def extract_cloudflare_tunnel_token(value: str) -> PreflightResult:
    raw = value.strip()
    if CLOUDFLARE_TOKEN.fullmatch(raw):
        return PreflightResult(True, "cloudflare_token_detected", "Tunnel token detected.", raw)
    match = CLOUDFLARE_TOKEN_SEARCH.search(raw)
    if match:
        return PreflightResult(True, "cloudflare_token_detected", "Tunnel token detected from the Cloudflare command.", match.group(1))
    return PreflightResult(
        False,
        "cloudflare_token_invalid_format",
        "Paste the Tunnel token or the Cloudflare command containing the eyJ... token.",
    )


def classify_cloudflared_probe(output: str, returncode: int | None, timed_out: bool) -> PreflightResult:
    lowered = output.lower()
    if "registered tunnel connection" in lowered or ("connection" in lowered and " registered" in lowered):
        return PreflightResult(True, "cloudflare_tunnel_valid", "Cloudflare accepted the Tunnel token and registered a connection.")
    invalid_markers = (
        "failed to parse token",
        "invalid tunnel token",
        "unauthorized",
        "authentication failed",
        "tunnel token is invalid",
    )
    if any(marker in lowered for marker in invalid_markers):
        return PreflightResult(False, "cloudflare_tunnel_rejected", "Cloudflare rejected the Tunnel token.")
    if timed_out:
        return PreflightResult(
            False,
            "cloudflare_tunnel_unconfirmed",
            "The token format is valid, but the VM could not confirm a Cloudflare connection in time.",
        )
    if returncode not in (None, 0):
        return PreflightResult(False, "cloudflare_probe_failed", "cloudflared could not start the Tunnel probe.")
    return PreflightResult(False, "cloudflare_tunnel_unconfirmed", "Cloudflare connection was not confirmed.")


def _docker_prefix() -> list[str] | None:
    probes = (["docker"], ["sudo", "-n", "docker"])
    for prefix in probes:
        try:
            result = subprocess.run(
                [*prefix, "ps"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
                timeout=5,
            )
        except (OSError, subprocess.TimeoutExpired):
            continue
        if result.returncode == 0:
            return list(prefix)
    return None


def probe_cloudflare_tunnel(
    value: str,
    *,
    timeout_seconds: float = 12.0,
    docker_prefix: list[str] | None = None,
    popen: Callable[..., subprocess.Popen[str]] = subprocess.Popen,
) -> PreflightResult:
    extracted = extract_cloudflare_tunnel_token(value)
    if not extracted.ok:
        return extracted

    prefix = list(docker_prefix) if docker_prefix is not None else _docker_prefix()
    if not prefix:
        return PreflightResult(False, "container_runtime_unavailable", "The installer cannot access the container runtime.")

    env_file = tempfile.NamedTemporaryFile("w", prefix="mypaas-cf-token-", delete=False, encoding="utf-8")
    try:
        os.chmod(env_file.name, 0o600)
        env_file.write("TUNNEL_TOKEN=" + extracted.value + "\n")
        env_file.close()
        command = [
            *prefix,
            "run",
            "--rm",
            "--network",
            "host",
            "--env-file",
            env_file.name,
            "cloudflare/cloudflared:latest",
            "tunnel",
            "--no-autoupdate",
            "run",
        ]
        try:
            process = popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
            )
        except OSError:
            return PreflightResult(False, "cloudflare_probe_failed", "Could not start cloudflared for validation.")

        timed_out = False
        try:
            output, _ = process.communicate(timeout=timeout_seconds)
        except subprocess.TimeoutExpired:
            timed_out = True
            process.terminate()
            try:
                output, _ = process.communicate(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
                output, _ = process.communicate()
        return classify_cloudflared_probe(output or "", process.returncode, timed_out)
    finally:
        try:
            env_file.close()
        except Exception:
            pass
        try:
            os.remove(env_file.name)
        except FileNotFoundError:
            pass
