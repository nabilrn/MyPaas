package project

import (
	"os"
	"path/filepath"
	"testing"
)

func writeStaticFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestIsStaticFrontendAstroDefaultBuild(t *testing.T) {
	dir := t.TempDir()
	writeStaticFixture(t, dir, "package.json", `{"scripts":{"build":"astro build"},"dependencies":{"astro":"^5.0.0"}}`)
	writeStaticFixture(t, dir, "astro.config.mjs", `import { defineConfig } from 'astro/config'; export default defineConfig({ base: '/' });`)
	if !isStaticFrontend(dir) {
		t.Fatal("expected default Astro build to be classified as static")
	}
}

func TestIsStaticFrontendRejectsAstroServerOutput(t *testing.T) {
	dir := t.TempDir()
	writeStaticFixture(t, dir, "package.json", `{"scripts":{"build":"astro build"},"dependencies":{"astro":"^5.0.0","@astrojs/node":"^9.0.0"}}`)
	writeStaticFixture(t, dir, "astro.config.mjs", `export default { output: 'server', adapter: node({ mode: 'standalone' }) };`)
	if isStaticFrontend(dir) {
		t.Fatal("expected Astro server output to require a runtime")
	}
}

func TestIsStaticFrontendRejectsBackendFrameworks(t *testing.T) {
	dir := t.TempDir()
	writeStaticFixture(t, dir, "package.json", `{"scripts":{"build":"next build"},"dependencies":{"next":"^15.0.0","react":"^19.0.0"}}`)
	if isStaticFrontend(dir) {
		t.Fatal("expected Next.js to remain runtime-backed")
	}
}

func TestIsStaticFrontendRecognizesViteBuild(t *testing.T) {
	dir := t.TempDir()
	writeStaticFixture(t, dir, "package.json", `{"scripts":{"build":"vite build"},"devDependencies":{"vite":"^6.0.0"}}`)
	if !isStaticFrontend(dir) {
		t.Fatal("expected Vite build to be classified as static")
	}
}
