#!/usr/bin/env python3
"""Full MyPaas backup/restore bundle tooling for production disaster-recovery operations.

The existing in-process backup scheduler remains the lightweight control-plane
PostgreSQL backup. This tool creates a fuller disaster-recovery bundle that also
captures production configuration, static artifacts, Compose workspaces, and
MyPaas-managed project volumes.

Backup bundles contain secrets in private/config.env and therefore must be
handled as sensitive operational artifacts. Reports and manifests never include
configuration values.
"""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import hmac
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tarfile
import tempfile
import time
from dataclasses import dataclass
from typing import Any, Iterable

SCHEMA_VERSION = 1
DEFAULT_INSTALL_DIR = "/opt/mypaas"
DEFAULT_BACKUP_ROOT = "/var/lib/mypaas/backups"
DEFAULT_STATIC_ROOT = "/var/lib/mypaas/static"
DEFAULT_COMPOSE_ROOT = "/var/lib/mypaas/compose"
MANAGED_LABEL = "mypaas.managed"
MANAGED_VALUE = "true"
DEFAULT_LOCAL_API = "http://127.0.0.1:8080"


class BackupRestoreError(RuntimeError):
    pass


@dataclass(frozen=True)
class ManagedVolume:
    name: str
    mountpoint: str
    source: str


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def parse_env_file(path: pathlib.Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
            value = value[1:-1]
        values[key] = value
    return values


def required_env(values: dict[str, str], key: str) -> str:
    value = values.get(key, "").strip()
    if not value:
        raise BackupRestoreError(f"required production setting {key} is missing")
    return value


def sql_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def run(
    args: list[str],
    *,
    input_bytes: bytes | None = None,
    stdout_file: pathlib.Path | None = None,
    check: bool = True,
) -> subprocess.CompletedProcess[bytes]:
    stdout: Any = subprocess.PIPE
    handle = None
    if stdout_file is not None:
        stdout_file.parent.mkdir(parents=True, exist_ok=True)
        handle = stdout_file.open("wb")
        stdout = handle
    try:
        completed = subprocess.run(
            args,
            input=input_bytes,
            stdout=stdout,
            stderr=subprocess.PIPE,
            check=False,
        )
    except OSError as exc:
        raise BackupRestoreError(f"cannot execute {args[0]}: {exc}") from exc
    finally:
        if handle is not None:
            handle.close()
    if check and completed.returncode != 0:
        message = completed.stderr.decode("utf-8", errors="replace").strip()
        raise BackupRestoreError(f"command failed ({args[0]}): {message[:1000]}")
    return completed


def run_text(args: list[str], *, check: bool = True) -> str:
    return run(args, check=check).stdout.decode("utf-8", errors="replace")


def sha256_file(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def file_record(kind: str, bundle: pathlib.Path, path: pathlib.Path, **extra: Any) -> dict[str, Any]:
    record = {
        "kind": kind,
        "path": path.relative_to(bundle).as_posix(),
        "sha256": sha256_file(path),
        "sizeBytes": path.stat().st_size,
    }
    record.update(extra)
    return record


def archive_directory(source: pathlib.Path, destination: pathlib.Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(destination, "w:gz", format=tarfile.PAX_FORMAT) as archive:
        if source.exists():
            for child in sorted(source.iterdir(), key=lambda p: p.name):
                archive.add(child, arcname=child.name, recursive=True)
    os.chmod(destination, 0o600)


def _safe_member_destination(root: pathlib.Path, name: str) -> pathlib.Path:
    candidate = (root / name).resolve()
    resolved_root = root.resolve()
    try:
        candidate.relative_to(resolved_root)
    except ValueError as exc:
        raise BackupRestoreError(f"unsafe archive member path: {name}") from exc
    return candidate


def safe_extract(archive_path: pathlib.Path, destination: pathlib.Path) -> None:
    destination.mkdir(parents=True, exist_ok=True)
    with tarfile.open(archive_path, "r:gz") as archive:
        for member in archive.getmembers():
            _safe_member_destination(destination, member.name)
            if member.issym() or member.islnk():
                link_target = pathlib.PurePosixPath(member.name).parent / member.linkname
                if link_target.is_absolute() or ".." in link_target.parts:
                    raise BackupRestoreError(f"unsafe archive link: {member.name}")
        archive.extractall(destination)


def clear_directory(path: pathlib.Path) -> None:
    path.mkdir(parents=True, exist_ok=True)
    for child in path.iterdir():
        if child.is_symlink() or child.is_file():
            child.unlink()
        else:
            shutil.rmtree(child)


def docker_json(args: list[str]) -> Any:
    completed = run(["docker", *args])
    raw = completed.stdout.decode("utf-8", errors="replace").strip()
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        raise BackupRestoreError(f"docker returned invalid JSON for {' '.join(args)}") from exc


def active_project_names(postgres_user: str, postgres_db: str) -> set[str]:
    query = "SELECT name FROM projects WHERE deleted_at IS NULL ORDER BY name"
    completed = run([
        "docker",
        "exec",
        "mypaas-postgres-prod",
        "psql",
        "-X",
        "-A",
        "-t",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
        "-c",
        query,
    ])
    return {line.strip() for line in completed.stdout.decode().splitlines() if line.strip()}


def active_project_volume_names(postgres_user: str, postgres_db: str) -> set[str]:
    query = "SELECT name, deploy_mode FROM projects WHERE deleted_at IS NULL ORDER BY name"
    completed = run([
        "docker",
        "exec",
        "mypaas-postgres-prod",
        "psql",
        "-X",
        "-A",
        "-F",
        "\t",
        "-t",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
        "-c",
        query,
    ])
    names: set[str] = set()
    for line in completed.stdout.decode().splitlines():
        parts = line.strip().split("\t")
        if not parts or not parts[0]:
            continue
        name = parts[0].strip()
        deploy_mode = parts[1].strip() if len(parts) > 1 else ""
        names.add(name)
        if deploy_mode == "compose":
            names.add(f"mypaas-{name}")
    return names


def control_json_query(postgres_user: str, postgres_db: str, sql: str) -> list[dict[str, Any]]:
    wrapped = f"SELECT COALESCE(json_agg(row_to_json(t)), '[]'::json) FROM ({sql}) t"
    raw = run_text([
        "docker",
        "exec",
        "mypaas-postgres-prod",
        "psql",
        "-X",
        "-A",
        "-t",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
        "-c",
        wrapped,
    ]).strip()
    data = json.loads(raw or "[]")
    if not isinstance(data, list):
        raise BackupRestoreError("control-plane query did not return a JSON array")
    return data


def project_by_name(postgres_user: str, postgres_db: str, name: str) -> dict[str, Any] | None:
    rows = control_json_query(
        postgres_user,
        postgres_db,
        "SELECT id::text, name, subdomain, deploy_mode, status, app_port, allocated_port "
        "FROM projects WHERE deleted_at IS NULL AND name = " + sql_literal(name),
    )
    if not rows:
        return None
    return rows[0]


def host_for_project(project: dict[str, Any], public_domain: str) -> str:
    subdomain = str(project.get("subdomain") or project.get("name") or "").strip()
    if not subdomain:
        raise BackupRestoreError("project is missing subdomain")
    return f"{subdomain}.{public_domain}"


def route_body(host: str, api_base: str = "http://127.0.0.1") -> str:
    return run_text(["curl", "-fsS", "--max-time", "15", "-H", f"Host: {host}", api_base + "/"])


def wait_for_route_body(host: str, api_base: str = "http://127.0.0.1", timeout_seconds: float = 120.0, interval_seconds: float = 3.0) -> str:
    deadline = time.monotonic() + max(0.0, timeout_seconds)
    last_error: BackupRestoreError | None = None
    while True:
        try:
            return route_body(host, api_base)
        except BackupRestoreError as exc:
            last_error = exc
        if time.monotonic() >= deadline:
            raise BackupRestoreError(f"route {host} did not become healthy within {timeout_seconds:g} seconds: {last_error}")
        time.sleep(interval_seconds)


def fixture_fail(failures: list[dict[str, str]], check: str, reason: str) -> None:
    failures.append({"check": check, "reason": reason})


def require_project(
    failures: list[dict[str, str]],
    postgres_user: str,
    postgres_db: str,
    name: str,
    deploy_mode: str | None = None,
    status: str | None = "running",
) -> dict[str, Any] | None:
    project = project_by_name(postgres_user, postgres_db, name)
    if project is None:
        fixture_fail(failures, name, "project is missing")
        return None
    if deploy_mode and project.get("deploy_mode") != deploy_mode:
        fixture_fail(failures, name, f"expected deploy_mode={deploy_mode}, got {project.get('deploy_mode')}")
    if status and project.get("status") != status:
        fixture_fail(failures, name, f"expected status={status}, got {project.get('status')}")
    return project


def verify_route_fixture(
    failures: list[dict[str, str]],
    check_name: str,
    project: dict[str, Any],
    public_domain: str,
    expected_content: str | None,
    route_timeout_seconds: float = 120.0,
) -> None:
    host = host_for_project(project, public_domain)
    try:
        body = wait_for_route_body(host, timeout_seconds=route_timeout_seconds)
    except BackupRestoreError as exc:
        fixture_fail(failures, check_name, f"route {host} is unhealthy: {exc}")
        return
    if expected_content is not None and expected_content not in body:
        fixture_fail(failures, check_name, f"route {host} did not contain expected sentinel content")


def verify_static_sentinel(
    failures: list[dict[str, str]],
    project: dict[str, Any],
    static_root: pathlib.Path,
    sentinel_path: str,
    expected_content: str | None,
) -> None:
    rel = pathlib.PurePosixPath(sentinel_path)
    if rel.is_absolute() or ".." in rel.parts:
        fixture_fail(failures, "static", "static sentinel path must be relative and stay inside the project static root")
        return
    path = static_root / str(project["id"]) / pathlib.Path(*rel.parts)
    if not path.is_file():
        fixture_fail(failures, "static", f"static sentinel file is missing: {path}")
        return
    if expected_content is not None and expected_content not in path.read_text(encoding="utf-8", errors="replace"):
        fixture_fail(failures, "static", "static sentinel file does not contain expected content")


def verify_persistent_volume_fixture(failures: list[dict[str, str]], fixture: dict[str, Any]) -> None:
    volume_name = str(fixture.get("volumeName") or "").strip()
    sentinel_path = str(fixture.get("sentinelPath") or "").strip()
    if not volume_name or not sentinel_path:
        fixture_fail(failures, "persistentVolume", "volumeName and sentinelPath are required")
        return
    rows = docker_json(["volume", "inspect", volume_name])
    if not isinstance(rows, list) or not rows:
        fixture_fail(failures, "persistentVolume", f"volume is missing: {volume_name}")
        return
    mountpoint = pathlib.Path(str(rows[0].get("Mountpoint") or ""))
    rel = pathlib.PurePosixPath(sentinel_path)
    if rel.is_absolute() or ".." in rel.parts:
        fixture_fail(failures, "persistentVolume", "sentinelPath must be relative and stay inside the volume")
        return
    sentinel = mountpoint / pathlib.Path(*rel.parts)
    if not sentinel.is_file():
        fixture_fail(failures, "persistentVolume", f"persistent sentinel file is missing in {volume_name}")
        return
    if "expectedSha256" in fixture:
        if sha256_file(sentinel) != str(fixture["expectedSha256"]):
            fixture_fail(failures, "persistentVolume", "persistent sentinel checksum mismatch")
    expected = fixture.get("expectedContent")
    if expected is not None and str(expected) not in sentinel.read_text(encoding="utf-8", errors="replace"):
        fixture_fail(failures, "persistentVolume", "persistent sentinel content mismatch")


def verify_compose_workspace(
    failures: list[dict[str, str]],
    project: dict[str, Any],
    compose_root: pathlib.Path,
    fixture: dict[str, Any],
) -> None:
    candidates = [compose_root / str(project["id"]), compose_root / str(project["name"])]
    if fixture.get("workspacePath"):
        candidates.insert(0, pathlib.Path(str(fixture["workspacePath"])))
    if not any(path.exists() and any(path.rglob("*")) for path in candidates):
        fixture_fail(failures, "composeDatabase", "compose workspace is missing or empty")


def verify_compose_database_sentinel(failures: list[dict[str, str]], fixture: dict[str, Any]) -> None:
    service = str(fixture.get("serviceContainer") or "").strip()
    command = fixture.get("sentinelCommand")
    expected = fixture.get("expectedOutput")
    if not service or not isinstance(command, list) or not command:
        fixture_fail(failures, "composeDatabase", "serviceContainer and sentinelCommand are required")
        return
    try:
        output = run_text(["docker", "exec", service, *[str(part) for part in command]]).strip()
    except BackupRestoreError as exc:
        fixture_fail(failures, "composeDatabase", f"database sentinel command failed: {exc}")
        return
    if expected is not None and str(expected).strip() not in output:
        fixture_fail(failures, "composeDatabase", "database sentinel row/value was not observed")


def jwt_segment(payload: dict[str, Any]) -> str:
    raw = json.dumps(payload, separators=(",", ":")).encode("utf-8")
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")


def issue_internal_access_token(env: dict[str, str], postgres_user: str, postgres_db: str) -> str:
    jwt_secret = required_env(env, "JWT_SECRET")
    rows = control_json_query(
        postgres_user,
        postgres_db,
        "SELECT id::text, email, role FROM users ORDER BY created_at LIMIT 1",
    )
    if not rows:
        raise BackupRestoreError("cannot verify authenticated fixture checks because no restored user exists")
    user = rows[0]
    now = int(time.time())
    claims = {
        "userId": user["id"],
        "email": user["email"],
        "role": user["role"],
        "tokenUse": "access",
        "sub": user["id"],
        "exp": now + 300,
        "iat": now,
    }
    signing_input = jwt_segment({"alg": "HS256", "typ": "JWT"}) + "." + jwt_segment(claims)
    signature = hmac.new(jwt_secret.encode("utf-8"), signing_input.encode("ascii"), hashlib.sha256).digest()
    return signing_input + "." + base64.urlsafe_b64encode(signature).rstrip(b"=").decode("ascii")


def api_json(path: str, token: str, api_base: str) -> Any:
    raw = run_text(["curl", "-fsS", "--max-time", "15", "-H", f"Authorization: Bearer {token}", api_base.rstrip("/") + path])
    return json.loads(raw or "{}")


def verify_dbstudio_fixture(
    failures: list[dict[str, str]],
    project: dict[str, Any],
    token: str,
    api_base: str,
) -> None:
    project_id = str(project["id"])
    try:
        status_payload = api_json(f"/projects/{project_id}/db/status", token, api_base)
        schemas_payload = api_json(f"/projects/{project_id}/db/schemas", token, api_base)
    except BackupRestoreError as exc:
        fixture_fail(failures, "dbStudio", f"DB Studio API check failed: {exc}")
        return
    status_data = status_payload.get("data", status_payload) if isinstance(status_payload, dict) else {}
    schemas_data = schemas_payload.get("data", schemas_payload) if isinstance(schemas_payload, dict) else {}
    if not status_data.get("configured"):
        fixture_fail(failures, "dbStudio", "DB Studio status is not configured")
    if not isinstance(schemas_data, list):
        fixture_fail(failures, "dbStudio", "DB Studio schemas response is not a list")
    write_access = status_data.get("writeAccess")
    if write_access not in (None, False):
        fixture_fail(failures, "dbStudio", "DB Studio write access is enabled before an explicit write session")


def verify_encrypted_env_fixture(
    failures: list[dict[str, str]],
    project: dict[str, Any],
    key: str,
    token: str,
    api_base: str,
) -> None:
    try:
        payload = api_json(f"/projects/{project['id']}/env/{key}/reveal", token, api_base)
    except BackupRestoreError as exc:
        fixture_fail(failures, "encryptedEnv", f"encrypted env reveal failed: {exc}")
        return
    data = payload.get("data", payload) if isinstance(payload, dict) else {}
    if not data.get("value"):
        fixture_fail(failures, "encryptedEnv", "encrypted env value did not decrypt to a non-empty value")


def load_fixture_spec(path: pathlib.Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise BackupRestoreError("fixture spec must be a JSON object")
    return data


def preflight_source_fixtures(args: argparse.Namespace) -> int:
    spec = load_fixture_spec(pathlib.Path(args.spec))
    install_dir = pathlib.Path(args.install_dir).resolve()
    env_file = pathlib.Path(args.env_file or install_dir / ".env").resolve()
    env = parse_env_file(env_file)
    postgres_user = required_env(env, "POSTGRES_USER")
    postgres_db = required_env(env, "POSTGRES_DB")
    public_domain = str(spec.get("publicDomain") or env.get("PUBLIC_DOMAIN") or "").strip()
    if not public_domain:
        raise BackupRestoreError("publicDomain is required in fixture spec or PUBLIC_DOMAIN")
    api_base = str(spec.get("apiBase") or DEFAULT_LOCAL_API)
    route_timeout_seconds = float(spec.get("routeTimeoutSeconds", 120))
    static_root = pathlib.Path(args.static_root)
    compose_root = pathlib.Path(args.compose_root)

    failures: list[dict[str, str]] = []
    checks: list[str] = []

    static_spec = spec.get("static")
    if isinstance(static_spec, dict):
        project = require_project(failures, postgres_user, postgres_db, str(static_spec.get("projectName") or ""), "static")
        if project:
            verify_static_sentinel(failures, project, static_root, str(static_spec.get("sentinelPath") or "index.html"), static_spec.get("expectedContent"))
            verify_route_fixture(failures, "static", project, public_domain, static_spec.get("expectedContent"), route_timeout_seconds)
        checks.append("static")
    else:
        fixture_fail(failures, "static", "static fixture is missing from spec")

    container_spec = spec.get("container")
    if isinstance(container_spec, dict):
        project = require_project(failures, postgres_user, postgres_db, str(container_spec.get("projectName") or ""))
        if project:
            mode = project.get("deploy_mode")
            if mode not in ("image", "dockerfile"):
                fixture_fail(failures, "container", f"expected image/dockerfile project, got {mode}")
            if not project.get("app_port"):
                fixture_fail(failures, "container", "container fixture has no app_port")
            verify_route_fixture(failures, "container", project, public_domain, container_spec.get("expectedContent"), route_timeout_seconds)
        checks.append("container")
    else:
        fixture_fail(failures, "container", "container fixture is missing from spec")

    volume_spec = spec.get("persistentVolume")
    if isinstance(volume_spec, dict):
        verify_persistent_volume_fixture(failures, volume_spec)
        checks.append("persistentVolume")
    else:
        fixture_fail(failures, "persistentVolume", "persistent volume fixture is missing from spec")

    compose_spec = spec.get("composeDatabase")
    compose_project = None
    if isinstance(compose_spec, dict):
        compose_project = require_project(failures, postgres_user, postgres_db, str(compose_spec.get("projectName") or ""), "compose")
        if compose_project:
            verify_route_fixture(failures, "composeDatabase", compose_project, public_domain, compose_spec.get("expectedContent"), route_timeout_seconds)
            verify_compose_workspace(failures, compose_project, compose_root, compose_spec)
        verify_compose_database_sentinel(failures, compose_spec)
        checks.append("composeDatabase")
    else:
        fixture_fail(failures, "composeDatabase", "compose database fixture is missing from spec")

    try:
        token = issue_internal_access_token(env, postgres_user, postgres_db)
    except BackupRestoreError as exc:
        token = ""
        fixture_fail(failures, "auth", str(exc))

    dbstudio_spec = spec.get("dbStudio")
    if isinstance(dbstudio_spec, dict) and token:
        db_project_name = str(dbstudio_spec.get("projectName") or (compose_spec or {}).get("projectName") or "")
        db_project = compose_project if compose_project and compose_project.get("name") == db_project_name else require_project(failures, postgres_user, postgres_db, db_project_name)
        if db_project:
            verify_dbstudio_fixture(failures, db_project, token, api_base)
        checks.append("dbStudio")
    elif not isinstance(dbstudio_spec, dict):
        fixture_fail(failures, "dbStudio", "DB Studio fixture is missing from spec")

    env_spec = spec.get("encryptedEnv")
    if isinstance(env_spec, dict) and token:
        env_project = require_project(failures, postgres_user, postgres_db, str(env_spec.get("projectName") or ""))
        key = str(env_spec.get("key") or "").strip()
        if not key:
            fixture_fail(failures, "encryptedEnv", "encrypted env key is required")
        elif env_project:
            verify_encrypted_env_fixture(failures, env_project, key, token, api_base)
        checks.append("encryptedEnv")
    elif not isinstance(env_spec, dict):
        fixture_fail(failures, "encryptedEnv", "encrypted env fixture is missing from spec")

    report = {
        "schemaVersion": 1,
        "kind": "backup-restore-source-preflight",
        "checkedAt": utc_now(),
        "sourceGitSha": git_sha(install_dir),
        "checks": checks,
        "status": "FAIL" if failures else "PASS",
        "failures": failures,
        "secretsExposed": False,
    }
    if args.report:
        write_json_private(pathlib.Path(args.report), report)
    print(json.dumps(report, indent=2, sort_keys=True))
    return 1 if failures else 0


def archive_has_entries(path: pathlib.Path) -> bool:
    with tarfile.open(path, "r:gz") as archive:
        return any(member.name and member.name != "." for member in archive.getmembers())


def validate_fixture_manifest(args: argparse.Namespace) -> int:
    spec = load_fixture_spec(pathlib.Path(args.spec))
    bundle = pathlib.Path(args.bundle).resolve()
    verify_bundle(bundle)
    manifest = load_manifest(bundle)
    failures: list[dict[str, str]] = []
    records = manifest.get("files", [])
    kinds = [record.get("kind") for record in records]
    if "static-artifacts" not in kinds:
        fixture_fail(failures, "manifest", "static-artifacts record is missing")
    if "compose-workspaces" not in kinds:
        fixture_fail(failures, "manifest", "compose-workspaces record is missing")

    for kind in ("static-artifacts", "compose-workspaces"):
        record = next((item for item in records if item.get("kind") == kind), None)
        if record and not archive_has_entries(bundle / str(record["path"])):
            fixture_fail(failures, "manifest", f"{kind} archive is empty")

    min_volumes = int(spec.get("manifest", {}).get("minManagedVolumes", 1) if isinstance(spec.get("manifest"), dict) else 1)
    volume_records = [record for record in records if record.get("kind") == "managed-volume"]
    if len(volume_records) < min_volumes:
        fixture_fail(failures, "manifest", f"expected at least {min_volumes} managed-volume records, got {len(volume_records)}")
    expected_volume = (spec.get("persistentVolume") or {}).get("volumeName") if isinstance(spec.get("persistentVolume"), dict) else None
    if expected_volume and all(record.get("volumeName") != expected_volume for record in volume_records):
        fixture_fail(failures, "manifest", f"expected persistent volume {expected_volume} was not captured")
    compose_spec = spec.get("composeDatabase")
    expected_compose_volume = str(compose_spec.get("volumeName") or "").strip() if isinstance(compose_spec, dict) else ""
    if not expected_compose_volume:
        fixture_fail(failures, "manifest", "composeDatabase.volumeName must identify the expected Compose DB volume")
    elif all(record.get("volumeName") != expected_compose_volume for record in volume_records):
        fixture_fail(failures, "manifest", f"expected Compose DB volume {expected_compose_volume} was not captured")

    report = {
        "schemaVersion": 1,
        "kind": "backup-restore-fixture-manifest-validation",
        "checkedAt": utc_now(),
        "sourceGitSha": manifest.get("sourceGitSha", "unknown"),
        "projectCount": manifest.get("projectCount"),
        "managedVolumeCount": manifest.get("managedVolumeCount"),
        "status": "FAIL" if failures else "PASS",
        "failures": failures,
        "secretsExposed": False,
    }
    if args.report:
        write_json_private(pathlib.Path(args.report), report)
    print(json.dumps(report, indent=2, sort_keys=True))
    return 1 if failures else 0


def classify_volume(info: dict[str, Any], project_names: set[str]) -> str | None:
    labels = info.get("Labels") or {}
    name = str(info.get("Name") or "")
    if str(labels.get(MANAGED_LABEL, "")).lower() == MANAGED_VALUE:
        return "mypaas-managed"
    compose_project = str(labels.get("com.docker.compose.project") or "")
    if compose_project and compose_project in project_names:
        return "mypaas-compose"
    # Older Compose-created volumes can survive label changes. The exact active
    # project-name prefix is still constrained by the control-plane project list.
    if any(name.startswith(project + "_") for project in project_names):
        return "mypaas-compose-legacy"
    return None


def discover_managed_volumes(project_names: set[str]) -> list[ManagedVolume]:
    completed = run(["docker", "volume", "ls", "-q"])
    names = sorted({line.strip() for line in completed.stdout.decode().splitlines() if line.strip()})
    result: list[ManagedVolume] = []
    for name in names:
        rows = docker_json(["volume", "inspect", name])
        if not isinstance(rows, list) or not rows:
            continue
        info = rows[0]
        source = classify_volume(info, project_names)
        if source is None:
            continue
        mountpoint = str(info.get("Mountpoint") or "").strip()
        if not mountpoint:
            raise BackupRestoreError(f"managed volume {name} has no mountpoint")
        result.append(ManagedVolume(name=name, mountpoint=mountpoint, source=source))
    return result


def running_containers_for_volume(name: str) -> list[str]:
    completed = run(["docker", "ps", "-q", "--filter", f"volume={name}"])
    return [line.strip() for line in completed.stdout.decode().splitlines() if line.strip()]


def stop_containers(ids: Iterable[str]) -> list[str]:
    unique = sorted(set(ids))
    if unique:
        run(["docker", "stop", *unique])
    return unique


def start_containers(ids: Iterable[str]) -> None:
    unique = sorted(set(ids))
    if unique:
        run(["docker", "start", *unique])


def git_sha(install_dir: pathlib.Path) -> str:
    completed = run(["git", "-c", f"safe.directory={install_dir}", "-C", str(install_dir), "rev-parse", "HEAD"])
    return completed.stdout.decode().strip()


def dump_control_database(output: pathlib.Path, postgres_user: str, postgres_db: str) -> None:
    run([
        "docker",
        "exec",
        "mypaas-postgres-prod",
        "pg_dump",
        "--format=custom",
        "--no-owner",
        "--no-privileges",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
    ], stdout_file=output)
    os.chmod(output, 0o600)


def restore_control_database(dump_path: pathlib.Path, postgres_user: str, postgres_db: str) -> None:
    payload = dump_path.read_bytes()
    run([
        "docker",
        "exec",
        "-i",
        "mypaas-postgres-prod",
        "pg_restore",
        "--clean",
        "--if-exists",
        "--no-owner",
        "--no-privileges",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
    ], input_bytes=payload)


def recreate_api_if_present(install_dir: pathlib.Path, env_file: pathlib.Path, compose_file: pathlib.Path) -> bool:
    completed = run([
        "docker",
        "compose",
        "-f",
        str(compose_file),
        "--env-file",
        str(env_file),
        "ps",
        "--all",
        "-q",
        "api",
    ], check=False)
    if completed.returncode != 0:
        message = completed.stderr.decode("utf-8", errors="replace").strip()
        raise BackupRestoreError(f"cannot inspect API service before restore reconciliation: {message[:1000]}")
    if not completed.stdout.decode().strip():
        return False
    run([
        "docker",
        "compose",
        "-f",
        str(compose_file),
        "--env-file",
        str(env_file),
        "up",
        "-d",
        "--force-recreate",
        "api",
    ])
    return True


def ensure_postgres(install_dir: pathlib.Path, env_file: pathlib.Path, compose_file: pathlib.Path, postgres_user: str, postgres_db: str) -> None:
    run([
        "docker",
        "compose",
        "-f",
        str(compose_file),
        "--env-file",
        str(env_file),
        "up",
        "-d",
        "postgres",
    ])
    deadline = time.monotonic() + 90
    while time.monotonic() < deadline:
        completed = run([
            "docker",
            "compose",
            "-f",
            str(compose_file),
            "--env-file",
            str(env_file),
            "exec",
            "-T",
            "postgres",
            "pg_isready",
            "-U",
            postgres_user,
            "-d",
            postgres_db,
        ], check=False)
        if completed.returncode == 0:
            return
        time.sleep(2)
    raise BackupRestoreError("restored PostgreSQL did not become ready within 90 seconds")


def manifest_path(bundle: pathlib.Path) -> pathlib.Path:
    return bundle / "manifest.json"


def write_json_private(path: pathlib.Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.chmod(path, 0o600)


def load_manifest(bundle: pathlib.Path) -> dict[str, Any]:
    path = manifest_path(bundle)
    if not path.is_file():
        raise BackupRestoreError(f"missing backup manifest: {path}")
    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("schemaVersion") != SCHEMA_VERSION:
        raise BackupRestoreError(f"unsupported manifest schemaVersion {data.get('schemaVersion')}")
    if not isinstance(data.get("files"), list):
        raise BackupRestoreError("manifest files must be an array")
    return data


def verify_bundle(bundle: pathlib.Path) -> dict[str, Any]:
    manifest = load_manifest(bundle)
    checked = 0
    for record in manifest["files"]:
        rel = pathlib.PurePosixPath(str(record.get("path") or ""))
        if rel.is_absolute() or ".." in rel.parts:
            raise BackupRestoreError(f"unsafe manifest path: {rel}")
        path = bundle / pathlib.Path(*rel.parts)
        if not path.is_file():
            raise BackupRestoreError(f"missing backup file: {rel}")
        expected = str(record.get("sha256") or "")
        actual = sha256_file(path)
        if not expected or actual != expected:
            raise BackupRestoreError(f"checksum mismatch: {rel}")
        checked += 1
    return {
        "schemaVersion": 1,
        "kind": "backup-verify",
        "verifiedAt": utc_now(),
        "sourceGitSha": manifest.get("sourceGitSha", "unknown"),
        "filesChecked": checked,
        "status": "PASS",
    }


def backup(args: argparse.Namespace) -> int:
    install_dir = pathlib.Path(args.install_dir).resolve()
    env_file = pathlib.Path(args.env_file or install_dir / ".env").resolve()
    if not env_file.is_file():
        raise BackupRestoreError(f"missing production config: {env_file}")
    env = parse_env_file(env_file)
    postgres_user = required_env(env, "POSTGRES_USER")
    postgres_db = required_env(env, "POSTGRES_DB")

    source_sha = git_sha(install_dir)
    stamp = utc_now().replace(":", "").replace("-", "")
    output = pathlib.Path(args.output or f"{DEFAULT_BACKUP_ROOT}/full-{stamp}-{source_sha[:12]}").resolve()
    if output.exists() and any(output.iterdir()):
        raise BackupRestoreError(f"backup output already exists and is not empty: {output}")
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(output, 0o700)

    files: list[dict[str, Any]] = []
    private_config = output / "private" / "config.env"
    private_config.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    shutil.copyfile(env_file, private_config)
    os.chmod(private_config, 0o600)
    files.append(file_record("production-config", output, private_config))

    database_dump = output / "control-plane" / "postgres.dump"
    dump_control_database(database_dump, postgres_user, postgres_db)
    files.append(file_record("control-plane-postgres", output, database_dump))

    static_archive = output / "filesystem" / "static.tar.gz"
    archive_directory(pathlib.Path(args.static_root), static_archive)
    files.append(file_record("static-artifacts", output, static_archive))

    compose_archive = output / "filesystem" / "compose.tar.gz"
    archive_directory(pathlib.Path(args.compose_root), compose_archive)
    files.append(file_record("compose-workspaces", output, compose_archive))

    projects = active_project_names(postgres_user, postgres_db)
    volume_project_names = active_project_volume_names(postgres_user, postgres_db)
    volumes = discover_managed_volumes(volume_project_names)
    active: dict[str, list[str]] = {volume.name: running_containers_for_volume(volume.name) for volume in volumes}
    in_use = sorted({container for rows in active.values() for container in rows})
    if in_use and not args.quiesce_managed_containers:
        raise BackupRestoreError(
            "managed project volumes are mounted by running containers; re-run with "
            "--quiesce-managed-containers for a controlled consistent snapshot"
        )

    stopped: list[str] = []
    try:
        if in_use:
            stopped = stop_containers(in_use)
        for volume in volumes:
            archive = output / "volumes" / f"{volume.name}.tar.gz"
            archive_directory(pathlib.Path(volume.mountpoint), archive)
            files.append(file_record(
                "managed-volume",
                output,
                archive,
                volumeName=volume.name,
                discovery=volume.source,
            ))
    finally:
        if stopped:
            start_containers(stopped)

    manifest = {
        "schemaVersion": SCHEMA_VERSION,
        "kind": "mypaas-full-backup",
        "createdAt": utc_now(),
        "sourceGitSha": source_sha,
        "configurationIncluded": True,
        "configurationValuesExposedInManifest": False,
        "quiescedManagedContainers": bool(stopped),
        "projectCount": len(projects),
        "managedVolumeCount": len(volumes),
        "files": files,
    }
    write_json_private(manifest_path(output), manifest)
    report = {
        "schemaVersion": 1,
        "kind": "backup-create",
        "createdAt": manifest["createdAt"],
        "sourceGitSha": source_sha,
        "bundle": str(output),
        "projectCount": len(projects),
        "managedVolumeCount": len(volumes),
        "files": len(files),
        "status": "PASS",
        "containsSensitiveConfiguration": True,
    }
    write_json_private(output / "backup-report.json", report)
    print(output)
    return 0


def restore_volume(bundle: pathlib.Path, record: dict[str, Any]) -> None:
    name = str(record.get("volumeName") or "").strip()
    if not name:
        raise BackupRestoreError("managed-volume record is missing volumeName")
    archive = bundle / str(record["path"])
    running = running_containers_for_volume(name)
    if running:
        raise BackupRestoreError(f"refusing to restore in-use volume {name}; running containers: {len(running)}")
    run(["docker", "volume", "create", "--label", f"{MANAGED_LABEL}={MANAGED_VALUE}", name])
    rows = docker_json(["volume", "inspect", name])
    if not isinstance(rows, list) or not rows or not rows[0].get("Mountpoint"):
        raise BackupRestoreError(f"cannot resolve mountpoint for restored volume {name}")
    mountpoint = pathlib.Path(str(rows[0]["Mountpoint"]))
    clear_directory(mountpoint)
    safe_extract(archive, mountpoint)


def restore(args: argparse.Namespace) -> int:
    if not args.confirm_restore:
        raise BackupRestoreError("restore is destructive; pass --confirm-restore after reviewing the bundle")
    bundle = pathlib.Path(args.bundle).resolve()
    verify = verify_bundle(bundle)
    manifest = load_manifest(bundle)

    install_dir = pathlib.Path(args.install_dir).resolve()
    compose_file = pathlib.Path(args.compose_file or install_dir / "docker-compose.prod.yml").resolve()
    target_env = pathlib.Path(args.env_file or install_dir / ".env").resolve()
    if not install_dir.joinpath(".git").is_dir():
        raise BackupRestoreError(f"restore target is not a Git checkout: {install_dir}")
    if not compose_file.is_file():
        raise BackupRestoreError(f"missing production Compose file: {compose_file}")

    config_record = next((item for item in manifest["files"] if item.get("kind") == "production-config"), None)
    db_record = next((item for item in manifest["files"] if item.get("kind") == "control-plane-postgres"), None)
    static_record = next((item for item in manifest["files"] if item.get("kind") == "static-artifacts"), None)
    compose_record = next((item for item in manifest["files"] if item.get("kind") == "compose-workspaces"), None)
    if not all((config_record, db_record, static_record, compose_record)):
        raise BackupRestoreError("bundle is missing one or more mandatory restore records")

    source_sha = str(manifest.get("sourceGitSha") or "").strip()
    if args.require_matching_git_sha:
        target_sha = git_sha(install_dir)
        if target_sha != source_sha:
            raise BackupRestoreError(f"target checkout {target_sha} does not match backup source {source_sha}")

    target_env.parent.mkdir(parents=True, exist_ok=True)
    if target_env.exists():
        backup_name = target_env.with_name(target_env.name + ".pre-restore-" + str(int(time.time())))
        shutil.copy2(target_env, backup_name)
        os.chmod(backup_name, 0o600)
    shutil.copyfile(bundle / str(config_record["path"]), target_env)
    os.chmod(target_env, 0o600)
    env = parse_env_file(target_env)
    postgres_user = required_env(env, "POSTGRES_USER")
    postgres_db = required_env(env, "POSTGRES_DB")

    static_root = pathlib.Path(args.static_root)
    compose_root = pathlib.Path(args.compose_root)
    clear_directory(static_root)
    safe_extract(bundle / str(static_record["path"]), static_root)
    clear_directory(compose_root)
    safe_extract(bundle / str(compose_record["path"]), compose_root)

    volume_records = [item for item in manifest["files"] if item.get("kind") == "managed-volume"]
    for record in volume_records:
        restore_volume(bundle, record)

    ensure_postgres(install_dir, target_env, compose_file, postgres_user, postgres_db)
    restore_control_database(bundle / str(db_record["path"]), postgres_user, postgres_db)
    api_recreated = recreate_api_if_present(install_dir, target_env, compose_file)

    report = {
        "schemaVersion": 1,
        "kind": "backup-restore",
        "restoredAt": utc_now(),
        "sourceGitSha": source_sha,
        "targetInstallDir": str(install_dir),
        "bundleVerified": verify["status"] == "PASS",
        "configurationRestored": True,
        "controlPlaneDatabaseRestored": True,
        "staticArtifactsRestored": True,
        "composeWorkspacesRestored": True,
        "managedVolumesRestored": len(volume_records),
        "apiRecreatedAfterDatabaseRestore": api_recreated,
        "status": "PASS",
        "nextStep": "run deploy-to-vm.sh, verify-production.sh, then execute workload-specific recovery checks",
    }
    report_path = pathlib.Path(args.report or bundle / "restore-report.json")
    write_json_private(report_path, report)
    print(report_path)
    return 0


def plan(args: argparse.Namespace) -> int:
    output = {
        "kind": "backup-restore-plan",
        "backupIncludes": [
            "production .env (sensitive, private bundle file)",
            "control-plane PostgreSQL custom dump",
            DEFAULT_STATIC_ROOT,
            DEFAULT_COMPOSE_ROOT,
            "volumes labeled mypaas.managed=true",
            "Compose volumes belonging to active MyPaas projects",
        ],
        "backupConsistency": "running managed-volume consumers require --quiesce-managed-containers",
        "restoreSafety": "restore requires --confirm-restore and verifies all manifest checksums before mutation",
        "runtimeAcceptance": "fresh-VM login/projects/env/routes/deployments/persistent-data/DB-Studio checks remain external evidence",
        "qualifyingSourcePreflight": "run source-preflight with the fixture spec before creating a fixture-qualified backup",
        "qualifyingManifestPreflight": "run validate-fixture-manifest after backup creation before any fixture-driven restore",
    }
    print(json.dumps(output, indent=2))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)

    backup_parser = sub.add_parser("backup", help="create a full disaster-recovery bundle")
    backup_parser.add_argument("--install-dir", default=os.getenv("MYPAAS_INSTALL_DIR", DEFAULT_INSTALL_DIR))
    backup_parser.add_argument("--env-file", default="")
    backup_parser.add_argument("--static-root", default=DEFAULT_STATIC_ROOT)
    backup_parser.add_argument("--compose-root", default=DEFAULT_COMPOSE_ROOT)
    backup_parser.add_argument("--output", default="")
    backup_parser.add_argument("--quiesce-managed-containers", action="store_true")
    backup_parser.set_defaults(func=backup)

    verify_parser = sub.add_parser("verify", help="verify bundle manifest and checksums without mutation")
    verify_parser.add_argument("--bundle", required=True)
    verify_parser.add_argument("--report", default="")

    restore_parser = sub.add_parser("restore", help="restore a verified bundle onto a prepared host")
    restore_parser.add_argument("--bundle", required=True)
    restore_parser.add_argument("--install-dir", default=os.getenv("MYPAAS_INSTALL_DIR", DEFAULT_INSTALL_DIR))
    restore_parser.add_argument("--env-file", default="")
    restore_parser.add_argument("--compose-file", default="")
    restore_parser.add_argument("--static-root", default=DEFAULT_STATIC_ROOT)
    restore_parser.add_argument("--compose-root", default=DEFAULT_COMPOSE_ROOT)
    restore_parser.add_argument("--report", default="")
    restore_parser.add_argument("--require-matching-git-sha", action="store_true", default=True)
    restore_parser.add_argument("--no-require-matching-git-sha", dest="require_matching_git_sha", action="store_false")
    restore_parser.add_argument("--confirm-restore", action="store_true")
    restore_parser.set_defaults(func=restore)

    preflight_parser = sub.add_parser("source-preflight", help="verify restore fixtures before backup")
    preflight_parser.add_argument("--spec", required=True, help="JSON file describing required source fixtures")
    preflight_parser.add_argument("--install-dir", default=os.getenv("MYPAAS_INSTALL_DIR", DEFAULT_INSTALL_DIR))
    preflight_parser.add_argument("--env-file", default="")
    preflight_parser.add_argument("--static-root", default=DEFAULT_STATIC_ROOT)
    preflight_parser.add_argument("--compose-root", default=DEFAULT_COMPOSE_ROOT)
    preflight_parser.add_argument("--report", default="")
    preflight_parser.set_defaults(func=preflight_source_fixtures)

    manifest_parser = sub.add_parser("validate-fixture-manifest", help="verify backup manifest covers the declared fixture set")
    manifest_parser.add_argument("--bundle", required=True)
    manifest_parser.add_argument("--spec", required=True, help="same JSON fixture spec used for source-preflight")
    manifest_parser.add_argument("--report", default="")
    manifest_parser.set_defaults(func=validate_fixture_manifest)

    plan_parser = sub.add_parser("plan", help="show the backup/restore contract without touching the host")
    plan_parser.set_defaults(func=plan)
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.command == "verify":
        report = verify_bundle(pathlib.Path(args.bundle).resolve())
        if args.report:
            write_json_private(pathlib.Path(args.report), report)
        else:
            print(json.dumps(report, indent=2))
        return 0
    try:
        return int(args.func(args))
    except BackupRestoreError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
