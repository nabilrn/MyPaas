package replica

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"mypaas/internal/container"
)

func TestScaleDownToOneDoesNotRequirePrimaryRouteIsolation(t *testing.T) {
	if shouldIsolatePrimaryBeforeReplicaChange(1, true) {
		t.Fatal("scale-down cleanup must not depend on the primary route being currently resolvable")
	}
}

func TestStaleReplicaReplacementStillIsolatesPrimary(t *testing.T) {
	if !shouldIsolatePrimaryBeforeReplicaChange(2, true) {
		t.Fatal("stale multi-replica replacement should isolate traffic to the primary first")
	}
}

func TestPersistentVolumeImagesRejectMultiReplicaMode(t *testing.T) {
	if err := validatePersistentReplicaMode(1, []string{"/data"}); err != nil {
		t.Fatalf("single runtime with persistent volume should remain allowed: %v", err)
	}
	if err := validatePersistentReplicaMode(2, []string{"/data"}); err == nil || !strings.Contains(err.Error(), "persistent VOLUME") {
		t.Fatalf("multi-replica persistent image must be rejected, got %v", err)
	}
}

func TestReplicaTopologyDetectsStaleImageUnhealthyAndDuplicateSlots(t *testing.T) {
	valid := []container.ReplicaInfo{
		{Name: "r2", Slot: 2, Image: "image:new", Running: true, Health: "healthy"},
		{Name: "r3", Slot: 3, Image: "image:new", Running: true},
	}
	if replicaTopologyNeedsRefresh(valid, 3, "image:new") {
		t.Fatal("healthy current-image topology should not be stale")
	}

	cases := [][]container.ReplicaInfo{
		{{Name: "r2", Slot: 2, Image: "image:old", Running: true}},
		{{Name: "r2", Slot: 2, Image: "image:new", Running: true, Health: "unhealthy"}},
		{
			{Name: "r2-a", Slot: 2, Image: "image:new", Running: true},
			{Name: "r2-b", Slot: 2, Image: "image:new", Running: true},
		},
	}
	for i, items := range cases {
		if !replicaTopologyNeedsRefresh(items, 3, "image:new") {
			t.Fatalf("case %d should require topology refresh", i)
		}
	}
}

func TestReadyReplicaUpstreamsAreStableAndHealthyOnly(t *testing.T) {
	projectID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	items := []container.ReplicaInfo{
		{Name: "r3", Slot: 3, Image: "image:new", Running: true},
		{Name: "r2", Slot: 2, Image: "image:new", Running: true, Health: "healthy"},
	}
	upstreams, err := readyReplicaUpstreams(projectID, items, 3, "image:new", 8080)
	if err != nil {
		t.Fatalf("readyReplicaUpstreams() error = %v", err)
	}
	if len(upstreams) != 2 {
		t.Fatalf("upstreams len = %d, want 2", len(upstreams))
	}
	if !strings.HasSuffix(upstreams[0].Dial, "-r2:8080") || !strings.HasSuffix(upstreams[1].Dial, "-r3:8080") {
		t.Fatalf("upstreams are not stable slot order: %+v", upstreams)
	}

	items[0].Health = "unhealthy"
	if _, err := readyReplicaUpstreams(projectID, items, 3, "image:new", 8080); err == nil {
		t.Fatal("unhealthy replica must never be accepted into the Caddy route")
	}
}

func TestReadyReplicaUpstreamsRejectStaleImage(t *testing.T) {
	projectID := uuid.New()
	items := []container.ReplicaInfo{{Name: "r2", Slot: 2, Image: "image:old", Running: true}}
	if _, err := readyReplicaUpstreams(projectID, items, 2, "image:new", 8080); err == nil {
		t.Fatal("stale-image replica must never enter the active route")
	}
}
