# Contributing to MyPaaS

Thanks for helping improve MyPaaS.

## Public contribution model

MyPaaS currently uses an **Issues-only public contribution workflow**.

Please use GitHub Issues for:

- reproducible bug reports;
- feature requests grounded in a real deployment or operator problem.

Unsolicited pull requests are not part of the supported public contribution path at this time. Implementation work is maintained through the repository owner's normal branch/PR workflow after an issue or internal task is accepted.

Before opening an issue, check existing issues to avoid duplicates.

## Bug reports

Use the Bug report issue form and include enough information to reproduce the problem:

- what failed;
- minimal reproduction steps;
- expected behavior;
- observed behavior;
- server OS;
- container runtime (rootful Podman or Docker compatibility mode);
- MyPaaS version or Git SHA when known;
- deployment mode involved (Dockerfile, Compose, Static, or OCI Image);
- relevant secret-safe logs or screenshots.

Never include tokens, cookies, passwords, registry credentials, OAuth credentials, decrypted environment values, production `.env` contents, or backup material containing secrets.

## Feature requests

Use the Feature request issue form and describe the **real application/operator problem first**.

A useful request explains:

- the workload or workflow that is blocked;
- which existing MyPaaS primitive was tried;
- why Dockerfile, Compose, Static, or OCI Image deployment cannot handle the requirement cleanly;
- the smallest reusable platform capability that would solve it;
- relevant security, persistence, routing, lifecycle, or recovery implications.

A workload-specific upstream/configuration issue or host-resource limit is not automatically a MyPaaS feature gap.

## Product boundaries

MyPaaS is intentionally a **single-host self-hosted PaaS** for an owner developer or small trusted team.

Requests for Kubernetes/Nomad/Swarm, distributed scheduling, automatic horizontal scaling, hostile multi-tenant isolation, arbitrary raw TCP/SSH/UDP exposure, generic host-port forwarding, or broad performance programs are outside the current product direction unless that direction changes explicitly.

Current product scope and engineering rules are documented in:

- [`README.md`](README.md)
- [`PRODUCT.md`](PRODUCT.md)
- [`ROADMAP.md`](ROADMAP.md)
- [`docs/README.md`](docs/README.md)
- [`AGENTS.md`](AGENTS.md)

## Security reports

Do not publish secrets or sensitive production material in an issue. If a security problem can be described safely without exposing credentials or private data, open a bug report with the minimum reproducible information and clearly identify the security impact.
