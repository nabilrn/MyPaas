package project

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/big"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

const maxRepoTreeEntries = 250

var errRepoTreeLimitReached = errors.New("repository tree limit reached")

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
	RepoURL     string
	Branch      string
	InspectOnly bool
}

type DetectResult struct {
	DeployMode    string
	Branch        string
	DefaultBranch string
	Branches      []string
	MainService   *string
	Services      []string
	ComposeFile   *string
	HasDockerfile bool
	EnvVars       []envdiscover.Var
	AppPort       int32
	ComposePlan   *ComposePlan
	Tree          []RepoTreeEntry
	TreeTruncated bool
}

type RepoTreeEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"`
	Depth int    `json:"depth"`
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

	ctx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()

	defaultBranch, branches, err := inspectRemoteBranches(ctx, repoURL)
	if err != nil {
		return DetectResult{}, err
	}

	branch := strings.TrimSpace(input.Branch)
	if branch == "" {
		branch = defaultBranch
	}
	if branch == "" {
		return DetectResult{}, fmt.Errorf("%w: branch is required", errs.ErrValidation)
	}

	if input.InspectOnly {
		tree, truncated, err := inspectRepositoryTree(ctx, repoURL, branch)
		if err != nil {
			return DetectResult{}, err
		}
		return DetectResult{
			Branch:        branch,
			DefaultBranch: defaultBranch,
			Branches:      branches,
			Tree:          tree,
			TreeTruncated: truncated,
		}, nil
	}

	result, err := detectModeOnBranch(ctx, repoURL, branch)
	if err != nil {
		return DetectResult{}, err
	}
	result.DefaultBranch = defaultBranch
	result.Branches = branches
	return result, nil
}

func detectModeOnBranch(ctx context.Context, repoURL, branch string) (DetectResult, error) {
	if branch == "" {
		return DetectResult{}, fmt.Errorf("%w: branch is required", errs.ErrValidation)
	}

	workspace, err := os.MkdirTemp("", "mypaas-detect-*")
	if err != nil {
		return DetectResult{}, fmt.Errorf("create detect workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	if err := cloneForDetect(ctx, workspace, repoURL, branch); err != nil {
		return DetectResult{}, err
	}
	tree, treeTruncated, err := listRepositoryTree(workspace, maxRepoTreeEntries)
	if err != nil {
		return DetectResult{}, fmt.Errorf("list repository tree: %w", err)
	}

	composeFile := detectComposeFile(workspace)
	hasDockerfile := fileExists(filepath.Join(workspace, "Dockerfile"))
	envVars, err := envdiscover.Discover(workspace, composeFile)
	if err != nil {
		return DetectResult{}, fmt.Errorf("discover env vars: %w", err)
	}
	if composeFile != "" {
		if err := prepareComposePreviewEnv(workspace, composeFile, envVars); err != nil {
			return DetectResult{}, err
		}
		services, err := detectComposeServices(ctx, workspace, composeFile)
		if err != nil {
			return DetectResult{}, err
		}
		mainService := pickMainService(ctx, workspace, composeFile, services)
		appPort := inferComposeAppPort(ctx, workspace, composeFile, mainService, envVars)
		composePlan, err := inspectComposePlan(ctx, workspace, composeFile, services, mainService, appPort, envVars)
		if err != nil {
			return DetectResult{}, err
		}
		return DetectResult{
			DeployMode:    "compose",
			Branch:        branch,
			MainService:   &mainService,
			Services:      services,
			ComposeFile:   &composeFile,
			HasDockerfile: hasDockerfile,
			EnvVars:       envVars,
			AppPort:       appPort,
			ComposePlan:   composePlan,
			Tree:          tree,
			TreeTruncated: treeTruncated,
		}, nil
	}
	if hasDockerfile {
		return DetectResult{DeployMode: "dockerfile", Branch: branch, HasDockerfile: true, EnvVars: envVars, AppPort: inferDockerfileAppPort(workspace, envVars), Tree: tree, TreeTruncated: treeTruncated}, nil
	}
	if _, _, err := staticdeploy.FindSiteRoot(workspace); err == nil {
		return DetectResult{DeployMode: "static", Branch: branch, HasDockerfile: false, EnvVars: envVars, AppPort: 80, Tree: tree, TreeTruncated: treeTruncated}, nil
	}
	return DetectResult{}, fmt.Errorf("%w: no deploy config found on branch %q", errs.ErrNoDeployConfig, branch)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (db.Project, error) {
	name := normalizeName(input.Name)
	if err := validateName(name); err != nil {
		return db.Project{}, err
	}
	if strings.TrimSpace(input.RepoURL) == "" {
		return db.Project{}, fmt.Errorf("%w: repository URL is required", errs.ErrValidation)
	}
	input.Branch = strings.TrimSpace(input.Branch)
	if input.Branch == "" {
		branch, err := resolveDefaultBranch(ctx, input.RepoURL)
		if err != nil {
			return db.Project{}, err
		}
		input.Branch = branch
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
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--single-branch", "--branch", branch, repoURL, ".")
	cmd.Dir = workspace
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: failed to clone repository branch %q", errs.ErrValidation, branch)
	}
	return nil
}

func resolveDefaultBranch(ctx context.Context, repoURL string) (string, error) {
	defaultBranch, _, err := inspectRemoteBranches(ctx, repoURL)
	return defaultBranch, err
}

func listRemoteBranches(ctx context.Context, repoURL string) ([]string, error) {
	_, branches, err := inspectRemoteBranches(ctx, repoURL)
	return branches, err
}

func inspectRemoteBranches(ctx context.Context, repoURL string) (string, []string, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return "", nil, fmt.Errorf("%w: repository URL is required", errs.ErrValidation)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "git", "ls-remote", "--symref", repoURL, "HEAD", "refs/heads/*").CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to inspect remote branches: %s", errs.ErrValidation, firstNonEmptyLine(string(out)))
	}
	defaultBranch := parseDefaultBranchRef(string(out))
	branches := prioritizeDefaultBranch(defaultBranch, parseRemoteBranchRefs(string(out)))
	if defaultBranch == "" && len(branches) > 0 {
		defaultBranch = branches[0]
	}
	if defaultBranch == "" {
		return "", nil, fmt.Errorf("%w: failed to detect default branch", errs.ErrValidation)
	}
	return defaultBranch, branches, nil
}

