package deployment

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/caddy"
	"mypaas/internal/compose"
	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/envdiscover"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
	"mypaas/internal/port"
	"mypaas/internal/staticdeploy"
)

type Service struct {
	cfg       *config.Config
	queries   *db.Queries
	envs      *envvar.Service
	ports     *port.Service
	caddy     *caddy.Client
	docker    *container.DockerCLI
	deploySem chan struct{}
	lockMu    sync.Mutex
	locks     map[uuid.UUID]*sync.Mutex
}

func NewService(cfg *config.Config, queries *db.Queries, envs *envvar.Service, ports *port.Service, caddyClient *caddy.Client, docker *container.DockerCLI) *Service {
	maxConcurrent := cfg.MaxConcurrentDeploys
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Service{
		cfg:       cfg,
		queries:   queries,
		envs:      envs,
		ports:     ports,
		caddy:     caddyClient,
		docker:    docker,
		deploySem: make(chan struct{}, maxConcurrent),
		locks:     make(map[uuid.UUID]*sync.Mutex),
	}
}

func (s *Service) TriggerDockerfile(ctx context.Context, projectID, userID uuid.UUID) (db.Deployment, error) {
	project, err := s.queries.GetProjectByID(ctx, projectID)
	if err == pgx.ErrNoRows {
		return db.Deployment{}, errs.ErrNotFound
	}
	if err != nil {
		return db.Deployment{}, err
	}
	lock := s.projectLock(project.ID)
	lock.Lock()
	defer lock.Unlock()

	if active, ok, err := s.activeDeployment(ctx, project.ID); err != nil {
		return db.Deployment{}, err
	} else if ok {
		return active, nil
	}

	deployment, err := s.queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		ProjectID:         project.ID,
		TriggeredBy:       "manual",
		TriggeredByUserID: pgUUID(userID),
	})
	if err != nil {
		return db.Deployment{}, err
	}

	go s.runDeployment(project.ID, deployment.ID)
	return deployment, nil
}

func (s *Service) TriggerWebhook(ctx context.Context, projectID uuid.UUID) (db.Deployment, error) {
	project, err := s.queries.GetProjectByID(ctx, projectID)
	if err == pgx.ErrNoRows {
		return db.Deployment{}, errs.ErrNotFound
	}
	if err != nil {
		return db.Deployment{}, err
	}
	lock := s.projectLock(project.ID)
	lock.Lock()
	defer lock.Unlock()

	if active, ok, err := s.activeDeployment(ctx, project.ID); err != nil {
		return db.Deployment{}, err
	} else if ok {
		return active, nil
	}

	deployment, err := s.queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		ProjectID:   project.ID,
		TriggeredBy: "webhook",
	})
	if err != nil {
		return db.Deployment{}, err
	}

	go s.runDeployment(project.ID, deployment.ID)
	return deployment, nil
}

func (s *Service) activeDeployment(ctx context.Context, projectID uuid.UUID) (db.Deployment, bool, error) {
	deployment, err := s.queries.GetActiveDeploymentByProject(ctx, projectID)
	if err == pgx.ErrNoRows {
		return db.Deployment{}, false, nil
	}
	if err != nil {
		return db.Deployment{}, false, err
	}
	return deployment, true, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]db.Deployment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.queries.ListDeploymentsByProject(ctx, db.ListDeploymentsByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.Deployment, error) {
	deployment, err := s.queries.GetDeploymentByID(ctx, id)
	if err == pgx.ErrNoRows {
		return db.Deployment{}, errs.ErrNotFound
	}
	return deployment, err
}

func (s *Service) Rollback(ctx context.Context, deploymentID, userID uuid.UUID) (db.Deployment, error) {
	target, err := s.Get(ctx, deploymentID)
	if err != nil {
		return db.Deployment{}, err
	}
	if target.Status != "running" {
		return db.Deployment{}, fmt.Errorf("%w: rollback target must be a successful deployment", errs.ErrValidation)
	}

	project, err := s.project(ctx, target.ProjectID)
	if err != nil {
		return db.Deployment{}, err
	}
	if project.DeployMode == "static" {
		return db.Deployment{}, fmt.Errorf("%w: static deployments are rolled forward by redeploying the target commit", errs.ErrValidation)
	}
	if project.DeployMode == "dockerfile" && (target.ImageTag == nil || strings.TrimSpace(*target.ImageTag) == "") {
		return db.Deployment{}, fmt.Errorf("%w: rollback target does not have an image tag", errs.ErrValidation)
	}

	lock := s.projectLock(project.ID)
	lock.Lock()
	defer lock.Unlock()

	return s.rollbackLocked(ctx, project, target, userID)
}

func (s *Service) Start(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if project.DeployMode == "static" {
		if err := s.caddy.AddFileServerRoute(ctx, s.host(project), s.staticCaddyPath(project)); err != nil {
			return err
		}
		return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
	}
	if project.DeployMode == "compose" {
		if err := s.docker.StartComposeProject(ctx, composeProjectName(project.Name)); err != nil {
			return err
		}
		if err := s.addRuntimeRoute(ctx, project); err != nil {
			return err
		}
		return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
	}
	if err := s.docker.Start(ctx, containerName(project.Name)); err != nil {
		return err
	}
	if err := s.addRuntimeRoute(ctx, project); err != nil {
		return err
	}
	return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
}

func (s *Service) Stop(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if project.DeployMode == "static" {
		if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
			return err
		}
		return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "stopped"})
	}
	if project.DeployMode == "compose" {
		if err := s.docker.StopComposeProject(ctx, composeProjectName(project.Name)); err != nil {
			return err
		}
		if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
			return err
		}
		return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "stopped"})
	}
	if err := s.docker.Stop(ctx, containerName(project.Name)); err != nil {
		return err
	}
	if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
		return err
	}
	return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "stopped"})
}

