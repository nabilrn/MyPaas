# Installer preflight contract

The browser installer performs live checks before production configuration is saved. These checks are diagnostic and do not mutate GitHub, DNS, or Cloudflare account configuration.

## Domain

`Check domain` validates a bare DNS hostname from the VM, resolves the dashboard hostname, and probes one randomized project subdomain to detect wildcard DNS visibility.

A missing wildcard is reported as a warning because the dashboard hostname may be valid before project wildcard routing is configured. This check proves DNS visibility from the VM; it does not prove Cloudflare zone ownership.

## GitHub OAuth

`Test GitHub configuration` validates the MyPaaS callback shape, then submits the configured Client ID, Client Secret, callback URL, and a deliberately invalid authorization code to GitHub's OAuth token endpoint.

The expected successful preflight result is `bad_verification_code`: GitHub accepted the supplied OAuth app credentials far enough to reject the intentionally fake authorization code. `incorrect_client_credentials` and `redirect_uri_mismatch`, when returned by GitHub, are surfaced as configuration failures.

A `bad_verification_code` response is **not** presented as proof that GitHub has stored the exact callback URL. Exact callback registration and owner identity are ultimately proven during a real GitHub sign-in. The installer only guarantees that the configured URL has the required `https://<host>/api/auth/github/callback` shape before that sign-in.

This check does **not** claim that the owner email is verified.

## Cloudflare Tunnel

`Test tunnel token` accepts either the `eyJ...` Tunnel token itself or the full Cloudflare Add-a-replica command containing it. The server extracts the token and starts a short-lived `cloudflared` connector.

The token is written to a mode-`0600` temporary env file and passed to Docker/Podman through `--env-file`. This keeps the secret out of process arguments and also works when the Docker-compatible CLI is reached through `sudo`, where environment forwarding cannot be assumed.

The check succeeds only when the connector readiness endpoint becomes healthy. The temporary connector and env file are removed after the test.

## Security

All `/preflight/*` requests require the same ephemeral wizard token as the installer. Responses never echo OAuth secrets or Tunnel tokens. Unexpected errors return a generic message so subprocess/network diagnostics cannot leak credentials into the browser.
