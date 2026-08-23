package container

import "testing"

func TestParseReplicaInspect(t *testing.T) {
	raw := []byte(`[
  {
    "Id": "abc123",
    "Name": "/mypaas-demo-r2",
    "Config": {
      "Image": "mypaas/demo:sha",
      "Labels": {
        "mypaas.replica.project": "project-1",
        "mypaas.replica.slot": "2",
        "mypaas.replica.image": "mypaas/demo:sha"
      }
    },
    "State": {
      "Running": true,
      "Status": "running",
      "Health": {"Status": "healthy"}
    }
  }
]`)
	items, err := parseReplicaInspect(raw)
	if err != nil {
		t.Fatalf("parseReplicaInspect() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	got := items[0]
	if got.Name != "mypaas-demo-r2" || got.Project != "project-1" || got.Slot != 2 || got.Image != "mypaas/demo:sha" || !got.Running || got.Health != "healthy" {
		t.Fatalf("unexpected replica: %+v", got)
	}
}
