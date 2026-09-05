#!/usr/bin/env python3
"""Safely merge a MyPaas backup .env into the fresh-host installer config.

The restored file is parsed as data, restricted to the current MyPaas environment
contract, and never evaluated as shell syntax. The merged file stays in the
existing KEY=VALUE format; shell scripts consume it through scripts/load-env.sh.
"""

from __future__ import annotations

import argparse
import os
import pathlib
import re
import tempfile

KEY_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")

# Host/runtime placement is owned by the destination VM, not the backup.
PROTECTED_KEYS = frozenset(
    {
        "DOCKER_SOCKET",
        "DOCKER_HOST",
        "DOCKER_BIND_HOST",
        "CONTROL_NETWORK",
        "PROJECT_NETWORK",
        "ROUTING_NETWORK",
        "CADDY_ADMIN",
        "CADDY_UPSTREAM_HOST",
        "STATIC_ROOT",
        "CADDY_STATIC_ROOT",
        "STATD_SOCKET",
    }
)

# Settings supported by the backend but not necessarily present in .env.example.
EXTRA_ALLOWED_KEYS = frozenset(
    {
        "MYPAAS_API_TOKEN",
        "CLOUDFLARE_API_TOKEN",
        "CLOUDFLARE_ZONE_ID",
        "S3_ENDPOINT",
        "S3_BUCKET",
        "S3_ACCESS_KEY",
        "S3_SECRET_KEY",
        "S3_REGION",
    }
)

PROTECTED_DEFAULTS = {
    "DOCKER_SOCKET": "/var/run/docker.sock",
    "DOCKER_HOST": "",
    "PROJECT_NETWORK": "mypaas-projects",
    "CADDY_ADMIN": "unix//run/mypaas/caddy-admin.sock",
    "CADDY_UPSTREAM_HOST": "runtime",
    "STATIC_ROOT": "/var/lib/mypaas/static",
    "CADDY_STATIC_ROOT": "/var/lib/mypaas/static",
    "STATD_SOCKET": "/run/mypaas/statd.sock",
}


class RestoreEnvError(ValueError):
    pass


def _decode_value(raw: str, *, source: str, line_number: int) -> str:
    raw = raw.strip()
    if not raw:
        return ""
    if raw[0] in {"'", '"'}:
        quote = raw[0]
        if len(raw) < 2 or raw[-1] != quote:
            raise RestoreEnvError(f"{source}:{line_number}: unterminated quoted value")
        value = raw[1:-1]
        if quote == '"':
            value = (
                value.replace(r"\\", "\0")
                .replace(r'\"', '"')
                .replace(r"\$", "$")
                .replace(r"\`", "`")
                .replace("\0", "\\")
            )
        return value
    return raw


def parse_env(path: pathlib.Path, *, allowed_keys: set[str] | frozenset[str] | None = None) -> dict[str, str]:
    values: dict[str, str] = {}
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeDecodeError) as exc:
        raise RestoreEnvError(f"cannot read environment file {path}") from exc

    for line_number, raw_line in enumerate(lines, start=1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            raise RestoreEnvError(f"{path}:{line_number}: expected KEY=VALUE")
        key, raw_value = line.split("=", 1)
        key = key.strip()
        if not KEY_RE.fullmatch(key):
            raise RestoreEnvError(f"{path}:{line_number}: invalid environment key")
        if allowed_keys is not None and key not in allowed_keys:
            raise RestoreEnvError(f"{path}:{line_number}: unsupported restored setting {key}")
        if key in values:
            raise RestoreEnvError(f"{path}:{line_number}: duplicate environment key {key}")
        value = _decode_value(raw_value, source=str(path), line_number=line_number)
        if "\x00" in value or "\n" in value or "\r" in value:
            raise RestoreEnvError(f"{path}:{line_number}: environment values must be single-line text")
        values[key] = value
    return values


def allowed_keys_from_example(example_path: pathlib.Path) -> frozenset[str]:
    keys = set(parse_env(example_path).keys())
    keys.update(EXTRA_ALLOWED_KEYS)
    return frozenset(keys)


def dotenv_line(key: str, value: str) -> str:
    # Keep the on-disk grammar compatible with existing grep/cut and Compose
    # consumers. MyPaas-generated values are already single-line tokens/URLs;
    # reject characters whose dotenv interpretation would be ambiguous.
    if any(character in value for character in ("\n", "\r", "\x00", "'", '"', "#", "$")):
        raise RestoreEnvError(
            f"restored setting {key} contains characters that cannot be represented safely in MyPaas .env"
        )
    if value != value.strip() or re.search(r"\s", value):
        raise RestoreEnvError(f"restored setting {key} contains unsupported whitespace")
    return f"{key}={value}"


def merge_values(
    current: dict[str, str],
    restored: dict[str, str],
    *,
    protected_keys: frozenset[str] = PROTECTED_KEYS,
) -> dict[str, str]:
    merged = dict(restored)
    for key in protected_keys:
        if key in current:
            merged[key] = current[key]
        elif key in PROTECTED_DEFAULTS:
            merged[key] = PROTECTED_DEFAULTS[key]
        else:
            merged.pop(key, None)
    return merged


def write_env(path: pathlib.Path, values: dict[str, str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary_name = tempfile.mkstemp(prefix=path.name + ".merge-", dir=str(path.parent))
    temporary = pathlib.Path(temporary_name)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\n") as handle:
            for key in sorted(values):
                handle.write(dotenv_line(key, values[key]) + "\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temporary, 0o600)
        os.replace(temporary, path)
        os.chmod(path, 0o600)
    except Exception:
        try:
            temporary.unlink()
        except FileNotFoundError:
            pass
        raise


def merge_env_files(current_path: pathlib.Path, restored_path: pathlib.Path, example_path: pathlib.Path) -> None:
    allowed = allowed_keys_from_example(example_path)
    current = parse_env(current_path) if current_path.exists() else {}
    restored = parse_env(restored_path, allowed_keys=allowed)
    merged = merge_values(current, restored)

    required = {
        "POSTGRES_USER",
        "POSTGRES_PASSWORD",
        "POSTGRES_DB",
        "PUBLIC_DOMAIN",
        "OWNER_EMAIL",
        "GITHUB_CLIENT_ID",
        "GITHUB_CLIENT_SECRET",
        "GITHUB_CALLBACK_URL",
        "JWT_SECRET",
        "ENCRYPTION_KEY",
        "CLOUDFLARE_TUNNEL_TOKEN",
    }
    missing = sorted(key for key in required if not merged.get(key, "").strip())
    if missing:
        raise RestoreEnvError("restored environment is missing required settings: " + ", ".join(missing))

    write_env(current_path, merged)


def main() -> None:
    parser = argparse.ArgumentParser(description="Safely merge a MyPaas backup environment")
    parser.add_argument("--current", required=True, type=pathlib.Path)
    parser.add_argument("--restored", required=True, type=pathlib.Path)
    parser.add_argument("--example", required=True, type=pathlib.Path)
    args = parser.parse_args()
    try:
        merge_env_files(args.current, args.restored, args.example)
    except RestoreEnvError as exc:
        parser.exit(2, f"restore env error: {exc}\n")


if __name__ == "__main__":
    main()
