# Product

## Register

product

## Users

MyPaas is used by the owner developer and a small whitelist of collaborators who deploy personal or small-team projects from Git repositories. They are usually in an operational context: checking whether an app is running, creating a deployment target, watching logs, changing environment variables, rolling back a bad deploy, or cleaning stale Compose resources on a constrained self-hosted VM.

## Product Purpose

MyPaas makes self-hosted deployment feel closer to Vercel or Railway while keeping ownership of infrastructure. It connects Git repositories to Dockerfile, Compose, or static deployments, manages routing through Caddy and Cloudflare Tunnel, tracks deployment history, exposes logs and metrics, and keeps common lifecycle actions close at hand. Success means a developer can deploy, diagnose, and recover a project without writing repetitive infrastructure glue.

## Brand Personality

Quiet, capable, and operationally precise. The interface should feel modern and elegant, but in the way a serious control plane feels elegant: dense enough for repeated use, clear under pressure, and confident without decorative noise.

## Anti-references

Avoid marketing-site composition, oversized hero sections, decorative gradients, nested card stacks, glassmorphism, cartoon illustrations, and repetitive wizard steps that slow down deployment. Avoid looking like a generic dark SaaS template, a purple-blue AI dashboard, or a consumer landing page. The UI should not hide deployment state behind vague spinners or imply secrets/ports are known when they are only fallbacks.

## Design Principles

1. Deployment state first: every screen should make current state, next action, and risk obvious before the user acts.
2. Calm density: compact, aligned information beats spacious decoration because the product is used repeatedly.
3. Honest automation: auto-detected values must be labeled as detected, fallback, static, or manual.
4. Recovery is a core flow: retry, rollback, reconnect, reveal, and reset states must be visible and trustworthy.
5. One control plane: shared components and consistent navigation matter more than page-local visual novelty.

## Accessibility & Inclusion

Target WCAG AA contrast for text and controls. Preserve keyboard access, visible focus, semantic buttons/links, reduced-motion safe interactions, and copy that does not rely on color alone for status. Dense operational screens must remain readable on small laptop and mobile viewports without clipped labels or hidden primary actions.
