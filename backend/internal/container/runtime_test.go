package container

import (
	"errors"
	"testing"
	"time"
)

func TestParseRuntimeProcesses(t *testing.T) {
	raw := []byte(`[
		{
			"Id":"abc123",
			"Name":"/mypaas-demo-web-1",
			"State":{"Pid":4321,"StartedAt":"2026-08-10T10:00:00.123456789Z"},
			"Config":{"Labels":{"com.docker.compose.service":"web"}}
		},
		{
			"Id":"def456",
			"Name":"/mypaas-demo-worker-1",
			"State":{"Pid":0,"StartedAt":"0001-01-01T00:00:00Z"},
			"Config":{"Labels":{"com.docker.compose.service":"worker"}}
		}
	]`)

	rows, err := parseRuntimeProcesses(raw)
	if err != nil {
		t.Fatalf("parseRuntimeProcesses() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].ID != "abc123" || rows[0].Service != "web" || rows[0].PID != 4321 {
		t.Fatalf("rows[0] = %+v", rows[0])
	}
	wantStartedAt := time.Date(2026, time.August, 10, 10, 0, 0, 123456789, time.UTC)
	if !rows[0].StartedAt.Equal(wantStartedAt) {
		t.Fatalf("StartedAt = %s, want %s", rows[0].StartedAt, wantStartedAt)
	}
	if rows[1].PID != 0 || !rows[1].StartedAt.IsZero() {
		t.Fatalf("rows[1] = %+v", rows[1])
	}
}

func TestParseRuntimeProcessesFallsBackToContainerName(t *testing.T) {
	raw := []byte(`[{"Id":"abc123","Name":"/mypaas-demo","State":{"Pid":123,"StartedAt":"2026-08-10T10:00:00Z"},"Config":{"Labels":{}}}]`)

	rows, err := parseRuntimeProcesses(raw)
	if err != nil {
		t.Fatalf("parseRuntimeProcesses() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Service != "mypaas-demo" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestParseRuntimeProcessesRejectsMalformedData(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "invalid json", raw: `{`},
		{name: "missing id", raw: `[{"Id":"","State":{"Pid":123}}]`},
		{name: "invalid startedAt", raw: `[{"Id":"abc","State":{"Pid":123,"StartedAt":"not-a-time"}}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseRuntimeProcesses([]byte(tt.raw)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseRuntimeProcessesEmptyIsNoContainer(t *testing.T) {
	_, err := parseRuntimeProcesses([]byte(`[]`))
	if !errors.Is(err, ErrNoContainer) {
		t.Fatalf("error = %v, want ErrNoContainer", err)
	}
}
