# ADR-018: VM update channels and release-safe updates

## Status

Accepted design; source-side implementation requires runtime qualification before the next release.

## Context

MyPaas has two different operational needs that must not be conflated:

- a **production installation** should remain on a published release until the owner explicitly chooses a newer published release;
- a **staging/development installation** may intentionally follow a mutable branch/ref while qualifying unreleased changes.

The original updater could follow `main` through `MYPAAS_REF`/`AUTO_UPDATE_REF`. That is useful for staging but unsafe as the default product update model because a production VM can advance merely because the development branch advanced.

MyPaas already publishes API/dashboard images with immutable full Git SHA tags and `scripts/update-vm.sh` can deploy an explicit ref while keeping source and image revisions aligned. The missing boundary is release discovery and update-channel intent.

## Decision

### 1. Two explicit update channels

`MYPAAS_UPDATE_CHANNEL` defines update intent:

- `release` — production/default dashboard behavior;
- `ref` — explicit staging/development behavior.

If unset, dashboard update behavior defaults to `release`.

### 2. Production release discovery is notification-first

For the `release` channel, the owner Settings surface checks published GitHub Releases, including prereleases used during beta.

A release is eligible only when:

- it is not a draft;
- it has a tag;
- its `target_commitish` is an immutable full Git commit SHA.

The dashboard reports one of:

- `Update available`;
- `Up to date`;
- `Release check unavailable`;
- `Unknown build` when the running build is not identifiable.

A failed external release check must not make platform settings unavailable. It must instead disable blind dashboard updates.

### 3. Dashboard updates are release-pinned

The dashboard update action is accepted only on the `release` channel and only after the backend verifies that a newer published release target differs from the running `MYPAAS_BUILD_SHA`.

When accepted, the backend starts `scripts/update-vm.sh` with:

```text
MYPAAS_REF=<verified-release-tag>
```

The existing updater then resolves that tag and waits for the corresponding immutable SHA-tagged API/dashboard images before deployment.

The dashboard must never translate `Update MyPaas` into an implicit `main` update.

### 4. Ref tracking remains explicit engineering behavior

A staging/development installation may use:

```text
MYPAAS_UPDATE_CHANNEL=ref
MYPAAS_REF=main
```

or another explicit candidate ref.

On this channel the dashboard shows the tracked ref and refuses the production release-update action. Host-side updater/timer workflows may still be used intentionally for staging qualification.

### 5. Existing updater safety remains

`scripts/update-vm.sh` continues to:

- serialize updates with a host lock;
- refuse dirty installer-managed checkouts;
- wait for immutable SHA-tagged API/dashboard images;
- reset the managed checkout only after target artifacts exist;
- run deployment and post-update verification;
- attempt best-effort rollback to the previous verified runtime when deployment fails.

No Watchtower-style privileged updater container is introduced.

## Consequences

### Positive

- Production and staging no longer share the same implicit update policy.
- A new commit on `main` alone cannot make the production dashboard perform an update.
- Production owners can see that a release exists before deciding to install it.
- Dashboard-triggered updates are pinned to a published release tag rather than a mutable branch.
- GitHub release discovery failing closed does not prevent normal Settings use.

### Trade-offs

- A production installation must already contain this release-aware code before it can notify about later releases; the first transition from an older beta therefore requires one explicit upgrade/bootstrap.
- GitHub Releases availability affects notification freshness, but not the running platform.
- Automatic rollback remains best effort when database migrations are not reversible.

## Qualification required before release

Source tests must prove:

1. draft releases are ignored;
2. prerelease/beta releases are considered;
3. mutable release targets such as `main` are rejected;
4. the running release is recognized by exact source SHA;
5. a newer published release produces `update_available=true`;
6. the dashboard update process receives the verified release tag through `MYPAAS_REF`;
7. `ref` channel does not query GitHub Releases and refuses dashboard release updates.

Runtime qualification on the staging VM must later prove the end-to-end behavior before publishing the release that production will consume.