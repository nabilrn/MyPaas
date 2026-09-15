#!/usr/bin/env python3
"""Deterministic checks for the current public/operator documentation contract.

This is intentionally a small set of high-signal invariants. Historical evidence is
allowed to preserve old commands/hostnames where explicitly allowlisted; current
operator surfaces must stay aligned with executable repository behavior.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
HISTORICAL_PERSONAL_HOST_ALLOWLIST = {
    Path("docs/ux/create-project-production-audit-2026-08-13.md"),
}

errors: list[str] = []


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def require(condition: bool, message: str) -> None:
    if not condition:
        errors.append(message)


def require_contains(relative: str, needle: str) -> None:
    require(needle in read(relative), f"{relative}: missing required contract text: {needle!r}")


def reject_contains(relative: str, needle: str) -> None:
    require(needle not in read(relative), f"{relative}: stale/unsupported contract text present: {needle!r}")


def extract_stable_sha(relative: str) -> str | None:
    match = re.search(r'^stable_sha="([0-9a-f]{40})"$', read(relative), flags=re.MULTILINE)
    if not match:
        errors.append(f'{relative}: stable install example must define stable_sha="<40-char-sha>"')
        return None
    return match.group(1)


def check_stable_install_contract() -> None:
    readme_sha = extract_stable_sha("README.md")
    install_sha = extract_stable_sha("docs/installation.md")
    if readme_sha and install_sha:
        require(readme_sha == install_sha, "README.md and docs/installation.md must pin the same stable source SHA")
        require(FULL_SHA.fullmatch(readme_sha) is not None, "stable install SHA must be a full lowercase Git SHA")

    for relative in ("README.md", "docs/installation.md"):
        text = read(relative)
        require(
            'https://raw.githubusercontent.com/nabilrn/MyPaas/${stable_sha}/scripts/bootstrap.sh' in text,
            f"{relative}: bootstrap download must be pinned through ${{stable_sha}}",
        )
        require(
            'MYPAAS_REF="$stable_sha" bash "$bootstrap_script"' in text,
            f"{relative}: bootstrap checkout must use the same exact stable SHA",
        )
        require("cp .env.example .env" not in text, f"{relative}: production install must not copy .env.example")
        require("MYPAAS_REF=v0.7.0" not in text, f"{relative}: release tag must not be used as bootstrap runtime authority")
        require(
            "release-identity hardening is tracked separately" not in text
            and "immutable release-identity verification requires a runtime/bootstrap change" not in text,
            f"{relative}: stale pre-hardening release-identity caveat remains",
        )

    bootstrap = read("scripts/bootstrap.sh")
    require("is_full_commit_sha()" in bootstrap, "scripts/bootstrap.sh: full-SHA bootstrap support is missing")
    require(
        'git -C "$INSTALL_DIR" checkout --detach FETCH_HEAD' in bootstrap,
        "scripts/bootstrap.sh: full-SHA bootstrap must use a detached verified checkout",
    )
    require(
        "does not match requested immutable SHA" in bootstrap,
        "scripts/bootstrap.sh: fetched full-SHA identity verification is missing",
    )


def check_mcp_contract() -> None:
    canonical = "go -C backend run ./cmd/mcp"
    stale = "go run ./backend/cmd/mcp"
    for relative in ("docs/mcp.md", ".agents/mcp/mypaas/README.md"):
        require_contains(relative, canonical)
        reject_contains(relative, stale)

    config = json.loads(read(".agents/mcp/mypaas/mcp_config.json"))
    server = config.get("mcpServers", {}).get("mypaas-server", {})
    require(server.get("command") == "go", ".agents/mcp/mypaas/mcp_config.json: MCP command must be go")
    require(
        server.get("args") == ["-C", "<path-to-MyPaas>/backend", "run", "./cmd/mcp"],
        ".agents/mcp/mypaas/mcp_config.json: MCP args must preserve the backend module working directory",
    )


def check_operator_commands() -> None:
    api = read("docs/api.md")
    require("GET /api/metrics" in api, "docs/api.md: public metrics path must be /api/metrics")

    update = read("docs/operations/update.md")
    require(
        "sudo systemctl start mypaas-update.service" in update,
        "docs/operations/update.md: manual production update must use mypaas-update.service",
    )
    require(
        "bash scripts/update-dispatch.sh" not in update,
        "docs/operations/update.md: do not document direct update-dispatch execution as an operator command",
    )

    migration = read("docs/operations/migration.md")
    require("bash scripts/install-migration.sh" in migration, "docs/operations/migration.md: protected migration helper is missing")
    require("--migrate-url" not in migration, "docs/operations/migration.md: remote migration URL must not be placed in argv")

    backup_tool = read("scripts/backup-restore.py").lower()
    require("beta" not in backup_tool, "scripts/backup-restore.py: current executable help/output must not use beta-readiness wording")

    verifier_docs = (
        "README.md",
        "docs/installation.md",
        "docs/operations/update.md",
        "docs/operations/backup-restore.md",
        "docs/operations/migration.md",
        "docs/STATD.md",
    )
    for relative in verifier_docs:
        text = read(relative)
        require("scripts/verify-production.sh" in text, f"{relative}: production verifier reference is missing")
        require(
            "sudo env ENV_FILE=" in text,
            f"{relative}: rootful production verification must preserve ENV_FILE through sudo",
        )


def check_audit_contract() -> None:
    for relative in (
        "frontend/playwright/audit/run-create-audit.mjs",
        "frontend/playwright/audit/create-project.spec.js",
    ):
        text = read(relative)
        require(
            "mode === 'production' && !process.env.MYPAAS_AUDIT_BASE_URL" in text
            or ("if (!process.env.MYPAAS_AUDIT_BASE_URL)" in text and "mode === 'production'" in text),
            f"{relative}: production audit entry point must fail closed without MYPAAS_AUDIT_BASE_URL",
        )
        require(
            "mode === 'production' && !process.env.MYPAAS_AUDIT_REPO_URL" in text,
            f"{relative}: production audit entry point must fail closed without MYPAAS_AUDIT_REPO_URL",
        )


def check_personal_host_hygiene() -> None:
    ignored_parts = {".git", "node_modules", ".svelte-kit", "artifacts"}
    for path in ROOT.rglob("*"):
        if not path.is_file() or any(part in ignored_parts for part in path.parts):
            continue
        relative = path.relative_to(ROOT)
        if relative in HISTORICAL_PERSONAL_HOST_ALLOWLIST:
            continue
        try:
            content = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue
        if "nabilrn.space" in content:
            errors.append(f"{relative}: personal production hostname is only allowed in explicit historical evidence")


def main() -> int:
    check_stable_install_contract()
    check_mcp_contract()
    check_operator_commands()
    check_audit_contract()
    check_personal_host_hygiene()

    if errors:
        print("Public/operator contract check failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print("Public/operator contract check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
