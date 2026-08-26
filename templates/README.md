# MyPaas application templates

`templates/` contains product-facing deployment manifests. They are intentionally separate from `compatibility/` fixtures.

- `compatibility/` proves whether a workload pattern works on a tested MyPaas host.
- `templates/` provides safe defaults for users creating real projects.
- Product templates must not contain fixed passwords, fixed encryption keys, or compatibility-only credentials.
- Secrets are supplied through MyPaas environment variables and encrypted by the existing environment-variable service.
- Templates reuse the existing project create/deploy lifecycle. They do not introduce a second deployment engine.

The first template set is intentionally small: Excalidraw, Uptime Kuma, n8n, Umami, Ghost, and NocoDB. Heavy workloads remain outside the default installer until their host requirements can be presented safely and clearly.
