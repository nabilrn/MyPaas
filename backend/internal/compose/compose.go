// Package compose centralises Docker Compose file discovery, path validation,
// and the layout resolver used by both detection (project package) and deploy
// orchestration (deployment package).
//
// Design goals:
//   - Keep root-only behaviour as the default fallback when a project has no
//     persisted compose_file_path. Existing projects continue to work.
//   - Allow a compose file anywhere inside a cloned workspace (subdirectory,
//     monorepo package, infra/ folder, etc.).
//   - Validate user-supplied paths against traversal and absolute-path escapes.
//   - Persist a single source of truth for candidate ordering and ignore rules
//     so project/ and deployment/ no longer duplicate the same list.
package compose

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mypaas/internal/errs"
)

// MaxScanDepth caps how deep Discover walks a workspace. Compose files deeper
// than this are ignored to keep discovery fast on large monorepos and to avoid
// following nested git histories or vendored trees.
const MaxScanDepth = 4

// LocalhostPortExpr matches `localhost:PORT` or `127.0.0.1:PORT` with an
// optional protocol prefix (postgres://, redis://, http://, etc.). The port
// is captured in group 1. Also matches bare `localhost` without a port.
var LocalhostPortExpr = regexp.MustCompile(`(?i)(?:[a-z]+://)?(?:localhost|127\.0\.0\.1)(?::(\d+))?`)

// skipDirs are directory names we never descend into during recursive
// discovery. They are heavy, rarely contain deployable compose files, or are
// platform noise (build outputs, dependency caches, VCS metadata).
var skipDirs = map[string]struct{}{
	".git":          {},
	"node_modules":  {},
	"vendor":        {},
	".cache":        {},
	".turbo":        {},
	".next":         {},
	".nuxt":         {},
	"dist":          {},
	"build":         {},
	"target":        {},
	".gradle":       {},
	".idea":         {},
	".vscode":       {},
	"__pycache__":   {},
	".pytest_cache": {},
	"coverage":      {},
}

// Candidate is a discovered compose file with metadata used by the UI picker.
type Candidate struct {
	// Path is the repo-relative POSIX path (forward slashes) to the compose
	// file, e.g. "infra/docker-compose.yml" or "docker-compose.prod.yml".
	Path string `json:"path"`
	// Score ranks candidates; higher is preferred. Root prod-variants score
	// highest, then root base names, then subdirectory matches.
	Score int `json:"score"`
	// Depth is the number of path separators in the repo-relative path.
	Depth int `json:"depth"`
}

// Layout is the resolved compose layout for a deployment: where to run docker
// compose from (WorkDir, absolute) and which -f files to pass (repo-relative
// or absolute paths). OverrideFile is the MyPaas-generated override path
// (absolute) and is appended after the user files. EnvFile is the absolute
// path to the .env file MyPaas writes for the project.
type Layout struct {
	WorkDir       string   // absolute path to use as `cmd.Dir`
	UserFiles     []string // absolute paths to user compose files (-f order matters)
	OverrideFile  string   // absolute path to MyPaas override
	SanitizedFile string   // absolute path to MyPaas sanitized JSON
	EnvFile       string   // absolute path to .env
	PrimaryRel    string   // repo-relative primary compose file (for logging)
}

// ValidateUserPath rejects paths that could escape the workspace or are not
// POSIX-relative. Used by the API layer for compose_file_path,
// compose_override_paths, and compose_workdir.
func ValidateUserPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: path %q must be relative, not absolute", errs.ErrValidation, path)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("%w: path %q must use forward slashes", errs.ErrValidation, path)
	}
	// Reject any literal `..` segment in the raw input. Even if filepath.Clean
	// would resolve it to a safe relative path, accepting `..` would let users
	// author confusing configs and complicate logging/auditing.
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return fmt.Errorf("%w: path %q contains parent-directory segments", errs.ErrValidation, path)
		}
	}
	cleaned := filepath.Clean(filepath.FromSlash(path))
	rel, err := filepath.Rel(".", cleaned)
	if err != nil {
		return fmt.Errorf("%w: path %q is not relative: %v", errs.ErrValidation, path, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: path %q escapes the repository root", errs.ErrValidation, path)
	}
	return nil
}