func (s *Service) Restart(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if project.DeployMode == "static" {
		if err := s.caddy.AddFileServerRoute(ctx, s.host(project), s.staticCaddyPath(project)); err != nil {
			return err
		}
		return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
	}
	if project.DeployMode == "compose" {
		if err := s.docker.RestartComposeProject(ctx, composeProjectName(project.Name)); err != nil {
			return err
		}
		if err := s.addRuntimeRoute(ctx, project); err != nil {
			return err
		}
		return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
	}
	if err := s.docker.Restart(ctx, containerName(project.Name)); err != nil {
		return err
	}
	if err := s.addRuntimeRoute(ctx, project); err != nil {
		return err
	}
	return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
}

func (s *Service) addRuntimeRoute(ctx context.Context, project db.Project) error {
	if project.AllocatedPort == nil {
		return fmt.Errorf("%w: project does not have an allocated port", errs.ErrValidation)
	}
	return s.caddy.AddRoute(ctx, s.host(project), *project.AllocatedPort)
}

func (s *Service) ContainerLogs(ctx context.Context, projectID uuid.UUID, tail int) ([]string, error) {
	items, err := s.ContainerLogLines(ctx, projectID, tail)
	if err != nil {
		return nil, err
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Line)
	}
	return lines, nil
}

func (s *Service) ContainerLogLines(ctx context.Context, projectID uuid.UUID, tail int) ([]container.ComposeLogLine, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.DeployMode == "static" {
		return []container.ComposeLogLine{}, nil
	}
	if project.DeployMode == "compose" {
		lines, err := s.docker.ComposeLogsAll(ctx, composeProjectName(project.Name), tail)
		if errors.Is(err, container.ErrNoContainer) {
			return nil, errs.ErrNotFound
		}
		return lines, err
	}
	lines, err := s.docker.Logs(ctx, containerName(project.Name), tail)
	if errors.Is(err, container.ErrNoContainer) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	service := "app"
	if project.MainService != nil && strings.TrimSpace(*project.MainService) != "" {
		service = strings.TrimSpace(*project.MainService)
	}
	items := make([]container.ComposeLogLine, 0, len(lines))
	for _, line := range lines {
		items = append(items, container.ComposeLogLine{Service: service, Line: line})
	}
	return items, nil
}

func (s *Service) ContainerMetrics(ctx context.Context, projectID uuid.UUID) (container.Metrics, error) {
	items, err := s.ContainerMetricsList(ctx, projectID)
	if err != nil {
		return container.Metrics{}, err
	}
	if len(items) == 0 {
		return container.Metrics{}, errs.ErrNotFound
	}
	return items[0], nil
}

func (s *Service) ContainerMetricsList(ctx context.Context, projectID uuid.UUID) ([]container.Metrics, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.DeployMode == "static" || !hasLiveRuntime(project.Status) {
		return idleMetrics(project), nil
	}
	if project.DeployMode == "compose" {
		metrics, err := s.docker.ComposeStatsAll(ctx, composeProjectName(project.Name))
		if errors.Is(err, container.ErrNoContainer) {
			return nil, errs.ErrNotFound
		}
		return metrics, err
	}

	metrics, err := s.docker.Stats(ctx, containerName(project.Name))
	if errors.Is(err, container.ErrNoContainer) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if project.MainService != nil && strings.TrimSpace(*project.MainService) != "" {
		metrics.Service = strings.TrimSpace(*project.MainService)
	} else {
		metrics.Service = "app"
	}
	return []container.Metrics{metrics}, nil
}

func hasLiveRuntime(status string) bool {
	return status == "running" || status == "building"
}

func idleMetrics(project db.Project) []container.Metrics {
	service := mainService(project)
	if project.DeployMode == "static" {
		service = "static"
	}
	return []container.Metrics{{
		Service:       service,
		CPUPercent:    0,
		MemoryMB:      0,
		MemoryLimitMB: float64(project.MemoryLimitMb),
		Uptime:        "n/a",
		CollectedAt:   time.Now().UTC(),
	}}
}

func (s *Service) ComposeResources(ctx context.Context, projectID uuid.UUID) (container.ComposeResourceSummary, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return container.ComposeResourceSummary{}, err
	}
	if project.DeployMode != "compose" {
		return container.ComposeResourceSummary{}, errs.ErrComposeUnsupported
	}
	return s.docker.ComposeResources(ctx, composeProjectName(project.Name))
}

func (s *Service) ResetComposeResources(ctx context.Context, projectID uuid.UUID) error {
	lock := s.projectLock(projectID)
	lock.Lock()
	defer lock.Unlock()

	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if project.DeployMode != "compose" {
		return errs.ErrComposeUnsupported
	}

	if err := s.docker.RemoveComposeProject(ctx, composeProjectName(project.Name)); err != nil {
		return err
	}
	if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
		slog.Warn("remove compose route during reset", "projectId", project.ID, "error", err)
	}
	if err := s.ports.Release(ctx, project.ID); err != nil {
		return err
	}
	if err := s.queries.SetProjectAllocatedPort(ctx, db.SetProjectAllocatedPortParams{ID: project.ID}); err != nil {
		return err
	}
	return s.queries.SetProjectActiveDeployment(ctx, db.SetProjectActiveDeploymentParams{
		ID:                 project.ID,
		ActiveDeploymentID: pgtype.UUID{},
		Status:             "stopped",
	})
}

