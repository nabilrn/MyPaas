package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"mypaas/internal/compose"
	"mypaas/internal/db"
	"mypaas/internal/envdiscover"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
	"mypaas/internal/staticdeploy"
)

var lifecycleEnvKeyPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type CreateValidationInput struct {
	Project        CreateInput
	EnvVars        []envvar.Value
	SharedPostgres bool
}

// CreateValidated is the authoritative project creation path used by the HTTP
// API. Repository/config validation happens before CreateProject is called, so
// invalid source, branch, base-directory, runtime, compose, static, and env
// configuration can never leave a project row behind.
func (s *Service) CreateValidated(ctx context.Context, request CreateValidationInput) (db.Project, error) {
	input := request.Project
	sourceType, err := normalizeSourceType(input.SourceType, input.DeployMode)
	if err != nil {
		return db.Project{}, err
	}
	input.SourceType = sourceType

	if err := validateLifecycleEnvVars(request.EnvVars); err != nil {
		return db.Project{}, err
	}

	if sourceType == SourceTypeRegistry {
		imageRef, err := normalizeImageRef(input.ImageRef)
		if err != nil {
			return db.Project{}, err
		}
		input.ImageRef = imageRef
		input.RepoURL = ""
		input.Branch = "main"
		input.DeployMode = "image"
		input.MainService = nil
		input.ComposeFilePath = nil
		input.ComposeOverridePaths = nil
		input.ComposeProfiles = nil
		input.ComposeWorkdir = nil
		input.StaticFrontendPath = nil
		input.BaseDirectory = nil
		return s.Create(ctx, input)
	}

	input.ImageRef = nil
	input.RepoURL = strings.TrimSpace(input.RepoURL)
	if input.RepoURL == "" {
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

	// Auto is resolved from the repository before persistence instead of being
	// silently coerced to dockerfile by the legacy Create method.
	if input.DeployMode == "" || input.DeployMode == "auto" {
		detected, err := s.DetectMode(ctx, DetectInput{
			RepoURL:       input.RepoURL,
			Branch:        input.Branch,
			BaseDirectory: valueOrEmpty(input.BaseDirectory),
		})
		if err != nil {
			return db.Project{}, err
		}
		input.DeployMode = detected.DeployMode
		if input.MainService == nil && detected.MainService != nil {
			input.MainService = detected.MainService
		}
		if input.AppPort <= 0 && detected.AppPort > 0 {
			input.AppPort = detected.AppPort
		}
		if input.ComposeFilePath == nil && detected.ComposeFile != nil {
			input.ComposeFilePath = detected.ComposeFile
		}
	}

	if input.DeployMode != "dockerfile" && input.DeployMode != "compose" && input.DeployMode != "static" {
		return db.Project{}, fmt.Errorf("%w: deploy mode must be dockerfile, compose, or static", errs.ErrValidation)
	}
	if input.DeployMode == "static" && input.AppPort <= 0 {
		input.AppPort = 80
	}
	if input.AppPort <= 0 || input.AppPort > 65535 {
		return db.Project{}, fmt.Errorf("%w: app port must be between 1 and 65535", errs.ErrValidation)
	}
	if input.DeployMode == "compose" && strings.TrimSpace(valueOrEmpty(input.MainService)) == "" {
		return db.Project{}, fmt.Errorf("%w: main service is required for compose projects", errs.ErrValidation)
	}
	if err := validateComposeConfigInput(input.DeployMode, input.ComposeFilePath, input.ComposeOverridePaths, input.ComposeWorkdir); err != nil {
		return db.Project{}, err
	}

	preflightCtx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	if err := preflightProjectWorkspace(preflightCtx, &input, request.EnvVars, request.SharedPostgres); err != nil {
		return db.Project{}, err
	}

	return s.Create(ctx, input)
}

// UpdateValidated keeps the project name immutable. Existing runtime resources
// are keyed by the creation-time project name, so allowing a rename would make
// stop/start/restart/log/cleanup target a different Docker resource. Keeping the
// identity immutable is the backwards-compatible fix until a persisted runtime
// identifier is introduced in a schema migration.
func (s *Service) UpdateValidated(ctx context.Context, input UpdateInput) (db.Project, error) {
	existing, err := s.Get(ctx, input.ID)
	if err != nil {
		return db.Project{}, err
	}
	requestedName := normalizeName(input.Name)
	if requestedName != "" && requestedName != existing.Name {
		return db.Project{}, fmt.Errorf("%w: project name cannot be changed after creation", errs.ErrValidation)
	}
	input.Name = existing.Name
	return s.Update(ctx, input)
}

// DetectModeValidated fixes inspect-only repository previews so baseDirectory
// scopes the tree just like full runtime detection does.
func (s *Service) DetectModeValidated(ctx context.Context, input DetectInput) (DetectResult, error) {
	if !input.InspectOnly {
		return s.DetectMode(ctx, input)
	}

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

	tree, truncated, err := inspectRepositoryTreeScoped(ctx, repoURL, branch, input.BaseDirectory)
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

func preflightProjectWorkspace(ctx context.Context, input *CreateInput, envVars []envvar.Value, sharedPostgres bool) error {
	root, err := os.MkdirTemp("", "mypaas-create-preflight-*")
	if err != nil {
		return fmt.Errorf("create project preflight workspace: %w", err)
	}
	defer os.RemoveAll(root)

	if err := cloneForDetect(ctx, root, input.RepoURL, input.Branch); err != nil {
		return err
	}
	workspace, err := resolveLifecycleDirectory(root, valueOrEmpty(input.BaseDirectory), "base directory")
	if err != nil {
		return err
	}

	if path := strings.TrimSpace(valueOrEmpty(input.StaticFrontendPath)); path != "" {
		staticWorkspace, err := resolveLifecycleDirectory(workspace, path, "static frontend path")
		if err != nil {
			return err
		}
		if err := validateStaticWorkspace(staticWorkspace); err != nil {
			return fmt.Errorf("%w: static frontend path %q is not deployable: %v", errs.ErrValidation, path, err)
		}
	}

	switch input.DeployMode {
	case "dockerfile":
		info, err := os.Stat(filepath.Join(workspace, "Dockerfile"))
		if err != nil || info.IsDir() {
			return fmt.Errorf("%w: Dockerfile was not found in the selected base directory", errs.ErrValidation)
		}
	case "static":
		if err := validateStaticWorkspace(workspace); err != nil {
			return fmt.Errorf("%w: selected base directory is not a deployable static site", errs.ErrValidation)
		}
	case "compose":
		if err := preflightComposeWorkspace(ctx, workspace, input, envVars, sharedPostgres); err != nil {
			return err
		}
	}
	return nil
}

func resolveLifecycleDirectory(root, value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return root, nil
	}
	if err := compose.ValidateUserPath(value); err != nil {
		return "", fmt.Errorf("%w: invalid %s: %v", errs.ErrValidation, label, err)
	}
	candidate := filepath.Join(root, filepath.FromSlash(value))
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s %q does not exist or is not a directory", errs.ErrValidation, label, value)
	}
	return candidate, nil
}

