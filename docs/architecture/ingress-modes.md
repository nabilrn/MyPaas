# Production ingress modes

MyPaaS supports three production delivery modes without changing its single-machine architecture.

## `tunnel`

Default and backwards-compatible mode.

```text
Cloudflare edge -> Cloudflare Tunnel -> Caddy :80 -> project/static runtime
```

Use this for private hosts, CGNAT, restricted campus networks, or deployments where opening inbound 443 is not acceptable. `cloudflared` is profile-gated and only starts in this mode.

Required:

- `PUBLIC_DELIVERY_MODE=tunnel`
- `CLOUDFLARE_TUNNEL_TOKEN`

Host Caddy ports are bound to loopback by default; the connector reaches Caddy over the internal Compose network.

## `cloudflare-origin`

Preferred public high-throughput mode when Cloudflare remains the edge/CDN/WAF but Tunnel should not be in the hot path.

```text
Cloudflare edge -> HTTPS origin :443 -> Caddy -> project/static runtime
```

Required:

- `PUBLIC_DELIVERY_MODE=cloudflare-origin`
- `/etc/mypaas/tls/cert.pem`
- `/etc/mypaas/tls/key.pem`

The certificate must cover both `PUBLIC_DOMAIN` and `*.PUBLIC_DOMAIN`. A Cloudflare Origin CA wildcard certificate is appropriate when the origin is reachable only through proxied Cloudflare DNS.

Before switching production traffic, restrict the host firewall so public application ingress is limited to the intended proxy/source addresses. Keep PostgreSQL, the MyPaaS API, dashboard ports, project runtime ports, and the Caddy admin socket private.

## `direct`

Direct public HTTPS to Caddy without Cloudflare Tunnel.

```text
client -> HTTPS :443 -> Caddy -> project/static runtime
```

Required:

- `PUBLIC_DELIVERY_MODE=direct`
- a publicly valid certificate at `/etc/mypaas/tls/cert.pem`
- matching key at `/etc/mypaas/tls/key.pem`
- certificate coverage for `PUBLIC_DOMAIN` and `*.PUBLIC_DOMAIN`

Use this for controlled origin benchmarking or organizations that intentionally operate their own public ingress. It does not provide Cloudflare CDN/WAF/DDoS proxy benefits.

## Safety behavior

`scripts/deploy-to-vm.sh` validates delivery mode and TLS material before switching the stack. HTTPS-origin modes verify certificate expiry, root hostname coverage, wildcard project-host coverage, and certificate/key matching.

When leaving Tunnel mode, known `cloudflared` containers are removed so an old connector cannot remain silently active.

`Caddyfile.prod.https` intentionally runs a single `:443` application server and disables automatic HTTP redirects. The current dynamic route writer targets Caddy server `srv0`; adding a second HTTP redirect server before that route writer becomes listener-aware would make the server identity ambiguous. HTTP redirect support should therefore be implemented together with server discovery, not guessed in this ingress change.

CI validates both Caddy configs and performs an HTTPS smoke test that inserts a project hostname through the same Caddy Admin Unix-socket route surface used by MyPaaS, then verifies the dynamic project route is reachable with the wildcard certificate.

## Firewall boundary

Recommended public exposure:

- `443/tcp`: public for `direct`; Cloudflare-source-only for `cloudflare-origin` where practical.
- `22/tcp`: management network/VPN/allowlist only.
- `80/tcp`: not required by the current HTTPS-origin profile.

Keep private:

- PostgreSQL `5432`
- MyPaaS API `8080`
- dashboard `3000`
- project runtime host ports
- Caddy Admin API (Unix socket only)

Docker/Podman published-port filtering must be validated at the engine/firewall layer; do not assume a host firewall rule automatically constrains container-published ports.
