package resourceprofile

import (
	"fmt"
	"strings"

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
	CPULimit float64
}

var profiles = map[string]Profile{
	Static: {
		ID:       Static,
		Label:    "Static/no-runtime",
		MemoryMB: 64,
		CPULimit: 0.10,
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

func Resolve(id, deployMode string, memoryMB int32, cpuLimit float64) (string, int32, float64, error) {
	profile, err := Get(defaultID(id, deployMode))
	if err != nil {
		return "", 0, 0, err
	}
	if memoryMB <= 0 {
		memoryMB = profile.MemoryMB
	}
	if cpuLimit <= 0 {
		cpuLimit = profile.CPULimit
	}
	if memoryMB <= 0 {
		return "", 0, 0, fmt.Errorf("%w: memory limit must be greater than 0", errs.ErrValidation)
	}
	if cpuLimit <= 0 {
		return "", 0, 0, fmt.Errorf("%w: CPU limit must be greater than 0", errs.ErrValidation)
	}
	return profile.ID, memoryMB, cpuLimit, nil
}

func Get(id string) (Profile, error) {
	id = strings.TrimSpace(id)
	profile, ok := profiles[id]
	if !ok {
		return Profile{}, fmt.Errorf("%w: unknown resource profile %q", errs.ErrValidation, id)
	}
	return profile, nil
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
	return 256, 0.25
}

func defaultID(id, deployMode string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return DefaultForDeployMode(deployMode)
}
