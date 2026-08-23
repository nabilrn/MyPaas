package caddy

import (
	"strings"
	"testing"
)

func TestEvaluateRuntimeReadiness(t *testing.T) {
	tests := []struct {
		name    string
		state   runtimeContainerState
		ready   bool
		wantErr string
	}{
		{
			name:  "running without healthcheck",
			state: runtimeContainerState{Status: "running", Running: true},
			ready: true,
		},
		{
			name: "healthy",
			state: runtimeContainerState{
				Status:  "running",
				Running: true,
				Health:  &runtimeHealthState{Status: "healthy"},
			},
			ready: true,
		},
		{
			name: "healthcheck starting",
			state: runtimeContainerState{
				Status:  "running",
				Running: true,
				Health:  &runtimeHealthState{Status: "starting"},
			},
		},
		{
			name: "unhealthy",
			state: runtimeContainerState{
				Status:  "running",
				Running: true,
				Health:  &runtimeHealthState{Status: "unhealthy"},
			},
			wantErr: "health status=unhealthy",
		},
		{
			name:    "exited",
			state:   runtimeContainerState{Status: "exited", Running: false},
			wantErr: "status=exited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready, err := evaluateRuntimeReadiness(tt.state)
			if ready != tt.ready {
				t.Fatalf("ready = %v, want %v", ready, tt.ready)
			}
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestRuntimeTargetFromInspectResolvesProjectRuntime(t *testing.T) {
	alias := runtimeRouteAlias(3456)
	raw := []byte(`[
  {
    "Id": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"Aliases": ["runtime"]}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", alias, 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if got.ContainerID != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("container ID = %q", got.ContainerID)
	}
	if got.ContainerPort != "8080" {
		t.Fatalf("container port = %q, want 8080", got.ContainerPort)
	}
	if got.RoutingAttached {
		t.Fatal("new runtime should not already be attached to routing network")
	}
	if got.RoutingAliasPresent {
		t.Fatal("new runtime should not already have the routing alias")
	}
	if alias != "mypaas-port-3456" {
		t.Fatalf("alias = %q", alias)
	}
}

func TestRuntimeTargetFromInspectDetectsExistingRoutingAlias(t *testing.T) {
	alias := runtimeRouteAlias(3456)
	raw := []byte(`[
  {
    "Id": "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"Aliases": ["runtime"]},
        "mypaas-routing": {"Aliases": ["runtime", "mypaas-port-3456"]}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", alias, 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !got.RoutingAttached || !got.RoutingAliasPresent {
		t.Fatalf("expected existing routing alias, got %+v", got)
	}
}

func TestRuntimeTargetFromInspectDetectsRoutingAttachmentWithoutAlias(t *testing.T) {
	alias := runtimeRouteAlias(3456)
	raw := []byte(`[
  {
    "Id": "bbbbbbbbbbbb1111111111111111111111111111111111111111111111111111",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"Aliases": ["runtime"]},
        "mypaas-routing": {"Aliases": ["runtime"]}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", alias, 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !got.RoutingAttached || got.RoutingAliasPresent {
		t.Fatalf("expected routing attachment without managed alias, got %+v", got)
	}
}

func TestRuntimeTargetFromInspectSelectsReplacementByAllocatedHostPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["old"]}}
    }
  },
  {
    "Id": "bbbbbbbbbbbb1111111111111111111111111111111111111111111111111111",
    "NetworkSettings": {
      "Ports": {"3000/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["replacement"]}}
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !strings.HasPrefix(got.ContainerID, "bbbbbbbbbbbb") || got.ContainerPort != "3000" {
		t.Fatalf("unexpected replacement target: %+v", got)
	}
}

func TestRuntimeTargetFromInspectSkipsSamePortOutsideProjectNetwork(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "cccccccccccc2222222222222222222222222222222222222222222222222222",
    "NetworkSettings": {
      "Ports": {"9000/tcp": [{"HostPort": "3456"}]},
      "Networks": {"other-network": {"Aliases": ["other"]}}
    }
  },
  {
    "Id": "dddddddddddd3333333333333333333333333333333333333333333333333333",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["runtime"]}}
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !strings.HasPrefix(got.ContainerID, "dddddddddddd") {
		t.Fatalf("unexpected project runtime: %+v", got)
	}
}

func TestRuntimeTargetFromInspectRequiresProjectNetworkAttachment(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "eeeeeeeeeeee4444444444444444444444444444444444444444444444444444",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"some-other-network": {"Aliases": ["runtime"]}}
    }
  }
]`)

	_, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err == nil {
		t.Fatal("expected missing project network to fail")
	}
	if !strings.Contains(err.Error(), "not attached to project network") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeTargetFromInspectRejectsUnknownPublishedPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "ffffffffffff5555555555555555555555555555555555555555555555555555",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"Aliases": ["runtime"]}}
    }
  }
]`)

	_, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", runtimeRouteAlias(3456), 3456)
	if err == nil {
		t.Fatal("expected unknown host port to fail")
	}
	if !strings.Contains(err.Error(), "no running container owns") {
		t.Fatalf("unexpected error: %v", err)
	}
}
