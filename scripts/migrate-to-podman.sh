#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
MyPaas Docker -> Podman in-place migration has been retired.

Why:
  Docker Engine and Podman keep engine-managed containers and named volumes in
  different storage. Removing Docker Engine and switching the socket can leave
  MyPaas control-plane or project data behind.

Supported path:
  1. Prepare a VM migration package from MyPaas Admin -> Settings -> VM Migration.
  2. Provision a fresh Ubuntu/Debian VM with Podman:

       curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env USE_PODMAN=true bash

  3. Restore the migration package on the new VM.

For disposable development hosts, reinstalling from scratch with USE_PODMAN=true
is also safe when no state needs to be preserved.
EOF
}

case "${1:-}" in
  -h|--help)
    usage
    exit 0
    ;;
  "")
    usage >&2
    printf '\nERROR: refusing destructive in-place engine migration; use a fresh Podman VM and the VM migration workflow.\n' >&2
    exit 2
    ;;
  *)
    printf 'ERROR: unknown argument: %s\n\n' "$1" >&2
    usage >&2
    exit 2
    ;;
esac
