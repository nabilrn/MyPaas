package deployment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"mypaas/internal/db"
)

func TestProjectHasContainerRuntimeSkipsStaticProjects(t *testing.T) {
	if projectHasContainerRuntime(db.Project{DeployMode: "static"}) {
		t.Fatal("static project must not be treated as a missing Docker runtime")
	}
	for _, mode := range []string{"dockerfile", "compose", "image"} {
		if !projectHasContainerRuntime(db.Project{DeployMode: mode}) {
			t.Fatalf("%s project should have a container runtime", mode)
		}
	}
}

func TestReplicaReconcilerOwnsSingleContainerRuntimeRoutes(t *testing.T) {
	for _, mode := range []string{"dockerfile", "image"} {
		if !routeOwnedByReplicaReconciler(db.Project{DeployMode: mode}) {
			t.Fatalf("%s route must be owned by replica reconciler to avoid primary-only route flapping", mode)
		}
	}
	for _, mode := range []string{"static", "compose"} {
		if routeOwnedByReplicaReconciler(db.Project{DeployMode: mode}) {
			t.Fatalf("%s route must stay with canonical reconciler", mode)
		}
	}
}

func TestRuntimeStackNameMatchesDeploymentNaming(t *testing.T) {
	project := db.Project{ID: uuid.New(), Name: "demo"}
	if got := runtimeStackName(project); got != "mypaas-demo" {
		t.Fatalf("runtimeStackName() = %q, want mypaas-demo", got)
	}
}

func TestStaticReleaseRollbackRestoresPreviousRelease(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "active")
	previous := filepath.Join(root, "active.previous")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(previous, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "version"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(previous, "version"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	release := &staticRelease{target: target, previous: previous, hadPrevious: true}
	release.Rollback()

	content, err := os.ReadFile(filepath.Join(target, "version"))
	if err != nil {
		t.Fatalf("read restored release: %v", err)
	}
	if string(content) != "old" {
		t.Fatalf("restored release = %q, want old", content)
	}
	if _, err := os.Stat(previous); !os.IsNotExist(err) {
		t.Fatalf("previous release still exists after rollback: %v", err)
	}
}

func TestStaticReleaseCommitKeepsNewRelease(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "active")
	previous := filepath.Join(root, "active.previous")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(previous, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "version"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	release := &staticRelease{target: target, previous: previous, hadPrevious: true}
	release.Commit()
	release.Rollback() // committed releases are no-ops on deferred rollback

	content, err := os.ReadFile(filepath.Join(target, "version"))
	if err != nil {
		t.Fatalf("read committed release: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("committed release = %q, want new", content)
	}
	if _, err := os.Stat(previous); !os.IsNotExist(err) {
		t.Fatalf("previous release still exists after commit: %v", err)
	}
}
