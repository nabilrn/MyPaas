package replica

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/caddy"
	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/envvar"
)

const (
	ReplicaCountKey   = "MYPAAS_PLATFORM_REPLICA_COUNT"
	DefaultMaxReplica = 4
)

type Service struct {
	cfg     *config.Config
	queries *db.Queries
	envs    *envvar.Service
	docker  *container.DockerCLI
	caddy   *caddy.Client

	mu     sync.RWMutex
	status map[uuid.UUID]Status
}

type Status struct {
	Desired      int       `json:"desired"`
	Ready        int       `json:"ready"`
	Image        string    `json:"image,omitempty"`
	LastError    string    `json:"lastError,omitempty"`
	ReconciledAt time.Time `json:"reconciledAt"`
}

func NewService(cfg *config.Config, queries *db.Queries, envs *envvar.Service, docker *container.DockerCLI, caddyClient *caddy.Client) *Service {
	return &Service{
		cfg:     cfg,
		queries: queries,
		envs:    envs,
		docker:  docker,
		caddy:   caddyClient,
		status:  make(map[uuid.UUID]Status),
	}
}

func (s *Service) Start(ctx context.Context, interval time.Duration) <-chan struct{} {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		run := func() {
			reconcileCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			if err := s.Reconcile(reconcileCtx); err != nil {
				slog.Warn("replica reconciliation incomplete", "error", err)
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return done
}

func (s *Service) Reconcile(ctx context.Context) error {
	projects, err := s.queries.ListRoutableProjects(ctx)
	if err != nil {
		return err
	}
	var failures []string
	for _, project := range projects {
		if err := s.reconcileProject(ctx, project); err != nil {
			s.setStatus(project.ID, Status{Desired: 1, LastError: err.Error(), ReconciledAt: time.Now()})
			failures = append(failures, project.Name+": "+err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d project replica reconciliation failure(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return nil
}

func (s *Service) CleanupProject(ctx context.Context, projectID uuid.UUID) error {
	if err := s.docker.RemoveReplicas(ctx, projectID.String()); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.status, projectID)
	s.mu.Unlock()
	return nil
}

func (s *Service) ProjectStatus(projectID uuid.UUID) (Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.status[projectID]
	return status, ok
}

func (s *Service) reconcileProject(ctx context.Context, project db.Project) error {
	if project.DeployMode == "static" || project.DeployMode == "compose" {
		if err := s.docker.RemoveReplicas(ctx, project.ID.String()); err != nil {
			return err
		}
		return nil
	}
	if project.DeployMode != "dockerfile" && project.DeployMode != "image" {
		return nil
	}
	if project.AllocatedPort == nil || *project.AllocatedPort <= 0 {
		return fmt.Errorf("running project has no allocated primary port")
	}

	values, err := s.envs.DecryptedMap(ctx, project.ID)
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, fmt.Errorf("read platform settings: %w", err))
	}
	desired, err := DesiredCount(values)
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, err)
	}
	if desired > 1 {
		if err := s.checkUserBudget(ctx, project.UserID); err != nil {
			return s.preservePrimaryOnError(ctx, project, err)
		}
	}

	deployment, err := s.queries.GetLatestRunningDeployment(ctx, project.ID)
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, fmt.Errorf("resolve active runtime image: %w", err))
	}
	if deployment.ImageTag == nil || strings.TrimSpace(*deployment.ImageTag) == "" {
		return s.preservePrimaryOnError(ctx, project, fmt.Errorf("latest running deployment has no image"))
	}
	image := strings.TrimSpace(*deployment.ImageTag)

	if desired > 1 {
		volumes, err := s.docker.ImageVolumeTargets(ctx, image)
		if err != nil {
			return s.preservePrimaryOnError(ctx, project, fmt.Errorf("inspect image persistence before scaling: %w", err))
		}
		if err := validatePersistentReplicaMode(desired, volumes); err != nil {
			return s.preservePrimaryOnError(ctx, project, err)
		}
	}

	existing, err := s.docker.ReplicaInfos(ctx, project.ID.String())
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, err)
	}
	sortReplicaInfos(existing)
	stale := replicaTopologyNeedsRefresh(existing, desired, image)
	if shouldIsolatePrimaryBeforeReplicaChange(desired, stale) {
		if err := s.routePrimary(ctx, project); err != nil {
			return fmt.Errorf("isolate primary before replica change: %w", err)
		}
	}

	bySlot := make(map[int]container.ReplicaInfo, len(existing))
	for _, item := range existing {
		remove := item.Slot < 2 || item.Slot > desired || item.Image != image || !replicaUsable(item)
		if !remove {
			_, remove = bySlot[item.Slot]
		}
		if remove {
			if err := s.docker.Remove(ctx, item.Name); err != nil {
				return s.preservePrimaryOnError(ctx, project, fmt.Errorf("remove stale replica %s: %w", item.Name, err))
			}
			continue
		}
		bySlot[item.Slot] = item
	}

	if desired == 1 {
		// Scale-down removes aliases that may still be present in the previous
		// multi-upstream route. Rewrite the route only after cleanup so scale
		// 2/3 -> 1 cannot leave Caddy pointing at removed secondary aliases.
		if err := s.routePrimary(ctx, project); err != nil {
			return fmt.Errorf("route primary after replica scale-down: %w", err)
		}
		s.setStatus(project.ID, Status{Desired: 1, Ready: 1, Image: image, ReconciledAt: time.Now()})
		return nil
	}

	tempDir, err := os.MkdirTemp("", "mypaas-replica-env-")
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, err)
	}
	defer os.RemoveAll(tempDir)
	envFile := filepath.Join(tempDir, ".env")
	if err := envvar.WriteEnvFile(envFile, values); err != nil {
		return s.preservePrimaryOnError(ctx, project, fmt.Errorf("write replica environment: %w", err))
	}

	for slot := 2; slot <= desired; slot++ {
		if _, ok := bySlot[slot]; ok {
			continue
		}
		name := replicaName(project.Name, slot)
		alias := replicaAlias(project.ID, slot)
		if err := s.docker.Remove(ctx, name); err != nil {
			return s.preservePrimaryOnError(ctx, project, err)
		}
		if err := s.docker.RunReplica(ctx, container.ReplicaRunOptions{
			Name:           name,
			ProjectID:      project.ID.String(),
			Slot:           slot,
			Image:          image,
			ContainerPort:  project.AppPort,
			MemoryMB:       project.MemoryLimitMb,
			CPULimit:       numericToFloat(project.CpuLimit),
			EnvFile:        envFile,
			RoutingNetwork: routingNetwork(s.cfg),
			RoutingAlias:   alias,
		}, func(line string) {
			slog.Info("replica runtime", "project", project.Name, "slot", slot, "message", line)
		}); err != nil {
			return s.preservePrimaryOnError(ctx, project, fmt.Errorf("start replica slot %d: %w", slot, err))
		}
		if err := s.waitReplicaReady(ctx, project.ID.String(), name, 60*time.Second); err != nil {
			_ = s.docker.Remove(context.Background(), name)
			return s.preservePrimaryOnError(ctx, project, fmt.Errorf("replica slot %d readiness: %w", slot, err))
		}
	}

	items, err := s.docker.ReplicaInfos(ctx, project.ID.String())
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, err)
	}
	upstreams, err := readyReplicaUpstreams(project.ID, items, desired, image, project.AppPort)
	if err != nil {
		return s.preservePrimaryOnError(ctx, project, err)
	}
	if err := s.routeReplicas(ctx, project, upstreams); err != nil {
		return err
	}
	s.setStatus(project.ID, Status{Desired: desired, Ready: desired, Image: image, ReconciledAt: time.Now()})
	return nil
}

