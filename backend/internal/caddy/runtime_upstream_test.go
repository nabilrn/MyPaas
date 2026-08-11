package caddy

import (
	"strings"
	"testing"
)

func TestRuntimeTargetFromInspectResolvesProjectRuntime(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "NetworkSettings": {
      "Ports": {
        "8080/tcp": [
          {"HostIp": "172.30.0.1", "HostPort": "3456"}
        ]
      },
      "Networks": {
        "mypaas-projects": {"IPAddress": "10.89.2.17"}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", 3456)
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
	if alias := runtimeRouteAlias(3456); alias != "mypaas-port-3456" {
		t.Fatalf("alias = %q", alias)
	}
}

func TestRuntimeTargetFromInspectDetectsExistingRoutingAttachment(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {
        "mypaas-projects": {"IPAddress": "10.89.2.10"},
        "mypaas-routing": {"IPAddress": "10.90.0.10"}
      }
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", 3456)
	if err != nil {
		t.Fatalf("runtimeTargetFromInspect returned error: %v", err)
	}
	if !got.RoutingAttached {
		t.Fatal("expected existing routing attachment")
	}
}

func TestRuntimeTargetFromInspectSelectsReplacementByAllocatedHostPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.10"}}
    }
  },
  {
    "Id": "bbbbbbbbbbbb1111111111111111111111111111111111111111111111111111",
    "NetworkSettings": {
      "Ports": {"3000/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.11"}}
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", 3456)
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
      "Networks": {"other-network": {"IPAddress": "10.10.0.5"}}
    }
  },
  {
    "Id": "dddddddddddd3333333333333333333333333333333333333333333333333333",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.23"}}
    }
  }
]`)

	got, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", 3456)
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
      "Networks": {"some-other-network": {"IPAddress": "10.10.0.5"}}
    }
  }
]`)

	_, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", 3456)
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
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.17"}}
    }
  }
]`)

	_, err := runtimeTargetFromInspect(raw, "mypaas-projects", "mypaas-routing", 3456)
	if err == nil {
		t.Fatal("expected unknown host port to fail")
	}
	if !strings.Contains(err.Error(), "no running container owns") {
		t.Fatalf("unexpected error: %v", err)
	}
}
