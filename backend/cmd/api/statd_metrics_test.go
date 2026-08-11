package main

import (
	"bytes"
	"strings"
	"testing"

	"mypaas/internal/statd"
)

func TestWriteStatdMetrics(t *testing.T) {
	statd.MarkAvailable(true)
	before := statd.Telemetry()
	statd.RecordFallback()
	statd.RecordSnapshotError()
	statd.RecordRegistrationError()

	var out bytes.Buffer
	writeStatdMetrics(&out)
	text := out.String()

	checks := []string{
		"mypaas_statd_available 1",
		"mypaas_statd_fallback_total " + uintString(before.Fallbacks+1),
		"mypaas_statd_snapshot_errors_total " + uintString(before.SnapshotErrors+1),
		"mypaas_statd_registration_errors_total " + uintString(before.RegistrationErrors+1),
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("metrics output missing %q:\n%s", check, text)
		}
	}
}

func uintString(value uint64) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}