func (s *Service) preservePrimaryOnError(ctx context.Context, project db.Project, cause error) error {
	if routeErr := s.routePrimary(ctx, project); routeErr != nil {
		return fmt.Errorf("%w; also failed to preserve primary route: %v", cause, routeErr)
	}
	return cause
}

func (s *Service) waitReplicaReady(ctx context.Context, projectID, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		items, err := s.docker.ReplicaInfos(ctx, projectID)
		if err != nil {
			return err
		}
		found := false
		for _, item := range items {
			if item.Name != name {
				continue
			}
			found = true
			health := strings.ToLower(strings.TrimSpace(item.Health))
			if health == "unhealthy" || (!item.Running && health != "starting") {
				return fmt.Errorf("container is not healthy (running=%v health=%s)", item.Running, item.Health)
			}
			if replicaUsable(item) {
				return nil
			}
		}
		if !found && time.Now().After(deadline) {
			return fmt.Errorf("replica container was not found before timeout")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("readiness timeout after %s", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func replicaUsable(item container.ReplicaInfo) bool {
	if !item.Running {
		return false
	}
	health := strings.ToLower(strings.TrimSpace(item.Health))
	return health == "" || health == "healthy"
}

func shouldIsolatePrimaryBeforeReplicaChange(desired int, stale bool) bool {
	return desired > 1 && stale
}

func validatePersistentReplicaMode(desired int, volumes []string) error {
	if desired > 1 && len(volumes) > 0 {
		return fmt.Errorf("replica count %d rejected: image declares persistent VOLUME targets", desired)
	}
	return nil
}

func replicaTopologyNeedsRefresh(items []container.ReplicaInfo, desired int, image string) bool {
	seen := make(map[int]struct{}, len(items))
	for _, item := range items {
		if item.Slot < 2 || item.Slot > desired || item.Image != image || !replicaUsable(item) {
			return true
		}
		if _, exists := seen[item.Slot]; exists {
			return true
		}
		seen[item.Slot] = struct{}{}
	}
	return false
}

func sortReplicaInfos(items []container.ReplicaInfo) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Slot == items[j].Slot {
			return items[i].Name < items[j].Name
		}
		return items[i].Slot < items[j].Slot
	})
}

