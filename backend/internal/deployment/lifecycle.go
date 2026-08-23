package deployment

import (
	"context"
	"log/slog"

	"mypaas/internal/db"
)

// ReconcileMissingRuntimes is the startup recovery path for container-backed
// projects. Static projects intentionally have no Docker stack and are restored
// by route reconciliation instead. Container/Compose projects use the same
// immutable creation-time project name as the rest of the deployment engine.
func (s *Service) ReconcileMissingRuntimes(ctx context.Context) error {
	projects, err := s.queries.ListRoutableProjects(ctx)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if !projectHasContainerRuntime(project) {
			continue
		}
		name := runtimeStackName(project)
		if s.docker.StackExists(ctx, name, project.DeployMode) {
			continue
		}
		slog.Info("reconciler: runtime missing for running project, triggering deployment",
			"project", project.Name,
			"id", project.ID,
			"mode", project.DeployMode,
			"runtime", name,
		)
		deployment, err := s.queries.CreateDeployment(ctx, db.CreateDeploymentParams{
			ProjectID:   project.ID,
			TriggeredBy: "manual",
		})
		if err != nil {
			slog.Error("reconciler: failed to create recovery deployment", "project", project.Name, "error", err)
			continue
		}
		go s.runRecoveryDeployment(project.ID, deployment.ID)
	}
	return nil
}

// ReconcileCanonicalRoutes repairs routes that have exactly one route owner.
// Dockerfile/image routes are deliberately excluded: the replica reconciler owns
// both their count=1 primary route and their count>1 multi-upstream route. This
// avoids a periodic primary-only write immediately before every replica pass,
// which made a healthy scaled Caddy route observably flap every reconciliation.
func (s *Service) ReconcileCanonicalRoutes(ctx context.Context) error {
	projects, err := s.queries.ListRoutableProjects(ctx)
	if err != nil {
		return err
	}

	var firstErr error
	reconciled := 0
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if routeOwnedByReplicaReconciler(project) {
			continue
		}
		if err := s.reconcileRoute(ctx, project); err != nil {
			slog.Warn("reconcile canonical caddy route failed",
				"projectId", project.ID,
				"name", project.Name,
				"mode", project.DeployMode,
				"error", err,
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		reconciled++
	}
	slog.Info("canonical caddy routes reconciled", "projects", len(projects), "reconciled", reconciled)
	return firstErr
}

func routeOwnedByReplicaReconciler(project db.Project) bool {
	return project.DeployMode == "dockerfile" || project.DeployMode == "image"
}

func projectHasContainerRuntime(project db.Project) bool {
	return project.DeployMode == "dockerfile" || project.DeployMode == "compose" || project.DeployMode == "image"
}

func runtimeStackName(project db.Project) string {
	return "mypaas-" + project.Name
}
