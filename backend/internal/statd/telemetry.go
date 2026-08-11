package statd

import "sync/atomic"

// TelemetrySnapshot is a process-local view of statd integration health.
// Counters intentionally have no labels so the /metrics endpoint stays
// low-cardinality regardless of project count.
type TelemetrySnapshot struct {
	Available          bool
	Fallbacks          uint64
	SnapshotErrors     uint64
	RegistrationErrors uint64
}

var (
	telemetryAvailable          atomic.Bool
	telemetryFallbacks          atomic.Uint64
	telemetrySnapshotErrors     atomic.Uint64
	telemetryRegistrationErrors atomic.Uint64
)

func MarkAvailable(available bool) {
	telemetryAvailable.Store(available)
}

func RecordFallback() {
	telemetryFallbacks.Add(1)
}

func recordSnapshotError() {
	telemetrySnapshotErrors.Add(1)
}

func recordRegistrationError() {
	telemetryRegistrationErrors.Add(1)
}

func Telemetry() TelemetrySnapshot {
	return TelemetrySnapshot{
		Available:          telemetryAvailable.Load(),
		Fallbacks:          telemetryFallbacks.Load(),
		SnapshotErrors:     telemetrySnapshotErrors.Load(),
		RegistrationErrors: telemetryRegistrationErrors.Load(),
	}
}
