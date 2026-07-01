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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/caddy"
	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
	"mypaas/internal/port"
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
	if project.DeployMode != "dockerfile" {
		return db.Deployment{}, errs.ErrComposeUnsupported
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
	if project.DeployMode != "dockerfile" {
		return db.Deployment{}, errs.ErrComposeUnsupported
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
	if target.ImageTag == nil || strings.TrimSpace(*target.ImageTag) == "" {
		return db.Deployment{}, fmt.Errorf("%w: rollback target does not have an image tag", errs.ErrValidation)
	}

	project, err := s.project(ctx, target.ProjectID)
	if err != nil {
		return db.Deployment{}, err
	}
	if project.DeployMode != "dockerfile" {
		return db.Deployment{}, errs.ErrComposeUnsupported
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
	if err := s.docker.Start(ctx, containerName(project.Name)); err != nil {
		return err
	}
	return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
}

func (s *Service) Stop(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if err := s.docker.Stop(ctx, containerName(project.Name)); err != nil {
		return err
	}
	return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "stopped"})
}

func (s *Service) Restart(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if err := s.docker.Restart(ctx, containerName(project.Name)); err != nil {
		return err
	}
	return s.queries.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{ID: project.ID, Status: "running"})
}

func (s *Service) ContainerLogs(ctx context.Context, projectID uuid.UUID, tail int) ([]string, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	lines, err := s.docker.Logs(ctx, containerName(project.Name), tail)
	if errors.Is(err, container.ErrNoContainer) {
		return nil, errs.ErrNotFound
	}
	return lines, err
}

func (s *Service) ContainerMetrics(ctx context.Context, projectID uuid.UUID) (container.Metrics, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return container.Metrics{}, err
	}
	if project.DeployMode != "dockerfile" {
		return container.Metrics{}, errs.ErrComposeUnsupported
	}

	metrics, err := s.docker.Stats(ctx, containerName(project.Name))
	if errors.Is(err, container.ErrNoContainer) {
		return container.Metrics{}, errs.ErrNotFound
	}
	if err != nil {
		return container.Metrics{}, err
	}
	if project.MainService != nil && strings.TrimSpace(*project.MainService) != "" {
		metrics.Service = strings.TrimSpace(*project.MainService)
	} else {
		metrics.Service = "app"
	}
	return metrics, nil
}

func (s *Service) CleanupProject(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	if err := s.docker.Remove(ctx, containerName(project.Name)); err != nil {
		return err
	}
	if err := s.caddy.RemoveRoute(ctx, s.host(project)); err != nil {
		slog.Warn("remove caddy route", "projectId", project.ID, "error", err)
	}
	return s.ports.Release(ctx, project.ID)
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

	if _, err := os.Stat(filepath.Join(workspace, "Dockerfile")); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, errs.ErrDockerfileNotFound)
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
	imageTag := fmt.Sprintf("mypaas/%s:%s", project.Name, shortSHA)
	if err := s.queries.SetDeploymentBuildInfo(ctx, db.SetDeploymentBuildInfoParams{
		ID:            deploymentID,
		CommitSha:     &commitSHA,
		CommitMessage: &commitMessage,
		ImageTag:      &imageTag,
	}); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	envs, err := s.envs.DecryptedMap(ctx, project.ID)
	if err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	envFile := filepath.Join(workspace, ".env")
	if err := envvar.WriteEnvFile(envFile, envs); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	if err := s.setStatus(ctx, deploymentID, "building"); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	log("Building image " + imageTag)
	if err := s.docker.Build(ctx, workspace, imageTag, log); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}

	if err := s.setStatus(ctx, deploymentID, "starting"); err != nil {
		s.fail(ctx, deploymentID, projectID, originalStatus, err)
		return
	}
	if err := s.switchDockerfileContainer(ctx, project, deploymentID, imageTag, envFile, log); err != nil {
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

	envs, err := s.envs.DecryptedMap(ctx, project.ID)
	if err != nil {
		return fail(err)
	}
	envFile := filepath.Join(workspace, ".env")
	if err := envvar.WriteEnvFile(envFile, envs); err != nil {
		return fail(err)
	}

	if err := s.switchDockerfileContainer(ctx, project, deployment.ID, *target.ImageTag, envFile, log); err != nil {
		return fail(err)
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

func containerName(projectName string) string {
	return "mypaas-" + projectName
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
