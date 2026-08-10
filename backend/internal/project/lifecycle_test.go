package project

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mypaas/internal/envvar"
)

func TestValidateLifecycleEnvVarsRejectsInvalidKeysBeforeCreate(t *testing.T) {
	if err := validateLifecycleEnvVars([]envvar.Value{{Key: "GOOD_KEY", Value: "ok"}}); err != nil {
		t.Fatalf("valid env key rejected: %v", err)
	}
	if err := validateLifecycleEnvVars([]envvar.Value{{Key: "BAD-KEY", Value: "nope"}}); err == nil {
		t.Fatal("invalid env key unexpectedly accepted")
	}
}

func TestPreflightProjectWorkspaceRejectsMissingBaseDirectory(t *testing.T) {
	repo := initLifecycleGitRepo(t, map[string]string{
		"Dockerfile": "FROM scratch\n",
	})
	input := CreateInput{
		RepoURL:       repo,
		Branch:        "main",
		DeployMode:    "dockerfile",
		AppPort:       80,
		BaseDirectory: stringPtrLifecycle("missing"),
	}
	if err := preflightProjectWorkspace(context.Background(), &input, nil, false); err == nil {
		t.Fatal("missing base directory unexpectedly passed preflight")
	}
}

func TestPreflightProjectWorkspaceRequiresSelectedRuntime(t *testing.T) {
	repo := initLifecycleGitRepo(t, map[string]string{
		"README.md": "demo\n",
	})
	input := CreateInput{
		RepoURL:    repo,
		Branch:     "main",
		DeployMode: "dockerfile",
		AppPort:    80,
	}
	if err := preflightProjectWorkspace(context.Background(), &input, nil, false); err == nil || !strings.Contains(err.Error(), "Dockerfile") {
		t.Fatalf("dockerfile preflight error = %v, want Dockerfile validation", err)
	}
}

func TestPreflightProjectWorkspaceWritesRootEnvForComposeEnvFile(t *testing.T) {
	repo := initLifecycleGitRepo(t, map[string]string{
		"docker-compose.yml": `
services:
  web:
    image: nginx:alpine
    env_file:
      - .env
    environment:
      APP_ENV: ${APP_ENV}
    expose:
      - "8080"
`,
	})
	mainService := "web"
	input := CreateInput{
		RepoURL:         repo,
		Branch:          "main",
		DeployMode:      "compose",
		MainService:     &mainService,
		AppPort:         8080,
		ResourceProfile: "compose-main",
	}

	err := preflightProjectWorkspace(context.Background(), &input, []envvar.Value{
		{Key: "APP_ENV", Value: "staging"},
	}, false)
	if err != nil {
		t.Fatalf("compose preflight with env_file .env failed: %v", err)
	}
}

func TestDetectModeValidatedScopesInspectTreeToBaseDirectory(t *testing.T) {
	repo := initLifecycleGitRepo(t, map[string]string{
		"README.md":           "root\n",
		"apps/api/Dockerfile": "FROM scratch\n",
		"apps/api/main.go":    "package main\n",
		"apps/web/index.html": "<html></html>\n",
	})
	service := &Service{}
	result, err := service.DetectModeValidated(context.Background(), DetectInput{
		RepoURL:       repo,
		Branch:        "main",
		InspectOnly:   true,
		BaseDirectory: "apps/api",
	})
	if err != nil {
		t.Fatalf("DetectModeValidated() error = %v", err)
	}
	if len(result.Tree) == 0 {
		t.Fatal("scoped tree is empty")
	}
	for _, entry := range result.Tree {
		if strings.HasPrefix(entry.Path, "apps/") || strings.Contains(entry.Path, "web") || entry.Path == "README.md" {
			t.Fatalf("tree entry escaped base directory: %+v", entry)
		}
	}
	if result.Tree[0].Depth < 0 {
		t.Fatalf("invalid tree depth: %+v", result.Tree[0])
	}
}

func initLifecycleGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for repository lifecycle tests")
	}
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runLifecycleGit(t, dir, "init", "-b", "main")
	runLifecycleGit(t, dir, "config", "user.email", "test@example.com")
	runLifecycleGit(t, dir, "config", "user.name", "MyPaas Test")
	runLifecycleGit(t, dir, "add", ".")
	runLifecycleGit(t, dir, "commit", "-m", "fixture")
	return dir
}

func runLifecycleGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

func stringPtrLifecycle(value string) *string {
	return &value
}