func (s *Service) CleanupProject(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if project.DeployMode == "static" {
		if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
			slog.Warn("remove static caddy route", "projectId", project.ID, "error", err)
		}
		if err := os.RemoveAll(s.staticProjectPath(project)); err != nil {
			return err
		}
	} else if project.DeployMode == "compose" {
		if err := s.docker.RemoveComposeProject(ctx, composeProjectName(project.Name)); err != nil {
			return err
		}
	} else if err := s.docker.Remove(ctx, containerName(project.Name)); err != nil {
		return err
	}
	if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
		slog.Warn("remove caddy route", "projectId", project.ID, "error", err)
	}
	return s.ports.Release(ctx, project.ID)
}

func (s *Service) ReconcileRoutes(ctx context.Context) error {
	projects, err := s.queries.ListRoutableProjects(ctx)
	if err != nil {
		return fmt.Errorf("list routable projects: %w", err)
	}

	reconciled := 0
	var firstErr error
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.reconcileRoute(ctx, project); err != nil {
			slog.Warn("reconcile caddy route failed",
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

	slog.Info("caddy routes reconciled",
		"projects", len(projects),
		"reconciled", reconciled,
	)
	return firstErr
}

func (s *Service) reconcileRoute(ctx context.Context, project db.Project) error {
	host := s.host(project)
	switch project.DeployMode {
	case "static":
		return s.caddy.AddFileServerRoute(ctx, host, s.staticCaddyPath(project))
	case "dockerfile", "compose":
		if project.AllocatedPort == nil {
			return fmt.Errorf("running %s project has no allocated port", project.DeployMode)
		}
		return s.caddy.AddRoute(ctx, host, *project.AllocatedPort)
	default:
		return fmt.Errorf("unsupported deploy mode %q", project.DeployMode)
	}
}

func (s *Service) UpdateProjectRoute(ctx context.Context, before, after db.Project) error {
	beforeHost := s.host(before)
	afterHost := s.host(after)
	if beforeHost == afterHost && samePort(before.AllocatedPort, after.AllocatedPort) {
		return nil
	}

	if beforeHost != afterHost {
		if err := s.caddy.RemoveRoute(ctx, beforeHost); err != nil {
			slog.Warn("remove old caddy route after project update", "projectId", after.ID, "host", beforeHost, "error", err)
		}
	}

	if after.AllocatedPort == nil {
		if after.DeployMode == "static" && after.Status == "running" {
			return s.caddy.AddFileServerRoute(ctx, afterHost, s.staticCaddyPath(after))
		}
		return nil
	}
	if after.DeployMode == "static" {
		if after.Status != "running" {
			return nil
		}
		return s.caddy.AddFileServerRoute(ctx, afterHost, s.staticCaddyPath(after))
	}
	return s.caddy.AddRoute(ctx, afterHost, *after.AllocatedPort)
}

func (s *Service) runDeployment(projectID, deploymentID uuid.UUID) {
	lock := s.projectLock(projectID)
	lock.Lock()
	defer lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.BuildTimeoutMinutes)*time.Minute)
	defer cancel()

	if err := s.acquireDeploySlot(ctx); err != nil {
		s.fail(ctx, deploymentID, projectID, "pending", err)
		return
	}
	defer s.releaseDeploySlot()

	project, err := s.project(ctx, projectID)
	if err != nil {
		s.fail(ctx, deploymentID, projectID, "pending", err)
		return
	}
	originalStatus := project.Status
	if err := s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "building"}); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	workspace := filepath.Join(os.TempDir(), "mypaas", "builds", deploymentID.String())
	defer os.RemoveAll(workspace)

	log := func(line string) {
		s.appendLog(ctx, deploymentID, line)
	}

	if err := os.MkdirAll(workspace, 0750); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	if err := s.setStatus(ctx, deploymentID, "cloning"); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	log("Cloning repository " + project.RepoUrl)
	if err := runGit(ctx, workspace, log, "clone", "--depth", "1", "--branch", project.Branch, project.RepoUrl, "."); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	commitSHA := gitOutput(ctx, workspace, "rev-parse", "HEAD")
	commitMessage := gitOutput(ctx, workspace, "log", "-1", "--pretty=%s")
	if commitSHA == "" {
		commitSHA = deploymentID.String()
	}
	shortSHA := commitSHA
	if len(shortSHA) > 12 {
		shortSHA = shortSHA[:12]
	}
	var imageTag *string
	if project.DeployMode == "dockerfile" {
		tag := fmt.Sprintf("mypaas/%s:%s", project.Name, shortSHA)
		imageTag = &tag
	} else if project.DeployMode == "compose" {
		tag := fmt.Sprintf("mypaas/%s-%s:%s", project.Name, mainService(project), shortSHA)
		imageTag = &tag
	} else if project.DeployMode == "static" {
		imageTag = nil
	} else {
		s.fail(ctx, deploymentID, projectID, originalStatus, fmt.Errorf("%w: unknown deploy mode %q", errs.ErrValidation, project.DeployMode))
		return
	}
	if err := s.queries.SetDeploymentBuildInfo(ctx, db.SetDeploymentBuildInfoParams{
		ID:            deploymentID,
		CommitSha:     &commitSHA,
		CommitMessage: &commitMessage,
		ImageTag:      imageTag,
	}); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	if project.DeployMode == "static" {
		if err := s.runStaticFromWorkspace(ctx, project, deploymentID, workspace, log); err != nil {
			s.fail(ctx, deploymentID, projectID, originalStatus, err)
			return
		}
		return
	}

	envs, err := s.envs.DecryptedMap(ctx, project.ID)
	if err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	logLocalhostEnvWarnings(log, envs, project.DeployMode)
	envFile := filepath.Join(workspace, ".env")
	if err := envvar.WriteEnvFile(envFile, envs); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	// For compose projects, auto-generate per-service .env files from
	// .env.example templates found in the repo. This lets microservice
	// repos where each service folder has its own .env.example deploy
	// without the user manually creating every .env file.
	if project.DeployMode == "compose" {
		if err := generatePerServiceEnvFiles(workspace, envs, log); err != nil {
			slog.Warn("generate per-service env files", "projectId", project.ID, "error", err)
		}
	}

	if project.DeployMode == "compose" {
		if err := s.runComposeFromWorkspace(ctx, project, deploymentID, workspace, envFile, stringValue(imageTag), log); err != nil {
			s.fail(ctx, deploymentID, projectID, originalStatus, err)
			return
		}
		return
	}

	if _, err := os.Stat(filepath.Join(workspace, "Dockerfile")); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, errs.ErrDockerfileNotFound)
		return
	}

	if err := s.setStatus(ctx, deploymentID, "building"); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	log("Building image " + *imageTag)
	if err := s.docker.Build(ctx, workspace, *imageTag, log); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	if err := s.setStatus(ctx, deploymentID, "starting"); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	if err := s.switchDockerfileContainer(ctx, project, deploymentID, *imageTag, envFile, log); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	if err := s.queries.SetProjectActiveDeployment(ctx, db.SetProjectActiveDeploymentParams{
		ID:                 project.ID,
		ActiveDeploymentID: pgUUID(deploymentID),
		Status:             "running",
	}); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	if err := s.queries.FinishDeployment(ctx, db.FinishDeploymentParams{ID: deploymentID, Status: "running"}); err != nil {
		slog.Error("finish deployment", "deploymentId", deploymentID, "error", err)
		return
	}
	log("Deployment running at " + s.publicURL(project))
}

