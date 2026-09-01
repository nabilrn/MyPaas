# ADR-024: GitHub repository picker and private-source access

**Status:** Accepted
**Date:** 2026-09-01

## Context

The New Project flow previously required an operator to paste a repository URL. That was also awkward for private repositories because repository inspection and deployment had no persisted GitHub credential path.

MyPaaS is a single-host platform for one owner or a small trusted team. GitHub OAuth is already the account connection used by the dashboard, so the same connection can provide repository discovery and private-source access without adding a separate credential form.

## Decision

The New Project form provides a **Choose a repository** picker backed by `GET /auth/github/repositories`. The API requests the repositories available to the signed-in GitHub account, including private and organization repositories, and returns a paginated list with the clone URL and default branch.

The GitHub OAuth flow requests the `repo` scope and stores the returned access token encrypted in the control-plane database. Repository inspection, project creation, deployment, rollback, and Compose route validation obtain the token through the project owner's control-plane record.

For GitHub HTTPS remotes, Git receives the token through a process-scoped HTTP authorization configuration. The token is never put in command arguments, repository configuration, deployment logs, or project workload environments. Non-GitHub remotes continue to use the existing unauthenticated path.

## Boundaries

- The picker is available only to an authenticated administrator with a connected GitHub account.
- Missing or revoked repository authorization returns an actionable reconnect response.
- The implementation supports GitHub HTTPS clone URLs; SSH remotes are not rewritten.
- The current OAuth `repo` scope is broader than read-only repository access. A future GitHub App or narrower OAuth design should reduce that permission without changing the picker contract.

## Consequences

Operators can choose a repository without copying a URL, and private GitHub repositories work through the same inspection and deployment paths as public repositories.

The control plane now holds an encrypted GitHub access token and must be treated as trusted host authority, consistent with the existing engine-socket boundary. Token revocation or expiry requires reconnecting GitHub before private repository operations can continue.