func validateStaticWorkspace(workspace string) error {
	if _, _, err := staticdeploy.FindSiteRoot(workspace); err == nil {
		return nil
	}
	if isStaticSPA(workspace) {
		return nil
	}
	return errs.ErrNoDeployConfig
}

func preflightComposeWorkspace(ctx context.Context, workspace string, input *CreateInput, envVars []envvar.Value, sharedPostgres bool) error {
	layout, err := compose.ResolveLayout(
		workspace,
		strings.TrimSpace(valueOrEmpty(input.ComposeFilePath)),
		input.ComposeOverridePaths,
		strings.TrimSpace(valueOrEmpty(input.ComposeWorkdir)),
		"docker-compose.mypaas.preflight.override.yml",
		"docker-compose.mypaas.preflight.sanitized.json",
		filepath.Join(workspace, ".mypaas-preflight.env"),
	)
	if err != nil {
		return err
	}
	if input.ComposeFilePath == nil || strings.TrimSpace(*input.ComposeFilePath) == "" {
		primary := layout.PrimaryRel
		input.ComposeFilePath = &primary
	}

	defaults, _ := envdiscover.Discover(workspace, layout.PrimaryRel)
	defaultKeys := make(map[string]struct{})
	previewValues := make(map[string]string)
	for _, item := range defaults {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		if item.DefaultValue != nil {
			defaultKeys[key] = struct{}{}
			previewValues[key] = *item.DefaultValue
		}
	}

	provided := make(map[string]struct{})
	for _, item := range envVars {
		key := strings.ToUpper(strings.TrimSpace(item.Key))
		if key == "" || strings.TrimSpace(item.Value) == "" {
			continue
		}
		provided[key] = struct{}{}
		previewValues[key] = "mypaas_preflight"
	}
	if sharedPostgres {
		provided["DATABASE_URL"] = struct{}{}
		previewValues["DATABASE_URL"] = "postgres://mypaas:mypaas@postgres:5432/mypaas"
	}

	required := make(map[string]struct{})
	for _, file := range layout.UserFiles {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			return fmt.Errorf("%w: could not read compose file %q", errs.ErrValidation, file)
		}
		for _, match := range composeEnvMatches(string(content)) {
			key := strings.TrimSpace(match.key)
			if key == "" {
				continue
			}
			if _, ok := previewValues[key]; !ok {
				previewValues[key] = "mypaas_preflight"
			}
			if match.operator == "-" || match.operator == ":-" {
				continue
			}
			if _, ok := defaultKeys[key]; ok {
				continue
			}
			required[key] = struct{}{}
		}
	}

	missing := make([]string, 0)
	for key := range required {
		if _, ok := provided[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%w: missing required compose environment values: %s", errs.ErrValidation, strings.Join(missing, ", "))
	}

	if err := envvar.WriteEnvFile(layout.EnvFile, previewValues); err != nil {
		return fmt.Errorf("write compose preflight env: %w", err)
	}
	rootEnvPath := filepath.Join(workspace, ".env")
	if rootEnvPath != layout.EnvFile && !pathExists(rootEnvPath) {
		if err := envvar.WriteEnvFile(rootEnvPath, previewValues); err != nil {
			return fmt.Errorf("write compose preflight root env: %w", err)
		}
	}
	rawConfig, err := composeConfigJSONMulti(ctx, layout)
	if err != nil {
		return err
	}
	var doc composeConfigDoc
	if err := json.Unmarshal(rawConfig, &doc); err != nil {
		return fmt.Errorf("%w: parse compose preflight config: %v", errs.ErrValidation, err)
	}
	main := strings.TrimSpace(valueOrEmpty(input.MainService))
	if _, ok := doc.Services[main]; !ok {
		return fmt.Errorf("%w: compose main service %q was not found", errs.ErrValidation, main)
	}

	composeDir := filepath.Dir(layout.UserFiles[0])
	plan := &ComposePlan{
		RecommendedMainService: main,
		RecommendedAppPort:     input.AppPort,
		RouteTarget:            fmt.Sprintf("%s:%d", main, input.AppPort),
		RequiredEnvVars:        sortedStringSet(required),
		Services:               make([]ComposeServicePlan, 0, len(doc.Services)),
		Issues:                 make([]ComposeIssue, 0),
	}
	serviceNames := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)
	for _, name := range serviceNames {
		spec := doc.Services[name]
		item := composeServicePlanFromConfig(composeDir, name, spec)
		if name == main {
			item.Role = "public"
		} else {
			item.Role = "internal"
		}
		plan.Services = append(plan.Services, item)
		addComposeServiceIssues(plan, composeDir, name, item, spec)
	}
	addComposePlanIssues(plan, doc, main, input.AppPort)
	for _, issue := range plan.Issues {
		if issue.Severity == "error" {
			return fmt.Errorf("%w: compose preflight failed: %s", errs.ErrValidation, issue.Message)
		}
	}
	return nil
}