func (s *Service) runStaticFromWorkspace(ctx context.Context, project db.Project, deploymentID uuid.UUID, workspace string, log func(string)) error {
	source, rel, err := staticdeploy.FindSiteRoot(workspace)
	if err != nil {
		return err
	}

	if err := s.setStatus(ctx, deploymentID, "building"); err != nil {
		return err
	}
	target := s.staticProjectPath(project)
	next := target + "-" + deploymentID.String()
	previous := target + ".previous"
	if err := os.RemoveAll(next); err != nil {
		return err
	}
	if err := os.RemoveAll(previous); err != nil {
		return err
	}

	log("Publishing static files from " + rel)
	if err := staticdeploy.CopyDir(source, next); err != nil {
		return err
	}

	hadPrevious := false
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, previous); err != nil {
			return fmt.Errorf("prepare previous static release: %w", err)
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}

	switched := false
	defer func() {
		if switched {
			if err := os.RemoveAll(previous); err != nil {
				slog.Warn("remove previous static release", "projectId", project.ID, "error", err)
			}
			return
		}
		if err := os.RemoveAll(target); err != nil {
			slog.Warn("remove failed static release", "projectId", project.ID, "error", err)
		}
		if hadPrevious {
			if err := os.Rename(previous, target); err != nil {
				slog.Warn("restore previous static release", "projectId", project.ID, "error", err)
			}
		}
	}()

	if err := os.Rename(next, target); err != nil {
		return fmt.Errorf("activate static release: %w", err)
	}

	if err := s.setStatus(ctx, deploymentID, "starting"); err != nil {
		return err
	}
	log("Updating static route " + s.host(project))
	if err := s.caddy.AddFileServerRoute(ctx, s.host(project), s.staticCaddyPath(project)); err != nil {
		return err
	}

	if err := s.queries.SetProjectActiveDeployment(ctx, db.SetProjectActiveDeploymentParams{
		ID:                 project.ID,
		ActiveDeploymentID: pgUUID(deploymentID),
		Status:             "running",
	}); err != nil {
		return err
	}
	if err := s.queries.FinishDeployment(ctx, db.FinishDeploymentParams{ID: deploymentID, Status: "running"}); err != nil {
		return err
	}
	log("Static deployment running at " + s.publicURL(project))
	switched = true
	return nil
}

