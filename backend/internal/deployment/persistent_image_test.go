package deployment

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPersistentImageVolumeNameIsStablePerProjectAndTarget(t *testing.T) {
	projectID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	first := persistentImageVolumeName(projectID, "/app/data")
	second := persistentImageVolumeName(projectID, "/app/data")
	if first != second {
		t.Fatalf("persistentImageVolumeName() is not stable: %q != %q", first, second)
	}
	if !strings.HasPrefix(first, "mypaas-11111111-2222-3333-4444-555555555555-") {
		t.Fatalf("persistentImageVolumeName() = %q, want project-scoped prefix", first)
	}
}

func TestPersistentImageVolumeNameDiffersByTarget(t *testing.T) {
	projectID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	if persistentImageVolumeName(projectID, "/app/data") == persistentImageVolumeName(projectID, "/var/lib/app") {
		t.Fatal("different targets must not share the same named volume")
	}
}

func TestNormalizeImageVolumeTargetRejectsRelativeAndRoot(t *testing.T) {
	for _, target := range []string{"app/data", "/"} {
		if _, err := normalizeImageVolumeTarget(target); err == nil {
			t.Fatalf("normalizeImageVolumeTarget(%q) error = nil, want validation error", target)
		}
	}
}

func TestNormalizeImageVolumeTargetCleansAbsolutePath(t *testing.T) {
	got, err := normalizeImageVolumeTarget("/app/./data/")
	if err != nil {
		t.Fatalf("normalizeImageVolumeTarget() error = %v", err)
	}
	if got != "/app/data" {
		t.Fatalf("normalizeImageVolumeTarget() = %q, want /app/data", got)
	}
}
