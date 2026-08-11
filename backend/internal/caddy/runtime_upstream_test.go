package caddy

import (
	"strings"
	"testing"
)

func TestRuntimeDialFromInspectResolvesProjectNetworkContainerPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "container-a",
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

	got, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err != nil {
		t.Fatalf("runtimeDialFromInspect returned error: %v", err)
	}
	if got != "10.89.2.17:8080" {
		t.Fatalf("dial = %q, want %q", got, "10.89.2.17:8080")
	}
}

func TestRuntimeDialFromInspectSelectsContainerByAllocatedHostPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "old-runtime",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.10"}}
    }
  },
  {
    "Id": "replacement-runtime",
    "NetworkSettings": {
      "Ports": {"3000/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.11"}}
    }
  }
]`)

	got, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err != nil {
		t.Fatalf("runtimeDialFromInspect returned error: %v", err)
	}
	if got != "10.89.2.11:3000" {
		t.Fatalf("dial = %q, want replacement runtime", got)
	}
}

func TestRuntimeDialFromInspectRequiresProjectNetworkAttachment(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "runtime-a",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"some-other-network": {"IPAddress": "10.10.0.5"}}
    }
  }
]`)

	_, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err == nil {
		t.Fatal("expected missing project network to fail")
	}
	if !strings.Contains(err.Error(), "not attached to project network") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeDialFromInspectRejectsUnknownPublishedPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "runtime-a",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3455"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.17"}}
    }
  }
]`)

	_, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err == nil {
		t.Fatal("expected unknown host port to fail")
	}
	if !strings.Contains(err.Error(), "no running container owns") {
		t.Fatalf("unexpected error: %v", err)
	}
}