func (s *Service) runComposeFromWorkspace(ctx context.Context, project db.Project, deploymentID uuid.UUID, workspace, envFile, imageTag string, log func(string)) error {
	layout, err := resolveComposeLayout(workspace, project, envFile)
	if err != nil {
		return err
	}
	main := mainService(project)
	if main == "" {
		return fmt.Errorf("%w: main service is required for compose projects", errs.ErrValidation)
	}

	services, err := s.docker.ComposeServices(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles...)
	if err != nil {
		return err
	}
	if !containsString(services, main) {
		return fmt.Errorf("%w: compose service %q was not found", errs.ErrValidation, main)
	}
	overrideImageTag := ""
	buildServices, err := s.docker.ComposeBuildServices(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles...)
	if err != nil {
		log("WARNING: could not detect Compose build services; rollback image tagging is disabled for this deployment")
		slog.Warn("detect compose build services", "projectId", project.ID, "error", err)
	} else if containsString(buildServices, main) {
		overrideImageTag = imageTag
	} else {
		log("Compose main service has no build context; rollback will reuse the upstream image declared by Compose")
	}
	if resources, err := s.docker.ComposeResources(ctx, composeProjectName(project.Name)); err == nil {
		total := resources.Containers + resources.Volumes + resources.Networks
		if total > 0 && !project.ActiveDeploymentID.Valid {
			log(fmt.Sprintf(
				"WARNING: found existing Compose resources before first tracked deploy (containers=%d volumes=%d networks=%d). Use Reset Compose resources in settings if these are stale.",
				resources.Containers,
				resources.Volumes,
				resources.Networks,
			))
		}
	} else {
		slog.Warn("check compose resources before deploy", "projectId", project.ID, "error", err)
	}

	port, allocatedNow, err := s.allocateProjectPort(ctx, project)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if succeeded || !allocatedNow {
			return
		}
		if err := s.ports.ReleasePort(context.Background(), port); err != nil {
			slog.Warn("release compose port after failed deploy", "projectId", project.ID, "port", port, "error", err)
		}
		if err := s.queries.SetProjectAllocatedPort(context.Background(), db.SetProjectAllocatedPortParams{ID: project.ID}); err != nil {
			slog.Warn("clear compose allocated port after failed deploy", "projectId", project.ID, "port", port, "error", err)
		}
	}()

	if err := writeComposeOverride(layout.OverrideFile, main, s.docker.ComposePortMapping(port, project.AppPort), project.MemoryLimitMb, numericToFloat(project.CpuLimit), s.cfg.ProjectNetwork, overrideImageTag); err != nil {
		return err
	}
	if err := s.docker.WriteSanitizedComposeConfigMulti(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles, layout.SanitizedFile); err != nil {
		return err
	}

	if err := s.setStatus(ctx, deploymentID, "building"); err != nil {
		return err
	}
	log("Starting compose project " + composeProjectName(project.Name) + " from " + layout.PrimaryRel)
	if err := s.docker.ComposeUp(ctx, container.ComposeUpOptions{
		ProjectName:  composeProjectName(project.Name),
		WorkDir:      layout.WorkDir,
		ComposeFiles: []string{layout.SanitizedFile},
		OverrideFile: layout.OverrideFile,
		EnvFile:      layout.EnvFile,
		Profiles:     project.ComposeProfiles,
	}, log); err != nil {
		return err
	}

	if err := s.setStatus(ctx, deploymentID, "starting"); err != nil {
		return err
	}
	log("Updating route " + s.host(project))
	if err := s.caddy.AddRoute(ctx, s.host(project), port); err != nil {
		return err
	}

	if err := s.queries.SetProjectActiveDeployment(ctx, db.SetProjectActiveDeploymentParams{
		ID:                 project.ID,
		ActiveDeploymentID: pgUUID(deploymentID),
		Status:             "running",
	}); err != nil {
		return err
	}
	if err := s.queries.FinishDeployment(ctx, db.FinishDeploymentParams{ID: deploymentID, Status: "running"}); err != nil {
		return err
	}
	log("Compose deployment running at " + s.publicURL(project))
	succeeded = true
	return nil
}

func (s *Service) rollbackLocked(ctx context.Context, project db.Project, target db.Deployment, userID uuid.UUID) (db.Deployment, error) {
	deployment, err := s.queries.CreateDeployment(ctx, db.CreateDeploymentParams{
		ProjectID:         project.ID,
		CommitSha:         target.CommitSha,
		CommitMessage:     target.CommitMessage,
		TriggeredBy:       "rollback",
		TriggeredByUserID: pgUUID(userID),
		ImageTag:          target.ImageTag,
	})
	if err != nil {
		return db.Deployment{}, err
	}

	originalStatus := project.Status
	log := func(line string) {
		s.appendLog(ctx, deployment.ID, line)
	}
	fail := func(err error) (db.Deployment, error) {
		s.fail(ctx, deployment.ID, project.ID, originalStatus, err)
		return db.Deployment{}, err
	}

	if err := s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "building"}); err != nil {
		return fail(err)
	}
	if err := s.setStatus(ctx, deployment.ID, "starting"); err != nil {
		return fail(err)
	}

	workspace := filepath.Join(os.TempDir(), "mypaas", "rollbacks", deployment.ID.String())
	defer os.RemoveAll(workspace)
	if err := os.MkdirAll(workspace, 0750); err != nil {
		return fail(err)
	}

	if project.DeployMode == "compose" {
		if err := s.checkoutDeploymentCommit(ctx, project, target, workspace, log); err != nil {
			return fail(err)
		}
		envFile, err := s.writeProjectEnvFile(ctx, project.ID, workspace)
		if err != nil {
			return fail(err)
		}
		if err := s.switchComposeRelease(ctx, project, target, workspace, envFile, log); err != nil {
			return fail(err)
		}
	} else {
		envFile, err := s.writeProjectEnvFile(ctx, project.ID, workspace)
		if err != nil {
			return fail(err)
		}
		if err := s.switchDockerfileContainer(ctx, project, deployment.ID, *target.ImageTag, envFile, log); err != nil {
			return fail(err)
		}
	}

	if err := s.queries.SetProjectActiveDeployment(ctx, db.SetProjectActiveDeploymentParams{
		ID:                 project.ID,
		ActiveDeploymentID: pgUUID(deployment.ID),
		Status:             "running",
	}); err != nil {
		return fail(err)
	}
	if err := s.queries.FinishDeployment(ctx, db.FinishDeploymentParams{ID: deployment.ID, Status: "running"}); err != nil {
		return fail(err)
	}
	log("Rollback running at " + s.publicURL(project))

	return s.Get(ctx, deployment.ID)
}

func (s *Service) checkoutDeploymentCommit(ctx context.Context, project db.Project, target db.Deployment, workspace string, log func(string)) error {
	if target.CommitSha == nil || strings.TrimSpace(*target.CommitSha) == "" {
		return fmt.Errorf("%w: rollback target does not have a commit SHA", errs.ErrValidation)
	}
	log("Cloning repository " + project.RepoUrl)
	if err := runGit(ctx, workspace, log, "clone", "--branch", project.Branch, project.RepoUrl, "."); err != nil {
		return err
	}
	log("Checking out commit " + *target.CommitSha)
	if err := runGit(ctx, workspace, log, "checkout", *target.CommitSha); err != nil {
		return err
	}
	return nil
}

