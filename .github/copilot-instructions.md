# MyPaaS Copilot Instructions

Use [`AGENTS.md`](../AGENTS.md) as the canonical persistent engineering context.

Source-of-truth order:

1. current code, migrations, tests, installers, and production configuration;
2. `AGENTS.md`;
3. `README.md` / `PRODUCT.md`;
4. current architecture/security docs;
5. accepted ADRs;
6. `ROADMAP.md`.

`docs/PRD.md` is historical and must not override current implementation.

Keep these current product boundaries in mind:

- single-host PaaS for an owner/small trusted team;
- Podman-first fresh installs with Docker Engine compatibility through the Docker-compatible contract;
- Dockerfile, Compose, static, and OCI image deployment;
- bounded image-mode private-registry authentication per ADR-022;
- bounded Compose additional HTTP routes per ADR-023;
- no raw TCP/SSH/UDP routing or arbitrary secondary host-port exposure;
- no Kubernetes/Nomad/Swarm/multi-node scheduler/autoscaling roadmap;
- no throughput/capacity claims from compatibility fixtures;
- fix reusable platform gaps discovered by real applications instead of adding speculative architecture.

Follow `AGENTS.md` for implementation, security, routing, testing, branching, and documentation rules.
