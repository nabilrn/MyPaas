package statd

import "testing"

func TestTelemetryCountersAndAvailability(t *testing.T) {
	before := Telemetry()

	MarkAvailable(true)
	RecordFallback()
	recordSnapshotError()
	recordRegistrationError()

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

	MarkAvailable(false)
	if Telemetry().Available {
		t.Fatal("Available = true after MarkAvailable(false)")
	}
}
