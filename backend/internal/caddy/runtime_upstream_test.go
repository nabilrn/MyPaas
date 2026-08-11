package caddy

import (
	"strings"
	"testing"
)

func TestRuntimeDialFromInspectResolvesProjectNetworkRuntimeDNS(t *testing.T) {
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

	got, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err != nil {
		t.Fatalf("runtimeDialFromInspect returned error: %v", err)
	}
	if got != "0123456789ab:8080" {
		t.Fatalf("dial = %q, want %q", got, "0123456789ab:8080")
	}
}

func TestRuntimeDialFromInspectSelectsReplacementByAllocatedHostPort(t *testing.T) {
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

	got, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err != nil {
		t.Fatalf("runtimeDialFromInspect returned error: %v", err)
	}
	if got != "bbbbbbbbbbbb:3000" {
		t.Fatalf("dial = %q, want replacement runtime DNS identity", got)
	}
}

func TestRuntimeDialFromInspectSkipsSamePortOutsideProjectNetwork(t *testing.T) {
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

	got, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err != nil {
		t.Fatalf("runtimeDialFromInspect returned error: %v", err)
	}
	if got != "dddddddddddd:8080" {
		t.Fatalf("dial = %q, want project runtime", got)
	}
}

func TestRuntimeDialFromInspectRequiresProjectNetworkAttachment(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "eeeeeeeeeeee4444444444444444444444444444444444444444444444444444",
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

func TestRuntimeDialFromInspectRejectsInvalidRuntimeID(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "short",
    "NetworkSettings": {
      "Ports": {"8080/tcp": [{"HostPort": "3456"}]},
      "Networks": {"mypaas-projects": {"IPAddress": "10.89.2.17"}}
    }
  }
]`)

	_, err := runtimeDialFromInspect(raw, "mypaas-projects", 3456)
	if err == nil {
		t.Fatal("expected invalid runtime ID to fail")
	}
	if !strings.Contains(err.Error(), "invalid runtime ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeDialFromInspectRejectsUnknownPublishedPort(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "ffffffffffff5555555555555555555555555555555555555555555555555555",
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
