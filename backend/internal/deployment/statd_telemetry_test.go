package deployment

import (
	"errors"
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
