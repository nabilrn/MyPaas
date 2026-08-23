package replica

import "testing"

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
