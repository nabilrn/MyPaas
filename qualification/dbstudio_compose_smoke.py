#!/usr/bin/env python3
"""Controlled MyPaas DB Studio smoke for PostgreSQL, MySQL, and MariaDB Compose projects."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import pathlib
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass
from typing import Any

from fixture_ref import FixtureRefError, resolve_fixture_ref

ENGINES = (
    ("postgres", "postgres", "qualification/fixtures/dbstudio/postgres"),
    ("mysql", "mysql", "qualification/fixtures/dbstudio/mysql"),
    ("mariadb", "mariadb", "qualification/fixtures/dbstudio/mariadb"),
)
TERMINAL = {"running", "failed", "stopped", "rolled_back"}


class SmokeError(RuntimeError):
    pass


@dataclass
class EngineResult:
    engine: str
    expected_driver: str
    project_id: str | None = None
    deployment_id: str | None = None
    deployment_status: str = "not_started"
    configured: bool = False
    connected: bool = False
    actual_driver: str = ""
    write_access_absent: bool = False
    schemas_readable: bool = False
    seconds: float = 0.0
    error: str | None = None


class Client:
    def __init__(self, base_url: str, token: str, timeout: float = 60.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.timeout = timeout

    def request(self, method: str, path: str, payload: dict[str, Any] | None = None) -> Any:
        body = None
        headers = {"Accept": "application/json", "Authorization": f"Bearer {self.token}"}
        if payload is not None:
            body = json.dumps(payload).encode("utf-8")
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(self.base_url + path, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")[:1000]
            raise SmokeError(f"{method} {path}: HTTP {exc.code}: {sanitize(detail)}") from exc
        except urllib.error.URLError as exc:
            raise SmokeError(f"{method} {path}: {exc.reason}") from exc
        if not raw:
            return None
        value = json.loads(raw.decode("utf-8"))
        return value.get("data") if isinstance(value, dict) and "data" in value else value


def sanitize(value: str) -> str:
    lowered = value.lower()
    if any(key in lowered for key in ("password", "secret", "token", "authorization")):
        return "<redacted-error>"
    return value[:1000]


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def project_prefix(timestamp: str) -> str:
    compact = "".join(c for c in timestamp.lower() if c.isalnum())
    return f"beta-db-{compact[:12]}"


def fixture_payload(engine: str, base_directory: str, prefix: str, repo: str, branch: str) -> dict[str, Any]:
    return {
        "name": f"{prefix}-{engine}",
        "sourceType": "git",
        "repoUrl": repo,
        "branch": branch,
        "deployMode": "compose",
        "resourceProfile": "compose-main",
        "mainService": "app",
        "appPort": 80,
        "baseDirectory": base_directory,
        "composeFilePath": "compose.yml",
    }


def wait_deployment(client: Client, deployment_id: str, timeout: float, poll: float) -> dict[str, Any]:
    deadline = time.monotonic() + timeout
    last: dict[str, Any] = {}
    while time.monotonic() < deadline:
        value = client.request("GET", f"/api/deployments/{deployment_id}")
        if isinstance(value, dict):
            last = value
            if value.get("status") in TERMINAL:
                return value
        time.sleep(poll)
    raise SmokeError(f"deployment {deployment_id} did not settle; last status={last.get('status', 'unknown')}")


def wait_dbstudio(client: Client, project_id: str, timeout: float, poll: float) -> dict[str, Any]:
    deadline = time.monotonic() + timeout
    last: dict[str, Any] = {}
    while time.monotonic() < deadline:
        value = client.request("GET", f"/api/projects/{project_id}/db/status")
        if isinstance(value, dict):
            last = value
            if value.get("configured") and value.get("connected"):
                return value
        time.sleep(poll)
    message = sanitize(str(last.get("message") or "DB Studio did not connect"))
    raise SmokeError(message)


def evaluate_status(status: dict[str, Any], expected_driver: str) -> list[str]:
    failures: list[str] = []
    if status.get("configured") is not True:
        failures.append("DB Studio did not report configured=true")
    if status.get("connected") is not True:
        failures.append("DB Studio did not report connected=true")
    connection = status.get("connection") if isinstance(status.get("connection"), dict) else {}
    if connection.get("driver") != expected_driver:
        failures.append(f"driver={connection.get('driver')!r}, expected {expected_driver!r}")
    if status.get("writeAccess") is not None:
        failures.append("writeAccess was active without an explicit write session")
    return failures


def run_engine(
    client: Client,
    engine: str,
    expected_driver: str,
    base_directory: str,
    prefix: str,
    repo: str,
    branch: str,
    timeout: float,
    poll: float,
) -> EngineResult:
    result = EngineResult(engine=engine, expected_driver=expected_driver)
    started = time.monotonic()
    try:
        project = client.request("POST", "/api/projects/", fixture_payload(engine, base_directory, prefix, repo, branch))
        if not isinstance(project, dict) or not project.get("id"):
            raise SmokeError("project create response did not include an id")
        result.project_id = str(project["id"])
        deployment = client.request("POST", f"/api/projects/{result.project_id}/deploy")
        if not isinstance(deployment, dict) or not deployment.get("id"):
            raise SmokeError("deployment trigger response did not include an id")
        result.deployment_id = str(deployment["id"])
        final = wait_deployment(client, result.deployment_id, timeout, poll)
        result.deployment_status = str(final.get("status", "unknown"))
        if result.deployment_status != "running":
            raise SmokeError(sanitize(str(final.get("errorMsg") or f"deployment ended as {result.deployment_status}")))

        status = wait_dbstudio(client, result.project_id, timeout, poll)
        failures = evaluate_status(status, expected_driver)
        result.configured = bool(status.get("configured"))
        result.connected = bool(status.get("connected"))
        connection = status.get("connection") if isinstance(status.get("connection"), dict) else {}
        result.actual_driver = str(connection.get("driver") or "")
        result.write_access_absent = status.get("writeAccess") is None
        if failures:
            raise SmokeError("; ".join(failures))

        schemas = client.request("GET", f"/api/projects/{result.project_id}/db/schemas")
        if not isinstance(schemas, list):
            raise SmokeError("DB Studio schemas endpoint did not return an array")
        result.schemas_readable = True
    except Exception as exc:
        result.error = sanitize(str(exc))
    finally:
        result.seconds = time.monotonic() - started
    return result


def markdown(report: dict[str, Any]) -> str:
    lines = [
        "# DB Studio Compose smoke report",
        "",
        f"- Run ID: `{report['runId']}`",
        f"- Git SHA: `{report['gitSha']}`",
        f"- Fixture branch: `{report.get('fixtureBranch', report['fixtureRef'])}`",
        f"- Fixture resolved SHA: `{report.get('fixtureResolvedSha', 'unknown')}`",
        f"- Result: **{'PASS' if report['pass'] else 'FAIL'}**",
        "",
        "| Engine | Deployment | Configured | Connected | Driver | Read-only default | Schemas | Result |",
        "| --- | --- | --- | --- | --- | --- | --- | --- |",
    ]
    for row in report["engines"]:
        ok = row["error"] is None
        lines.append(
            f"| {row['engine']} | {row['deployment_status']} | {row['configured']} | {row['connected']} | "
            f"{row['actual_driver'] or 'n/a'} | {row['write_access_absent']} | {row['schemas_readable']} | {'PASS' if ok else 'FAIL'} |"
        )
    lines.append("")
    if not report["pass"]:
        lines.append("## Failures")
        lines.append("")
        for row in report["engines"]:
            if row["error"]:
                lines.append(f"- {row['engine']}: {row['error']}")
        lines.append("")
    return "\n".join(lines)


def write_report(report: dict[str, Any], output: pathlib.Path) -> None:
    output.mkdir(parents=True, exist_ok=True)
    (output / "report.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    (output / "report.md").write_text(markdown(report), encoding="utf-8")


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base-url", default=os.getenv("MYPAAS_BETA_BASE_URL", ""))
    p.add_argument("--token", default=os.getenv("MYPAAS_API_TOKEN", ""))
    p.add_argument("--fixture-repo", default=os.getenv("MYPAAS_BETA_FIXTURE_REPO", "https://github.com/nabilrn/MyPaas"))
    p.add_argument("--fixture-ref", default=os.getenv("MYPAAS_BETA_FIXTURE_REF", "main"), help="immutable SHA or branch identity to verify")
    p.add_argument("--fixture-branch", default=os.getenv("MYPAAS_BETA_FIXTURE_BRANCH", ""), help="optional clonable branch; required only when auto-resolution is ambiguous")
    p.add_argument("--git-sha", default=os.getenv("MYPAAS_BETA_GIT_SHA", "unknown"))
    p.add_argument("--timeout", type=float, default=900)
    p.add_argument("--poll-seconds", type=float, default=3)
    p.add_argument("--output", default="")
    p.add_argument("--plan", action="store_true")
    p.add_argument("--confirm-destructive", action="store_true")
    return p


def main(argv: list[str] | None = None) -> int:
    args = parser().parse_args(argv)
    if args.plan:
        print(json.dumps({
            "engines": [engine for engine, _, _ in ENGINES],
            "fixtureRepo": args.fixture_repo,
            "fixtureRef": args.fixture_ref,
            "fixtureBranch": args.fixture_branch or "<auto-resolve-before-run>",
            "checks": ["deploy running", "db configured", "db connected", "driver", "writeAccess null", "schemas readable"],
            "destructive": True,
        }, indent=2))
        return 0
    if not args.confirm_destructive:
        raise SystemExit("refusing to create/deploy DB Studio fixtures without --confirm-destructive; use --plan first")
    if not args.base_url or not args.token:
        raise SystemExit("--base-url and --token (or matching env vars) are required")

    try:
        fixture_ref = resolve_fixture_ref(args.fixture_repo, args.fixture_ref, args.fixture_branch)
    except FixtureRefError as exc:
        raise SystemExit(f"fixture ref preflight failed: {exc}") from exc

    started = utc_now()
    compact = started.replace(":", "").replace("-", "")
    prefix = project_prefix(compact)
    run_id = f"{compact}-{''.join(c for c in args.git_sha if c.isalnum())[:12] or 'unknown'}"
    client = Client(args.base_url, args.token)
    results = [
        run_engine(client, engine, driver, directory, prefix, args.fixture_repo, fixture_ref.branch, args.timeout, args.poll_seconds)
        for engine, driver, directory in ENGINES
    ]
    report = {
        "schemaVersion": 2,
        "kind": "dbstudio-compose-smoke",
        "runId": run_id,
        "gitSha": args.git_sha,
        "baseUrl": args.base_url,
        "fixtureRepo": args.fixture_repo,
        "fixtureRef": fixture_ref.requested_ref,
        "fixtureBranch": fixture_ref.branch,
        "fixtureResolvedSha": fixture_ref.resolved_sha,
        "startedAt": started,
        "finishedAt": utc_now(),
        "engines": [asdict(result) for result in results],
        "pass": all(result.error is None for result in results),
    }
    output = pathlib.Path(args.output or f"artifacts/beta-readiness/{run_id}/dbstudio-compose")
    write_report(report, output)
    print(output)
    return 0 if report["pass"] else 1


if __name__ == "__main__":
    sys.exit(main())
