#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
The standalone MyPaas migration exporter has been retired.

Use the migration workflow in:
  MyPaas Dashboard -> Admin -> Settings -> VM Migration

The backend migration service is the single supported exporter because it:
  - performs storage safety preflight checks;
  - temporarily quiesces running container-backed project runtimes;
  - dumps the MyPaas and shared PostgreSQL databases;
  - archives supported host-managed persistent directories;
  - resumes project runtimes before marking the export ready.

Engine-managed Compose named/external volumes are intentionally rejected by the
preflight instead of being silently omitted. Move that persistent data to bind
mounts under /var/lib/mypaas/volumes or migrate the listed volumes separately.
EOF
}

case "${1:-}" in
  -h|--help)
    usage
    exit 0
    ;;
  "")
    usage >&2
    printf '\nERROR: refusing the retired standalone exporter; use Admin -> Settings -> VM Migration.\n' >&2
    exit 2
    ;;
  *)
    printf 'ERROR: unknown argument: %s\n\n' "$1" >&2
    usage >&2
    exit 2
    ;;
esac