func (s *Service) writeProjectEnvFile(ctx context.Context, projectID uuid.UUID, workspace string) (string, error) {
	envs, err := s.envs.DecryptedMap(ctx, projectID)
	if err != nil {
		return "", err
	}
	envFile := filepath.Join(workspace, ".env")
	if err := envvar.WriteEnvFile(envFile, envs); err != nil {
		return "", err
	}
	return envFile, nil
}

func (s *Service) switchComposeRelease(ctx context.Context, project db.Project, target db.Deployment, workspace, envFile string, log func(string)) error {
	imageTag := strings.TrimSpace(stringValue(target.ImageTag))
	layout, err := resolveComposeLayout(workspace, project, envFile)
	if err != nil {
		return err
	}
	main := mainService(project)
	if main == "" {
		return fmt.Errorf("%w: main service is required for compose projects", errs.ErrValidation)
	}
	services, err := s.docker.ComposeServices(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles...)
	if err != nil {
		return err
	}
	if !containsString(services, main) {
		return fmt.Errorf("%w: compose service %q was not found", errs.ErrValidation, main)
	}
	overrideImageTag := ""
	buildServices, err := s.docker.ComposeBuildServices(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles...)
	if err != nil {
		log("WARNING: could not detect Compose build services; rollback will not override the main service image")
		slog.Warn("detect compose build services during rollback", "projectId", project.ID, "error", err)
	} else if containsString(buildServices, main) {
		if imageTag == "" {
			return fmt.Errorf("%w: compose rollback target does not have an image tag", errs.ErrValidation)
		}
		exists, err := s.docker.ImageExists(ctx, imageTag)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: compose rollback image %q is no longer available on this host", errs.ErrValidation, imageTag)
		}
		overrideImageTag = imageTag
	} else {
		log("Compose rollback will use the upstream image declared by Compose")
	}

	port, allocatedNow, err := s.allocateProjectPort(ctx, project)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if succeeded || !allocatedNow {
			return
		}
		if err := s.ports.ReleasePort(context.Background(), port); err != nil {
			slog.Warn("release compose rollback port after failed deploy", "projectId", project.ID, "port", port, "error", err)
		}
		if err := s.queries.SetProjectAllocatedPort(context.Background(), db.SetProjectAllocatedPortParams{ID: project.ID}); err != nil {
			slog.Warn("clear compose rollback allocated port after failed deploy", "projectId", project.ID, "port", port, "error", err)
		}
	}()

	if err := writeComposeOverride(layout.OverrideFile, main, s.docker.ComposePortMapping(port, project.AppPort), project.MemoryLimitMb, numericToFloat(project.CpuLimit), s.cfg.ProjectNetwork, overrideImageTag); err != nil {
		return err
	}
	if err := s.docker.WriteSanitizedComposeConfigMulti(ctx, layout.WorkDir, layout.EnvFile, layout.UserFiles, layout.SanitizedFile); err != nil {
		return err
	}

	log("Starting compose rollback " + composeProjectName(project.Name) + " from " + layout.PrimaryRel)
	if err := s.docker.ComposeUp(ctx, container.ComposeUpOptions{
		ProjectName:  composeProjectName(project.Name),
		WorkDir:      layout.WorkDir,
		ComposeFiles: []string{layout.SanitizedFile},
		OverrideFile: layout.OverrideFile,
		EnvFile:      layout.EnvFile,
		NoBuild:      true,
		Profiles:     project.ComposeProfiles,
	}, log); err != nil {
		return err
	}

	log("Updating route " + s.host(project))
	if err := s.caddy.AddRoute(ctx, s.host(project), port); err != nil {
		return err
	}
	succeeded = true
	return nil
}

func (s *Service) switchDockerfileContainer(ctx context.Context, project db.Project, deploymentID uuid.UUID, imageTag, envFile string, log func(string)) error {
	stableName := containerName(project.Name)
	tempName := temporaryContainerName(project.Name, deploymentID)
	host := s.host(project)

	newPort, err := s.ports.Allocate(ctx, project.ID)
	if err != nil {
		return err
	}
	releaseNewPort := true
	defer func() {
		if releaseNewPort {
			if err := s.ports.ReleasePort(context.Background(), newPort); err != nil {
				slog.Warn("release new port after failed switch", "projectId", project.ID, "port", newPort, "error", err)
			}
		}
	}()

	log("Starting replacement container " + tempName)
	if err := s.docker.Remove(ctx, tempName); err != nil {
		return err
	}
	if err := s.docker.Run(ctx, container.RunOptions{
		Name:          tempName,
		Image:         imageTag,
		HostPort:      newPort,
		ContainerPort: project.AppPort,
		MemoryMB:      project.MemoryLimitMb,
		CPULimit:      numericToFloat(project.CpuLimit),
		EnvFile:       envFile,
	}, log); err != nil {
		return err
	}
	keepTempContainer := false
	defer func() {
		if !keepTempContainer {
			if err := s.docker.Remove(context.Background(), tempName); err != nil {
				slog.Warn("remove temp container after failed switch", "projectId", project.ID, "container", tempName, "error", err)
			}
		}
	}()

	log("Updating route " + host)
	if err := s.caddy.AddRoute(ctx, host, newPort); err != nil {
		return err
	}

	log("Replacing container " + stableName)
	if err := s.docker.Remove(ctx, stableName); err != nil {
		s.restoreRoute(ctx, project, host)
		return err
	}
	if err := s.docker.Rename(ctx, tempName, stableName); err != nil {
		return err
	}

	if err := s.queries.SetProjectAllocatedPort(ctx, db.SetProjectAllocatedPortParams{
		ID:            project.ID,
		AllocatedPort: &newPort,
	}); err != nil {
		return err
	}

	keepTempContainer = true
	releaseNewPort = false
	if project.AllocatedPort != nil && *project.AllocatedPort != newPort {
		if err := s.ports.ReleasePort(ctx, *project.AllocatedPort); err != nil {
			slog.Warn("release old project port", "projectId", project.ID, "port", *project.AllocatedPort, "error", err)
		}
	}
	return nil
}

