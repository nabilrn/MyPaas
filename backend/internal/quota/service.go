package quota

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

const runtimeUsageTimeout = 1200 * time.Millisecond

type Service struct {
	queries *db.Queries
	cfg     *config.Config
	docker  *container.DockerCLI
}

type Usage struct {
	MemoryLimitMb   int32   `json:"memoryLimitMb"`
	MemoryUsedMb    int32   `json:"memoryUsedMb"`
	MemoryRuntimeMb int32   `json:"memoryRuntimeMb"`
	CPULimit        float64 `json:"cpuLimit"`
	CPUUsed         float64 `json:"cpuUsed"`
	CPURuntime      float64 `json:"cpuRuntime"`
	ProjectLimit    int32   `json:"projectLimit"`
	ProjectCount    int32   `json:"projectCount"`
}

func NewService(queries *db.Queries, cfg *config.Config, dockerClient ...*container.DockerCLI) *Service {
	var docker *container.DockerCLI
	if len(dockerClient) > 0 {
		docker = dockerClient[0]
	}
	return &Service{queries: queries, cfg: cfg, docker: docker}
}

func (s *Service) Usage(ctx context.Context, userID uuid.UUID) (Usage, error) {
	return s.usage(ctx, userID, false)
}

func (s *Service) UsageWithRuntime(ctx context.Context, userID uuid.UUID) (Usage, error) {
	return s.usage(ctx, userID, true)
}

func (s *Service) usage(ctx context.Context, userID uuid.UUID, includeRuntime bool) (Usage, error) {
	resources, err := s.queries.GetTotalResourcesByUser(ctx, userID)
	if err != nil {
		return Usage{}, err
	}
	count, err := s.queries.CountProjectsByUser(ctx, userID)
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
		MemoryUsedMb:    resources.TotalMemoryMb,
		MemoryRuntimeMb: runtimeMemoryMb,
		CPULimit:        s.cfg.UserCPUQuota,
		CPUUsed:         numericToFloat(resources.TotalCpu),
		CPURuntime:      runtimeCPU,
		ProjectLimit:    s.cfg.MaxProjects,
		ProjectCount:    int32(count),
	}, nil
}

func (s *Service) CheckCreate(ctx context.Context, userID uuid.UUID, memoryMb int32, cpu float64) error {
	usage, err := s.Usage(ctx, userID)
	if err != nil {
		return err
	}
	return checkUsage(usage, memoryMb, cpu, 1)
}

func (s *Service) CheckUpdate(ctx context.Context, project db.Project, memoryMb int32, cpu float64) error {
	resources, err := s.queries.GetTotalResourcesByUserExcludingProject(ctx, db.GetTotalResourcesByUserExcludingProjectParams{
		UserID: project.UserID,
		ID:     project.ID,
	})
	if err != nil {
		return err
	}
	count, err := s.queries.CountProjectsByUser(ctx, project.UserID)
	if err != nil {
		return err
	}
	usage := Usage{
		MemoryLimitMb: s.cfg.UserRAMQuotaMB,
		MemoryUsedMb:  resources.TotalMemoryMb,
		CPULimit:      s.cfg.UserCPUQuota,
		CPUUsed:       numericToFloat(resources.TotalCpu),
		ProjectLimit:  s.cfg.MaxProjects,
		ProjectCount:  int32(count),
	}
	return checkUsage(usage, memoryMb, cpu, 0)
}

func checkUsage(usage Usage, addedMemoryMb int32, addedCPU float64, addedProjects int32) error {
	if usage.ProjectLimit > 0 && usage.ProjectCount+addedProjects > usage.ProjectLimit {
		return fmt.Errorf("%w: project count %d would exceed limit %d", errs.ErrQuotaExceeded, usage.ProjectCount+addedProjects, usage.ProjectLimit)
	}
	if usage.MemoryLimitMb > 0 && usage.MemoryUsedMb+addedMemoryMb > usage.MemoryLimitMb {
		return fmt.Errorf("%w: memory %dMB would exceed limit %dMB", errs.ErrQuotaExceeded, usage.MemoryUsedMb+addedMemoryMb, usage.MemoryLimitMb)
	}
	if usage.CPULimit > 0 && usage.CPUUsed+addedCPU > usage.CPULimit {
		return fmt.Errorf("%w: CPU %.2f would exceed limit %.2f", errs.ErrQuotaExceeded, usage.CPUUsed+addedCPU, usage.CPULimit)
	}
	return nil
}

func numericToFloat(value pgtype.Numeric) float64 {
	if !value.Valid || value.Int == nil {
		return 0
	}
	f, _ := new(big.Rat).SetFrac(value.Int, big.NewInt(1)).Float64()
	return f * math.Pow10(int(value.Exp))
}

func (s *Service) runtimeUsage(ctx context.Context, userID uuid.UUID) (int32, float64) {
	if s.docker == nil {
		return 0, 0
	}
	projects, err := s.queries.ListProjectsByUser(ctx, userID)
	if err != nil {
		return 0, 0
	}
	var memoryMb int32
	var cpuPercent float64
	for _, project := range projects {
		if project.Status != "running" {
			continue
		}
		metrics, err := s.projectMetrics(ctx, project)
		if err != nil {
			continue
		}
		memoryMb += int32(math.Round(metrics.MemoryMB))
		cpuPercent += metrics.CPUPercent
	}
	return memoryMb, cpuPercent
}

func (s *Service) projectMetrics(ctx context.Context, project db.Project) (container.Metrics, error) {
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
