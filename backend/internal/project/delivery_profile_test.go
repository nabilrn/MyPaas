package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectDeliveryProfileNextDockerfile(t *testing.T) {
	dir := writePackageJSON(t, `{
		"dependencies": {"next": "15.0.0", "react": "19.0.0"},
		"scripts": {"build": "next build"}
	}`)

	got := detectDeliveryProfile(dir, "dockerfile")
	if got.Framework != frameworkNextJS {
		t.Fatalf("framework = %q, want %q", got.Framework, frameworkNextJS)
	}
	if got.Profile != deliveryNextStandalone {
		t.Fatalf("profile = %q, want %q", got.Profile, deliveryNextStandalone)
	}
	if !warningsContain(got.Warnings, "/_next/static/*") {
		t.Fatalf("warnings do not mention Next static cache path: %#v", got.Warnings)
	}
}

func TestDetectDeliveryProfileStaticVite(t *testing.T) {
	dir := writePackageJSON(t, `{
		"devDependencies": {"vite": "5.0.0"},
		"scripts": {"build": "vite build"}
	}`)

	got := detectDeliveryProfile(dir, "static")
	if got.Framework != frameworkVite {
		t.Fatalf("framework = %q, want %q", got.Framework, frameworkVite)
	}
	if got.Profile != deliverySPAStatic {
		t.Fatalf("profile = %q, want %q", got.Profile, deliverySPAStatic)
	}
}

func TestDetectDeliveryProfileNodeAPI(t *testing.T) {
	dir := writePackageJSON(t, `{
		"dependencies": {"express": "4.18.0"},
		"scripts": {"start": "node server.js"}
	}`)

	got := detectDeliveryProfile(dir, "dockerfile")
	if got.Framework != frameworkNodeAPI {
		t.Fatalf("framework = %q, want %q", got.Framework, frameworkNodeAPI)
	}
	if got.Profile != deliveryAPIOnly {
		t.Fatalf("profile = %q, want %q", got.Profile, deliveryAPIOnly)
	}
}

func TestDetectDeliveryProfileCompose(t *testing.T) {
	dir := writePackageJSON(t, `{
		"dependencies": {"next": "15.0.0"},
		"scripts": {"build": "next build"}
	}`)

	got := detectDeliveryProfile(dir, "compose")
	if got.Profile != deliveryCompose {
		t.Fatalf("profile = %q, want %q", got.Profile, deliveryCompose)
	}
}

func writePackageJSON(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return dir
}

func warningsContain(warnings []string, want string) bool {
	for _, item := range warnings {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
