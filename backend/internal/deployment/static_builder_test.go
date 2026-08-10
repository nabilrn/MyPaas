package deployment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticBuildPlanUsesDeclaredPnpmForSubdirectory(t *testing.T) {
	workspace := t.TempDir()
	writeStaticTestFile(t, workspace, "package.json", `{
  "packageManager": "pnpm@11.21.0",
  "scripts": {"build": "astro build"}
}`)
	writeStaticTestFile(t, workspace, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")

	plan, err := staticBuildPlanForWorkspace(workspace)
	if err != nil {
		t.Fatalf("staticBuildPlanForWorkspace() error = %v", err)
	}
	if plan.Image != "node:22-alpine" {
		t.Fatalf("plan.Image = %q, want node:22-alpine", plan.Image)
	}
	if plan.PackageManager != "pnpm" {
		t.Fatalf("plan.PackageManager = %q, want pnpm", plan.PackageManager)
	}
	want := "corepack enable && pnpm install --frozen-lockfile && pnpm run build"
	if plan.Command != want {
		t.Fatalf("plan.Command = %q, want %q", plan.Command, want)
	}
}

func TestStaticBuildPlanInfersPnpmFromLockfile(t *testing.T) {
	workspace := t.TempDir()
	writeStaticTestFile(t, workspace, "package.json", `{"scripts":{"build":"vite build"}}`)
	writeStaticTestFile(t, workspace, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")

	plan, err := staticBuildPlanForWorkspace(workspace)
	if err != nil {
		t.Fatalf("staticBuildPlanForWorkspace() error = %v", err)
	}
	if plan.PackageManager != "pnpm" {
		t.Fatalf("plan.PackageManager = %q, want pnpm", plan.PackageManager)
	}
}

func TestStaticBuildPlanUsesNpmCIWhenLockfileExists(t *testing.T) {
	workspace := t.TempDir()
	writeStaticTestFile(t, workspace, "package.json", `{
  "packageManager": "npm@12.0.2",
  "scripts": {"build": "vite build"}
}`)
	writeStaticTestFile(t, workspace, "package-lock.json", `{}`)

	plan, err := staticBuildPlanForWorkspace(workspace)
	if err != nil {
		t.Fatalf("staticBuildPlanForWorkspace() error = %v", err)
	}
	if plan.PackageManager != "npm" {
		t.Fatalf("plan.PackageManager = %q, want npm", plan.PackageManager)
	}
	if plan.Command != "npm ci && npm run build" {
		t.Fatalf("plan.Command = %q, want npm ci command", plan.Command)
	}
}

func TestStaticBuildPlanUsesYarnClassicFrozenLockfile(t *testing.T) {
	workspace := t.TempDir()
	writeStaticTestFile(t, workspace, "package.json", `{
  "packageManager": "yarn@1.22.22",
  "scripts": {"build": "vite build"}
}`)
	writeStaticTestFile(t, workspace, "yarn.lock", "# yarn lockfile v1\n")

	plan, err := staticBuildPlanForWorkspace(workspace)
	if err != nil {
		t.Fatalf("staticBuildPlanForWorkspace() error = %v", err)
	}
	if !strings.Contains(plan.Command, "yarn install --frozen-lockfile") {
		t.Fatalf("plan.Command = %q, want Yarn classic frozen lockfile install", plan.Command)
	}
}

func TestStaticBuildPlanRejectsUnsupportedPackageManager(t *testing.T) {
	workspace := t.TempDir()
	writeStaticTestFile(t, workspace, "package.json", `{
  "packageManager": "bun@1.2.0",
  "scripts": {"build": "bun run build"}
}`)

	_, err := staticBuildPlanForWorkspace(workspace)
	if err == nil || !strings.Contains(err.Error(), "unsupported static package manager") {
		t.Fatalf("staticBuildPlanForWorkspace() error = %v, want unsupported package manager error", err)
	}
}

func TestNeedsStaticBuildRequiresBuildScript(t *testing.T) {
	workspace := t.TempDir()
	writeStaticTestFile(t, workspace, "package.json", `{"scripts":{"dev":"astro dev"}}`)
	if needsStaticBuild(workspace) {
		t.Fatal("needsStaticBuild() = true without build script")
	}

	writeStaticTestFile(t, workspace, "package.json", `{"scripts":{"build":"astro build"}}`)
	if !needsStaticBuild(workspace) {
		t.Fatal("needsStaticBuild() = false with build script")
	}
}

func writeStaticTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
