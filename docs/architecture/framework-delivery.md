# Framework delivery optimization

MyPaaS keeps the single-host architecture and optimizes delivery at the Caddy/runtime boundary before considering replicas or orchestration.

## Implemented defaults

### Pure static deployments

Static releases are published atomically under `STATIC_ROOT` and served directly by Caddy.

Caddy applies:

- `Cache-Control: public, max-age=0, must-revalidate` by default;
- `Cache-Control: public, max-age=31536000, immutable` only to known fingerprinted build namespaces and only when the requested file exists:
  - Next.js `/_next/static/*` when present in a published static tree;
  - Astro `/_astro/*`;
  - Nuxt `/_nuxt/*`;
  - SvelteKit `/_app/immutable/*`;
  - Vite default hash-named `/assets/*-*.*` output;
- gzip and Zstandard response encoding for eligible responses;
- existing `.br` and `.gz` sidecars through Caddy `file_server` precompressed support;
- build-time `.gz` sidecar generation for sufficiently large textual static assets.

Unhashed files are deliberately not promoted to immutable caching. This avoids stale `public/` assets and other user-controlled filenames. Missing requests in known static asset namespaces return `404` instead of falling through to an SPA shell.

### Static routing semantics

MyPaaS no longer applies `/index.html` history fallback to every static project.

During static publication it recognizes obvious client-rendered Vite/legacy SPA builds and writes a reserved `.mypaas-spa-fallback` release marker. Caddy uses that marker only as a route condition:

- React/Vue/Svelte-style Vite SPA builds receive history fallback after real-file lookup fails;
- Astro static output, plain HTML, Vite multi-page builds, and SvelteKit static output keep ordinary file-server `404` behavior by default;
- real files and directory indexes always win before SPA fallback;
- missing `/assets/*`, `/_astro/*`, `/_nuxt/*`, `/_app/*`, `/_next/static/*`, and `/static/*` requests return `404` and never become `index.html`;
- direct requests to `.mypaas-*` metadata are blocked;
- source-controlled `.mypaas-*` files are discarded during publication so application code cannot forge routing metadata.

This keeps deep-route behavior explicit enough for framework qualification without persisting another database field or guessing that every static build is an SPA.

### Dockerfile, registry, Compose and API runtimes

Caddy performs response compression before `reverse_proxy` for generic container, SSR and API traffic. Application cache semantics are preserved; MyPaaS does not add blanket caching to dynamic API or SSR responses.

Express/Fastify/Nest and PostgreSQL-backed Node APIs therefore keep application/database behavior intact while compression is offloaded to the proxy layer.

### Framework detection

Repository inspection records framework-specific delivery guidance for:

- Vite static builds;
- Astro static builds;
- Next.js standalone/runtime deployments;
- Nuxt SSR/static builds;
- SvelteKit Node/static builds;
- NestJS and common Node API frameworks;
- Compose projects.

The detector does not claim direct Caddy delivery for assets that exist only inside a runtime image.

## Deliberately not implemented yet

### Next.js and Nuxt runtime image asset extraction

Direct Caddy delivery for `/_next/static/*` or `/_nuxt/*` from an SSR Docker image requires a first-class artifact-publication lifecycle:

1. identify the framework build artifact inside the final image without assuming `WORKDIR`;
2. extract only the documented static subtree;
3. publish it atomically alongside the deployment;
4. make route activation and rollback switch runtime and static artifacts together;
5. preserve framework cache headers and dynamic SSR semantics.

MyPaaS does not guess an image path or point Caddy at a non-existent host directory. Until that lifecycle exists, SSR assets remain behind the runtime reverse proxy and can still be compressed and cached by an upstream CDN according to framework response headers.

### Cloudflare Tunnel scaling

The default connector count is unchanged. Additional `cloudflared` replicas are availability/failover controls, not a proven latency optimization, so they remain opt-in.

### Application/process scaling

No Node clustering, application replicas, Kubernetes, PgBouncer, arbitrary PostgreSQL pool changes, or Caddy low-level tuning is added without evidence from the final qualification run.

## Qualification boundary

The framework benchmark should test all fixtures in parallel only after this code is deployed. Results must separate:

- Caddy-served static delivery;
- runtime/SSR delivery;
- API latency;
- database CRUD latency;
- Cloudflare/Tunnel behavior;
- host CPU, RAM and NIC throughput.

No capacity or superiority claim is implied by these delivery defaults.
