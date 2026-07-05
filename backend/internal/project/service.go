package project

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
	"mypaas/internal/envdiscover"
	"mypaas/internal/errs"
	"mypaas/internal/quota"
	"mypaas/internal/resourceprofile"
	"mypaas/internal/staticdeploy"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$`)

type Service struct {
	queries *db.Queries
	domain  string
	quota   *quota.Service
}

type CreateInput struct {
	UserID          uuid.UUID
	Name            string
	RepoURL         string
	Branch          string
	DeployMode      string
	ResourceProfile string
	MainService     *string
	AppPort         int32
	MemoryLimitMb   int32
	CPULimit        float64
}

type UpdateInput struct {
	ID              uuid.UUID
	Name            string
	Branch          string
	ResourceProfile string
	AppPort         int32
	MemoryLimitMb   int32
	CPULimit        float64
}

type DetectInput struct {
	RepoURL string
	Branch  string
}

type DetectResult struct {
	DeployMode    string
	MainService   *string
	Services      []string
	ComposeFile   *string
	HasDockerfile bool
	EnvVars       []envdiscover.Var
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

func (s *Service) DetectMode(ctx context.Context, input DetectInput) (DetectResult, error) {
	repoURL := strings.TrimSpace(input.RepoURL)
	if repoURL == "" {
		return DetectResult{}, fmt.Errorf("%w: repository URL is required", errs.ErrValidation)
	}
	branch := strings.TrimSpace(input.Branch)
	if branch == "" {
		branch = "main"
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	workspace, err := os.MkdirTemp("", "mypaas-detect-*")
	if err != nil {
		return DetectResult{}, fmt.Errorf("create detect workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	if err := cloneForDetect(ctx, workspace, repoURL, branch); err != nil {
		return DetectResult{}, err
	}

	composeFile := detectComposeFile(workspace)
	hasDockerfile := fileExists(filepath.Join(workspace, "Dockerfile"))
	envVars, err := envdiscover.Discover(workspace, composeFile)
	if err != nil {
		return DetectResult{}, fmt.Errorf("discover env vars: %w", err)
	}
	if composeFile != "" {
		services, err := detectComposeServices(ctx, workspace, composeFile)
		if err != nil {
			return DetectResult{}, err
		}
		mainService := pickMainService(services)
		return DetectResult{
			DeployMode:    "compose",
			MainService:   &mainService,
			Services:      services,
			ComposeFile:   &composeFile,
			HasDockerfile: hasDockerfile,
			EnvVars:       envVars,
		}, nil
	}
	if hasDockerfile {
		return DetectResult{DeployMode: "dockerfile", HasDockerfile: true, EnvVars: envVars}, nil
	}
	if _, _, err := staticdeploy.FindSiteRoot(workspace); err == nil {
		return DetectResult{DeployMode: "static", HasDockerfile: false, EnvVars: envVars}, nil
	}
	return DetectResult{}, errs.ErrNoDeployConfig
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
	if input.DeployMode != "dockerfile" && input.DeployMode != "compose" && input.DeployMode != "static" {
		return db.Project{}, fmt.Errorf("%w: deploy mode must be dockerfile, compose, or static", errs.ErrValidation)
	}
	if input.DeployMode == "compose" {
		mainService := strings.TrimSpace(valueOrEmpty(input.MainService))
		if mainService == "" {
			return db.Project{}, fmt.Errorf("%w: main service is required for compose projects", errs.ErrValidation)
		}
		input.MainService = &mainService
	} else {
		input.MainService = nil
	}
	if input.DeployMode == "static" && input.AppPort <= 0 {
		input.AppPort = 80
	}
	if input.AppPort <= 0 || input.AppPort > 65535 {
		return db.Project{}, fmt.Errorf("%w: app port must be between 1 and 65535", errs.ErrValidation)
	}
	profileID, memoryLimitMb, cpuLimit, err := resourceprofile.Resolve(input.ResourceProfile, input.DeployMode, input.MemoryLimitMb, input.CPULimit)
	if err != nil {
		return db.Project{}, err
	}
	input.ResourceProfile = profileID
	input.MemoryLimitMb = memoryLimitMb
	input.CPULimit = cpuLimit
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
		UserID:          input.UserID,
		Name:            name,
		RepoUrl:         strings.TrimSpace(input.RepoURL),
		Branch:          strings.TrimSpace(input.Branch),
		Subdomain:       name,
		DeployMode:      input.DeployMode,
		ResourceProfile: input.ResourceProfile,
		MainService:     input.MainService,
		AppPort:         input.AppPort,
		WebhookSecret:   secret,
		MemoryLimitMb:   input.MemoryLimitMb,
		CpuLimit:        numericFromFloat(input.CPULimit),
	})
	if err != nil {
		if isProjectUniqueViolation(err) {
			return db.Project{}, errs.ErrProjectNameTaken
		}
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
	if strings.TrimSpace(input.ResourceProfile) == "" {
		input.ResourceProfile = existing.ResourceProfile
	}
	profileID, memoryLimitMb, cpuLimit, err := resourceprofile.Resolve(input.ResourceProfile, existing.DeployMode, input.MemoryLimitMb, input.CPULimit)
	if err != nil {
		return db.Project{}, err
	}
	input.ResourceProfile = profileID
	input.MemoryLimitMb = memoryLimitMb
	input.CPULimit = cpuLimit
	if s.quota != nil {
		if err := s.quota.CheckUpdate(ctx, existing, input.MemoryLimitMb, input.CPULimit); err != nil {
			return db.Project{}, err
		}
	}

	if err := s.queries.UpdateProject(ctx, db.UpdateProjectParams{
		ID:              input.ID,
		Name:            name,
		Subdomain:       name,
		Branch:          strings.TrimSpace(input.Branch),
		ResourceProfile: input.ResourceProfile,
		AppPort:         input.AppPort,
		MemoryLimitMb:   input.MemoryLimitMb,
		CpuLimit:        numericFromFloat(input.CPULimit),
	}); err != nil {
		if isProjectUniqueViolation(err) {
			return db.Project{}, errs.ErrProjectNameTaken
		}
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

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneForDetect(ctx context.Context, workspace, repoURL, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, repoURL, ".")
	cmd.Dir = workspace
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: failed to clone repository branch %q", errs.ErrValidation, branch)
	}
	return nil
}

func detectComposeFile(workspace string) string {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if fileExists(filepath.Join(workspace, name)) {
			return name
		}
	}
	return ""
}

func detectComposeServices(ctx context.Context, workspace, composeFile string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "config", "--services")
	cmd.Dir = workspace
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: compose file exists but services could not be detected", errs.ErrValidation)
	}
	services := splitNonEmptyLines(string(out))
	if len(services) == 0 {
		return nil, fmt.Errorf("%w: compose file does not define any services", errs.ErrValidation)
	}
	return services, nil
}

func pickMainService(services []string) string {
	for _, service := range services {
		if service == "app" {
			return service
		}
	}
	return services[0]
}

func splitNonEmptyLines(value string) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

func isProjectUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	switch pgErr.ConstraintName {
	case "projects_name_key",
		"projects_subdomain_key",
		"idx_projects_name_active_unique",
		"idx_projects_subdomain_active_unique":
		return true
	default:
		return false
	}
}