func (s *Service) restoreRoute(ctx context.Context, project db.Project, host string) {
	if project.AllocatedPort == nil {
		if err := s.caddy.RemoveRoute(ctx, host); err != nil {
			slog.Warn("restore route after failed switch", "projectId", project.ID, "error", err)
		}
		return
	}
	if err := s.caddy.AddRoute(ctx, host, *project.AllocatedPort); err != nil {
		slog.Warn("restore route after failed switch", "projectId", project.ID, "port", *project.AllocatedPort, "error", err)
	}
}

func (s *Service) ensurePort(ctx context.Context, project db.Project) (int32, error) {
	if project.AllocatedPort != nil {
		return *project.AllocatedPort, nil
	}
	port, err := s.ports.Allocate(ctx, project.ID)
	if err != nil {
		return 0, err
	}
	if err := s.queries.SetProjectAllocatedPort(ctx, db.SetProjectAllocatedPortParams{
		ID:            project.ID,
		AllocatedPort: &port,
	}); err != nil {
		return 0, err
	}
	return port, nil
}

func (s *Service) allocateProjectPort(ctx context.Context, project db.Project) (int32, bool, error) {
	if project.AllocatedPort != nil {
		return *project.AllocatedPort, false, nil
	}
	port, err := s.ports.Allocate(ctx, project.ID)
	if err != nil {
		return 0, false, err
	}
	if err := s.queries.SetProjectAllocatedPort(ctx, db.SetProjectAllocatedPortParams{
		ID:            project.ID,
		AllocatedPort: &port,
	}); err != nil {
		if releaseErr := s.ports.ReleasePort(context.Background(), port); releaseErr != nil {
			slog.Warn("release port after failed project allocation", "projectId", project.ID, "port", port, "error", releaseErr)
		}
		return 0, false, err
	}
	return port, true, nil
}

// resolveComposeLayout turns the cloned workspace + persisted project compose
// fields into an absolute compose.Layout for docker compose. When the project
// has no persisted compose_file_path, falls back to recursive discovery so
// existing projects keep working with the same root-only behaviour as before.
func resolveComposeLayout(workspace string, project db.Project, envFile string) (*compose.Layout, error) {
	primaryRel := stringValue(project.ComposeFilePath)
	workdirRel := stringValue(project.ComposeWorkdir)
	overrideRel := project.ComposeOverridePaths
	if overrideRel == nil {
		overrideRel = nil
	}
	return compose.ResolveLayout(
		workspace,
		primaryRel,
		overrideRel,
		workdirRel,
		"docker-compose.mypaas.override.yml",
		"docker-compose.mypaas.sanitized.json",
		envFile,
	)
}

// logLocalhostEnvWarnings scans decrypted env vars for localhost/127.0.0.1
// references and logs actionable warnings before docker compose up. Inside a
// container, localhost means the container itself — not the host or another
// service — so a DATABASE_URL=postgres://...@localhost:5432 will fail. When
// the port matches a compose service, the warning includes a concrete
// suggestion (e.g. "use db:5432 instead").
func logLocalhostEnvWarnings(log func(string), envs map[string]string, deployMode string) {
	warnings := compose.DetectLocalhostInEnv(envs, nil)
	for _, w := range warnings {
		base := fmt.Sprintf("WARNING: env var %s contains localhost", w.Key)
		if w.Port > 0 {
			base += fmt.Sprintf(":%d", w.Port)
		}
		base += ". In Docker, localhost means the container itself, not the host or another service."
		if w.Service != "" {
			log(fmt.Sprintf("%s Compose service %q exposes port %d. Use %s instead of %s", base, w.Service, w.Port, strconv.Quote(w.Suggested), strconv.Quote(w.Value)))
		} else {
			log(fmt.Sprintf("%s Replace localhost with the compose service name (e.g. db, redis, nats).", base))
		}
	}
}

// generatePerServiceEnvFiles walks the workspace for .env.example templates
// and generates a .env file next to each one, substituting values from the
// user's decrypted env vars. This supports microservice/monorepo repos where
// each service folder has its own .env.example with service-specific vars.
//
// Behavior:
//   - If a .env file already exists next to .env.example, it is NOT
//     overwritten — the user's committed .env takes precedence.
//   - If no .env.example exists in a directory, no .env is generated there.
//   - Keys in the user's env vars that aren't in the template are appended.
//   - The root workspace .env (already written above) is skipped.
func generatePerServiceEnvFiles(workspace string, envs map[string]string, log func(string)) error {
	templateNames := map[string]struct{}{
		".env.example":         {},
		".env.sample":          {},
		".env.template":        {},
		".env.local.example":   {},
	}

	generated := 0
	err := filepath.WalkDir(workspace, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if _, isTemplate := templateNames[entry.Name()]; !isTemplate {
			return nil
		}
		dir := filepath.Dir(path)
		// Skip the root workspace — its .env is already written by the caller.
		if dir == filepath.Clean(workspace) {
			return nil
		}
		envPath := filepath.Join(dir, ".env")
		// Don't overwrite an existing .env — user's committed file wins.
		if _, err := os.Stat(envPath); err == nil {
			return nil
		}
		content, err := envdiscover.GenerateEnvFromTemplate(path, envs)
		if err != nil {
			return nil // skip this template, don't fail the whole deploy
		}
		if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
			return nil
		}
		relDir, _ := filepath.Rel(workspace, dir)
		log(fmt.Sprintf("Generated %s/.env from %s", filepath.ToSlash(relDir), entry.Name()))
		generated++
		return nil
	})
	if err != nil {
		return err
	}
	if generated > 0 {
		log(fmt.Sprintf("Generated %d per-service .env file(s) from .env.example templates", generated))
	}
	return nil
}

