package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mypaas/internal/envdiscover"
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
      INTERNAL_PASSWORD: $${INTERNAL_PASSWORD}
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

func TestComposeServicePlanEmptyCollectionsMarshalAsArrays(t *testing.T) {
	item := ComposeServicePlan{
		Name:      "db",
		Role:      "internal",
		Ports:     composePortPlans(nil),
		Expose:    composeExposePorts(nil),
		DependsOn: composeDependsOn(nil),
	}

	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{`"ports":[]`, `"expose":[]`, `"dependsOn":[]`} {
		if !strings.Contains(content, want) {
			t.Fatalf("ComposeServicePlan JSON missing %s in %s", want, content)
		}
	}
}

func TestComposeServicePlanHandlesAbsoluteBuildContextFromComposeConfig(t *testing.T) {
	workspace := t.TempDir()
	serverDir := filepath.Join(workspace, "server")
	if err := os.MkdirAll(serverDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "Dockerfile"), []byte("FROM alpine\n"), 0640); err != nil {
		t.Fatal(err)
	}
	rawBuild, err := json.Marshal(map[string]any{
		"context":    serverDir,
		"dockerfile": "Dockerfile",
	})
	if err != nil {
		t.Fatal(err)
	}

	item := composeServicePlanFromConfig(workspace, "backend", composeServiceConfig{Build: rawBuild})
	if item.BuildContext == nil || *item.BuildContext != "server" {
		t.Fatalf("BuildContext = %#v, want server", item.BuildContext)
	}

	plan := &ComposePlan{Issues: make([]ComposeIssue, 0)}
	addComposeServiceIssues(plan, workspace, "backend", item, composeServiceConfig{})
	if hasComposeIssue(plan.Issues, "BUILD_CONTEXT_MISSING") || hasComposeIssue(plan.Issues, "DOCKERFILE_MISSING") {
		t.Fatalf("absolute build context should resolve inside workspace, got issues: %#v", plan.Issues)
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

func TestPrepareComposePreviewEnvWritesPlaceholders(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "compose.yml"), []byte(`
services:
  db:
    image: mariadb
    environment:
      MARIADB_ROOT_PASSWORD: ${DB_ROOT_PASSWORD}
      MARIADB_DATABASE: ${DB_NAME}
      MARIADB_PORT: ${DB_PORT:-3306}
      HEALTHCHECK_PASSWORD: $${MARIADB_ROOT_PASSWORD}
`), 0640); err != nil {
		t.Fatal(err)
	}

	if err := prepareComposePreviewEnv(workspace, "compose.yml", []envdiscover.Var{
		{Key: "DB_ROOT_PASSWORD", Sensitive: true},
		{Key: "DB_NAME"},
	}); err != nil {
		t.Fatalf("prepareComposePreviewEnv() error = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(workspace, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"DB_ROOT_PASSWORD=mypaas_preview", "DB_NAME=mypaas_preview", "DB_PORT=3000"} {
		if !strings.Contains(content, want) {
			t.Fatalf(".env preview missing %q in:\n%s", want, content)
		}
	}
	if strings.Contains(content, "MARIADB_ROOT_PASSWORD=") {
		t.Fatalf(".env preview should not include escaped Compose variable:\n%s", content)
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
