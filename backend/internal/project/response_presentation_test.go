package project

import "testing"

func TestDetectResponseKeepsBlockingIssuesAndCompactsNonBlockingDiagnostics(t *testing.T) {
	plan := &ComposePlan{Issues: []ComposeIssue{
		{Severity: "warning", Code: "W1", Message: "warning one"},
		{Severity: "error", Code: "E1", Message: "blocking one"},
		{Severity: "info", Code: "I1", Message: "info one"},
		{Severity: "warning", Code: "W2", Message: "warning two"},
		{Severity: "error", Code: "E2", Message: "blocking two"},
		{Severity: "info", Code: "I2", Message: "info two"},
		{Severity: "warning", Code: "W3", Message: "warning three"},
	}}

	resp := DetectResponseFromResult(DetectResult{ComposePlan: plan})
	if resp.ComposePlan == nil {
		t.Fatal("expected compose plan")
	}

	var blocking int
	for _, issue := range resp.ComposePlan.Issues {
		if issue.Severity == "error" {
			blocking++
		}
	}
	if blocking != 2 {
		t.Fatalf("expected all 2 blocking issues, got %d", blocking)
	}
	if len(resp.ComposePlan.Issues) != 6 {
		t.Fatalf("expected 2 blockers + 3 non-blockers + summary, got %d issues", len(resp.ComposePlan.Issues))
	}
	last := resp.ComposePlan.Issues[len(resp.ComposePlan.Issues)-1]
	if last.Code != "ADDITIONAL_DIAGNOSTICS" {
		t.Fatalf("expected compact diagnostics summary, got %q", last.Code)
	}
	if len(plan.Issues) != 7 {
		t.Fatalf("presentation compaction must not mutate source plan, got %d source issues", len(plan.Issues))
	}
}

func TestDetectResponseLeavesConciseDiagnosticsUntouched(t *testing.T) {
	plan := &ComposePlan{Issues: []ComposeIssue{
		{Severity: "error", Code: "E1", Message: "blocking"},
		{Severity: "warning", Code: "W1", Message: "warning"},
		{Severity: "info", Code: "I1", Message: "info"},
	}}

	resp := DetectResponseFromResult(DetectResult{ComposePlan: plan})
	if resp.ComposePlan != plan {
		t.Fatal("expected concise compose plan to remain unchanged")
	}
}
