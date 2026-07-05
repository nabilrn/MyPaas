package staticdeploy

import (
	"os"
	"path/filepath"
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