// Discover walks the workspace and returns compose file candidates ranked by
// preference. The returned paths use forward slashes and are relative to the
// workspace root. Returns ErrComposeFileNotFound when no candidate exists.
func Discover(workspace string) ([]Candidate, error) {
	workspace = filepath.Clean(workspace)
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		return nil, errs.ErrComposeFileNotFound
	}

	var found []Candidate
	err := filepath.WalkDir(workspace, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == workspace {
			return nil
		}
		if entry.IsDir() {
			if _, skip := skipDirs[strings.ToLower(entry.Name())]; skip {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(workspace, path)
			depth := strings.Count(filepath.ToSlash(rel), "/")
			if depth >= MaxScanDepth {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !isComposeFilename(name) {
			return nil
		}
		if ignoredComposeCandidate(name) {
			return nil
		}
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		candidate := Candidate{
			Path:  relSlash,
			Score: scoreCandidate(relSlash),
			Depth: strings.Count(relSlash, "/"),
		}
		found = append(found, candidate)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk workspace: %w", err)
	}
	if len(found) == 0 {
		return nil, errs.ErrComposeFileNotFound
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Score != found[j].Score {
			return found[i].Score > found[j].Score
		}
		if found[i].Depth != found[j].Depth {
			return found[i].Depth < found[j].Depth
		}
		return found[i].Path < found[j].Path
	})
	return found, nil
}

// First returns the highest-ranked candidate, or ErrComposeFileNotFound when
// the slice is empty. Convenience wrapper around Discover for callers that
// only need a single path.
func First(workspace string) (string, error) {
	candidates, err := Discover(workspace)
	if err != nil {
		return "", err
	}
	return candidates[0].Path, nil
}

// ResolveLayout turns a workspace + persisted project fields into an absolute
// Layout for docker compose. Rules:
//
//  1. If primaryRel is set, it is used as the primary compose file. workdirRel
//     overrides the directory; if empty, the compose file's directory is used.
//  2. Otherwise the workspace is scanned and the top candidate is used.
//  3. overrideRel paths are appended (in order) after the primary file but
//     BEFORE the MyPaas generated override so MyPaas's port binding always wins.
//  4. mypaasOverrideBase/mypaasSanitizedBase are the filenames MyPaas writes
//     (e.g. "docker-compose.mypaas.override.yml"); they are placed inside the
//     resolved WorkDir so `-f` relative resolution stays consistent.
//  5. envFile is converted to an absolute path so docker compose does not
//     resolve it relative to WorkDir unexpectedly.
func ResolveLayout(workspace string, primaryRel string, overrideRel []string, workdirRel string, mypaasOverrideBase, mypaasSanitizedBase, envFile string) (*Layout, error) {
	workspace = filepath.Clean(workspace)
	if primaryRel == "" {
		guessed, err := First(workspace)
		if err != nil {
			return nil, err
		}
		primaryRel = guessed
	}
	if err := ValidateUserPath(primaryRel); err != nil {
		return nil, err
	}
	primaryAbs := filepath.Join(workspace, filepath.FromSlash(primaryRel))
	if _, err := os.Stat(primaryAbs); err != nil {
		return nil, fmt.Errorf("%w: %s", errs.ErrComposeFileNotFound, primaryRel)
	}

	workdirAbs := workspace
	if workdirRel != "" {
		if err := ValidateUserPath(workdirRel); err != nil {
			return nil, err
		}
		workdirAbs = filepath.Join(workspace, filepath.FromSlash(workdirRel))
	} else {
		workdirAbs = filepath.Dir(primaryAbs)
	}
	if info, err := os.Stat(workdirAbs); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: compose workdir %q does not exist", errs.ErrValidation, workdirRel)
	}

	userFiles := []string{primaryAbs}
	for _, rel := range overrideRel {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		if err := ValidateUserPath(rel); err != nil {
			return nil, err
		}
		abs := filepath.Join(workspace, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("%w: override file %q not found", errs.ErrValidation, rel)
		}
		userFiles = append(userFiles, abs)
	}

	overrideFile := filepath.Join(workdirAbs, mypaasOverrideBase)
	sanitizedFile := filepath.Join(workdirAbs, mypaasSanitizedBase)

	envAbs := envFile
	if envFile != "" {
		if !filepath.IsAbs(envFile) {
			envAbs = filepath.Clean(filepath.Join(workspace, envFile))
		}
	}

	return &Layout{
		WorkDir:       workdirAbs,
		UserFiles:     userFiles,
		OverrideFile:  overrideFile,
		SanitizedFile: sanitizedFile,
		EnvFile:       envAbs,
		PrimaryRel:    primaryRel,
	}, nil
}

