#!/usr/bin/env python3
"""Controlled concurrent-deploy resilience harness for MyPaas beta readiness."""

from __future__ import annotations

import argparse
import concurrent.futures
import datetime as dt
import hashlib
import hmac
import json
import os
import pathlib
import sys
import time
import urllib.error
import urllib.request
import uuid
from dataclasses import asdict, dataclass
from typing import Any

from fixture_ref import FixtureRefError, resolve_fixture_ref

TERMINAL = {"running", "failed", "stopped", "rolled_back"}
FIXTURES = (
    {
        "kind": "dockerfile",
        "deployMode": "dockerfile",
        "resourceProfile": "node-python",
        "appPort": 8080,
        "baseDirectory": "qualification/fixtures/beta/dockerfile",
    },
    {
        "kind": "compose",
        "deployMode": "compose",
        "resourceProfile": "compose-main",
        "appPort": 8080,
        "baseDirectory": "qualification/fixtures/beta/compose",
        "mainService": "app",
        "composeFilePath": "compose.yml",
    },
)


class HarnessError(RuntimeError):
    pass


class ApiClient:
    def __init__(self, base_url: str, token: str, timeout: float = 60.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.timeout = timeout

    def request(
        self,
        method: str,
        path: str,
        payload: dict[str, Any] | None = None,
        headers: dict[str, str] | None = None,
        authenticated: bool = True,
    ) -> tuple[int, Any]:
        body = None
        request_headers = {"Accept": "application/json"}
        if authenticated:
            request_headers["Authorization"] = f"Bearer {self.token}"
        if headers:
            request_headers.update(headers)
        if payload is not None:
            body = json.dumps(payload).encode()
            request_headers["Content-Type"] = "application/json"
        req = urllib.request.Request(self.base_url + path, data=body, headers=request_headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as response:
                status = response.status
                raw = response.read()
        except urllib.error.HTTPError as exc:
            raw = exc.read()
            status = exc.code
        except urllib.error.URLError as exc:
            raise HarnessError(f"{method} {path}: {exc.reason}") from exc

        decoded: Any = None
        if raw:
            try:
                decoded = json.loads(raw.decode())
            except json.JSONDecodeError:
                decoded = raw.decode(errors="replace")[:2000]
        if status >= 400:
            raise HarnessError(f"{method} {path}: HTTP {status}: {sanitize(decoded)}")
        if isinstance(decoded, dict) and "data" in decoded:
            decoded = decoded["data"]
        return status, decoded


@dataclass
class ManagedProject:
    name: str
    kind: str
    id: str
    subdomain: str
    webhook_secret: str
    allocated_port: int | None
    should_fail: bool = False


@dataclass
class DeploymentObservation:
    project_id: str
    project_name: str
    phase: str
    deployment_id: str | None = None
    status: str = "not_started"
    seconds: float | None = None
    error: str | None = None


def sanitize(value: Any) -> str:
    text = json.dumps(value, sort_keys=True) if not isinstance(value, str) else value
    for marker in ("Bearer ", "token", "password", "secret"):
        if marker.lower() in text.lower():
            return "<redacted-error>"
    return text[:2000]


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def fixture_payload(index: int, prefix: str, repo: str, branch: str) -> dict[str, Any]:
    fixture = FIXTURES[index % len(FIXTURES)]
    payload: dict[str, Any] = {
        "name": f"{prefix}-{index + 1:02d}-{fixture['kind']}",
        "sourceType": "git",
        "repoUrl": repo,
        "branch": branch,
        "deployMode": fixture["deployMode"],
        "resourceProfile": fixture["resourceProfile"],
        "appPort": fixture["appPort"],
        "baseDirectory": fixture["baseDirectory"],
    }
    for key in ("mainService", "composeFilePath"):
        if key in fixture:
            payload[key] = fixture[key]
    return payload


def failing_payload(prefix: str, run_id: str) -> dict[str, Any]:
    return {
        "name": f"{prefix}-intentional-failure",
        "sourceType": "registry",
        "imageRef": f"ghcr.io/nabilrn/mypaas-beta-intentionally-missing:{run_id.lower()}",
        "branch": "main",
        "deployMode": "image",
        "resourceProfile": "node-python",
        "appPort": 8080,
    }


def project_from_response(row: dict[str, Any], kind: str, should_fail: bool = False) -> ManagedProject:
    return ManagedProject(
        name=str(row["name"]),
        kind=kind,
        id=str(row["id"]),
        subdomain=str(row.get("subdomain") or ""),
        webhook_secret=str(row.get("webhookSecret") or ""),
        allocated_port=int(row["allocatedPort"]) if row.get("allocatedPort") is not None else None,
        should_fail=should_fail,
    )


def create_projects(client: ApiClient, count: int, prefix: str, repo: str, branch: str, run_id: str, parallel: int) -> list[ManagedProject]:
    payloads = [(fixture_payload(i, prefix, repo, branch), FIXTURES[i % len(FIXTURES)]["kind"], False) for i in range(count)]
    payloads.append((failing_payload(prefix, run_id), "intentional-failure", True))

    def create(item: tuple[dict[str, Any], str, bool]) -> ManagedProject:
        payload, kind, should_fail = item
        _, row = client.request("POST", "/api/projects/", payload)
        if not isinstance(row, dict):
            raise HarnessError("project create did not return an object")
        return project_from_response(row, str(kind), should_fail)

    with concurrent.futures.ThreadPoolExecutor(max_workers=parallel) as pool:
        rows = list(pool.map(create, payloads))
    return rows


def project_prefix(run_id: str) -> str:
    compact = "".join(c for c in run_id.lower() if c.isalnum())
    return f"br-{(compact or 'run')[:7]}"


def wait_deployment(client: ApiClient, deployment_id: str, timeout: float, poll: float) -> dict[str, Any]:
    deadline = time.monotonic() + timeout
    last: dict[str, Any] = {}
    while time.monotonic() < deadline:
        _, row = client.request("GET", f"/api/deployments/{deployment_id}")
        if isinstance(row, dict):
            last = row
            if row.get("status") in TERMINAL:
                return row
        time.sleep(poll)
    raise HarnessError(f"deployment {deployment_id} stuck at {last.get('status', 'unknown')}")


def deploy_one(client: ApiClient, project: ManagedProject, phase: str, timeout: float, poll: float) -> DeploymentObservation:
    observation = DeploymentObservation(project.id, project.name, phase)
    started = time.monotonic()
    try:
        _, row = client.request("POST", f"/api/projects/{project.id}/deploy")
        if not isinstance(row, dict) or not row.get("id"):
            raise HarnessError("deployment trigger did not return an id")
        observation.deployment_id = str(row["id"])
        final = wait_deployment(client, observation.deployment_id, timeout, poll)
        observation.status = str(final.get("status", "unknown"))
        if project.should_fail and observation.status != "failed":
            observation.error = f"intentional failure project unexpectedly ended as {observation.status}"
        if not project.should_fail and observation.status != "running":
            observation.error = sanitize(final.get("errorMsg") or f"unexpected status {observation.status}")
    except Exception as exc:
        observation.status = "harness_error"
        observation.error = sanitize(str(exc))
    observation.seconds = time.monotonic() - started
    return observation


def deploy_concurrently(client: ApiClient, projects: list[ManagedProject], phase: str, parallel: int, timeout: float, poll: float) -> list[DeploymentObservation]:
    with concurrent.futures.ThreadPoolExecutor(max_workers=parallel) as pool:
        futures = [pool.submit(deploy_one, client, project, phase, timeout, poll) for project in projects]
        rows = [future.result() for future in concurrent.futures.as_completed(futures)]
    return sorted(rows, key=lambda item: item.project_name)


def webhook_headers(secret: str, body: bytes) -> dict[str, str]:
    signature = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return {
        "Content-Type": "application/json",
        "X-GitHub-Event": "push",
        "X-GitHub-Delivery": str(uuid.uuid4()),
        "X-Hub-Signature-256": f"sha256={signature}",
    }


def send_webhook_burst(client: ApiClient, project: ManagedProject, branch: str, count: int, parallel: int) -> list[str]:
    if count <= 0:
        return []
    payload = {"ref": f"refs/heads/{branch}", "after": "0" * 40, "repository": {"full_name": "beta/readiness-fixture"}}
    body = json.dumps(payload).encode()
    headers = webhook_headers(project.webhook_secret, body)

    def send(_: int) -> str:
        req = urllib.request.Request(
            f"{client.base_url}/api/webhook/{project.id}",
            data=body,
            headers=headers | {"X-GitHub-Delivery": str(uuid.uuid4())},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=client.timeout) as response:
                return f"HTTP {response.status}"
        except urllib.error.HTTPError as exc:
            return f"HTTP {exc.code}"
        except urllib.error.URLError as exc:
            return f"ERROR {exc.reason}"

    with concurrent.futures.ThreadPoolExecutor(max_workers=parallel) as pool:
        return list(pool.map(send, range(count)))


def duplicate_ports(projects: list[ManagedProject]) -> list[int]:
    seen: set[int] = set()
    duplicates: set[int] = set()
    for project in projects:
        if project.allocated_port is None:
            continue
        if project.allocated_port in seen:
            duplicates.add(project.allocated_port)
        seen.add(project.allocated_port)
    return sorted(duplicates)


def probe_url(url: str, timeout: float = 10.0) -> dict[str, Any]:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            return {"url": url, "status": response.status, "ok": 200 <= response.status < 500}
    except urllib.error.HTTPError as exc:
        return {"url": url, "status": exc.code, "ok": exc.code < 500}
    except urllib.error.URLError as exc:
        return {"url": url, "status": 0, "ok": False, "error": str(exc.reason)}


def project_route(project: ManagedProject, public_domain: str) -> str | None:
    if not project.subdomain or not public_domain:
        return None
    return f"https://{project.subdomain}.{public_domain.strip('.')}"


def read_path_probes(client: ApiClient, project: ManagedProject) -> list[dict[str, Any]]:
    paths = [
        f"/api/projects/{project.id}/logs?tail=20",
        f"/api/projects/{project.id}/metrics",
        f"/api/projects/{project.id}/db/status",
    ]
    out: list[dict[str, Any]] = []
    for path in paths:
        try:
            status, _ = client.request("GET", path)
            out.append({"path": path, "status": status, "serverError": False})
        except HarnessError as exc:
            text = str(exc)
            status = 0
            if "HTTP " in text:
                try:
                    status = int(text.split("HTTP ", 1)[1].split(":", 1)[0])
                except ValueError:
                    pass
            out.append({"path": path, "status": status, "serverError": status >= 500 or status == 0})
    return out


def evaluate(projects: list[ManagedProject], deployments: list[DeploymentObservation], route_probes: list[dict[str, Any]], read_probes: list[dict[str, Any]]) -> list[str]:
    failures: list[str] = []
    duplicates = duplicate_ports(projects)
    if duplicates:
        failures.append(f"duplicate allocated ports: {duplicates}")
    for observation in deployments:
        project = next(p for p in projects if p.id == observation.project_id)
        expected = "failed" if project.should_fail else "running"
        if observation.status != expected:
            failures.append(f"{observation.phase}/{observation.project_name}: expected {expected}, got {observation.status}")
        if observation.error:
            failures.append(f"{observation.phase}/{observation.project_name}: {observation.error}")
    for probe in route_probes:
        if not probe.get("ok"):
            failures.append(f"route unavailable: {probe.get('url')}")
    for probe in read_probes:
        if probe.get("serverError"):
            failures.append(f"read path failed during load: {probe.get('path')} status={probe.get('status')}")
    return failures


def markdown_report(report: dict[str, Any]) -> str:
    lines = [
        "# MyPaas concurrent-deploy resilience report",
        "",
        f"- Run ID: `{report['runId']}`",
        f"- Git SHA: `{report['gitSha']}`",
        f"- Target: `{report['baseUrl']}`",
        f"- Fixture branch: `{report.get('fixtureBranch', report.get('fixtureRef', 'unknown'))}`",
        f"- Fixture resolved SHA: `{report.get('fixtureResolvedSha', 'unknown')}`",
        f"- Result: **{'PASS' if report['pass'] else 'FAIL'}**",
        "",
        "| Phase | Project | Expected | Actual | Seconds |",
        "| --- | --- | --- | --- | ---: |",
    ]
    expected = {p["id"]: ("failed" if p["should_fail"] else "running") for p in report["projects"]}
    for row in report["deployments"]:
        lines.append(
            f"| {row['phase']} | {row['project_name']} | {expected[row['project_id']]} | "
            f"{row['status']} | {row['seconds']:.2f} |"
        )
    lines.extend(["", "## Findings", ""])
    if report["failures"]:
        lines.extend(f"- {item}" for item in report["failures"])
    else:
        lines.append("- No consistency failures detected.")
    if report["blockedReasons"]:
        lines.extend(["", "## Blocked observations", ""])
        lines.extend(f"- {item}" for item in report["blockedReasons"])
    lines.append("")
    return "\n".join(lines)


def write_report(report: dict[str, Any], output: pathlib.Path) -> None:
    output.mkdir(parents=True, exist_ok=True)
    (output / "report.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    (output / "report.md").write_text(markdown_report(report), encoding="utf-8")


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base-url", default=os.getenv("MYPAAS_BETA_BASE_URL", ""))
    p.add_argument("--token", default=os.getenv("MYPAAS_API_TOKEN", ""))
    p.add_argument("--fixture-repo", default=os.getenv("MYPAAS_BETA_FIXTURE_REPO", "https://github.com/nabilrn/MyPaas"))
    p.add_argument("--fixture-ref", default=os.getenv("MYPAAS_BETA_FIXTURE_REF", "main"), help="immutable SHA or branch identity to verify")
    p.add_argument("--fixture-branch", default=os.getenv("MYPAAS_BETA_FIXTURE_BRANCH", ""), help="optional clonable branch; required only when auto-resolution is ambiguous")
    p.add_argument("--git-sha", default=os.getenv("MYPAAS_BETA_GIT_SHA", "unknown"))
    p.add_argument("--public-domain", default=os.getenv("MYPAAS_BETA_PUBLIC_DOMAIN", ""))
    p.add_argument("--project-count", type=int, default=4)
    p.add_argument("--parallel", type=int, default=6)
    p.add_argument("--deploy-timeout", type=float, default=900)
    p.add_argument("--poll-seconds", type=float, default=3)
    p.add_argument("--webhook-burst", type=int, default=0)
    p.add_argument("--protect-url", action="append", default=[])
    p.add_argument("--output", default="")
    p.add_argument("--plan", action="store_true")
    p.add_argument("--confirm-destructive", action="store_true")
    return p


def main(argv: list[str] | None = None) -> int:
    args = parser().parse_args(argv)
    if args.project_count < 1 or args.parallel < 1:
        raise SystemExit("--project-count and --parallel must be positive")
    if args.plan:
        print(json.dumps({
            "controlledProjects": args.project_count,
            "intentionalFailureProjects": 1,
            "phases": ["concurrent-create", "concurrent-deploy", "concurrent-redeploy"],
            "fixtureRepo": args.fixture_repo,
            "fixtureRef": args.fixture_ref,
            "fixtureBranch": args.fixture_branch or "<auto-resolve-before-run>",
            "webhookBurst": args.webhook_burst,
            "destructive": True,
        }, indent=2))
        return 0
    if not args.confirm_destructive:
        raise SystemExit("refusing to mutate projects without --confirm-destructive; use --plan first")
    if not args.base_url or not args.token:
        raise SystemExit("--base-url and --token (or matching env vars) are required")

    try:
        fixture_ref = resolve_fixture_ref(args.fixture_repo, args.fixture_ref, args.fixture_branch)
    except FixtureRefError as exc:
        raise SystemExit(f"fixture ref preflight failed: {exc}") from exc

    started = utc_now()
    run_id = f"{started.replace(':', '').replace('-', '')}-{''.join(c for c in args.git_sha if c.isalnum())[:12] or 'unknown'}"
    prefix = project_prefix(run_id)
    output = pathlib.Path(args.output or f"artifacts/beta-readiness/{run_id}/resilience-concurrent-deploys")
    client = ApiClient(args.base_url, args.token)
    blocked: list[str] = []

    protected_before = [probe_url(url) for url in args.protect_url]
    projects = create_projects(client, args.project_count, prefix, args.fixture_repo, fixture_ref.branch, run_id, args.parallel)
    baseline = deploy_concurrently(client, projects, "initial-deploy", args.parallel, args.deploy_timeout, args.poll_seconds)
    healthy = [p for p in projects if not p.should_fail]
    redeploy = deploy_concurrently(client, healthy, "redeploy", args.parallel, args.deploy_timeout, args.poll_seconds)

    webhook_results: list[str] = []
    if args.webhook_burst:
        target = next((p for p in healthy if p.webhook_secret), None)
        if target:
            webhook_results = send_webhook_burst(client, target, fixture_ref.branch, args.webhook_burst, args.parallel)
        else:
            blocked.append("webhook burst requested but no fixture project exposed a webhook secret")

    read_probes: list[dict[str, Any]] = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.parallel) as pool:
        for rows in pool.map(lambda p: read_path_probes(client, p), healthy):
            read_probes.extend(rows)

    route_probes: list[dict[str, Any]] = []
    if args.public_domain:
        route_probes.extend(probe_url(url) for p in healthy if (url := project_route(p, args.public_domain)))
    else:
        blocked.append("project route verification skipped because --public-domain was not configured")
    protected_after = [probe_url(url) for url in args.protect_url]
    route_probes.extend(protected_after)

    deployments = baseline + redeploy
    failures = evaluate(projects, deployments, route_probes, read_probes)
    for before, after in zip(protected_before, protected_after):
        if before.get("ok") and not after.get("ok"):
            failures.append(f"pre-existing protected route regressed: {after.get('url')}")

    report = {
        "schemaVersion": 2,
        "kind": "resilience-concurrent-deploys",
        "runId": run_id,
        "gitSha": args.git_sha,
        "baseUrl": args.base_url,
        "fixtureRepo": args.fixture_repo,
        "fixtureRef": fixture_ref.requested_ref,
        "fixtureBranch": fixture_ref.branch,
        "fixtureResolvedSha": fixture_ref.resolved_sha,
        "startedAt": started,
        "finishedAt": utc_now(),
        "projects": [asdict(p) | {"webhook_secret": "<redacted>"} for p in projects],
        "deployments": [asdict(row) for row in deployments],
        "webhookBurst": {"requested": args.webhook_burst, "results": webhook_results},
        "readProbes": read_probes,
        "routeProbes": route_probes,
        "protectedBefore": protected_before,
        "blockedReasons": blocked,
        "failures": failures,
        "pass": not failures,
    }
    write_report(report, output)
    print(output)
    return 0 if report["pass"] else 1


if __name__ == "__main__":
    sys.exit(main())
