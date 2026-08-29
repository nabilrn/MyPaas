package project

import (
	"strings"
	"testing"
)

func TestComposeCommandErrorDetailPreservesDecoderCause(t *testing.T) {
	stderr := "1 error(s) decoding:\n\n* error decoding 'services.frontend.ports': invalid type\n"

	got := composeCommandErrorDetail(stderr, "")

	if !strings.Contains(got, "1 error(s) decoding:") {
		t.Fatalf("composeCommandErrorDetail() lost summary: %q", got)
	}
	if !strings.Contains(got, "services.frontend.ports") {
		t.Fatalf("composeCommandErrorDetail() lost actionable decoder cause: %q", got)
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("composeCommandErrorDetail() should be toast-friendly single-line text: %q", got)
	}
}

func TestComposeCommandErrorDetailPrefersStderrAndFallsBackToStdout(t *testing.T) {
	if got := composeCommandErrorDetail("stderr detail", "stdout detail"); got != "stderr detail" {
		t.Fatalf("stderr should win, got %q", got)
	}
	if got := composeCommandErrorDetail("", "stdout detail"); got != "stdout detail" {
		t.Fatalf("stdout fallback = %q, want stdout detail", got)
	}
	if got := composeCommandErrorDetail("", ""); got != "docker compose config failed without diagnostic output" {
		t.Fatalf("empty diagnostic fallback = %q", got)
	}
}

func TestComposeCommandErrorDetailCapsDiagnosticSize(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = strings.Repeat("x", 400)
	}

	got := composeCommandErrorDetail(strings.Join(lines, "\n"), "")

	if len(got) > 2003 {
		t.Fatalf("diagnostic should be bounded, got %d chars", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("bounded diagnostic should signal truncation: %q", got[len(got)-20:])
	}
}
