#!/usr/bin/env python3
"""Real-world OSS compatibility runner for MyPaaS.

The default commands only validate and describe the suite. Live deployment is
explicit and requires an authenticated MyPaaS API plus a public domain.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import pathlib
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any

ROOT = pathlib.Path(__file__).resolve().parents[1]
COMPAT_DIR = ROOT / "compatibility"
CATALOG_PATH = COMPAT_DIR / "catalog.json"
TERMINAL_DEPLOYMENT_STATES = {"running", "failed", "stopped", "rolled_back"}
VALID_RESULTS = {
    "untested",
    "pass",
    "fail-unclassified",
    "fail-platform",
    "fail-app",
    "fail-resource",
    "blocked",
}
PROJECT_NAME_RE = re.compile(r"^[a-z0-9](?:[a-z0-9-]{1,28}[a-z0-9])$")
ROUTE_NAME_RE = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,18}[a-z0-9])?$")
MAX_ADDITIONAL_ROUTES = 4
CORE_REPO_SUFFIX = "/nabilrn/MyPaas.git"


class SuiteError(RuntimeError):
    pass


@dataclass
class APIError(SuiteError):
    status: int
    code: str
    message: str

    def __str__(self) -> str:
        return f"HTTP {self.status} {self.code}: {self.message}"


def load_catalog(path: pathlib.Path = CATALOG_PATH) -> dict[str, Any]:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise SuiteError(f"cannot load catalog {path}: {exc}") from exc


def validate_catalog(catalog: dict[str, Any], compose: bool = False) -> list[str]:
    errors: list[str] = []
    if catalog.get("schemaVersion") != 1:
        errors.append("schemaVersion must be 1")
    defaults = catalog.get("defaults")
    if not isinstance(defaults, dict):
        errors.append("defaults must be an object")
        defaults = {}
    apps = catalog.get("applications")
    if not isinstance(apps, list) or not apps:
        errors.append("applications must be a non-empty array")
        return errors

    seen: set[str] = set()
    for index, app in enumerate(apps):
        prefix = f"applications[{index}]"
        if not isinstance(app, dict):
            errors.append(f"{prefix} must be an object")
            continue
        app_id = app.get("id")
        if not isinstance(app_id, str) or not app_id:
            errors.append(f"{prefix}.id is required")
            continue
        if app_id in seen:
            errors.append(f"duplicate application id: {app_id}")
        seen.add(app_id)
        if app.get("resourceTier") not in {"light", "moderate", "heavy"}:
            errors.append(f"{app_id}: invalid resourceTier")
        execution = app.get("execution")
        if not isinstance(execution, dict):
            errors.append(f"{app_id}: execution must be an object")
            continue
        source_type = execution.get("sourceType")
        deploy_mode = execution.get("deployMode")
        if source_type not in {"git", "registry"}:
            errors.append(f"{app_id}: invalid sourceType")
        if deploy_mode not in {"dockerfile", "compose", "static", "image"}:
            errors.append(f"{app_id}: invalid deployMode")
        if source_type == "registry" and not execution.get("imageRef"):
            errors.append(f"{app_id}: registry entry requires imageRef")
        if source_type == "git" and not (execution.get("repoUrl") or defaults.get("repoUrl")):
            errors.append(f"{app_id}: git entry requires repoUrl")
        port = execution.get("appPort")
        if not isinstance(port, int) or not 1 <= port <= 65535:
            errors.append(f"{app_id}: appPort must be 1..65535")
        if deploy_mode == "compose":
            if not execution.get("mainService"):
                errors.append(f"{app_id}: compose entry requires mainService")
            base_dir = execution.get("baseDirectory")
            compose_file = execution.get("composeFilePath", "compose.yml")
            if base_dir and (execution.get("repoUrl") or defaults.get("repoUrl", "")).endswith("/MyPaas.git"):
                path = ROOT / base_dir / compose_file
                if not path.is_file():
                    errors.append(f"{app_id}: manifest not found: {path.relative_to(ROOT)}")
                elif compose:
                    errors.extend(validate_compose_file(app_id, path))
        errors.extend(validate_route_contract(app_id, execution, deploy_mode))
        smoke_path = execution.get("smokePath")
        if not isinstance(smoke_path, str) or not smoke_path.startswith("/"):
            errors.append(f"{app_id}: smokePath must start with /")
        statuses = execution.get("expectedStatus")
        if not isinstance(statuses, list) or not statuses or not all(isinstance(s, int) for s in statuses):
            errors.append(f"{app_id}: expectedStatus must be a non-empty integer array")
    return errors


def validate_route_contract(app_id: str, execution: dict[str, Any], deploy_mode: Any) -> list[str]:
    errors: list[str] = []
    routes = execution.get("additionalRoutes", [])
    if routes is None:
        routes = []
    if not isinstance(routes, list):
        return [f"{app_id}: additionalRoutes must be an array"]
    if routes and deploy_mode != "compose":
        errors.append(f"{app_id}: additionalRoutes are only valid for compose entries")
    if len(routes) > MAX_ADDITIONAL_ROUTES:
        errors.append(f"{app_id}: additionalRoutes may contain at most {MAX_ADDITIONAL_ROUTES} routes")

    route_names: set[str] = set()
    for index, route in enumerate(routes):
        prefix = f"{app_id}: additionalRoutes[{index}]"
        if not isinstance(route, dict):
            errors.append(f"{prefix} must be an object")
            continue
        name = route.get("name")
        service = route.get("service")
        port = route.get("containerPort")
        if not isinstance(name, str) or not ROUTE_NAME_RE.fullmatch(name):
            errors.append(f"{prefix}.name is invalid")
        elif name in route_names:
            errors.append(f"{app_id}: duplicate additional route name: {name}")
        else:
            route_names.add(name)
        if not isinstance(service, str) or not service.strip():
            errors.append(f"{prefix}.service is required")
        if not isinstance(port, int) or not 1 <= port <= 65535:
            errors.append(f"{prefix}.containerPort must be 1..65535")

    route_smoke = execution.get("routeSmoke", [])
    if route_smoke is None:
        route_smoke = []
    if not isinstance(route_smoke, list):
        return errors + [f"{app_id}: routeSmoke must be an array"]
    for index, check in enumerate(route_smoke):
        prefix = f"{app_id}: routeSmoke[{index}]"
        if not isinstance(check, dict):
            errors.append(f"{prefix} must be an object")
            continue
        route = check.get("route")
        path = check.get("path")
        statuses = check.get("expectedStatus")
        if route not in route_names:
            errors.append(f"{prefix}.route must reference an additional route")
        if not isinstance(path, str) or not path.startswith("/"):
            errors.append(f"{prefix}.path must start with /")
        if not isinstance(statuses, list) or not statuses or not all(isinstance(status, int) for status in statuses):
            errors.append(f"{prefix}.expectedStatus must be a non-empty integer array")
    return errors


def validate_compose_file(app_id: str, path: pathlib.Path) -> list[str]:
    cmd = ["docker", "compose", "-f", str(path), "config", "--quiet"]
    result = subprocess.run(cmd, cwd=path.parent, capture_output=True, text=True)
    if result.returncode == 0:
        return []
    detail = (result.stderr or result.stdout).strip().replace("\n", " | ")
    return [f"{app_id}: docker compose config failed: {detail}"]


def merged_execution(catalog: dict[str, Any], app: dict[str, Any]) -> dict[str, Any]:
    merged = dict(catalog.get("defaults", {}))
    merged.update(app["execution"])
    return merged


def selected_apps(catalog: dict[str, Any], ids: list[str], include_heavy: bool, include_blocked: bool) -> list[dict[str, Any]]:
    apps = catalog["applications"]
    if ids:
        wanted = set(ids)
        missing = wanted - {app["id"] for app in apps}
        if missing:
            raise SuiteError(f"unknown application ids: {', '.join(sorted(missing))}")
        apps = [app for app in apps if app["id"] in wanted]
    selected: list[dict[str, Any]] = []
    for app in apps:
        execution = app["execution"]
        if not execution.get("enabled", True) and not include_blocked:
            continue
        if execution.get("defaultRun") is False and not include_heavy:
            continue
        selected.append(app)
    return selected


class Client:
    def __init__(self, base: str, token: str, timeout: float = 60.0) -> None:
        self.base = base.rstrip("/")
        self.token = token
        self.timeout = timeout

    def request(self, method: str, path: str, payload: dict[str, Any] | None = None) -> Any:
        body = None
        headers = {"Accept": "application/json", "Authorization": f"Bearer {self.token}"}
        if payload is not None:
            body = json.dumps(payload).encode("utf-8")
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(self.base + path, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                if response.status == 204:
                    return None
                raw = response.read()
        except urllib.error.HTTPError as exc:
            raw = exc.read()
            try:
                payload_error = json.loads(raw.decode("utf-8"))
                error = payload_error.get("error", {})
                code = str(error.get("code", "HTTP_ERROR"))
                message = str(error.get("message", raw.decode("utf-8", errors="replace")))
            except json.JSONDecodeError:
                code = "HTTP_ERROR"
                message = raw.decode("utf-8", errors="replace")
            raise APIError(exc.code, code, message) from exc
        except urllib.error.URLError as exc:
            raise SuiteError(f"API request failed: {exc}") from exc
        if not raw:
            return None
        decoded = json.loads(raw.decode("utf-8"))
        return decoded.get("data", decoded)


def project_payload(
    catalog: dict[str, Any],
    app: dict[str, Any],
    name: str,
    core_repo_branch: str = "",
) -> dict[str, Any]:
    execution = merged_execution(catalog, app)
    branch = execution.get("branch", "main")
    repo_url = execution.get("repoUrl", "")
    if execution["sourceType"] == "git" and not repo_url:
        repo_url = catalog.get("defaults", {}).get("repoUrl", "")
    if core_repo_branch and repo_url.endswith(CORE_REPO_SUFFIX):
        branch = core_repo_branch
    payload: dict[str, Any] = {
        "name": name,
        "sourceType": execution["sourceType"],
        "branch": branch,
        "deployMode": execution["deployMode"],
        "resourceProfile": execution.get("resourceProfile", "custom"),
        "appPort": execution["appPort"],
        "memoryLimitMb": execution.get("memoryLimitMb", 512),
        "cpuLimit": execution.get("cpuLimit", 0.5),
        "envVars": execution.get("envVars", []),
        "sharedPostgres": False,
    }
    if execution["sourceType"] == "registry":
        payload["repoUrl"] = ""
        payload["imageRef"] = execution["imageRef"]
    else:
        payload["repoUrl"] = repo_url
    for key in ("mainService", "composeFilePath", "composeOverridePaths", "composeProfiles", "composeWorkdir", "staticFrontendPath", "baseDirectory"):
        if key in execution:
            payload[key] = execution[key]
    return payload


def wait_for_deployment(client: Client, deployment_id: str, timeout: float) -> dict[str, Any]:
    deadline = time.monotonic() + timeout
    last: dict[str, Any] = {}
    while time.monotonic() < deadline:
        current = client.request("GET", f"/deployments/{deployment_id}")
        if isinstance(current, dict):
            last = current
            if current.get("status") in TERMINAL_DEPLOYMENT_STATES:
                return current
        time.sleep(3)
    raise SuiteError(f"deployment {deployment_id} did not reach a terminal state within {timeout:.0f}s; last={last}")


def smoke(url: str, expected: list[int], timeout: float = 30.0) -> tuple[int, str]:
    request = urllib.request.Request(url, headers={"User-Agent": "MyPaaS-Compatibility/1"})
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            status = response.status
            final_url = response.geturl()
    except urllib.error.HTTPError as exc:
        status = exc.code
        final_url = exc.geturl()
    except urllib.error.URLError as exc:
        raise SuiteError(f"smoke request failed for {url}: {exc}") from exc
    if status not in expected:
        raise SuiteError(f"smoke request {url} returned {status}; expected {expected}")
    return status, final_url


def smoke_retry(url: str, expected: list[int], ready_timeout: float = 45.0) -> tuple[int, str]:
    deadline = time.monotonic() + max(0.0, ready_timeout)
    last_error: Exception | None = None
    while True:
        try:
            return smoke(url, expected)
        except SuiteError as exc:
            last_error = exc
            if time.monotonic() >= deadline:
                raise last_error
            time.sleep(2)


def smoke_checks(execution: dict[str, Any], name: str, public_domain: str) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    domain = public_domain.strip(".")
    primary_url = f"https://{name}.{domain}{execution['smokePath']}"
    status, final_url = smoke_retry(primary_url, execution["expectedStatus"])
    primary = {"url": primary_url, "finalUrl": final_url, "status": status}

    route_results: list[dict[str, Any]] = []
    for check in execution.get("routeSmoke", []):
        route_name = check["route"]
        url = f"https://{name}-{route_name}.{domain}{check['path']}"
        status, final_url = smoke_retry(url, check["expectedStatus"])
        route_results.append({
            "route": route_name,
            "url": url,
            "finalUrl": final_url,
            "status": status,
        })
    return primary, route_results


def likely_blocked(exc: Exception) -> bool:
    text = str(exc).lower()
    markers = ("unsupported", "unsafe", "host bind", "external network", "external volume", "privileged", "socket mount")
    return any(marker in text for marker in markers)


def run_one(client: Client, catalog: dict[str, Any], app: dict[str, Any], public_domain: str, timeout: float, keep: bool, lifecycle: bool) -> dict[str, Any]:
    execution = merged_execution(catalog, app)
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%H%M%S")
    raw_name = f"compat-{app['id']}-{stamp}"[:30].rstrip("-")
    name = raw_name if PROJECT_NAME_RE.match(raw_name) else f"compat-{stamp}"
    result: dict[str, Any] = {
        "application": app["id"],
        "projectName": name,
        "result": "fail-unclassified",
        "phase": "create",
        "startedAt": dt.datetime.now(dt.timezone.utc).isoformat(),
    }
    project_id: str | None = None
    try:
        core_branch = os.environ.get("MYPAAS_COMPAT_REPO_BRANCH", "").strip()
        project = client.request("POST", "/projects/", project_payload(catalog, app, name, core_branch))
        project_id = project["id"]
        result["projectId"] = project_id
        if core_branch and project.get("repoUrl", "").endswith(CORE_REPO_SUFFIX):
            result["sourceBranch"] = core_branch

        service_resources = execution.get("serviceResources")
        if service_resources:
            result["phase"] = "resource-config"
            client.request("PATCH", f"/projects/{project_id}", {"serviceResources": service_resources})

        additional_routes = execution.get("additionalRoutes", [])
        if additional_routes:
            result["phase"] = "route-config"
            client.request("PUT", f"/projects/{project_id}/routes", {"routes": additional_routes})
            result["additionalRoutes"] = additional_routes

        result["phase"] = "deploy"
        deployment = client.request("POST", f"/projects/{project_id}/deploy")
        result["deploymentId"] = deployment["id"]
        deployment = wait_for_deployment(client, deployment["id"], timeout)
        if deployment.get("status") != "running":
            result["deploymentStatus"] = deployment.get("status")
            result["error"] = deployment.get("errorMsg") or deployment.get("buildLog") or "deployment did not reach running"
            return result

        result["phase"] = "smoke"
        primary, route_results = smoke_checks(execution, name, public_domain)
        result["smoke"] = primary
        if route_results:
            result["routeSmoke"] = route_results

        if lifecycle:
            result["phase"] = "restart"
            client.request("POST", f"/projects/{project_id}/restart")
            time.sleep(3)
            primary, route_results = smoke_checks(execution, name, public_domain)
            result["restartSmoke"] = primary
            if route_results:
                result["restartRouteSmoke"] = route_results

        result["phase"] = "complete"
        result["result"] = "pass"
        return result
    except (APIError, SuiteError) as exc:
        result["error"] = str(exc)
        if likely_blocked(exc):
            result["result"] = "blocked"
        return result
    finally:
        result["finishedAt"] = dt.datetime.now(dt.timezone.utc).isoformat()
        if project_id and not keep:
            try:
                client.request("DELETE", f"/projects/{project_id}")
                result["cleanup"] = "deleted"
            except Exception as exc:  # cleanup must not hide the compatibility result
                result["cleanup"] = f"failed: {exc}"


def command_list(catalog: dict[str, Any]) -> int:
    print(f"{'ID':20} {'TIER':10} {'MODE':12} {'DEFAULT':8} NAME")
    for app in catalog["applications"]:
        execution = app["execution"]
        default = execution.get("enabled", True) and execution.get("defaultRun", True)
        print(f"{app['id']:20} {app['resourceTier']:10} {execution['deployMode']:12} {str(default):8} {app['name']}")
    return 0


def command_plan(catalog: dict[str, Any], args: argparse.Namespace) -> int:
    apps = selected_apps(catalog, args.apps, args.include_heavy, args.include_blocked)
    plan = []
    for app in apps:
        execution = merged_execution(catalog, app)
        plan.append({
            "id": app["id"],
            "name": app["name"],
            "class": app["class"],
            "resourceTier": app["resourceTier"],
            "sourceType": execution["sourceType"],
            "deployMode": execution["deployMode"],
            "appPort": execution["appPort"],
            "additionalRoutes": execution.get("additionalRoutes", []),
            "defaultRun": execution.get("enabled", True) and execution.get("defaultRun", True),
            "limitations": app.get("limitations", []),
        })
    print(json.dumps(plan, indent=2))
    return 0


def command_run(catalog: dict[str, Any], args: argparse.Namespace) -> int:
    base = os.environ.get("MYPAAS_API_BASE", "").strip()
    token = os.environ.get("MYPAAS_TOKEN", "").strip()
    domain = os.environ.get("MYPAAS_PUBLIC_DOMAIN", "").strip()
    if not base or not token or not domain:
        raise SuiteError("live run requires MYPAAS_API_BASE, MYPAAS_TOKEN and MYPAAS_PUBLIC_DOMAIN")
    client = Client(base, token)
    apps = selected_apps(catalog, args.apps, args.include_heavy, args.include_blocked)
    results = []
    for app in apps:
        print(f"==> {app['name']} ({app['id']})", file=sys.stderr, flush=True)
        result = run_one(client, catalog, app, domain, args.timeout, args.keep, args.lifecycle)
        results.append(result)
        print(f"    {result['result']} at {result['phase']}", file=sys.stderr, flush=True)
        if args.stop_on_failure and result["result"] != "pass":
            break
    output = {
        "schemaVersion": 1,
        "generatedAt": dt.datetime.now(dt.timezone.utc).isoformat(),
        "host": base,
        "results": results,
    }
    encoded = json.dumps(output, indent=2)
    print(encoded)
    if args.output:
        pathlib.Path(args.output).write_text(encoded + "\n", encoding="utf-8")
    return 0 if all(item["result"] == "pass" for item in results) else 1


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser(description="MyPaaS real-world OSS compatibility suite")
    sub = root.add_subparsers(dest="command", required=True)
    validate = sub.add_parser("validate", help="validate catalog and local compatibility manifests")
    validate.add_argument("--compose", action="store_true", help="also run docker compose config --quiet")
    sub.add_parser("list", help="list catalogued applications")
    for name in ("plan", "run"):
        command = sub.add_parser(name, help=f"{name} selected compatibility applications")
        command.add_argument("apps", nargs="*", help="application ids; default is the safe baseline set")
        command.add_argument("--include-heavy", action="store_true")
        command.add_argument("--include-blocked", action="store_true")
        if name == "run":
            command.add_argument("--timeout", type=float, default=1200)
            command.add_argument("--keep", action="store_true", help="keep created MyPaaS projects instead of deleting them")
            command.add_argument("--lifecycle", action="store_true", help="restart the project and repeat primary/additional route smoke checks")
            command.add_argument("--stop-on-failure", action="store_true")
            command.add_argument("--output", help="optional JSON result path; do not commit live results as capacity evidence")
    return root


def main(argv: list[str] | None = None) -> int:
    args = parser().parse_args(argv)
    catalog = load_catalog()
    errors = validate_catalog(catalog, compose=args.compose if args.command == "validate" else False)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 2
    if args.command == "validate":
        print(f"validated {len(catalog['applications'])} compatibility entries")
        return 0
    if args.command == "list":
        return command_list(catalog)
    if args.command == "plan":
        return command_plan(catalog, args)
    if args.command == "run":
        return command_run(catalog, args)
    raise AssertionError(args.command)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except SuiteError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(2)
