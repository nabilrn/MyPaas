#!/usr/bin/env python3
"""Minimal host-side firewall helper for MyPaaS.

The API talks to this process through a Unix socket. The helper intentionally
supports only three operations: status, allow a MyPaaS-managed rule, and delete
that exact MyPaaS-managed rule. It never enables/disables UFW and never accepts
arbitrary command arguments.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import socket
import subprocess
from pathlib import Path

SOCKET_PATH = Path(os.environ.get("MYPAAS_FIREWALL_SOCKET", "/run/mypaas/firewall.sock"))
MARKER = "mypaas-managed"
PROTECTED_PORTS = {22, 80, 443}
MAX_REQUEST = 4096
PORT_PROTO_RE = re.compile(r"\b(?P<port>\d+)/(?:tcp|udp)\b", re.IGNORECASE)


def ufw_binary() -> str | None:
    for candidate in (shutil.which("ufw"), "/usr/sbin/ufw", "/usr/bin/ufw"):
        if candidate and Path(candidate).is_file():
            return candidate
    return None


def run_ufw(*args: str) -> subprocess.CompletedProcess[str]:
    binary = ufw_binary()
    if not binary:
        raise RuntimeError("ufw is not installed")
    return subprocess.run(
        [binary, *args],
        check=False,
        text=True,
        capture_output=True,
        timeout=10,
    )


def validate_rule(port: object, protocol: object) -> tuple[int, str]:
    if isinstance(port, bool):
        raise ValueError("invalid port")
    try:
        parsed_port = int(port)
    except (TypeError, ValueError) as exc:
        raise ValueError("invalid port") from exc
    if parsed_port < 1 or parsed_port > 65535:
        raise ValueError("port must be between 1 and 65535")
    if parsed_port in PROTECTED_PORTS:
        raise ValueError(f"port {parsed_port} is protected by MyPaaS")
    parsed_protocol = str(protocol or "").strip().lower()
    if parsed_protocol not in {"tcp", "udp"}:
        raise ValueError("protocol must be tcp or udp")
    return parsed_port, parsed_protocol


def ufw_active() -> bool:
    result = run_ufw("status")
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip() or "ufw status failed"
        raise RuntimeError(detail)
    return any(line.strip().lower() == "status: active" for line in result.stdout.splitlines())


def configured_rule_lines() -> list[str]:
    result = run_ufw("show", "added")
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip() or "ufw show added failed"
        raise RuntimeError(detail)
    return result.stdout.splitlines()


def managed_rules(lines: list[str]) -> list[dict[str, object]]:
    rules: list[dict[str, object]] = []
    seen: set[tuple[int, str]] = set()
    for line in lines:
        if MARKER not in line:
            continue
        match = PORT_PROTO_RE.search(line)
        if not match:
            continue
        port = int(match.group("port"))
        lowered = line.lower()
        protocol = "udp" if f"{port}/udp" in lowered else "tcp"
        key = (port, protocol)
        if key in seen:
            continue
        seen.add(key)
        rules.append({"port": port, "protocol": protocol})
    rules.sort(key=lambda item: (int(item["port"]), str(item["protocol"])))
    return rules


def current_managed_rules() -> list[dict[str, object]]:
    return managed_rules(configured_rule_lines())


def firewall_status() -> dict[str, object]:
    if not ufw_binary():
        return {"ok": True, "available": False, "active": False, "rules": []}
    return {"ok": True, "available": True, "active": ufw_active(), "rules": current_managed_rules()}


def allow_rule(port: int, protocol: str) -> dict[str, object]:
    if not ufw_binary():
        raise RuntimeError("ufw is not installed")
    if any(rule["port"] == port and rule["protocol"] == protocol for rule in current_managed_rules()):
        return {"ok": True}
    comment = f"{MARKER}:{port}/{protocol}"
    result = run_ufw("allow", f"{port}/{protocol}", "comment", comment)
    if result.returncode != 0:
        raise RuntimeError((result.stderr or result.stdout).strip() or "ufw allow failed")
    return {"ok": True}


def delete_rule(port: int, protocol: str) -> dict[str, object]:
    if not ufw_binary():
        raise RuntimeError("ufw is not installed")
    if not any(rule["port"] == port and rule["protocol"] == protocol for rule in current_managed_rules()):
        return {"ok": True}
    comment = f"{MARKER}:{port}/{protocol}"
    result = run_ufw("--force", "delete", "allow", f"{port}/{protocol}", "comment", comment)
    if result.returncode != 0:
        raise RuntimeError((result.stderr or result.stdout).strip() or "ufw delete failed")
    return {"ok": True}


def handle(request: object) -> dict[str, object]:
    if not isinstance(request, dict):
        raise ValueError("request must be an object")
    op = request.get("op")
    if op == "status":
        return firewall_status()
    if op in {"allow", "delete"}:
        port, protocol = validate_rule(request.get("port"), request.get("protocol"))
        return allow_rule(port, protocol) if op == "allow" else delete_rule(port, protocol)
    raise ValueError("unsupported operation")


def serve() -> None:
    SOCKET_PATH.parent.mkdir(parents=True, exist_ok=True)
    try:
        SOCKET_PATH.unlink()
    except FileNotFoundError:
        pass

    server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    server.bind(str(SOCKET_PATH))
    os.chmod(SOCKET_PATH, 0o660)
    server.listen(16)
    try:
        while True:
            conn, _ = server.accept()
            with conn:
                try:
                    raw = b""
                    while b"\n" not in raw and len(raw) <= MAX_REQUEST:
                        chunk = conn.recv(1024)
                        if not chunk:
                            break
                        raw += chunk
                    if len(raw) > MAX_REQUEST:
                        raise ValueError("request too large")
                    line = raw.split(b"\n", 1)[0]
                    request = json.loads(line.decode("utf-8"))
                    response = handle(request)
                except Exception as exc:  # bounded protocol surface; return sanitized error text
                    response = {"ok": False, "error": str(exc)[:512]}
                conn.sendall(json.dumps(response, separators=(",", ":")).encode("utf-8") + b"\n")
    finally:
        server.close()
        try:
            SOCKET_PATH.unlink()
        except FileNotFoundError:
            pass


if __name__ == "__main__":
    serve()
