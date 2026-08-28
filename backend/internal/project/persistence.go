package project

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/db"
	"mypaas/internal/errs"
	"mypaas/internal/quota"
)

func (s *Service) createProjectRecord(ctx context.Context, input CreateInput, name, secret string) (db.Project, error) {
	create := func(queries *db.Queries) (db.Project, error) {
		inUse, err := queries.ProjectHTTPHostLabelInUse(ctx, name, uuid.Nil)
		if err != nil {
			return db.Project{}, err
		}
		if inUse {
			return db.Project{}, errs.ErrProjectNameTaken
		}
		if _, err := queries.GetProjectByName(ctx, name); err == nil {
			return db.Project{}, errs.ErrProjectNameTaken
		} else if err != pgx.ErrNoRows {
			return db.Project{}, err
		}

		project, err := queries.CreateProject(ctx, db.CreateProjectParams{
			UserID:               input.UserID,
			Name:                 name,
			RepoUrl:              strings.TrimSpace(input.RepoURL),
			Branch:               strings.TrimSpace(input.Branch),
			Subdomain:            name,
			DeployMode:           input.DeployMode,
			ResourceProfile:      input.ResourceProfile,
			MainService:          input.MainService,
			AppPort:              input.AppPort,
			WebhookSecret:        secret,
			MemoryLimitMb:        input.MemoryLimitMb,
			CpuLimit:             numericFromFloat(input.CPULimit),
			ComposeFilePath:      input.ComposeFilePath,
			ComposeOverridePaths: normalizeStringSlice(input.ComposeOverridePaths),
			ComposeProfiles:      normalizeStringSlice(input.ComposeProfiles),
			ComposeWorkdir:       input.ComposeWorkdir,
			ServiceResources:     input.ServiceResources,
			StaticFrontendPath:   input.StaticFrontendPath,
			BaseDirectory:        input.BaseDirectory,
			ImageRef:             input.ImageRef,
		})
		if err != nil {
			if isProjectUniqueViolation(err) {
				return db.Project{}, errs.ErrProjectNameTaken
			}
			return db.Project{}, err
		}
		return project, nil
	}

	if s.quota == nil {
		return create(s.queries)
	}

	declaredMemory, declaredCPU, err := quota.DeclaredResources(
		input.MemoryLimitMb,
		input.CPULimit,
		valueOrEmpty(input.MainService),
		input.ServiceResources,
	)
	if err != nil {
		return db.Project{}, err
	}

	var project db.Project
	err = s.quota.WithinCreateQuota(ctx, input.UserID, declaredMemory, declaredCPU, func(queries *db.Queries) error {
		var err error
		project, err = create(queries)
		return err
	})
	if err != nil {
		return db.Project{}, err
	}
	return project, nil
}

func (s *Service) updateProjectRecord(
	ctx context.Context,
	existing db.Project,
	input UpdateInput,
	params db.UpdateProjectParams,
) (db.Project, error) {
	update := func(queries *db.Queries) error {
		if err := queries.UpdateProject(ctx, params); err != nil {
			if isProjectUniqueViolation(err) {
				return errs.ErrProjectNameTaken
			}
			return err
		}
		return nil
	}

	var err error
	if s.quota == nil {
		err = update(s.queries)
	} else {
		declaredMemory, declaredCPU, resourceErr := quota.DeclaredResources(
			input.MemoryLimitMb,
			input.CPULimit,
			valueOrEmpty(params.MainService),
			params.ServiceResources,
		)
		if resourceErr != nil {
			return db.Project{}, resourceErr
		}
		err = s.quota.WithinUpdateQuota(ctx, existing, declaredMemory, declaredCPU, update)
	}
	if err != nil {
		return db.Project{}, err
	}
	return s.Get(ctx, input.ID)
}