func writeComposeOverride(path, service, portMapping string, memoryMB int32, cpuLimit float64, projectNetwork, imageTag string) error {
	if memoryMB <= 0 {
		memoryMB = 512
	}
	if cpuLimit <= 0 {
		cpuLimit = 0.5
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`services:
  %q:
    ports:
      - %q
    mem_limit: %dm
    cpus: %.2f
`, service, portMapping, memoryMB, cpuLimit))
	imageTag = strings.TrimSpace(imageTag)
	if imageTag != "" {
		b.WriteString(fmt.Sprintf("    image: %q\n", imageTag))
	}
	projectNetwork = strings.TrimSpace(projectNetwork)
	if projectNetwork != "" {
		b.WriteString(`    networks:
      - default
      - mypaas_platform
`)
	} else {
		b.WriteString(`    extra_hosts:
      - "host.docker.internal:host-gateway"
`)
	}
	b.WriteString(`    restart: unless-stopped
`)
	if projectNetwork != "" {
		b.WriteString(fmt.Sprintf(`
networks:
  mypaas_platform:
    external: true
    name: %q
`, projectNetwork))
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func (s *Service) fail(ctx context.Context, deploymentID, projectID uuid.UUID, originalStatus string, err error) {
	msg := err.Error()
	s.appendLog(ctx, deploymentID, "ERROR: "+msg)
	if updateErr := s.queries.FailDeployment(ctx, db.FailDeploymentParams{ID: deploymentID, ErrorMsg: &msg}); updateErr != nil {
		slog.Error("fail deployment", "deploymentId", deploymentID, "error", updateErr)
	}
	if originalStatus == "" || originalStatus == "building" {
		originalStatus = "pending"
	}
	if updateErr := s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: projectID, Status: originalStatus}); updateErr != nil {
		slog.Error("restore project status", "projectId", projectID, "error", updateErr)
	}
}

func (s *Service) setStatus(ctx context.Context, deploymentID uuid.UUID, status string) error {
	return s.queries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{ID: deploymentID, Status: status})
}

func (s *Service) appendLog(ctx context.Context, deploymentID uuid.UUID, line string) {
	line = strings.TrimRight(line, "\r\n") + "\n"
	if err := s.queries.AppendBuildLog(ctx, db.AppendBuildLogParams{ID: deploymentID, BuildLog: &line}); err != nil {
		slog.Warn("append build log", "deploymentId", deploymentID, "error", err)
	}
}

func (s *Service) project(ctx context.Context, id uuid.UUID) (db.Project, error) {
	project, err := s.queries.GetProjectByID(ctx, id)
	if err == pgx.ErrNoRows {
		return db.Project{}, errs.ErrNotFound
	}
	return project, err
}

func (s *Service) projectLock(projectID uuid.UUID) *sync.Mutex {
	s.lockMu.Lock()
	defer s.lockMu.Unlock()
	lock, ok := s.locks[projectID]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[projectID] = lock
	}
	return lock
}

func (s *Service) acquireDeploySlot(ctx context.Context) error {
	select {
	case s.deploySem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for deployment slot: %w", ctx.Err())
	}
}

func (s *Service) releaseDeploySlot() {
	<-s.deploySem
}

func (s *Service) host(project db.Project) string {
	domain := strings.TrimSpace(s.cfg.PublicDomain)
	if domain == "" {
		domain = "localhost"
	}
	return project.Subdomain + "." + domain
}

func (s *Service) publicURL(project db.Project) string {
	scheme := "https"
	if s.cfg.IsDevelopment() {
		scheme = "http"
	}
	return scheme + "://" + s.host(project)
}

func (s *Service) staticProjectPath(project db.Project) string {
	root := strings.TrimSpace(s.cfg.StaticRoot)
	if root == "" {
		root = "/var/lib/mypaas/static"
	}
	return filepath.Join(root, project.ID.String())
}

func (s *Service) staticCaddyPath(project db.Project) string {
	root := strings.TrimSpace(s.cfg.CaddyStaticRoot)
	if root == "" {
		root = strings.TrimSpace(s.cfg.StaticRoot)
	}
	if root == "" {
		root = "/var/lib/mypaas/static"
	}
	return path.Join(strings.ReplaceAll(root, "\\", "/"), project.ID.String())
}

func containerName(projectName string) string {
	return "mypaas-" + projectName
}

func composeProjectName(projectName string) string {
	return "mypaas-" + projectName
}

func mainService(project db.Project) string {
	if project.MainService == nil {
		return "app"
	}
	service := strings.TrimSpace(*project.MainService)
	if service == "" {
		return "app"
	}
	return service
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func samePort(a, b *int32) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func temporaryContainerName(projectName string, deploymentID uuid.UUID) string {
	return containerName(projectName) + "-" + deploymentID.String()[:12]
}

func runGit(ctx context.Context, dir string, log func(string), args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git start: %w", err)
	}
	done := make(chan struct{}, 2)
	go scan(stdout, log, done)
	go scan(stderr, log, done)
	<-done
	<-done
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func gitOutput(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func scan(pipe io.Reader, log func(string), done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		log(scanner.Text())
	}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func numericToFloat(value pgtype.Numeric) float64 {
	if !value.Valid || value.Int == nil {
		return 0.5
	}
	f, _ := new(big.Rat).SetFrac(value.Int, big.NewInt(1)).Float64()
	result := f * math.Pow10(int(value.Exp))
	if result <= 0 {
		return 0.5
	}
	return result
}
