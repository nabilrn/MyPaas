package quota

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

type Service struct {
	queries *db.Queries
	cfg     *config.Config
}

type Usage struct {
	MemoryLimitMb int32   `json:"memoryLimitMb"`
	MemoryUsedMb  int32   `json:"memoryUsedMb"`
	CPULimit      float64 `json:"cpuLimit"`
	CPUUsed       float64 `json:"cpuUsed"`
	ProjectLimit  int32   `json:"projectLimit"`
	ProjectCount  int32   `json:"projectCount"`
}

func NewService(queries *db.Queries, cfg *config.Config) *Service {
	return &Service{queries: queries, cfg: cfg}
}

func (s *Service) Usage(ctx context.Context, userID uuid.UUID) (Usage, error) {
	resources, err := s.queries.GetTotalResourcesByUser(ctx, userID)
	if err != nil {
		return Usage{}, err
	}
	count, err := s.queries.CountProjectsByUser(ctx, userID)
	if err != nil {
		return Usage{}, err
	}
	return Usage{
		MemoryLimitMb: s.cfg.UserRAMQuotaMB,
		MemoryUsedMb:  resources.TotalMemoryMb,
		CPULimit:      s.cfg.UserCPUQuota,
		CPUUsed:       numericToFloat(resources.TotalCpu),
		ProjectLimit:  s.cfg.MaxProjects,
		ProjectCount:  int32(count),
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