func parseDefaultBranchRef(output string) string {
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "ref:" || fields[2] != "HEAD" || !strings.HasPrefix(fields[1], "refs/heads/") {
			continue
		}
		return strings.TrimPrefix(fields[1], "refs/heads/")
	}
	return ""
}

func parseRemoteBranchRefs(output string) []string {
	seen := make(map[string]struct{})
	branches := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || !strings.HasPrefix(fields[1], "refs/heads/") {
			continue
		}
		branch := strings.TrimPrefix(fields[1], "refs/heads/")
		if branch == "" {
			continue
		}
		if _, ok := seen[branch]; ok {
			continue
		}
		seen[branch] = struct{}{}
		branches = append(branches, branch)
	}
	return branches
}

func prioritizeDefaultBranch(defaultBranch string, branches []string) []string {
	if len(branches) == 0 {
		return branches
	}
	rest := make([]string, 0, len(branches))
	hasDefault := false
	for _, branch := range branches {
		if branch == defaultBranch {
			hasDefault = true
			continue
		}
		rest = append(rest, branch)
	}
	sort.Strings(rest)
	if !hasDefault || defaultBranch == "" {
		return rest
	}
	return append([]string{defaultBranch}, rest...)
}

func inspectRepositoryTree(ctx context.Context, repoURL, branch string) ([]RepoTreeEntry, bool, error) {
	workspace, err := os.MkdirTemp("", "mypaas-repo-preview-*")
	if err != nil {
		return nil, false, fmt.Errorf("create repository preview workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	if err := cloneForTreePreview(ctx, workspace, repoURL, branch); err != nil {
		return nil, false, err
	}
	tree, truncated, err := listGitRepositoryTree(ctx, workspace, maxRepoTreeEntries)
	if err != nil {
		return nil, false, err
	}
	return tree, truncated, nil
}

func cloneForTreePreview(ctx context.Context, workspace, repoURL, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--filter", "blob:none", "--no-checkout", "--single-branch", "--branch", branch, repoURL, ".")
	cmd.Dir = workspace
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: failed to clone repository branch %q for preview", errs.ErrValidation, branch)
	}
	return nil
}

func listGitRepositoryTree(ctx context.Context, workspace string, limit int) ([]RepoTreeEntry, bool, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "-t", "--full-tree", "HEAD")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, false, fmt.Errorf("%w: failed to read repository tree: %s", errs.ErrValidation, firstNonEmptyLine(string(out)))
	}
	tree, truncated := parseGitTreeEntries(string(out), limit)
	return tree, truncated, nil
}

