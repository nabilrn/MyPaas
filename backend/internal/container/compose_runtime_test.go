package container

import (
	"errors"
	"strings"
	"testing"
)

func TestComposePullArgsRefreshesImageOnlyServices(t *testing.T) {
	opts := ComposeUpOptions{
		ProjectName:  "mypaas-wago",
		ComposeFiles: []string{"sanitized.json"},
		OverrideFile: "override.yml",
		EnvFile:      "/tmp/wago/.env",
	}

	got := composePullArgs(opts)
	wantParts := []string{
		"compose", "--env-file", "/tmp/wago/.env",
		"-p", "mypaas-wago",
		"-f", "sanitized.json",
		"-f", "override.yml",
		"pull", "--ignore-buildable",
	}
	if strings.Join(got, "|") != strings.Join(wantParts, "|") {
		t.Fatalf("composePullArgs() = %v, want %v", got, wantParts)
	}
}

func TestEvaluateComposeReadiness(t *testing.T) {
	tests := []struct {
		name      string
		state     composeContainerState
		wantReady bool
		wantErr   bool
	}{
		{
			name:      "running without healthcheck",
			state:     composeContainerState{Status: "running", Running: true},
			wantReady: true,
		},
		{
			name: "healthy",
			state: composeContainerState{
				Status:  "running",
				Running: true,
				Health:  &composeHealthState{Status: "healthy"},
			},
			wantReady: true,
		},
		{
			name: "healthcheck starting",
			state: composeContainerState{
				Status:  "running",
				Running: true,
				Health:  &composeHealthState{Status: "starting"},
			},
		},
		{
			name: "unhealthy",
			state: composeContainerState{
				Status:  "running",
				Running: true,
				Health:  &composeHealthState{Status: "unhealthy"},
			},
			wantErr: true,
		},
		{
			name:    "exited",
			state:   composeContainerState{Status: "exited", Running: false},
			wantErr: true,
		},
		{
			name:  "restarting",
			state: composeContainerState{Status: "restarting", Running: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready, err := evaluateComposeReadiness(tt.state)
			if ready != tt.wantReady {
				t.Fatalf("ready = %v, want %v", ready, tt.wantReady)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvaluateComposeReadinessReportsTerminalState(t *testing.T) {
	_, err := evaluateComposeReadiness(composeContainerState{
		Status:  "running",
		Running: true,
		Health:  &composeHealthState{Status: "unhealthy"},
	})
	if err == nil {
		t.Fatal("expected unhealthy state to fail")
	}
	if !errors.Is(err, ErrComposeServiceUnhealthy) {
		t.Fatalf("error = %v, want ErrComposeServiceUnhealthy", err)
	}
}
