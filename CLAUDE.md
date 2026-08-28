# CLAUDE.md — MyPaaS

Use [`AGENTS.md`](AGENTS.md) as the canonical persistent engineering context for this repository.

Before changing architecture or product behavior, read in this order:

1. current code, migrations, tests, installers, and production configuration;
2. `AGENTS.md`;
3. `README.md` and `PRODUCT.md`;
4. current architecture/security docs under `docs/`;
5. accepted ADRs;
6. `ROADMAP.md`.

`docs/PRD.md` is historical and is **not** the current runtime source of truth.

Critical current boundaries:

- single-host PaaS for an owner/small trusted team;
- Podman-first fresh installs, Docker Engine compatibility through one Docker-compatible contract;
- Dockerfile, Compose, static, and OCI image deployment modes;
- one bounded configured registry credential for image-mode authenticated pulls;
- bounded Compose additional HTTP routes are delivered and qualified (ADR-023);
- no generic raw TCP/SSH/UDP routing or extra secondary host-port publication;
- no Kubernetes/Nomad/Swarm/multi-node scheduler/autoscaling roadmap;
- no broad performance matrices or capacity claims;
- compatibility work should discover real reusable product gaps, not create speculative features.

Follow `AGENTS.md` for detailed security, routing, testing, documentation, and branching rules.