func parseGitTreeEntries(output string, limit int) ([]RepoTreeEntry, bool) {
	entries := make([]RepoTreeEntry, 0)
	truncated := false
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if len(entries) >= limit {
			truncated = true
			break
		}
		meta, rel, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) < 2 {
			continue
		}
		entryType := "file"
		switch fields[1] {
		case "tree":
			entryType = "directory"
		case "blob":
			entryType = "file"
		default:
			continue
		}
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		entries = append(entries, RepoTreeEntry{
			Name:  pathpkg.Base(rel),
			Path:  rel,
			Type:  entryType,
			Depth: strings.Count(rel, "/"),
		})
	}
	return entries, truncated
}

func listRepositoryTree(root string, limit int) ([]RepoTreeEntry, bool, error) {
	root = filepath.Clean(root)
	entries := make([]RepoTreeEntry, 0)
	truncated := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if shouldSkipRepoPreview(entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if len(entries) >= limit {
			truncated = true
			return errRepoTreeLimitReached
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		entryType := "file"
		if entry.IsDir() {
			entryType = "directory"
		}
		entries = append(entries, RepoTreeEntry{
			Name:  entry.Name(),
			Path:  rel,
			Type:  entryType,
			Depth: strings.Count(rel, "/"),
		})
		return nil
	})
	if errors.Is(err, errRepoTreeLimitReached) {
		err = nil
	}
	return entries, truncated, err
}

func shouldSkipRepoPreview(entry fs.DirEntry) bool {
	if !entry.IsDir() {
		return false
	}
	switch strings.ToLower(entry.Name()) {
	case ".git", "node_modules", "vendor", ".cache", ".turbo":
		return true
	default:
		return false
	}
}

func firstNonEmptyLine(value string) string {
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "unknown error"
}

func detectComposeFile(workspace string) string {
	for _, name := range composeFileCandidates(workspace) {
		if fileExists(filepath.Join(workspace, name)) {
			return name
		}
	}
	return ""
}

func composeFileCandidates(workspace string) []string {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"docker-compose.prod.yml",
		"docker-compose.prod.yaml",
		"compose.prod.yml",
		"compose.prod.yaml",
		"docker-compose.production.yml",
		"docker-compose.production.yaml",
		"compose.production.yml",
		"compose.production.yaml",
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, name := range candidates {
		seen[name] = struct{}{}
	}

	for _, pattern := range []string{"docker-compose.*.yml", "docker-compose.*.yaml", "compose.*.yml", "compose.*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(workspace, pattern))
		if err != nil {
			continue
		}
		sort.Strings(matches)
		for _, match := range matches {
			name := filepath.Base(match)
			if ignoredComposeCandidate(name) {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			candidates = append(candidates, name)
		}
	}
	return candidates
}

func ignoredComposeCandidate(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(normalized, "override") || strings.Contains(normalized, "test")
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

func pickMainService(ctx context.Context, workspace, composeFile string, services []string) string {
	if service := pickServiceWithPublishedPort(ctx, workspace, composeFile, services); service != "" {
		return service
	}
	for _, preferred := range []string{"app", "web", "frontend", "api", "backend", "server"} {
		for _, service := range services {
			if service == preferred {
				return service
			}
		}
	}
	if len(services) == 0 {
		return ""
	}
	return services[0]
}

func pickServiceWithPublishedPort(ctx context.Context, workspace, composeFile string, services []string) string {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "config", "--format", "json")
	cmd.Dir = workspace
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return pickMainServiceFromComposeConfig(out, services)
}

func pickMainServiceFromComposeConfig(rawConfig []byte, services []string) string {
	var config struct {
		Services map[string]struct {
			Ports  json.RawMessage `json:"ports"`
			Expose json.RawMessage `json:"expose"`
		} `json:"services"`
	}
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return ""
	}
	for _, service := range services {
		raw, ok := config.Services[service]
		if !ok {
			continue
		}
		if parseComposePorts(raw.Ports, nil) > 0 {
			return service
		}
	}
	for _, service := range services {
		raw, ok := config.Services[service]
		if !ok {
			continue
		}
		if parseComposeExpose(raw.Expose, nil) > 0 {
			return service
		}
	}
	return ""
}

func inferDockerfileAppPort(workspace string, envVars []envdiscover.Var) int32 {
	env := envDefaultsFromDiscovery(envVars)
	file, err := os.Open(filepath.Join(workspace, "Dockerfile"))
	if err != nil {
		return portFromEnv(env)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := stripDockerfileComment(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "ENV":
			mergeDockerfileEnv(env, strings.TrimSpace(line[len(fields[0]):]))
		case "EXPOSE":
			for _, token := range fields[1:] {
				if port := parsePortToken(token, env); port > 0 {
					return port
				}
			}
		}
	}
	return portFromEnv(env)
}

