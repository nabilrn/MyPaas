package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRequiredComposeEnvVarsSkipsDefaults(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "compose.yml"), []byte(`
services:
  app:
    image: demo:${APP_VERSION:-local}
    environment:
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      PORT: ${PORT:-3000}
`), 0640); err != nil {
		t.Fatal(err)
	}

	got := requiredComposeEnvVars(workspace, "compose.yml", nil)
	want := []string{"DB_PASSWORD", "DB_USER"}
	if len(got) != len(want) {
		t.Fatalf("requiredComposeEnvVars() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("requiredComposeEnvVars()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestComposePortPlansParsesPublishedHostPort(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{{
		"target":    80,
		"published": "80",
		"protocol":  "tcp",
	}})
	if err != nil {
		t.Fatal(err)
	}

	got := composePortPlans(raw)
	if len(got) != 1 {
		t.Fatalf("composePortPlans() returned %d ports, want 1", len(got))
	}
	if got[0].Target != 80 || got[0].Published == nil || *got[0].Published != "80" {
		t.Fatalf("composePortPlans()[0] = %#v", got[0])
	}
}

func TestComposeServiceIssuesDetectReservedPortAndMissingDockerfile(t *testing.T) {
	workspace := t.TempDir()
	context := "client"
	dockerfile := "Dockerfile"
	published := "80"
	service := "frontend"
	plan := &ComposePlan{}
	item := ComposeServicePlan{
		Name:         service,
		BuildContext: &context,
		Dockerfile:   &dockerfile,
		Ports:        []ComposePortPlan{{Target: 80, Published: &published}},
	}

	addComposeServiceIssues(plan, workspace, service, item, composeServiceConfig{})

	if !hasComposeIssue(plan.Issues, "BUILD_CONTEXT_MISSING") {
		t.Fatalf("missing BUILD_CONTEXT_MISSING issue: %#v", plan.Issues)
	}
	if !hasComposeIssue(plan.Issues, "DOCKERFILE_MISSING") {
		t.Fatalf("missing DOCKERFILE_MISSING issue: %#v", plan.Issues)
	}
	if !hasComposeIssue(plan.Issues, "HOST_PORT_RESERVED") {
		t.Fatalf("missing HOST_PORT_RESERVED issue: %#v", plan.Issues)
	}
}

func hasComposeIssue(issues []ComposeIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
