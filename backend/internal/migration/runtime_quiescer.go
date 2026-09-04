package migration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/db"
)

type ResumeFunc func(context.Context) error

type RuntimeQuiescer interface {
	Quiesce(context.Context) (ResumeFunc, error)
}

type runtimeEngine interface {
	StackExists(context.Context, string, string) bool
	Stop(context.Context, string) error
	Start(context.Context, string) error
	StopComposeProject(context.Context, string) error
	StartComposeProject(context.Context, string) error
}

type runtimeTarget struct {
	name string
	mode string
}

type dockerRuntimeQuiescer struct {
	cfg            *config.Config
	engine         runtimeEngine
	preflight      StoragePreflight
	captureVolumes func(context.Context) error
	cleanupVolumes func() error
}

func newRuntimeQuiescer(cfg *config.Config) RuntimeQuiescer {
	return &dockerRuntimeQuiescer{
		cfg:            cfg,
		engine:         container.NewDockerCLI(cfg.DockerBindHost, cfg.ProjectNetwork),
		preflight:      newStoragePreflight(),
		captureVolumes: captureProjectEngineVolumes,
		cleanupVolumes: cleanupProjectEngineVolumeStage,
	}
}

func (q *dockerRuntimeQuiescer) Quiesce(ctx context.Context) (ResumeFunc, error) {
	if q.preflight != nil {
		if err := q.preflight.Check(ctx); err != nil {
			return nil, err
		}
	}

	pool, err := db.Connect(ctx, q.cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect for runtime quiesce: %w", err)
	}
	defer pool.Close()

	projects, err := db.New(pool).ListRoutableProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("list running projects: %w", err)
	}

	resume, err := quiesceTargets(ctx, q.engine, runtimeTargets(projects))
	if err != nil {
		return nil, err
	}
	return prepareMigrationResume(ctx, resume, q.captureVolumes, q.cleanupVolumes)
}

func prepareMigrationResume(
	ctx context.Context,
	resume ResumeFunc,
	capture func(context.Context) error,
	cleanup func() error,
) (ResumeFunc, error) {
	if resume == nil {
		resume = func(context.Context) error { return nil }
	}
	if capture != nil {
		if err := capture(ctx); err != nil {
			resumeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			resumeErr := resume(resumeCtx)
			cancel()
			var cleanupErr error
			if cleanup != nil {
				cleanupErr = cleanup()
			}
			return nil, errors.Join(fmt.Errorf("capture engine-managed project volumes: %w", err), resumeErr, cleanupErr)
		}
	}

	return func(ctx context.Context) error {
		resumeErr := resume(ctx)
		var cleanupErr error
		if cleanup != nil {
			cleanupErr = cleanup()
		}
		return errors.Join(resumeErr, cleanupErr)
	}, nil
}

func runtimeTargets(projects []db.Project) []runtimeTarget {
	targets := make([]runtimeTarget, 0, len(projects))
	for _, project := range projects {
		if project.DeployMode == "static" {
			continue
		}
		targets = append(targets, runtimeTarget{
			name: "mypaas-" + project.Name,
			mode: project.DeployMode,
		})
	}
	return targets
}

func quiesceTargets(ctx context.Context, engine runtimeEngine, targets []runtimeTarget) (ResumeFunc, error) {
	stopped := make([]runtimeTarget, 0, len(targets))
	for _, target := range targets {
		if !engine.StackExists(ctx, target.name, target.mode) {
			continue
		}
		if err := stopTarget(ctx, engine, target); err != nil {
			resumeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			rollbackErr := resumeTargets(resumeCtx, engine, stopped)
			cancel()
			return nil, errors.Join(fmt.Errorf("stop runtime %s: %w", target.name, err), rollbackErr)
		}
		stopped = append(stopped, target)
	}

	return func(ctx context.Context) error {
		return resumeTargets(ctx, engine, stopped)
	}, nil
}

func stopTarget(ctx context.Context, engine runtimeEngine, target runtimeTarget) error {
	if target.mode == "compose" {
		return engine.StopComposeProject(ctx, target.name)
	}
	return engine.Stop(ctx, target.name)
}

func resumeTargets(ctx context.Context, engine runtimeEngine, targets []runtimeTarget) error {
	var errs []error
	for _, target := range targets {
		var err error
		if target.mode == "compose" {
			err = engine.StartComposeProject(ctx, target.name)
		} else {
			err = engine.Start(ctx, target.name)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("start runtime %s: %w", target.name, err))
		}
	}
	return errors.Join(errs...)
}
