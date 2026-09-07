package resourceprofile

import (
	"fmt"
	"strings"
	"sync"

	"mypaas/internal/errs"
)

const (
	Static      = "static"
	GoSmall     = "go-small"
	NodePython  = "node-python"
	ComposeMain = "compose-main"
	Custom      = "custom"
)

type Profile struct {
	ID       string
	Label    string
	MemoryMB int32
	// CPULimit is retained for settings/API compatibility with older clients.
	// Runtime CPU is shared across the host and Resolve always returns 0,
	// which is the Docker/Podman no-limit value.
	CPULimit float64
}

var minimumProfiles = map[string]Profile{
	Static: {
		ID:       Static,
		Label:    "Static/no-runtime",
		MemoryMB: 64,
		CPULimit: 0.01,
	},
	GoSmall: {
		ID:       GoSmall,
		Label:    "Go small",
		MemoryMB: 128,
		CPULimit: 0.20,
	},
	NodePython: {
		ID:       NodePython,
		Label:    "Node/Python",
		MemoryMB: 256,
		CPULimit: 0.35,
	},
	ComposeMain: {
		ID:       ComposeMain,
		Label:    "Compose main",
		MemoryMB: 256,
		CPULimit: 0.35,
	},
	Custom: {
		ID:       Custom,
		Label:    "Custom",
		MemoryMB: 512,
		CPULimit: 0.50,
	},
}

var (
	profilesMu sync.RWMutex
	profiles   = cloneProfiles(minimumProfiles)
)

func Resolve(id, deployMode string, memoryMB int32, cpuLimit float64) (string, int32, float64, error) {
	profile, err := Get(defaultID(id, deployMode))
	if err != nil {
		return "", 0, 0, err
	}
	if memoryMB <= 0 {
		memoryMB = profile.MemoryMB
	}
	if memoryMB <= 0 {
		return "", 0, 0, fmt.Errorf("%w: memory limit must be greater than 0", errs.ErrValidation)
	}

	// CPU is intentionally not reserved per project. Docker and Podman both
	// define --cpus=0 as unlimited, so 0 is the persisted/runtime contract for
	// shared CPU. Keep accepting the legacy argument so older API clients remain
	// source-compatible while the platform stops enforcing it.
	_ = cpuLimit
	return profile.ID, memoryMB, 0, nil
}

func Get(id string) (Profile, error) {
	id = strings.TrimSpace(id)
	profilesMu.RLock()
	defer profilesMu.RUnlock()
	profile, ok := profiles[id]
	if !ok {
		return Profile{}, fmt.Errorf("%w: unknown resource profile %q", errs.ErrValidation, id)
	}
	return profile, nil
}

// ConfigureDefaults changes the platform memory defaults used when a project
// does not provide an explicit memory limit. CPU values are retained only for
// compatibility with existing settings payloads and are not runtime limits.
func ConfigureDefaults(configured map[string]Profile) error {
	for id, profile := range configured {
		minimum, ok := minimumProfiles[id]
		if !ok || id == Custom {
			return fmt.Errorf("%w: resource profile %q is not configurable", errs.ErrValidation, id)
		}
		if profile.MemoryMB < minimum.MemoryMB || profile.MemoryMB > 32768 {
			return fmt.Errorf("%w: %s memory must be between %d and 32768 MB", errs.ErrValidation, minimum.Label, minimum.MemoryMB)
		}
		if profile.CPULimit < minimum.CPULimit || profile.CPULimit > 32 {
			return fmt.Errorf("%w: %s legacy CPU value must be between %.2f and 32", errs.ErrValidation, minimum.Label, minimum.CPULimit)
		}
	}

	profilesMu.Lock()
	defer profilesMu.Unlock()
	for id, configuredProfile := range configured {
		profile := profiles[id]
		profile.MemoryMB = configuredProfile.MemoryMB
		profile.CPULimit = configuredProfile.CPULimit
		profiles[id] = profile
	}
	return nil
}

func DefaultForDeployMode(deployMode string) string {
	switch strings.TrimSpace(deployMode) {
	case "compose":
		return ComposeMain
	case "static":
		return Static
	default:
		return NodePython
	}
}

func ComposeSideLimits() (int32, float64) {
	return 256, 0
}

func defaultID(id, deployMode string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return DefaultForDeployMode(deployMode)
}

func cloneProfiles(source map[string]Profile) map[string]Profile {
	cloned := make(map[string]Profile, len(source))
	for id, profile := range source {
		cloned[id] = profile
	}
	return cloned
}