func composeConfigJSONMulti(ctx context.Context, layout *compose.Layout) ([]byte, error) {
	args := []string{"compose", "--env-file", layout.EnvFile}
	for _, file := range layout.UserFiles {
		args = append(args, "-f", file)
	}
	args = append(args, "config", "--format", "json")
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = layout.WorkDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: compose config could not be validated: %s", errs.ErrValidation, firstNonEmptyLine(string(out)))
	}
	return out, nil
}

func validateLifecycleEnvVars(values []envvar.Value) error {
	for _, item := range values {
		key := strings.ToUpper(strings.TrimSpace(item.Key))
		if !lifecycleEnvKeyPattern.MatchString(key) {
			return fmt.Errorf("%w: invalid env var key %q", errs.ErrValidation, item.Key)
		}
	}
	return nil
}

func sortedStringSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func inspectRepositoryTreeScoped(ctx context.Context, repoURL, branch, baseDirectory string) ([]RepoTreeEntry, bool, error) {
	workspace, err := os.MkdirTemp("", "mypaas-repo-preview-*")
	if err != nil {
		return nil, false, fmt.Errorf("create repository preview workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	if err := cloneForTreePreview(ctx, workspace, repoURL, branch); err != nil {
		return nil, false, err
	}
	base := strings.TrimSpace(baseDirectory)
	if base == "" || base == "." {
		return listGitRepositoryTree(ctx, workspace, maxRepoTreeEntries)
	}
	if err := compose.ValidateUserPath(base); err != nil {
		return nil, false, err
	}
	base = filepath.ToSlash(filepath.Clean(filepath.FromSlash(base)))
	typeCmd := exec.CommandContext(ctx, "git", "cat-file", "-t", "HEAD:"+base)
	typeCmd.Dir = workspace
	kind, err := typeCmd.Output()
	if err != nil || strings.TrimSpace(string(kind)) != "tree" {
		return nil, false, fmt.Errorf("%w: base directory %q was not found in branch %q", errs.ErrValidation, base, branch)
	}

	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "-t", "--full-tree", "HEAD", "--", base)
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, false, fmt.Errorf("%w: failed to read repository tree: %s", errs.ErrValidation, firstNonEmptyLine(string(out)))
	}
	prefix := strings.TrimSuffix(base, "/") + "/"
	entries := make([]RepoTreeEntry, 0)
	truncated := false
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if len(entries) >= maxRepoTreeEntries {
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
		rel = strings.TrimSpace(rel)
		if !strings.HasPrefix(rel, prefix) {
			continue
		}
		rel = strings.TrimPrefix(rel, prefix)
		if rel == "" {
			continue
		}
		entryType := "file"
		if fields[1] == "tree" {
			entryType = "directory"
		} else if fields[1] != "blob" {
			continue
		}
		entries = append(entries, RepoTreeEntry{
			Name:  filepath.Base(rel),
			Path:  filepath.ToSlash(rel),
			Type:  entryType,
			Depth: strings.Count(filepath.ToSlash(rel), "/"),
		})
	}
	return entries, truncated, nil
}
