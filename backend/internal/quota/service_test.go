package quota

import (
	"encoding/json"
	"errors"
	"testing"

	"mypaas/internal/errs"
)

func TestCheckUsage(t *testing.T) {
	tests := []struct {
		name          string
		usage         Usage
		addedMemoryMb int32
		addedCPU      float64
		addedProjects int32
		wantErr       bool
	}{
		{
			name: "within quota",
			usage: Usage{
				MemoryLimitMb: 6144,
				MemoryUsedMb:  512,
				CPULimit:      3,
				CPUUsed:       0.5,
				ProjectLimit:  20,
				ProjectCount:  1,
			},
			addedMemoryMb: 512,
			addedCPU:      0.5,
			addedProjects: 1,
		},
		{
			name: "exceeds memory",
			usage: Usage{
				MemoryLimitMb: 1024,
				MemoryUsedMb:  768,
				CPULimit:      3,
				ProjectLimit:  20,
			},
			addedMemoryMb: 512,
			wantErr:       true,
		},
		{
			name: "exceeds cpu",
			usage: Usage{
				MemoryLimitMb: 6144,
				CPULimit:      1,
				CPUUsed:       0.75,
				ProjectLimit:  20,
			},
			addedCPU: 0.5,
			wantErr:  true,
		},
		{
			name: "allows cpu floating point boundary noise",
			usage: Usage{
				MemoryLimitMb: 6144,
				CPULimit:      3,
				CPUUsed:       2.0000000000000004,
				ProjectLimit:  20,
			},
			addedCPU: 1,
		},
		{
			name: "exceeds project count",
			usage: Usage{
				MemoryLimitMb: 6144,
				CPULimit:      3,
				ProjectLimit:  2,
				ProjectCount:  2,
			},
			addedProjects: 1,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkUsage(tt.usage, tt.addedMemoryMb, tt.addedCPU, tt.addedProjects)
			if tt.wantErr {
				if !errors.Is(err, errs.ErrQuotaExceeded) {
					t.Fatalf("expected ErrQuotaExceeded, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestDeclaredResources(t *testing.T) {
	raw := json.RawMessage(`{
		"app": {"memoryLimitMb": 9999, "cpuLimit": 9},
		"worker": {"memoryLimitMb": 768, "cpuLimit": 0.75},
		"db": {"memoryLimitMb": 768, "cpuLimit": 0.5},
		"redis": {"memoryLimitMb": 256, "cpuLimit": 0.25}
	}`)
	memory, cpu, err := DeclaredResources(1536, 1.25, "app", raw)
	if err != nil {
		t.Fatalf("DeclaredResources returned error: %v", err)
	}
	if memory != 3328 {
		t.Fatalf("memory = %d, want 3328", memory)
	}
	if cpu != 2.75 {
		t.Fatalf("cpu = %.2f, want 2.75", cpu)
	}
}

func TestDeclaredResourcesUsesDeploymentDefaultsForSecondaryServices(t *testing.T) {
	memory, cpu, err := DeclaredResources(512, 0.5, "app", json.RawMessage(`{"worker": {}}`))
	if err != nil {
		t.Fatalf("DeclaredResources returned error: %v", err)
	}
	if memory != 768 {
		t.Fatalf("memory = %d, want 768", memory)
	}
	if cpu != 0.75 {
		t.Fatalf("cpu = %.2f, want 0.75", cpu)
	}
}

func TestDeclaredResourcesRejectsMalformedJSON(t *testing.T) {
	if _, _, err := DeclaredResources(512, 0.5, "app", json.RawMessage(`{"worker":`)); err == nil {
		t.Fatal("expected malformed service resources to fail")
	}
}
