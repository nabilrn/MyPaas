package staticdeploy

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindSiteRootPrefersBuildOutput(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "dist"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "dist", "index.html"), []byte("ok"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "index.html"), []byte("root"), 0640); err != nil {
		t.Fatal(err)
	}

	root, rel, err := FindSiteRoot(workspace)
	if err != nil {
		t.Fatalf("FindSiteRoot returned error: %v", err)
	}
	if rel != "dist" {
		t.Fatalf("rel = %q, want dist", rel)
	}
	if root != filepath.Join(workspace, "dist") {
		t.Fatalf("root = %q, want %q", root, filepath.Join(workspace, "dist"))
	}
}

func TestFindSiteRootSupportsCommonStaticOutputDirs(t *testing.T) {
	for _, rel := range []string{"out", ".output/public", "_site", "site", "www"} {
		t.Run(rel, func(t *testing.T) {
			workspace := t.TempDir()
			root := filepath.Join(workspace, rel)
			if err := os.MkdirAll(root, 0750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0640); err != nil {
				t.Fatal(err)
			}

			gotRoot, gotRel, err := FindSiteRoot(workspace)
			if err != nil {
				t.Fatalf("FindSiteRoot returned error: %v", err)
			}
			if gotRel != rel {
				t.Fatalf("rel = %q, want %q", gotRel, rel)
			}
			if gotRoot != root {
				t.Fatalf("root = %q, want %q", gotRoot, root)
			}
		})
	}
}

func TestCopyDirSkipsGitAndNodeModules(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	for _, dir := range []string{"assets", ".git", "node_modules/pkg"} {
		if err := os.MkdirAll(filepath.Join(src, dir), 0750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(src, "index.html"), []byte("ok"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "assets", "app.css"), []byte("css"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".git", "config"), []byte("secret"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "node_modules", "pkg", "index.js"), []byte("module"), 0640); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "index.html")); err != nil {
		t.Fatalf("index.html was not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "assets", "app.css")); err != nil {
		t.Fatalf("asset was not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".git", "config")); !os.IsNotExist(err) {
		t.Fatalf(".git should not be copied")
	}
	if _, err := os.Stat(filepath.Join(dst, "node_modules", "pkg", "index.js")); !os.IsNotExist(err) {
		t.Fatalf("node_modules should not be copied")
	}
	if _, err := os.Stat(filepath.Join(dst, spaFallbackMarker)); !os.IsNotExist(err) {
		t.Fatalf("plain static site must not receive SPA fallback metadata")
	}
}

func TestCopyDirPrecompressesLargeTextAssets(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(filepath.Join(src, "assets"), 0750); err != nil {
		t.Fatal(err)
	}
	js := strings.Repeat("const value = 'benchmark';\n", 200)
	if err := os.WriteFile(filepath.Join(src, "assets", "app.js"), []byte(js), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "assets", "image.png"), bytes.Repeat([]byte{1, 2, 3, 4}, 400), 0640); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}

	gzPath := filepath.Join(dst, "assets", "app.js.gz")
	file, err := os.Open(gzPath)
	if err != nil {
		t.Fatalf("precompressed app.js.gz was not created: %v", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if string(body) != js {
		t.Fatalf("precompressed content mismatch")
	}
	if _, err := os.Stat(filepath.Join(dst, "assets", "image.png.gz")); !os.IsNotExist(err) {
		t.Fatalf("binary image should not be precompressed")
	}
}

func TestCopyDirMarksReactViteSPAForHistoryFallback(t *testing.T) {
	workspace := t.TempDir()
	dist := filepath.Join(workspace, "dist")
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"dependencies":{"react":"19.0.0"},"devDependencies":{"vite":"7.0.0"}}`), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("ok"), 0640); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "release")
	if err := CopyDir(dist, dst); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, spaFallbackMarker)); err != nil {
		t.Fatalf("Vite SPA routing marker missing: %v", err)
	}
}

func TestCopyDirDoesNotMarkAstroStaticForSPAFallback(t *testing.T) {
	workspace := t.TempDir()
	dist := filepath.Join(workspace, "dist")
	if err := os.MkdirAll(filepath.Join(dist, "about"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"dependencies":{"astro":"5.0.0"}}`), 0640); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dist, "index.html"), filepath.Join(dist, "about", "index.html")} {
		if err := os.WriteFile(path, []byte("ok"), 0640); err != nil {
			t.Fatal(err)
		}
	}

	dst := filepath.Join(t.TempDir(), "release")
	if err := CopyDir(dist, dst); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, spaFallbackMarker)); !os.IsNotExist(err) {
		t.Fatalf("Astro static site must keep normal 404 semantics")
	}
}

func TestDetectSPAFallbackRejectsViteMultiPageBuild(t *testing.T) {
	workspace := t.TempDir()
	dist := filepath.Join(workspace, "dist")
	if err := os.MkdirAll(dist, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"dependencies":{"vue":"3.0.0"},"devDependencies":{"vite":"7.0.0"}}`), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "vite.config.js"), []byte(`export default { build: { rollupOptions: { input: { main: 'index.html', nested: 'nested/index.html' } } } }`), 0640); err != nil {
		t.Fatal(err)
	}
	if detectSPAFallback(dist) {
		t.Fatalf("Vite multi-page input must not use SPA history fallback")
	}
}

func TestDetectSPAFallbackDoesNotGuessSvelteKitStatic(t *testing.T) {
	workspace := t.TempDir()
	build := filepath.Join(workspace, "build")
	if err := os.MkdirAll(build, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"dependencies":{"@sveltejs/kit":"2.0.0","@sveltejs/adapter-static":"3.0.0"}}`), 0640); err != nil {
		t.Fatal(err)
	}
	if detectSPAFallback(build) {
		t.Fatalf("SvelteKit static output must not receive an implicit index fallback")
	}
}
