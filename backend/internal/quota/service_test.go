package quota

import (
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
