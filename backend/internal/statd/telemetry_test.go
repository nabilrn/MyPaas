package statd

import "testing"

func TestTelemetryCountersAndAvailabilityTransitions(t *testing.T) {
	telemetryAvailability.Store(0)
	before := Telemetry()

	if got := MarkAvailable(true); got != AvailabilityInitialAvailable {
		t.Fatalf("initial available transition = %v", got)
	}
	if got := MarkAvailable(true); got != AvailabilityUnchanged {
		t.Fatalf("repeated available transition = %v", got)
	}
	if got := MarkAvailable(false); got != AvailabilityLost {
		t.Fatalf("lost transition = %v", got)
	}
	if got := MarkAvailable(false); got != AvailabilityUnchanged {
		t.Fatalf("repeated unavailable transition = %v", got)
	}
	if got := MarkAvailable(true); got != AvailabilityRecovered {
		t.Fatalf("recovery transition = %v", got)
	}

	RecordFallback()
	RecordSnapshotError()
	RecordRegistrationError()

	after := Telemetry()
	if !after.Available {
		t.Fatal("Available = false, want true")
	}
	if after.Fallbacks != before.Fallbacks+1 {
		t.Fatalf("Fallbacks = %d, want %d", after.Fallbacks, before.Fallbacks+1)
	}
	if after.SnapshotErrors != before.SnapshotErrors+1 {
		t.Fatalf("SnapshotErrors = %d, want %d", after.SnapshotErrors, before.SnapshotErrors+1)
	}
	if after.RegistrationErrors != before.RegistrationErrors+1 {
		t.Fatalf("RegistrationErrors = %d, want %d", after.RegistrationErrors, before.RegistrationErrors+1)
	}
}

func TestTelemetryInitialUnavailable(t *testing.T) {
	telemetryAvailability.Store(0)
	if got := MarkAvailable(false); got != AvailabilityInitialUnavailable {
		t.Fatalf("initial unavailable transition = %v", got)
	}
	if Telemetry().Available {
		t.Fatal("Available = true after initial unavailable state")
	}
}
