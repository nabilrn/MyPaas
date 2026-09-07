package quota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

const runtimeUsageTimeout = 10000 * time.Millisecond

const defaultSecondaryMemoryMB = int32(256)

type Service struct {
	queries *db.Queries
	cfg     *config.Config
	docker  *container.DockerCLI
}

type Usage struct {
	MemoryLimitMb   int32   `json:"memoryLimitMb"`
	MemoryUsedMb    int32   `json:"memoryUsedMb"`
	MemoryRuntimeMb int32   `json:"memoryRuntimeMb"`
	// CPULimit and CPUUsed are retained in the API shape for compatibility.
	// A zero CPU limit means host-shared CPU with no declared reservation.
	CPULimit   float64 `json:"cpuLimit"`
	CPUUsed    float64 `json:"cpuUsed"`
	CPURuntime float64 `json:"cpuRuntime"`
	ProjectLimit int32 `json:"projectLimit"`
	ProjectCount int32 `json:"projectCount"`
}

type serviceResourceLimit struct {
	MemoryLimitMb int32 `json:"memoryLimitMb"`
}

func NewService(queries *db.Queries, cfg *config.Config, dockerClient ...*container.DockerCLI) *Service {
	var docker *container.DockerCLI
	if len(dockerClient) > 0 {
		docker = dockerClient[0]
	}
	return &Service{queries: queries, cfg: cfg, docker: docker}
}

// DeclaredResources returns the hard resource reservation represented by a
// project. Memory remains bounded per runtime. CPU is host-shared, so the
// declared CPU reservation is always zero. The cpu argument and cpuLimit JSON
// fields are accepted for backwards compatibility with existing clients/data.
func DeclaredResources(memoryMb int32, cpu float64, main string, raw json.RawMessage) (int32, float64, error) {
	_ = cpu
	main = strings.TrimSpace(main)
	if main == "" {
		main = "app"
	}

	totalMemory := int64(memoryMb)
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return memoryMb, 0, nil
	}

	var resources map[string]serviceResourceLimit
	if err := json.Unmarshal(raw, &resources); err != nil {
		return 0, 0, fmt.Errorf("decode service resources for quota: %w", err)
	}
	for serviceName, resource := range resources {
		if strings.TrimSpace(serviceName) == main {
			continue
		}
		memory := resource.MemoryLimitMb
		if memory <= 0 {
			memory = defaultSecondaryMemoryMB
		}
		totalMemory += int64(memory)
	}
	if totalMemory > math.MaxInt32 {
		return 0, 0, fmt.Errorf("declared project memory exceeds supported range")
	}
	return int32(totalMemory), 0, nil
}

func (s *Service) Usage(ctx context.Context, userID uuid.UUID) (Usage, error) {
	return s.usage(ctx, userID, false)
}

func (s *Service) UsageWithRuntime(ctx context.Context, userID uuid.UUID) (Usage, error) {
	return s.usage(ctx, userID, true)
}

func (s *Service) usage(ctx context.Context, userID uuid.UUID, includeRuntime bool) (Usage, error) {
	projects, err := s.queries.ListProjectsByUser(ctx, userID)
	if err != nil {
		return Usage{}, err
	}
	declaredMemoryMb, _, err := declaredUsage(projects, uuid.Nil)
	if err != nil {
		return Usage{}, err
	}

	var runtimeMemoryMb int32
	var runtimeCPU float64
	if includeRuntime {
		runtimeCtx, cancel := context.WithTimeout(ctx, runtimeUsageTimeout)
		defer cancel()
		runtimeMemoryMb, runtimeCPU = s.runtimeUsage(runtimeCtx, userID)
	}
	return Usage{
		MemoryLimitMb:   s.cfg.UserRAMQuotaMB,
		MemoryUsedMb:    declaredMemoryMb,
		MemoryRuntimeMb: runtimeMemoryMb,
		CPULimit:        0,
		CPUUsed:         0,
		CPURuntime:      runtimeCPU,
		ProjectLimit:    s.cfg.MaxProjects,
		ProjectCount:    int32(len(projects)),
	}, nil
}

