# Installer preflight contract

The browser installer performs live checks before production configuration is saved. These checks are diagnostic and do not mutate GitHub, DNS, or Cloudflare account configuration.

## Domain

`Check domain` validates a bare DNS hostname from the VM, resolves the dashboard hostname, and probes one randomized project subdomain to detect wildcard DNS visibility.

A missing wildcard is reported as a warning because the dashboard hostname may be valid before project wildcard routing is configured.

## GitHub OAuth

`Test GitHub configuration` submits the configured Client ID, Client Secret, callback URL, and a deliberately invalid authorization code to GitHub's OAuth token endpoint.

The expected successful preflight result is `bad_verification_code`: GitHub accepted the OAuth app credentials and callback, but the intentionally fake authorization code cannot be exchanged. `incorrect_client_credentials` and `redirect_uri_mismatch` are surfaced as configuration failures.

This check does **not** claim that the owner email is verified. Owner identity is proven during a real GitHub sign-in.

## Cloudflare Tunnel

`Test tunnel token` accepts either the `eyJ...` Tunnel token itself or the full Cloudflare Add-a-replica command containing it. The server extracts the token and starts a short-lived `cloudflared` connector using `TUNNEL_TOKEN` in the child environment, never in process arguments.

The check succeeds only when the connector readiness endpoint becomes healthy. The temporary connector is terminated after the test.

## Security

All `/preflight/*` requests require the same ephemeral wizard token as the installer. Responses never echo OAuth secrets or Tunnel tokens. Unexpected errors return a generic message so subprocess/network diagnostics cannot leak credentials into the browser.
