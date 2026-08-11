package statd

import "sync/atomic"

type AvailabilityTransition uint8

const (
	AvailabilityUnchanged AvailabilityTransition = iota
	AvailabilityInitialAvailable
	AvailabilityInitialUnavailable
	AvailabilityRecovered
	AvailabilityLost
)

// TelemetrySnapshot is a process-local view of statd integration health.
// Counters intentionally have no labels so the /metrics endpoint stays
// low-cardinality regardless of project count.
type TelemetrySnapshot struct {
	Available          bool
	Fallbacks          uint64
	SnapshotErrors     uint64
	RegistrationErrors uint64
}

// 0 = unknown/not exercised yet, 1 = available, -1 = unavailable.
var (
	telemetryAvailability       atomic.Int32
	telemetryFallbacks          atomic.Uint64
	telemetrySnapshotErrors     atomic.Uint64
	telemetryRegistrationErrors atomic.Uint64
)

// MarkAvailable updates process-local statd health and reports only meaningful
// state transitions so callers can log degradation/recovery without emitting a
// warning on every metrics refresh.
func MarkAvailable(available bool) AvailabilityTransition {
	next := int32(-1)
	if available {
		next = 1
	}
	previous := telemetryAvailability.Swap(next)
	if previous == next {
		return AvailabilityUnchanged
	}
	if previous == 0 {
		if available {
			return AvailabilityInitialAvailable
		}
		return AvailabilityInitialUnavailable
	}
	if available {
		return AvailabilityRecovered
	}
	return AvailabilityLost
}

func RecordFallback() {
	telemetryFallbacks.Add(1)
}

func RecordSnapshotError() {
	telemetrySnapshotErrors.Add(1)
}

func RecordRegistrationError() {
	telemetryRegistrationErrors.Add(1)
}

func Telemetry() TelemetrySnapshot {
	return TelemetrySnapshot{
		Available:          telemetryAvailability.Load() == 1,
		Fallbacks:          telemetryFallbacks.Load(),
		SnapshotErrors:     telemetrySnapshotErrors.Load(),
		RegistrationErrors: telemetryRegistrationErrors.Load(),
	}
}
