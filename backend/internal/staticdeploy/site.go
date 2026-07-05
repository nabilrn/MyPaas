package staticdeploy

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"mypaas/internal/errs"
)

var candidateDirs = []string{"dist", "build", "public", "."}

func FindSiteRoot(workspace string) (string, string, error) {
	workspace = filepath.Clean(workspace)
	for _, rel := range candidateDirs {
		root := filepath.Join(workspace, rel)
		if hasIndex(root) {
			return root, rel, nil
		}
	}
	return "", "", errs.ErrNoDeployConfig
}

func CopyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("clear static target: %w", err)
	}
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if shouldSkip(path, entry) {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0750)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		return copyFile(path, target)
	})
}

func hasIndex(root string) bool {
	info, err := os.Stat(filepath.Join(root, "index.html"))
	return err == nil && !info.IsDir()
}

func shouldSkip(path string, entry fs.DirEntry) bool {
	if !entry.IsDir() {
		return false
	}
	name := strings.ToLower(entry.Name())
	return name == ".git" || name == "node_modules"
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
