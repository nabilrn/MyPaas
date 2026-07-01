package project

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
	"mypaas/internal/errs"
	"mypaas/internal/quota"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$`)

type Service struct {
	queries *db.Queries
	domain  string
	quota   *quota.Service
}

type CreateInput struct {
	UserID        uuid.UUID
	Name          string
	RepoURL       string
	Branch        string
	DeployMode    string
	MainService   *string
	AppPort       int32
	MemoryLimitMb int32
	CPULimit      float64
}

type UpdateInput struct {
	ID            uuid.UUID
	Name          string
	Branch        string
	AppPort       int32
	MemoryLimitMb int32
	CPULimit      float64
}

func NewService(queries *db.Queries, domain string, quotaService *quota.Service) *Service {
	return &Service{queries: queries, domain: domain, quota: quotaService}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]db.Project, error) {
	return s.queries.ListProjectsByUser(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.Project, error) {
	project, err := s.queries.GetProjectByID(ctx, id)
	if err == pgx.ErrNoRows {
		return db.Project{}, errs.ErrNotFound
	}
	return project, err
}

func (s *Service) Create(ctx context.Context, input CreateInput) (db.Project, error) {
	name := normalizeName(input.Name)
	if err := validateName(name); err != nil {
		return db.Project{}, err
	}
	if strings.TrimSpace(input.RepoURL) == "" {
		return db.Project{}, fmt.Errorf("%w: repository URL is required", errs.ErrValidation)
	}
	if input.Branch == "" {
		input.Branch = "main"
	}
	if input.DeployMode == "" || input.DeployMode == "auto" {
		input.DeployMode = "dockerfile"
	}
	if input.DeployMode == "compose" {
		return db.Project{}, errs.ErrComposeUnsupported
	}
	if input.DeployMode != "dockerfile" {
		return db.Project{}, fmt.Errorf("%w: deploy mode must be dockerfile for MVP", errs.ErrValidation)
	}
	if input.AppPort <= 0 || input.AppPort > 65535 {
		return db.Project{}, fmt.Errorf("%w: app port must be between 1 and 65535", errs.ErrValidation)
	}
	if input.MemoryLimitMb <= 0 {
		input.MemoryLimitMb = 512
	}
	if input.CPULimit <= 0 {
		input.CPULimit = 0.5
	}
	if s.quota != nil {
		if err := s.quota.CheckCreate(ctx, input.UserID, input.MemoryLimitMb, input.CPULimit); err != nil {
			return db.Project{}, err
		}
	}

	if _, err := s.queries.GetProjectByName(ctx, name); err == nil {
		return db.Project{}, errs.ErrProjectNameTaken
	} else if err != pgx.ErrNoRows {
		return db.Project{}, err
	}

	secret, err := randomSecret()
	if err != nil {
		return db.Project{}, fmt.Errorf("generate webhook secret: %w", err)
	}

	project, err := s.queries.CreateProject(ctx, db.CreateProjectParams{
		UserID:        input.UserID,
		Name:          name,
		RepoUrl:       strings.TrimSpace(input.RepoURL),
		Branch:        strings.TrimSpace(input.Branch),
		Subdomain:     name,
		DeployMode:    input.DeployMode,
		MainService:   input.MainService,
		AppPort:       input.AppPort,
		WebhookSecret: secret,
		MemoryLimitMb: input.MemoryLimitMb,
		CpuLimit:      numericFromFloat(input.CPULimit),
	})
	if err != nil {
		return db.Project{}, err
	}
	return project, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (db.Project, error) {
	existing, err := s.Get(ctx, input.ID)
	if err != nil {
		return db.Project{}, err
	}

	name := normalizeName(input.Name)
	if name == "" {
		name = existing.Name
	}
	if err := validateName(name); err != nil {
		return db.Project{}, err
	}
	if name != existing.Name {
		if _, err := s.queries.GetProjectByName(ctx, name); err == nil {
			return db.Project{}, errs.ErrProjectNameTaken
		} else if err != pgx.ErrNoRows {
			return db.Project{}, err
		}
	}
	if input.Branch == "" {
		input.Branch = existing.Branch
	}
	if input.AppPort == 0 {
		input.AppPort = existing.AppPort
	}
	if input.AppPort < 0 || input.AppPort > 65535 {
		return db.Project{}, fmt.Errorf("%w: app port must be between 1 and 65535", errs.ErrValidation)
	}
	if input.MemoryLimitMb <= 0 {
		input.MemoryLimitMb = existing.MemoryLimitMb
	}
	if input.CPULimit <= 0 {
		input.CPULimit = numericToFloat(existing.CpuLimit)
	}
	if s.quota != nil {
		if err := s.quota.CheckUpdate(ctx, existing, input.MemoryLimitMb, input.CPULimit); err != nil {
			return db.Project{}, err
		}
	}

	if err := s.queries.UpdateProject(ctx, db.UpdateProjectParams{
		ID:            input.ID,
		Name:          name,
		Subdomain:     name,
		Branch:        strings.TrimSpace(input.Branch),
		AppPort:       input.AppPort,
		MemoryLimitMb: input.MemoryLimitMb,
		CpuLimit:      numericFromFloat(input.CPULimit),
	}); err != nil {
		return db.Project{}, err
	}

	return s.Get(ctx, input.ID)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.queries.SoftDeleteProject(ctx, id)
}

func (s *Service) RegenerateWebhookSecret(ctx context.Context, id uuid.UUID) (string, error) {
	secret, err := randomSecret()
	if err != nil {
		return "", fmt.Errorf("generate webhook secret: %w", err)
	}
	updated, err := s.queries.UpdateProjectWebhookSecret(ctx, db.UpdateProjectWebhookSecretParams{
		ID:            id,
		WebhookSecret: secret,
	})
	if err == pgx.ErrNoRows {
		return "", errs.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return updated, nil
}

func validateName(name string) error {
	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("%w: project name must be 3-30 chars, lowercase alphanumeric or dash, and start/end with alphanumeric", errs.ErrValidation)
	}
	return nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func randomSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func numericFromFloat(value float64) pgtype.Numeric {
	scaled := int64(math.Round(value * 100))
	return pgtype.Numeric{Int: big.NewInt(scaled), Exp: -2, Valid: true}
}

func numericToFloat(value pgtype.Numeric) float64 {
	if !value.Valid || value.Int == nil {
		return 0
	}
	f, _ := new(big.Rat).SetFrac(value.Int, big.NewInt(1)).Float64()
	return f * math.Pow10(int(value.Exp))
}
