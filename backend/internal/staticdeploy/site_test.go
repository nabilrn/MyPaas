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
		t.Fatalf("root = %q", root)
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
