package container

import (
	"math"
	"testing"
	"time"
)

func TestParseStatsLine(t *testing.T) {
	line := `{"Name":"mypaas-demo","CPUPerc":"3.45%","MemUsage":"27.5MiB / 512MiB"}`

	metrics, err := parseStatsLine(line)
	if err != nil {
		t.Fatalf("parseStatsLine() error = %v", err)
	}

	if metrics.Service != "mypaas-demo" {
		t.Fatalf("Service = %q, want mypaas-demo", metrics.Service)
	}
	assertFloat(t, metrics.CPUPercent, 3.45)
	assertFloat(t, metrics.MemoryMB, 27.5)
	assertFloat(t, metrics.MemoryLimitMB, 512)
}

func TestParseMemoryUsage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantUsed  float64
		wantLimit float64
	}{
		{name: "mib", input: "27.5MiB / 512MiB", wantUsed: 27.5, wantLimit: 512},
		{name: "gib", input: "1.25GiB / 8GiB", wantUsed: 1280, wantLimit: 8192},
		{name: "bytes", input: "1048576B / 536870912B", wantUsed: 1, wantLimit: 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, limit, err := parseMemoryUsage(tt.input)
			if err != nil {
				t.Fatalf("parseMemoryUsage() error = %v", err)
			}
			assertFloat(t, used, tt.wantUsed)
			assertFloat(t, limit, tt.wantLimit)
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{name: "seconds", duration: 45 * time.Second, want: "<1m"},
		{name: "minutes", duration: 17 * time.Minute, want: "17m"},
		{name: "hours", duration: 2*time.Hour + 8*time.Minute, want: "2h 8m"},
		{name: "days", duration: 49*time.Hour + 30*time.Minute, want: "2d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatUptime(tt.duration); got != tt.want {
				t.Fatalf("formatUptime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
