package migration

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"mypaas/internal/db"
)

type fakeRuntimeEngine struct {
	exists     map[string]bool
	stopErr    map[string]error
	startErr   map[string]error
	operations []string
}

type fakeStoragePreflight struct {
	err error
}

func (f fakeStoragePreflight) Check(context.Context) error {
	return f.err
}

func (f *fakeRuntimeEngine) StackExists(_ context.Context, name, _ string) bool {
	return f.exists[name]
}

func (f *fakeRuntimeEngine) Stop(_ context.Context, name string) error {
	f.operations = append(f.operations, "stop:"+name)
	return f.stopErr[name]
}

func (f *fakeRuntimeEngine) Start(_ context.Context, name string) error {
	f.operations = append(f.operations, "start:"+name)
	return f.startErr[name]
}

func (f *fakeRuntimeEngine) StopComposeProject(_ context.Context, name string) error {
	f.operations = append(f.operations, "stop-compose:"+name)
	return f.stopErr[name]
}

func (f *fakeRuntimeEngine) StartComposeProject(_ context.Context, name string) error {
	f.operations = append(f.operations, "start-compose:"+name)
	return f.startErr[name]
}

func TestRuntimeTargetsSkipsStaticProjects(t *testing.T) {
	projects := []db.Project{
		{Name: "demo", DeployMode: "dockerfile"},
		{Name: "stack", DeployMode: "compose"},
		{Name: "site", DeployMode: "static"},
		{Name: "image", DeployMode: "image"},
	}

	got := runtimeTargets(projects)
	want := []runtimeTarget{
		{name: "mypaas-demo", mode: "dockerfile"},
		{name: "mypaas-stack", mode: "compose"},
		{name: "mypaas-image", mode: "image"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtimeTargets() = %#v, want %#v", got, want)
	}
}

func TestRuntimeQuiescerPreflightFailsBeforeRuntimeOrDatabaseWork(t *testing.T) {
	preflightErr := errors.New("unportable volume")
	engine := &fakeRuntimeEngine{
		exists:   map[string]bool{"mypaas-demo": true},
		stopErr:  map[string]error{},
		startErr: map[string]error{},
	}
	quiescer := &dockerRuntimeQuiescer{
		engine:    engine,
		preflight: fakeStoragePreflight{err: preflightErr},
	}

	resume, err := quiescer.Quiesce(context.Background())
	if !errors.Is(err, preflightErr) {
		t.Fatalf("Quiesce() error = %v, want preflight error", err)
	}
	if resume != nil {
		t.Fatal("resume must be nil when storage preflight fails")
	}
	if len(engine.operations) != 0 {
		t.Fatalf("runtime operations = %#v, want none", engine.operations)
	}
}

func TestQuiesceTargetsStopsOnlyExistingAndResumes(t *testing.T) {
	engine := &fakeRuntimeEngine{
		exists: map[string]bool{
			"mypaas-demo":  true,
			"mypaas-stack": true,
			"mypaas-gone":  false,
		},
		stopErr:  map[string]error{},
		startErr: map[string]error{},
	}
	targets := []runtimeTarget{
		{name: "mypaas-demo", mode: "dockerfile"},
		{name: "mypaas-stack", mode: "compose"},
		{name: "mypaas-gone", mode: "image"},
	}

	resume, err := quiesceTargets(context.Background(), engine, targets)
	if err != nil {
		t.Fatalf("quiesceTargets() error = %v", err)
	}
	if err := resume(context.Background()); err != nil {
		t.Fatalf("resume() error = %v", err)
	}

	want := []string{
		"stop:mypaas-demo",
		"stop-compose:mypaas-stack",
		"start:mypaas-demo",
		"start-compose:mypaas-stack",
	}
	if !reflect.DeepEqual(engine.operations, want) {
		t.Fatalf("operations = %#v, want %#v", engine.operations, want)
	}
}

func TestQuiesceTargetsRollsBackAlreadyStoppedTargetsOnFailure(t *testing.T) {
	stopFailure := errors.New("stop failed")
	engine := &fakeRuntimeEngine{
		exists: map[string]bool{
			"mypaas-one": true,
			"mypaas-two": true,
		},
		stopErr: map[string]error{
			"mypaas-two": stopFailure,
		},
		startErr: map[string]error{},
	}

	resume, err := quiesceTargets(context.Background(), engine, []runtimeTarget{
		{name: "mypaas-one", mode: "dockerfile"},
		{name: "mypaas-two", mode: "compose"},
	})
	if err == nil || !errors.Is(err, stopFailure) {
		t.Fatalf("quiesceTargets() error = %v, want stop failure", err)
	}
	if resume != nil {
		t.Fatal("resume must be nil when quiescing fails")
	}

	want := []string{
		"stop:mypaas-one",
		"stop-compose:mypaas-two",
		"start:mypaas-one",
	}
	if !reflect.DeepEqual(engine.operations, want) {
		t.Fatalf("operations = %#v, want %#v", engine.operations, want)
	}
}

func TestResumeTargetsAttemptsEveryRuntime(t *testing.T) {
	firstErr := errors.New("first start failed")
	secondErr := errors.New("second start failed")
	engine := &fakeRuntimeEngine{
		exists:  map[string]bool{},
		stopErr: map[string]error{},
		startErr: map[string]error{
			"mypaas-one": firstErr,
			"mypaas-two": secondErr,
		},
	}

	err := resumeTargets(context.Background(), engine, []runtimeTarget{
		{name: "mypaas-one", mode: "dockerfile"},
		{name: "mypaas-two", mode: "compose"},
	})
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("resumeTargets() error = %v, want both start errors", err)
	}
	if len(engine.operations) != 2 {
		t.Fatalf("operations = %#v, want both start attempts", engine.operations)
	}
}