func readyReplicaUpstreams(projectID uuid.UUID, items []container.ReplicaInfo, desired int, image string, appPort int32) ([]caddy.ReplicaUpstream, error) {
	ordered := append([]container.ReplicaInfo(nil), items...)
	sortReplicaInfos(ordered)
	bySlot := make(map[int]container.ReplicaInfo, len(ordered))
	for _, item := range ordered {
		if item.Slot < 2 || item.Slot > desired || item.Image != image || !replicaUsable(item) {
			return nil, fmt.Errorf("replica %s is not eligible for the active route", item.Name)
		}
		if _, exists := bySlot[item.Slot]; exists {
			return nil, fmt.Errorf("duplicate ready replica slot %d", item.Slot)
		}
		bySlot[item.Slot] = item
	}

	upstreams := make([]caddy.ReplicaUpstream, 0, desired-1)
	for slot := 2; slot <= desired; slot++ {
		if _, ok := bySlot[slot]; !ok {
			return nil, fmt.Errorf("only %d/%d runtime replicas are ready", len(bySlot)+1, desired)
		}
		upstreams = append(upstreams, caddy.ReplicaUpstream{
			Dial: replicaAlias(projectID, slot) + ":" + strconv.Itoa(int(appPort)),
		})
	}
	if len(bySlot) != desired-1 {
		return nil, fmt.Errorf("unexpected ready replica topology: got %d secondaries, want %d", len(bySlot), desired-1)
	}
	return upstreams, nil
}

func DesiredCount(values map[string]string) (int, error) {
	raw := strings.TrimSpace(values[ReplicaCountKey])
	if raw == "" {
		return 1, nil
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", ReplicaCountKey)
	}
	if count < 1 || count > DefaultMaxReplica {
		return 0, fmt.Errorf("%s must be between 1 and %d", ReplicaCountKey, DefaultMaxReplica)
	}
	return count, nil
}

func (s *Service) checkUserBudget(ctx context.Context, userID uuid.UUID) error {
	projects, err := s.queries.ListProjectsByUser(ctx, userID)
	if err != nil {
		return err
	}
	var memory int64
	var cpu float64
	for _, project := range projects {
		count := 1
		if project.DeployMode == "dockerfile" || project.DeployMode == "image" {
			if values, err := s.envs.DecryptedMap(ctx, project.ID); err == nil {
				if desired, err := DesiredCount(values); err == nil {
					count = desired
				}
			}
		}
		memory += int64(project.MemoryLimitMb) * int64(count)
		cpu += numericToFloat(project.CpuLimit) * float64(count)
	}
	if s.cfg.UserRAMQuotaMB > 0 && memory > int64(s.cfg.UserRAMQuotaMB) {
		return fmt.Errorf("replica reservation %dMB exceeds user memory quota %dMB", memory, s.cfg.UserRAMQuotaMB)
	}
	if s.cfg.UserCPUQuota > 0 && cpu > s.cfg.UserCPUQuota+0.0001 {
		return fmt.Errorf("replica reservation %.2f CPU exceeds user CPU quota %.2f", cpu, s.cfg.UserCPUQuota)
	}
	return nil
}

func (s *Service) routePrimary(ctx context.Context, project db.Project) error {
	host := project.Subdomain + "." + strings.TrimSpace(s.cfg.PublicDomain)
	if project.StaticFrontendPath != nil && strings.TrimSpace(*project.StaticFrontendPath) != "" {
		return s.caddy.AddHybridRoute(ctx, host, staticCaddyPath(s.cfg, project), *project.AllocatedPort)
	}
	return s.caddy.AddRoute(ctx, host, *project.AllocatedPort)
}

func (s *Service) routeReplicas(ctx context.Context, project db.Project, upstreams []caddy.ReplicaUpstream) error {
	host := project.Subdomain + "." + strings.TrimSpace(s.cfg.PublicDomain)
	if project.StaticFrontendPath != nil && strings.TrimSpace(*project.StaticFrontendPath) != "" {
		return s.caddy.AddHybridReplicaRoute(ctx, host, staticCaddyPath(s.cfg, project), *project.AllocatedPort, upstreams)
	}
	return s.caddy.AddReplicaRoute(ctx, host, *project.AllocatedPort, upstreams)
}

func staticCaddyPath(cfg *config.Config, project db.Project) string {
	root := strings.TrimSpace(cfg.CaddyStaticRoot)
	if root == "" {
		root = strings.TrimSpace(cfg.StaticRoot)
	}
	if root == "" {
		root = "/var/lib/mypaas/static"
	}
	return path.Join(strings.ReplaceAll(root, "\\", "/"), project.ID.String())
}

func routingNetwork(cfg *config.Config) string {
	if value := strings.TrimSpace(cfg.RoutingNetwork); value != "" {
		return value
	}
	return "mypaas-routing"
}

func replicaName(projectName string, slot int) string {
	return fmt.Sprintf("mypaas-%s-r%d", projectName, slot)
}

func replicaAlias(projectID uuid.UUID, slot int) string {
	compact := strings.ReplaceAll(projectID.String(), "-", "")
	if len(compact) > 12 {
		compact = compact[:12]
	}
	return fmt.Sprintf("mypaas-replica-%s-r%d", compact, slot)
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

func (s *Service) setStatus(projectID uuid.UUID, status Status) {
	s.mu.Lock()
	s.status[projectID] = status
	s.mu.Unlock()
}