func inferComposeAppPort(ctx context.Context, workspace, composeFile, serviceName string, envVars []envdiscover.Var) int32 {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "config", "--format", "json")
	cmd.Dir = workspace
	out, err := cmd.Output()
	if err != nil {
		return portFromEnv(envDefaultsFromDiscovery(envVars))
	}

	var config struct {
		Services map[string]json.RawMessage `json:"services"`
	}
	if err := json.Unmarshal(out, &config); err != nil {
		return portFromEnv(envDefaultsFromDiscovery(envVars))
	}
	raw, ok := config.Services[serviceName]
	if !ok {
		return portFromEnv(envDefaultsFromDiscovery(envVars))
	}

	var service struct {
		Ports       json.RawMessage `json:"ports"`
		Expose      json.RawMessage `json:"expose"`
		Environment json.RawMessage `json:"environment"`
	}
	if err := json.Unmarshal(raw, &service); err != nil {
		return portFromEnv(envDefaultsFromDiscovery(envVars))
	}

	env := envDefaultsFromDiscovery(envVars)
	mergeComposeEnvironment(env, service.Environment)
	if port := parseComposePorts(service.Ports, env); port > 0 {
		return port
	}
	if port := parseComposeExpose(service.Expose, env); port > 0 {
		return port
	}
	return portFromEnv(env)
}

func envDefaultsFromDiscovery(vars []envdiscover.Var) map[string]string {
	env := make(map[string]string)
	for _, item := range vars {
		if item.DefaultValue == nil {
			continue
		}
		env[strings.ToUpper(strings.TrimSpace(item.Key))] = strings.TrimSpace(*item.DefaultValue)
	}
	return env
}

func portFromEnv(env map[string]string) int32 {
	for _, key := range []string{"PORT", "APP_PORT", "SERVER_PORT", "LISTEN_PORT"} {
		if port := parsePortToken(env[key], env); port > 0 {
			return port
		}
	}
	return 0
}

func stripDockerfileComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

func mergeDockerfileEnv(env map[string]string, raw string) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return
	}
	if len(fields) == 2 && !strings.Contains(fields[0], "=") {
		env[strings.ToUpper(fields[0])] = trimPortValue(fields[1])
		return
	}
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		env[strings.ToUpper(strings.TrimSpace(key))] = trimPortValue(value)
	}
}

func mergeComposeEnvironment(env map[string]string, raw json.RawMessage) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err == nil {
		for key, value := range asMap {
			if value == nil {
				continue
			}
			env[strings.ToUpper(key)] = trimPortValue(fmt.Sprint(value))
		}
		return
	}

	var asList []string
	if err := json.Unmarshal(raw, &asList); err == nil {
		for _, item := range asList {
			key, value, ok := strings.Cut(item, "=")
			if !ok {
				continue
			}
			env[strings.ToUpper(strings.TrimSpace(key))] = trimPortValue(value)
		}
	}
}

func parseComposePorts(raw json.RawMessage, env map[string]string) int32 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var ports []json.RawMessage
	if err := json.Unmarshal(raw, &ports); err != nil {
		return 0
	}
	for _, item := range ports {
		if port := parseComposePort(item, env); port > 0 {
			return port
		}
	}
	return 0
}

func parseComposePort(raw json.RawMessage, env map[string]string) int32 {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, key := range []string{"target", "container_port"} {
			if value, ok := obj[key]; ok {
				if port := parsePortToken(fmt.Sprint(value), env); port > 0 {
					return port
				}
			}
		}
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return 0
	}
	parts := strings.Split(asString, ":")
	return parsePortToken(parts[len(parts)-1], env)
}

func parseComposeExpose(raw json.RawMessage, env map[string]string) int32 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		var value string
		if err := json.Unmarshal(raw, &value); err == nil {
			return parsePortToken(value, env)
		}
		return 0
	}
	for _, item := range values {
		if port := parseComposeExposeValue(item, env); port > 0 {
			return port
		}
	}
	return 0
}

func parseComposeExposeValue(raw json.RawMessage, env map[string]string) int32 {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return parsePortToken(value, env)
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return parsePortToken(strconv.Itoa(number), env)
	}
	return 0
}

func parsePortToken(token string, env map[string]string) int32 {
	value := trimPortValue(token)
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		key := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		value = env[strings.ToUpper(key)]
	} else if strings.HasPrefix(value, "$") {
		value = env[strings.ToUpper(strings.TrimPrefix(value, "$"))]
	}
	value = trimPortValue(value)
	if strings.Contains(value, "/") {
		value = strings.SplitN(value, "/", 2)[0]
	}
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return int32(port)
}

func trimPortValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
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
