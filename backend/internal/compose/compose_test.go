package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUserPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "empty allowed", path: "", wantErr: false},
		{name: "root relative", path: "docker-compose.yml", wantErr: false},
		{name: "subdir relative", path: "infra/docker-compose.yml", wantErr: false},
		{name: "nested", path: "apps/api/docker/compose.yaml", wantErr: false},
		{name: "absolute rejected", path: "/etc/passwd", wantErr: true},
		{name: "backslash rejected", path: "infra\\docker-compose.yml", wantErr: true},
		{name: "parent traversal rejected", path: "../escape.yml", wantErr: true},
		{name: "mid traversal rejected", path: "infra/../escape.yml", wantErr: true},
		{name: "deep traversal rejected", path: "a/b/../../c.yml", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserPath(tt.path)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateUserPath(%q) = nil, want error", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateUserPath(%q) = %v, want nil", tt.path, err)
			}
		})
	}
}

func TestDiscoverRootOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "README.md"), "ok")

	candidates, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("Discover() = %d candidates, want 1", len(candidates))
	}
	if candidates[0].Path != "docker-compose.yml" {
		t.Fatalf("Discover()[0].Path = %q, want docker-compose.yml", candidates[0].Path)
	}
}

func TestDiscoverPrefersProdVariantAtRoot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.test.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "docker-compose.prod.yml"), "services: {}\n")

	candidates, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if candidates[0].Path != "docker-compose.prod.yml" {
		t.Fatalf("Discover()[0].Path = %q, want docker-compose.prod.yml", candidates[0].Path)
	}
}

func TestDiscoverFindsSubdirectoryComposeFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "infra"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "infra", "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "package.json"), "{}")

	candidates, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("Discover() returned no candidates for subdir compose file")
	}
	if candidates[0].Path != "infra/docker-compose.yml" {
		t.Fatalf("Discover()[0].Path = %q, want infra/docker-compose.yml", candidates[0].Path)
	}
}

func TestDiscoverSkipsHeavyDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "node_modules", "pkg", "docker-compose.yml"), "services: {}\n")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "src", "compose.yml"), "services: {}\n")

	candidates, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	for _, c := range candidates {
		if strings.HasPrefix(c.Path, "node_modules/") {
			t.Fatalf("Discover() should skip node_modules, got %q", c.Path)
		}
	}
	if len(candidates) == 0 {
		t.Fatal("Discover() should find src/compose.yml")
	}
}

func TestDiscoverIgnoresOverrideAndTestFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.override.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "docker-compose.test.yml"), "services: {}\n")

	if _, err := Discover(dir); err == nil {
		t.Fatal("Discover() should return ErrComposeFileNotFound when only override/test files exist")
	}
}

func TestDiscoverRootWinsOverSubdirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "infra"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "infra", "docker-compose.yml"), "services: {}\n")

	candidates, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if candidates[0].Path != "docker-compose.yml" {
		t.Fatalf("Discover()[0].Path = %q, want root docker-compose.yml", candidates[0].Path)
	}
}

func TestDiscoverReturnsErrWhenNoComposeFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "ok")
	if _, err := Discover(dir); err == nil {
		t.Fatal("Discover() should return error when no compose file exists")
	}
}

func TestResolveLayoutUsesPrimaryFromSubdir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "infra"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "infra", "docker-compose.yml"), "services: {}\n")
	envPath := filepath.Join(dir, ".env")
	writeFile(t, envPath, "FOO=bar\n")

	layout, err := ResolveLayout(dir, "infra/docker-compose.yml", nil, "", "docker-compose.mypaas.override.yml", "docker-compose.mypaas.sanitized.json", envPath)
	if err != nil {
		t.Fatalf("ResolveLayout() error = %v", err)
	}
	if layout.WorkDir != filepath.Join(dir, "infra") {
		t.Fatalf("WorkDir = %q, want %q", layout.WorkDir, filepath.Join(dir, "infra"))
	}
	if len(layout.UserFiles) != 1 || layout.UserFiles[0] != filepath.Join(dir, "infra", "docker-compose.yml") {
		t.Fatalf("UserFiles = %#v, want [%q]", layout.UserFiles, filepath.Join(dir, "infra", "docker-compose.yml"))
	}
	if layout.OverrideFile != filepath.Join(layout.WorkDir, "docker-compose.mypaas.override.yml") {
		t.Fatalf("OverrideFile = %q, want it inside WorkDir", layout.OverrideFile)
	}
	if layout.SanitizedFile != filepath.Join(layout.WorkDir, "docker-compose.mypaas.sanitized.json") {
		t.Fatalf("SanitizedFile = %q, want it inside WorkDir", layout.SanitizedFile)
	}
	if layout.EnvFile != envPath {
		t.Fatalf("EnvFile = %q, want %q", layout.EnvFile, envPath)
	}
}

func TestResolveLayoutAcceptsExplicitWorkdir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "infra"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "deploy"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "infra", "docker-compose.yml"), "services: {}\n")

	layout, err := ResolveLayout(dir, "infra/docker-compose.yml", nil, "deploy", "override.yml", "sanitized.json", "")
	if err != nil {
		t.Fatalf("ResolveLayout() error = %v", err)
	}
	if layout.WorkDir != filepath.Join(dir, "deploy") {
		t.Fatalf("WorkDir = %q, want %q", layout.WorkDir, filepath.Join(dir, "deploy"))
	}
}

func TestResolveLayoutAppendsUserOverridesBeforeMyPaas(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "infra"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "infra", "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "infra", "docker-compose.prod.yml"), "services: {}\n")

	layout, err := ResolveLayout(dir, "infra/docker-compose.yml", []string{"infra/docker-compose.prod.yml"}, "", "override.yml", "sanitized.json", "")
	if err != nil {
		t.Fatalf("ResolveLayout() error = %v", err)
	}
	if len(layout.UserFiles) != 2 {
		t.Fatalf("UserFiles len = %d, want 2", len(layout.UserFiles))
	}
	if !strings.HasSuffix(layout.UserFiles[0], "docker-compose.yml") {
		t.Fatalf("UserFiles[0] = %q, want primary first", layout.UserFiles[0])
	}
	if !strings.HasSuffix(layout.UserFiles[1], "docker-compose.prod.yml") {
		t.Fatalf("UserFiles[1] = %q, want override second", layout.UserFiles[1])
	}
}

func TestResolveLayoutFallsBackToDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "compose.yml"), "services: {}\n")

	layout, err := ResolveLayout(dir, "", nil, "", "override.yml", "sanitized.json", "")
	if err != nil {
		t.Fatalf("ResolveLayout() error = %v", err)
	}
	if layout.PrimaryRel != "compose.yml" {
		t.Fatalf("PrimaryRel = %q, want compose.yml", layout.PrimaryRel)
	}
}

func TestResolveLayoutRejectsTraversalPrimary(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolveLayout(dir, "../escape.yml", nil, "", "override.yml", "sanitized.json", ""); err == nil {
		t.Fatal("ResolveLayout() should reject traversal primary path")
	}
}

func TestResolveLayoutRejectsMissingOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
	if _, err := ResolveLayout(dir, "docker-compose.yml", []string{"missing.yml"}, "", "override.yml", "sanitized.json", ""); err == nil {
		t.Fatal("ResolveLayout() should reject missing override path")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