func (s *Service) CheckCreate(ctx context.Context, userID uuid.UUID, memoryMb int32, cpu float64) error {
	_ = cpu
	usage, err := s.Usage(ctx, userID)
	if err != nil {
		return err
	}
	return checkUsage(usage, memoryMb, 0, 1)
}

func (s *Service) CheckUpdate(ctx context.Context, project db.Project, memoryMb int32, cpu float64) error {
	_ = cpu
	projects, err := s.queries.ListProjectsByUser(ctx, project.UserID)
	if err != nil {
		return err
	}
	declaredMemoryMb, _, err := declaredUsage(projects, project.ID)
	if err != nil {
		return err
	}
	usage := Usage{
		MemoryLimitMb: s.cfg.UserRAMQuotaMB,
		MemoryUsedMb:  declaredMemoryMb,
		CPULimit:      0,
		CPUUsed:       0,
		ProjectLimit:  s.cfg.MaxProjects,
		ProjectCount:  int32(len(projects)),
	}
	return checkUsage(usage, memoryMb, 0, 0)
}

func declaredUsage(projects []db.Project, excludeID uuid.UUID) (int32, float64, error) {
	var totalMemory int64
	for _, project := range projects {
		if excludeID != uuid.Nil && project.ID == excludeID {
			continue
		}
		memory, _, err := DeclaredResources(
			project.MemoryLimitMb,
			0,
			mainService(project),
			project.ServiceResources,
		)
		if err != nil {
			return 0, 0, err
		}
		totalMemory += int64(memory)
	}
	if totalMemory > math.MaxInt32 {
		return 0, 0, fmt.Errorf("declared user memory exceeds supported range")
	}
	return int32(totalMemory), 0, nil
}

func checkUsage(usage Usage, addedMemoryMb int32, addedCPU float64, addedProjects int32) error {
	_ = addedCPU
	if usage.ProjectLimit > 0 && usage.ProjectCount+addedProjects > usage.ProjectLimit {
		return fmt.Errorf("%w: project count %d would exceed limit %d", errs.ErrQuotaExceeded, usage.ProjectCount+addedProjects, usage.ProjectLimit)
	}
	if usage.MemoryLimitMb > 0 && usage.MemoryUsedMb+addedMemoryMb > usage.MemoryLimitMb {
		return fmt.Errorf("%w: memory %dMB would exceed limit %dMB", errs.ErrQuotaExceeded, usage.MemoryUsedMb+addedMemoryMb, usage.MemoryLimitMb)
	}
	return nil
}

func (s *Service) runtimeUsage(ctx context.Context, userID uuid.UUID) (int32, float64) {
	if s.docker == nil {
		return 0, 0
	}
	projects, err := s.queries.ListProjectsByUser(ctx, userID)
	if err != nil {
		return 0, 0
	}

	type result struct {
		mem int32
		cpu float64
	}
	results := make(chan result, len(projects))

	for _, p := range projects {
		if p.Status != "running" {
			continue
		}
		go func(project db.Project) {
			metrics, err := s.projectMetrics(ctx, project)
			if err != nil {
				results <- result{0, 0}
				return
			}
			results <- result{
				mem: int32(math.Round(metrics.MemoryMB)),
				cpu: metrics.CPUPercent,
			}
		}(p)
	}

	var memoryMb int32
	var cpuPercent float64

	runningCount := 0
	for _, p := range projects {
		if p.Status == "running" {
			runningCount++
		}
	}

	for i := 0; i < runningCount; i++ {
		res := <-results
		memoryMb += res.mem
		cpuPercent += res.cpu
	}

	return memoryMb, cpuPercent
}

func (s *Service) projectMetrics(ctx context.Context, project db.Project) (container.Metrics, error) {
	if project.DeployMode == "static" {
		return container.Metrics{}, nil
	}
	name := "mypaas-" + project.Name
	if project.DeployMode == "compose" {
		metrics, err := s.docker.ComposeStats(ctx, name, mainService(project))
		if errors.Is(err, container.ErrNoContainer) {
			return container.Metrics{}, err
		}
		return metrics, err
	}
	metrics, err := s.docker.Stats(ctx, name)
	if errors.Is(err, container.ErrNoContainer) {
		return container.Metrics{}, err
	}
	return metrics, err
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
