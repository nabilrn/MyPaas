package main

import (
	"fmt"
	"io"

	"mypaas/internal/statd"
)

func writeStatdMetrics(w io.Writer) {
	telemetry := statd.Telemetry()
	available := 0
	if telemetry.Available {
		available = 1
	}

	_, _ = fmt.Fprintf(w, "# HELP mypaas_statd_available Whether the preferred statd metrics path is currently available.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mypaas_statd_available gauge\nmypaas_statd_available %d\n", available)
	_, _ = fmt.Fprintf(w, "# HELP mypaas_statd_fallback_total Number of runtime metric requests that fell back to the Docker-compatible path.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mypaas_statd_fallback_total counter\nmypaas_statd_fallback_total %d\n", telemetry.Fallbacks)
	_, _ = fmt.Fprintf(w, "# HELP mypaas_statd_snapshot_errors_total Unexpected statd snapshot failures observed by the MyPaaS integration.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mypaas_statd_snapshot_errors_total counter\nmypaas_statd_snapshot_errors_total %d\n", telemetry.SnapshotErrors)
	_, _ = fmt.Fprintf(w, "# HELP mypaas_statd_registration_errors_total Terminal statd runtime registration failures observed by the MyPaaS integration.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mypaas_statd_registration_errors_total counter\nmypaas_statd_registration_errors_total %d\n", telemetry.RegistrationErrors)
}