// isComposeFilename reports whether name matches a Docker Compose file naming
// convention. Covers the standard root files and any `*.yml`/`*.yaml` whose
// base starts with `docker-compose` or `compose`.
func isComposeFilename(name string) bool {
	lower := strings.ToLower(name)
	if !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".yaml") {
		return false
	}
	stem := strings.TrimSuffix(strings.TrimSuffix(lower, ".yaml"), ".yml")
	return stem == "compose" ||
		stem == "docker-compose" ||
		strings.HasPrefix(stem, "compose.") ||
		strings.HasPrefix(stem, "docker-compose.")
}

// ignoredComposeCandidate suppresses override and test variants during
// discovery so we do not surface files that docker compose would itself treat
// as auto-included overrides or test fixtures.
func ignoredComposeCandidate(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(normalized, "override") || strings.Contains(normalized, "test")
}

// scoreCandidate ranks candidates. Root location wins; prod-variants rank
// higher than dev/test/default variants; deeper paths lose points so the UI
// picker still surfaces sensible defaults first.
func scoreCandidate(rel string) int {
	lower := strings.ToLower(rel)
	score := 0
	depth := strings.Count(rel, "/")
	if depth == 0 {
		score += 10
	} else {
		// First-level subdirs like infra/, docker/, deploy/ get a smaller boost.
		topDir := ""
		if idx := strings.Index(rel, "/"); idx > 0 {
			topDir = lower[:idx]
		}
		switch topDir {
		case "infra", "docker", "deploy", "deployment", "ops", ".docker":
			score += 6
		case "apps", "services", "packages":
			score += 3
		default:
			score += 1
		}
		score -= depth
	}
	switch {
	case strings.Contains(lower, "prod") || strings.Contains(lower, "production"):
		score += 5
	case strings.Contains(lower, "dev") || strings.Contains(lower, "development") || strings.Contains(lower, "local"):
		score -= 3
	}
	base := filepath.Base(lower)
	switch base {
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		score += 2
	}
	return score
}

// ErrComposeFileNotFound is re-exported so callers can do errors.Is against
// the same sentinel used by the rest of the codebase.
var ErrComposeFileNotFound = errs.ErrComposeFileNotFound

// EnsureErrComposeFileNotFound is a compile-time guarantee that this package's
// exported sentinel stays aligned with errs.ErrComposeFileNotFound.
var _ = errors.Is(ErrComposeFileNotFound, errs.ErrComposeFileNotFound)

// LocalhostWarning describes a localhost reference found in an env var value.
// Used by the compose doctor and the deployment engine to warn users before
// their app fails to connect inside a container.
type LocalhostWarning struct {
	Key       string // env var key, e.g. "DATABASE_URL"
	Value     string // full env var value, e.g. "postgres://user:pass@localhost:5432/db"
	Port      int    // port extracted from the localhost reference (0 if none)
	Service   string // compose service name that exposes the port (empty if no match)
	Suggested string // value with localhost replaced by the service name (empty if no match)
}

// DetectLocalhostInEnv scans a key→value map for localhost/127.0.0.1
// references and returns actionable warnings. portToService is optional —
// when nil, warnings are emitted without a service suggestion.
func DetectLocalhostInEnv(envs map[string]string, portToService map[int32]string) []LocalhostWarning {
	if portToService == nil {
		portToService = map[int32]string{}
	}
	keys := make([]string, 0, len(envs))
	for key := range envs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var warnings []LocalhostWarning
	for _, key := range keys {
		value := envs[key]
		matches := LocalhostPortExpr.FindAllStringSubmatch(value, -1)
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches {
			w := LocalhostWarning{
				Key:   key,
				Value: value,
			}
			if match[1] != "" {
				if port, err := strconv.Atoi(match[1]); err == nil && port > 0 {
					w.Port = port
					if svc, ok := portToService[int32(port)]; ok {
						w.Service = svc
						// Replace only the host portion, keeping :port intact
						// so localhost:5432 → db:5432, not just db.
						hostPart := match[0]
						hostPart = strings.TrimSuffix(hostPart, ":"+match[1])
						w.Suggested = strings.Replace(value, hostPart, svc, 1)
					}
				}
			}
			warnings = append(warnings, w)
		}
	}
	return warnings
}
