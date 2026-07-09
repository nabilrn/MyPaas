package project

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mypaas/internal/envdiscover"
)

func TestInferDockerfileAppPortFromExpose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine\nEXPOSE 8080/tcp\n")

	got := inferDockerfileAppPort(dir, nil)
	if got != 8080 {
		t.Fatalf("inferDockerfileAppPort() = %d, want 8080", got)
	}
}

func TestInferDockerfileAppPortFromEnvReference(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM node:20\nENV PORT=4173\nEXPOSE $PORT\n")

	got := inferDockerfileAppPort(dir, nil)
	if got != 4173 {
		t.Fatalf("inferDockerfileAppPort() = %d, want 4173", got)
	}
}

func TestInferDockerfileAppPortFromDiscoveredEnv(t *testing.T) {
	dir := t.TempDir()
	value := "9000"
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine\n")

	got := inferDockerfileAppPort(dir, []envdiscover.Var{{Key: "PORT", DefaultValue: &value}})
	if got != 9000 {
		t.Fatalf("inferDockerfileAppPort() = %d, want 9000", got)
	}
}

func TestParseComposePortsUsesContainerTarget(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{{
		"published": "3000",
		"target":    8080,
		"protocol":  "tcp",
	}})
	if err != nil {
		t.Fatal(err)
	}

	got := parseComposePorts(raw, nil)
	if got != 8080 {
		t.Fatalf("parseComposePorts() = %d, want 8080", got)
	}
}

func TestParseComposeExposeUsesFirstPort(t *testing.T) {
	raw, err := json.Marshal([]string{"7000", "9000"})
	if err != nil {
		t.Fatal(err)
	}

	got := parseComposeExpose(raw, nil)
	if got != 7000 {
		t.Fatalf("parseComposeExpose() = %d, want 7000", got)
	}
}

func TestParseDefaultBranchRef(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "main",
			input: "ref: refs/heads/main\tHEAD\n" +
				"8f8c2 HEAD\n",
			want: "main",
		},
		{
			name: "master",
			input: "ref: refs/heads/master HEAD\n" +
				"8f8c2 HEAD\n",
			want: "master",
		},
		{
			name: "branch with slash",
			input: "ref: refs/heads/release/v1 HEAD\n" +
				"8f8c2 HEAD\n",
			want: "release/v1",
		},
		{
			name:  "missing symref",
			input: "8f8c2 HEAD\n",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDefaultBranchRef(tt.input); got != tt.want {
				t.Fatalf("parseDefaultBranchRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRemoteBranchRefs(t *testing.T) {
	input := "8f8c2 refs/heads/main\n" +
		"9a9c1 refs/heads/fina\n" +
		"aaaa1 refs/heads/release/v1\n" +
		"9a9c1 refs/heads/fina\n" +
		"bbbb2 refs/tags/v1.0.0\n"

	got := parseRemoteBranchRefs(input)
	want := []string{"main", "fina", "release/v1"}
	if len(got) != len(want) {
		t.Fatalf("parseRemoteBranchRefs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseRemoteBranchRefs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPrioritizeDefaultBranch(t *testing.T) {
	got := prioritizeDefaultBranch("main", []string{"feature/api", "main", "dev"})
	want := []string{"main", "dev", "feature/api"}
	if len(got) != len(want) {
		t.Fatalf("prioritizeDefaultBranch() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("prioritizeDefaultBranch()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestListRepositoryTreeSkipsHeavyDirsAndTruncates(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "README.md"), "ok")
	writeFile(t, filepath.Join(dir, "src", "main.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "node_modules", "pkg", "index.js"), "module")

	tree, truncated, err := listRepositoryTree(dir, 2)
	if err != nil {
		t.Fatalf("listRepositoryTree() returned error: %v", err)
	}
	if !truncated {
		t.Fatalf("listRepositoryTree() truncated = false, want true")
	}
	for _, item := range tree {
		if item.Path == "node_modules" || strings.HasPrefix(item.Path, "node_modules/") {
			t.Fatalf("listRepositoryTree() included skipped path %q", item.Path)
		}
	}
}

func TestParseGitTreeEntries(t *testing.T) {
	input := "040000 tree abc123\tcmd\n" +
		"100644 blob def456\tcmd/api/main.go\n" +
		"100644 blob 111111\tREADME.md\n" +
		"120000 blob 222222\tlink\n"

	got, truncated := parseGitTreeEntries(input, 3)
	if !truncated {
		t.Fatalf("parseGitTreeEntries() truncated = false, want true")
	}
	want := []RepoTreeEntry{
		{Name: "cmd", Path: "cmd", Type: "directory", Depth: 0},
		{Name: "main.go", Path: "cmd/api/main.go", Type: "file", Depth: 2},
		{Name: "README.md", Path: "README.md", Type: "file", Depth: 0},
	}
	if len(got) != len(want) {
		t.Fatalf("parseGitTreeEntries() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseGitTreeEntries()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestDetectComposeFilePrefersProdVariant(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.test.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "docker-compose.prod.yml"), "services: {}\n")

	got := detectComposeFile(dir)
	if got != "docker-compose.prod.yml" {
		t.Fatalf("detectComposeFile() = %q, want docker-compose.prod.yml", got)
	}
}

func TestPickMainServicePrefersFrontendOverDatabase(t *testing.T) {
	got := pickMainService(context.Background(), t.TempDir(), "docker-compose.prod.yml", []string{"db", "backend", "frontend"})
	if got != "frontend" {
		t.Fatalf("pickMainService() = %q, want frontend", got)
	}
}

func TestPickMainServiceFromComposeConfigPrefersPortsOverExpose(t *testing.T) {
	raw := []byte(`{
		"services": {
			"db": {},
			"backend": {"expose": ["3000"]},
			"frontend": {"ports": [{"target": 80, "published": "80"}]}
		}
	}`)

	got := pickMainServiceFromComposeConfig(raw, []string{"db", "backend", "frontend"})
	if got != "frontend" {
		t.Fatalf("pickMainServiceFromComposeConfig() = %q, want frontend", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
