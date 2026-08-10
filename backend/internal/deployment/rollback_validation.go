package deployment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

func (s *Service) RollbackHistorical(ctx context.Context, deploymentID, userID uuid.UUID) (db.Deployment, error) {
	target, err := s.Get(ctx, deploymentID)
	if err != nil {
		return db.Deployment{}, err
	}
	project, err := s.project(ctx, target.ProjectID)
	if err != nil {
		return db.Deployment{}, err
	}
	if isCurrentDeploymentID(project.ActiveDeploymentID, target.ID) {
		return db.Deployment{}, fmt.Errorf("%w: deployment is already the current release", errs.ErrValidation)
	}
	return s.Rollback(ctx, deploymentID, userID)
}

func isCurrentDeploymentID(activeDeploymentID pgtype.UUID, deploymentID uuid.UUID) bool {
	if !activeDeploymentID.Valid {
		return false
	}
	return uuid.UUID(activeDeploymentID.Bytes) == deploymentID
}
