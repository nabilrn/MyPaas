# ADR-022: Bounded Private Registry Authentication

## Status

Accepted

## Context

MyPaas already deploys public OCI images, but a real single-node PaaS also needs to pull private images without requiring an operator to run a persistent `docker login` on the host. The platform should not turn this into a registry proxy, mirror, or cache subsystem.

## Decision

MyPaas supports one optional platform-level registry credential through:

- `MYPAAS_REGISTRY_HOST`
- `MYPAAS_REGISTRY_USERNAME`
- `MYPAAS_REGISTRY_PASSWORD`

All three values must be configured together. Credentials are only applied when the image registry host matches `MYPAAS_REGISTRY_HOST`; they are never sent to an unrelated registry.

For each authenticated pull, MyPaas:

1. creates a private temporary Docker configuration directory;
2. runs `docker login <host> --username <username> --password-stdin` using that isolated `DOCKER_CONFIG`;
3. pulls the image with the same isolated configuration;
4. removes the temporary configuration after the pull.

This deliberately avoids changing the host user's persistent Docker credential store.

Docker Hub aliases (`index.docker.io` and `registry-1.docker.io`) normalize to `docker.io` for credential matching.

Pull failures are classified into actionable product errors where evidence is available:

- authentication failure;
- permission denial;
- registry rate limiting;
- missing image/tag;
- generic registry command failure.

## Boundaries

- This phase supports one configured registry credential at a time.
- Registry credentials are installation-level operational secrets, not project environment variables.
- No pull-through cache, registry mirror, image proxy, credential broker, or registry UI is introduced.
- Compose services continue to use the normal Compose image-pull behavior; this contract initially targets MyPaas `sourceType=registry` / `deployMode=image` pulls.
- The control-plane registry variables are intentionally excluded from project Compose environment inheritance by the existing Compose host-environment allowlist.

## Consequences

- Private GHCR, Docker Hub, or another single authenticated registry can be used for OCI-image projects.
- Anonymous public pulls continue to work when no registry credential is configured.
- Rate-limit failures become distinguishable from authentication and missing-tag failures without adding speculative caching infrastructure.
- A future dashboard credential manager can replace the installation-level input contract without changing image-pull semantics.
