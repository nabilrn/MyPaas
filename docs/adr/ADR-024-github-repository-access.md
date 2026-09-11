# ADR-024: GitHub repository picker and private-source access

**Status:** Accepted
**Date:** 2026-09-01
**Amended:** 2026-09-11

## Context

The New Project flow previously required an operator to paste a repository URL. That was also awkward for private repositories because repository inspection and deployment had no persisted GitHub credential path.

MyPaaS is a single-host platform for one owner or a small trusted team. GitHub OAuth is already the account connection used by the dashboard, so the same connection can provide repository discovery and private-source access without adding a separate credential form.

GitHub OAuth can issue expiring access tokens together with rotating refresh tokens. Treating every access-token expiry as a broken connection makes repository selection and deployment repeatedly require an interactive reconnect even though GitHub has provided a non-interactive recovery path.

## Decision

The New Project form provides a **Choose a repository** picker backed by `GET /auth/github/repositories`. The API requests the repositories available to the signed-in GitHub account, including private and organization repositories, and returns a paginated list with the clone URL and default branch.

The GitHub OAuth flow requests `repo` plus `offline_access` alongside the identity scopes. MyPaaS stores the returned OAuth credential encrypted in the existing control-plane token columns. The encrypted payload can include the access token, refresh token, token type, and access-token expiry. Existing rows that contain only a legacy long-lived access token remain readable.

Repository inspection, project creation, deployment, rollback, and Compose route validation obtain an access token through the project owner's control-plane record. When an expiring access token is no longer valid, the control plane uses the refresh token to rotate the credential and persists the replacement before continuing. Repository discovery may make one forced refresh-and-retry after a GitHub `401` to cover expiry or rotation between the local validity check and the API request.

Interactive reconnect is reserved for missing credentials, credentials without a usable refresh path, an invalid/expired refresh token, or authorization that remains rejected after refresh. Transient refresh failures do not erase the stored credential.

For GitHub HTTPS remotes, Git receives the access token through a process-scoped HTTP authorization configuration. Tokens are never put in command arguments, repository configuration, deployment logs, or project workload environments. Non-GitHub remotes continue to use the existing unauthenticated path.

## Boundaries

- The picker is available only to an authenticated administrator with a connected GitHub account.
- Access and refresh tokens remain encrypted at rest in the control-plane database.
- Missing or revoked repository authorization returns an actionable reconnect response only after the available refresh path has been exhausted.
- Refresh-token rotation is serialized inside the GitHub credential service so concurrent repository/deployment reads cannot spend the same rotating refresh token twice.
- The implementation supports GitHub HTTPS clone URLs; SSH remotes are not rewritten.
- The current OAuth `repo` scope is broader than read-only repository access. A future GitHub App or narrower OAuth design should reduce that permission without changing the picker contract.

## Consequences

Operators can choose a repository without copying a URL, private GitHub repositories work through the same inspection and deployment paths as public repositories, and normal access-token expiry does not require repeated manual reconnects.

The control plane holds encrypted GitHub OAuth credentials and must be treated as trusted host authority, consistent with the existing engine-socket boundary. A genuinely revoked authorization or an exhausted refresh credential still requires reconnecting GitHub.
