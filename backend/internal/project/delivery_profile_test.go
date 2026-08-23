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
	if !warningsContain(got.Warnings, "explicitly published") {
		t.Fatalf("warnings must not pretend image assets are already published: %#v", got.Warnings)
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
	if !warningsContain(got.Warnings, "/assets/*") {
		t.Fatalf("warnings do not mention Vite immutable asset path: %#v", got.Warnings)
	}
}

func TestDetectDeliveryProfileStaticAstro(t *testing.T) {
	dir := writePackageJSON(t, `{
		"dependencies": {"astro": "5.0.0"},
		"scripts": {"build": "astro build"}
	}`)

	got := detectDeliveryProfile(dir, "static")
	if got.Framework != frameworkAstro {
		t.Fatalf("framework = %q, want %q", got.Framework, frameworkAstro)
	}
	if got.Profile != deliveryStaticSite {
		t.Fatalf("profile = %q, want %q", got.Profile, deliveryStaticSite)
	}
	if !warningsContain(got.Warnings, "/_astro/*") {
		t.Fatalf("warnings do not mention Astro immutable asset path: %#v", got.Warnings)
	}
}

func TestDetectDeliveryProfileNuxtRuntime(t *testing.T) {
	dir := writePackageJSON(t, `{
		"dependencies": {"nuxt": "4.0.0"},
		"scripts": {"build": "nuxt build"}
	}`)

	got := detectDeliveryProfile(dir, "dockerfile")
	if got.Framework != frameworkNuxt {
		t.Fatalf("framework = %q, want %q", got.Framework, frameworkNuxt)
	}
	if !warningsContain(got.Warnings, "/_nuxt/*") {
		t.Fatalf("warnings do not mention Nuxt static namespace: %#v", got.Warnings)
	}
	if !warningsContain(got.Warnings, "explicit published artifact") {
		t.Fatalf("warnings must preserve the image/publication boundary: %#v", got.Warnings)
	}
}

func TestDetectDeliveryProfileSvelteKitRuntime(t *testing.T) {
	dir := writePackageJSON(t, `{
		"dependencies": {"@sveltejs/kit": "2.0.0"},
		"scripts": {"build": "vite build"}
	}`)

	got := detectDeliveryProfile(dir, "dockerfile")
	if got.Framework != frameworkSvelteKit {
		t.Fatalf("framework = %q, want %q", got.Framework, frameworkSvelteKit)
	}
	if !warningsContain(got.Warnings, "/_app/immutable/*") {
		t.Fatalf("warnings do not mention SvelteKit immutable namespace: %#v", got.Warnings)
	}
	if !warningsContain(got.Warnings, "Brotli/gzip") {
		t.Fatalf("warnings do not mention preserved precompression: %#v", got.Warnings)
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
	if !warningsContain(got.Warnings, "Caddy compresses") {
		t.Fatalf("warnings do not reflect proxy compression: %#v", got.Warnings)
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
