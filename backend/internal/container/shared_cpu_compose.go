package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// stripGeneratedComposeCPUHardCaps removes historical MyPaas-generated `cpus:`
// entries before Compose evaluates the override. Runtime normalization remains
// as a compatibility backstop for older or externally managed containers.
func stripGeneratedComposeCPUHardCaps(opts ComposeUpOptions) error {
	path := strings.TrimSpace(opts.OverrideFile)
	if path == "" {
		return nil
	}
	if !filepath.IsAbs(path) && strings.TrimSpace(opts.WorkDir) != "" {
		path = filepath.Join(opts.WorkDir, path)
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read compose override for shared CPU: %w", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "cpus:") {
			changed = true
			continue
		}
		filtered = append(filtered, line)
	}
	if !changed {
		return nil
	}

	return os.WriteFile(path, []byte(strings.Join(filtered, "\n")), 0o600)
}

func stripComposeCPUFields(service map[string]any) {
	for _, key := range []string{
		"cpus",
		"cpu_count",
		"cpu_percent",
		"cpu_period",
		"cpu_quota",
		"cpu_rt_period",
		"cpu_rt_runtime",
		"cpuset",
	} {
		delete(service, key)
	}

	deploy, ok := service["deploy"].(map[string]any)
	if !ok {
		return
	}
	resources, ok := deploy["resources"].(map[string]any)
	if !ok {
		return
	}
	for _, bucket := range []string{"limits", "reservations"} {
		values, ok := resources[bucket].(map[string]any)
		if ok {
			delete(values, "cpus")
		}
	}
}
