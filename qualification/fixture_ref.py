#!/usr/bin/env python3
"""Resolve an immutable fixture revision to a clonable remote branch."""

from __future__ import annotations

import re
import subprocess
from dataclasses import dataclass

FULL_SHA = re.compile(r"^[0-9a-fA-F]{40}$")
PREFERRED_CANDIDATE_BRANCH = "test/beta-readiness-candidate"


class FixtureRefError(RuntimeError):
    pass


@dataclass(frozen=True)
class FixtureRef:
    requested_ref: str
    branch: str
    resolved_sha: str


def _remote_branch_heads(repo: str) -> dict[str, str]:
    repo = repo.strip()
    if not repo:
        raise FixtureRefError("fixture repository is required")
    try:
        completed = subprocess.run(
            ["git", "ls-remote", "--heads", repo],
            capture_output=True,
            text=True,
            timeout=30,
            check=False,
        )
    except (OSError, subprocess.SubprocessError) as exc:
        raise FixtureRefError(f"failed to inspect fixture repository branches: {exc}") from exc
    if completed.returncode != 0:
        detail = (completed.stderr or completed.stdout).strip().splitlines()
        suffix = f": {detail[0]}" if detail else ""
        raise FixtureRefError(f"failed to inspect fixture repository branches{suffix}")

    heads: dict[str, str] = {}
    for raw in completed.stdout.splitlines():
        fields = raw.strip().split()
        if len(fields) < 2 or not fields[1].startswith("refs/heads/"):
            continue
        sha = fields[0].lower()
        branch = fields[1][len("refs/heads/") :]
        if FULL_SHA.fullmatch(sha) and branch:
            heads[branch] = sha
    if not heads:
        raise FixtureRefError("fixture repository has no remote branches")
    return heads


def resolve_fixture_ref(repo: str, requested_ref: str, explicit_branch: str = "") -> FixtureRef:
    """Resolve fixture identity without ever sending a raw commit SHA as API branch.

    A full 40-character SHA is treated as the immutable expected revision. The
    resolver finds a remote branch whose head equals that SHA, preferring the
    beta candidate branch when multiple branches match. A non-SHA ref continues
    to behave as a branch name, but is verified to exist remotely before the
    destructive harness starts.
    """

    requested = requested_ref.strip()
    branch_override = explicit_branch.strip()
    if not requested:
        raise FixtureRefError("fixture ref is required")

    heads = _remote_branch_heads(repo)
    if FULL_SHA.fullmatch(requested):
        expected = requested.lower()
        if branch_override:
            actual = heads.get(branch_override)
            if actual is None:
                raise FixtureRefError(f"fixture branch {branch_override!r} does not exist on the remote")
            if actual != expected:
                raise FixtureRefError(
                    f"fixture branch {branch_override!r} resolves to {actual}, expected immutable SHA {expected}"
                )
            return FixtureRef(requested_ref=expected, branch=branch_override, resolved_sha=actual)

        matches = sorted(branch for branch, sha in heads.items() if sha == expected)
        if not matches:
            raise FixtureRefError(
                "fixture SHA is not the head of any remote branch; publish or select a branch at that exact revision"
            )
        branch = PREFERRED_CANDIDATE_BRANCH if PREFERRED_CANDIDATE_BRANCH in matches else matches[0]
        return FixtureRef(requested_ref=expected, branch=branch, resolved_sha=expected)

    branch = branch_override or requested
    actual = heads.get(branch)
    if actual is None:
        raise FixtureRefError(f"fixture branch {branch!r} does not exist on the remote")
    return FixtureRef(requested_ref=requested, branch=branch, resolved_sha=actual)
