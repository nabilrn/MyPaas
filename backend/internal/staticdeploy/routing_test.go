package staticdeploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirIgnoresSourceOwnedRoutingMetadata(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "release")
	if err := os.WriteFile(filepath.Join(src, "index.html"), []byte("plain static"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, spaFallbackMarker), []byte("forged"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, spaFallbackMarker)); !os.IsNotExist(err) {
		t.Fatalf("source-owned routing metadata must not be published")
	}
}
