package deployment

import (
	"errors"
	"fmt"
	"testing"

	"mypaas/internal/statd"
)

func TestRecordUnexpectedSnapshotErrorIgnoresColdMiss(t *testing.T) {
	before := statd.Telemetry().SnapshotErrors
	recordUnexpectedSnapshotError(&statd.ProtocolError{Code: "NOT_FOUND"})
	if got := statd.Telemetry().SnapshotErrors; got != before {
		t.Fatalf("SnapshotErrors = %d, want unchanged %d for cold NOT_FOUND", got, before)
	}
}

func TestRecordUnexpectedSnapshotErrorCountsOperationalFailure(t *testing.T) {
	before := statd.Telemetry().SnapshotErrors
	recordUnexpectedSnapshotError(errors.New("socket read failed"))
	if got := statd.Telemetry().SnapshotErrors; got != before+1 {
		t.Fatalf("SnapshotErrors = %d, want %d", got, before+1)
	}
}

func TestRuntimeIdentityMayBeStale(t *testing.T) {
	if runtimeIdentityMayBeStale(errors.New("statd connect: connection refused")) {
		t.Fatal("connection failure must not trigger Docker runtime rediscovery")
	}
	if runtimeIdentityMayBeStale(&statd.ProtocolError{Code: "NOT_FOUND"}) {
		t.Fatal("cold NOT_FOUND must not trigger Docker runtime rediscovery")
	}
	wrapped := fmt.Errorf("replace statd registration: register: %w", &statd.ProtocolError{Code: "REGISTER_FAILED"})
	if !runtimeIdentityMayBeStale(wrapped) {
		t.Fatal("terminal REGISTER_FAILED must permit runtime rediscovery")
	}
}
