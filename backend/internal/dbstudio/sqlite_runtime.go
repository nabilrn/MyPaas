package dbstudio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"sort"
	"strings"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

type sqliteRuntimeInspect struct {
	Config struct {
		WorkingDir string `json:"WorkingDir"`
	} `json:"Config"`
	Mounts []struct {
		Type        string `json:"Type"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
}

type sqliteRuntimeClient struct {
	container    string
	databasePath string
}

type sqliteRuntimeCandidate struct {
	container string
	path      string
}

type sqliteHelperEnvelope struct {
	OK       bool                 `json:"ok"`
	Response SQLiteHelperResponse `json:"response"`
	Error    string               `json:"error"`
}

func resolveSQLiteConnection(ctx context.Context, project db.Project, envs map[string]string) (Connection, bool, error) {
	rawPath, source, ok := sqlitePathFromEnv(envs)
	if !ok {
		return Connection{}, false, nil
	}
	return resolveSQLiteConfiguredPath(ctx, project, rawPath, source)
}

func resolveSQLiteConfiguredPath(ctx context.Context, project db.Project, rawPath, source string) (Connection, bool, error) {
	if project.DeployMode == "static" {
		return Connection{}, true, fmt.Errorf("%w: SQLite DB Studio requires a container-backed project", errs.ErrValidation)
	}

	containers, err := sqliteRuntimeCandidates(ctx, project)
	if err != nil {
		return Connection{}, true, err
	}
	if len(containers) == 0 {
		return Connection{}, true, fmt.Errorf("%w: SQLite runtime container is not available yet", errs.ErrValidation)
	}

	var lastReason string
	for _, container := range containers {
		inspect, err := inspectSQLiteRuntime(ctx, container)
		if err != nil {
			lastReason = err.Error()
			continue
		}
		resolved, err := resolveSQLiteContainerPath(rawPath, inspect.Config.WorkingDir)
		if err != nil {
			lastReason = err.Error()
			continue
		}
		if !sqlitePathOnPersistentMount(resolved, inspect) {
			lastReason = "SQLite database is not stored on a persistent runtime mount"
			continue
		}
		return Connection{
			Driver: DriverSQLite, Database: resolved, Source: source,
			runtimeContainer: container, databasePath: resolved,
		}, true, nil
	}
	if lastReason == "" {
		lastReason = "SQLite database is not available from the project runtime"
	}
	return Connection{}, true, fmt.Errorf("%w: %s", errs.ErrValidation, lastReason)
}

func sqlitePathFromEnv(envs map[string]string) (string, string, bool) {
	if value := strings.TrimSpace(envs["DATABASE_URL"]); value != "" {
		if path, ok := sqlitePathFromURL(value); ok {
			return path, "DATABASE_URL", true
		}
		if _, err := connectionFromURL(value, "DATABASE_URL"); err == nil {
			return "", "", false
		}
	}
	if value := strings.TrimSpace(envs["SQLITE_URL"]); value != "" {
		if path, ok := sqlitePathFromURL(value); ok {
			return path, "SQLITE_URL", true
		}
	}
	for _, key := range []string{"SQLITE_PATH", "SQLITE_DATABASE", "SQLITE_DB"} {
		if value := strings.TrimSpace(envs[key]); value != "" {
			return value, key, true
		}
	}
	hint := strings.ToLower(firstEnv(envs, "DB_CONNECTION", "DB_DRIVER", "DATABASE_CLIENT"))
	if strings.Contains(hint, "sqlite") {
		if value := strings.TrimSpace(envs["DATABASE_PATH"]); value != "" {
			return value, "DATABASE_PATH", true
		}
		if value := firstEnv(envs, "DB_DATABASE", "DB_NAME", "DATABASE_NAME"); value != "" {
			return value, "env-parts", true
		}
	}
	return "", "", false
}

func resolveSQLiteRuntimeConnection(ctx context.Context, project db.Project) (Connection, bool, error) {
	if project.DeployMode == "static" {
		return Connection{}, false, nil
	}

	containers, err := sqliteRuntimeCandidates(ctx, project)
	if err != nil {
		if isDockerUnavailable(err.Error()) {
			return Connection{}, false, nil
		}
		return Connection{}, true, err
	}

	candidates := make([]sqliteRuntimeCandidate, 0)
	seen := make(map[string]struct{})
	for _, container := range containers {
		inspect, err := inspectSQLiteRuntime(ctx, container)
		if err != nil {
			if isDockerUnavailable(err.Error()) || isNoSuchContainer(err.Error()) {
				continue
			}
			return Connection{}, true, err
		}
		roots := persistentSQLiteMountRoots(inspect)
		if len(roots) == 0 {
			continue
		}

		client := &sqliteRuntimeClient{container: container}
		paths, err := client.Discover(ctx, roots)
		if err != nil {
			return Connection{}, true, err
		}
		for _, path := range paths {
			path = pathpkg.Clean(strings.TrimSpace(path))
			if !sqlitePathOnPersistentMount(path, inspect) {
				continue
			}
			key := container + "\x00" + path
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, sqliteRuntimeCandidate{container: container, path: path})
		}
	}

	candidate, ambiguous := selectSQLiteRuntimeCandidate(candidates)
	if ambiguous {
		return Connection{}, true, fmt.Errorf("%w: multiple persistent SQLite databases were found; configure SQLITE_PATH, SQLITE_DATABASE, SQLITE_DB, or DATABASE_URL explicitly", errs.ErrValidation)
	}
	if candidate.path == "" {
		return Connection{}, false, nil
	}
	return Connection{
		Driver: DriverSQLite, Database: candidate.path, Source: "runtime-discovery",
		runtimeContainer: candidate.container, databasePath: candidate.path,
	}, true, nil
}

func selectSQLiteRuntimeCandidate(candidates []sqliteRuntimeCandidate) (sqliteRuntimeCandidate, bool) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].container != candidates[j].container {
			return candidates[i].container < candidates[j].container
		}
		return candidates[i].path < candidates[j].path
	})
	if len(candidates) == 0 {
		return sqliteRuntimeCandidate{}, false
	}
	if len(candidates) > 1 {
		return sqliteRuntimeCandidate{}, true
	}
	return candidates[0], false
}

func sqlitePathFromURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	lowered := strings.ToLower(value)
	for _, prefix := range []string{"file:", "sqlite:", "sqlite3:"} {
		if !strings.HasPrefix(lowered, prefix) {
			continue
		}
		rest := strings.TrimSpace(value[len(prefix):])
		if strings.HasPrefix(rest, "//") {
			rest = strings.TrimPrefix(rest, "//")
			if !strings.HasPrefix(rest, "/") {
				rest = "/" + rest
			}
		}
		if query := strings.IndexByte(rest, '?'); query >= 0 {
			rest = rest[:query]
		}
		if fragment := strings.IndexByte(rest, '#'); fragment >= 0 {
			rest = rest[:fragment]
		}
		rest = strings.TrimSpace(rest)
		return rest, rest != ""
	}
	return "", false
}

func sqliteRuntimeCandidates(ctx context.Context, project db.Project) ([]string, error) {
	if project.DeployMode != "compose" {
		return []string{"mypaas-" + project.Name}, nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	add := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	if project.MainService != nil && strings.TrimSpace(*project.MainService) != "" {
		for _, composeProject := range composeProjectCandidates(project.Name) {
			ids, err := composeServiceContainerIDs(ctx, composeProject, strings.TrimSpace(*project.MainService))
			if err != nil {
				return nil, err
			}
			add(ids...)
		}
	}
	for _, composeProject := range composeProjectCandidates(project.Name) {
		raw, err := sqliteRuntimeCommand(ctx, "ps", "-aq", "--filter", "label=com.docker.compose.project="+composeProject)
		if err != nil {
			return nil, fmt.Errorf("%w: find SQLite Compose containers: %s", errs.ErrValidation, strings.TrimSpace(string(raw)))
		}
		add(strings.Fields(string(raw))...)
	}
	sort.Strings(out)
	return out, nil
}

func inspectSQLiteRuntime(ctx context.Context, container string) (sqliteRuntimeInspect, error) {
	out, err := sqliteRuntimeCommand(ctx, "inspect", container)
	if err != nil {
		return sqliteRuntimeInspect{}, fmt.Errorf("inspect SQLite runtime: %s", strings.TrimSpace(string(out)))
	}
	var rows []sqliteRuntimeInspect
	if err := json.Unmarshal(out, &rows); err != nil || len(rows) == 0 {
		return sqliteRuntimeInspect{}, fmt.Errorf("parse SQLite runtime inspection")
	}
	return rows[0], nil
}

var sqliteRuntimeCommand = func(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

func resolveSQLiteContainerPath(rawPath, workingDir string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" || strings.ContainsRune(rawPath, '\x00') {
		return "", fmt.Errorf("SQLite database path is invalid")
	}
	if pathpkg.IsAbs(rawPath) {
		cleaned := pathpkg.Clean(rawPath)
		if cleaned == "/" {
			return "", fmt.Errorf("SQLite database path cannot be the container root")
		}
		return cleaned, nil
	}
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" || !pathpkg.IsAbs(workingDir) {
		return "", fmt.Errorf("SQLite database uses a relative path but the runtime has no absolute working directory")
	}
	return pathpkg.Clean(pathpkg.Join(workingDir, rawPath)), nil
}

func sqlitePathOnPersistentMount(databasePath string, inspect sqliteRuntimeInspect) bool {
	databasePath = pathpkg.Clean(strings.TrimSpace(databasePath))
	for _, destination := range persistentSQLiteMountRoots(inspect) {
		if databasePath == destination || strings.HasPrefix(databasePath, destination+"/") {
			return true
		}
	}
	return false
}

func persistentSQLiteMountRoots(inspect sqliteRuntimeInspect) []string {
	seen := make(map[string]struct{})
	roots := make([]string, 0)
	for _, mount := range inspect.Mounts {
		mountType := strings.ToLower(strings.TrimSpace(mount.Type))
		if mountType != "bind" && mountType != "volume" {
			continue
		}
		destination := pathpkg.Clean(strings.TrimSpace(mount.Destination))
		if destination == "." || destination == "/" || destination == "" {
			continue
		}
		if _, ok := seen[destination]; ok {
			continue
		}
		seen[destination] = struct{}{}
		roots = append(roots, destination)
	}
	sort.Strings(roots)
	return roots
}

func (c *sqliteRuntimeClient) call(ctx context.Context, request SQLiteHelperRequest) (SQLiteHelperResponse, error) {
	if c.databasePath != "" {
		request.DatabasePath = c.databasePath
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return SQLiteHelperResponse{}, err
	}
	image, err := sqliteHelperImage(ctx)
	if err != nil {
		return SQLiteHelperResponse{}, err
	}
	args := []string{
		"run", "--rm", "--network", "none", "--read-only",
		"--tmpfs", "/tmp:rw,nosuid,nodev,size=16m",
		"--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
		"--volumes-from", c.container,
		"--entrypoint", "/app/mypaas-sqlite-helper", image,
	}
	out, err := sqliteRuntimeHelperCommand(ctx, args, payload)
	if err != nil {
		return SQLiteHelperResponse{}, fmt.Errorf("SQLite helper failed: %s", strings.TrimSpace(string(out)))
	}
	var envelope sqliteHelperEnvelope
	if err := json.Unmarshal(bytes.TrimSpace(out), &envelope); err != nil {
		return SQLiteHelperResponse{}, fmt.Errorf("parse SQLite helper response: %w", err)
	}
	if !envelope.OK {
		if strings.TrimSpace(envelope.Error) == "" {
			envelope.Error = "SQLite helper request failed"
		}
		return SQLiteHelperResponse{}, fmt.Errorf("%s", envelope.Error)
	}
	return envelope.Response, nil
}

var sqliteRuntimeHelperCommand = func(ctx context.Context, args []string, payload []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = bytes.NewReader(append(payload, '\n'))
	return cmd.CombinedOutput()
}

func sqliteHelperImage(ctx context.Context) (string, error) {
	if value := strings.TrimSpace(os.Getenv("MYPAAS_SQLITE_HELPER_IMAGE")); value != "" {
		return value, nil
	}
	out, err := sqliteRuntimeCommand(ctx, "inspect", "--format", "{{.Image}}", "mypaas-api")
	if err != nil {
		return "", fmt.Errorf("resolve SQLite helper image: %s", strings.TrimSpace(string(out)))
	}
	image := strings.TrimSpace(string(out))
	if image == "" {
		return "", fmt.Errorf("resolve SQLite helper image: empty image id")
	}
	return image, nil
}

func (c *sqliteRuntimeClient) Discover(ctx context.Context, roots []string) ([]string, error) {
	response, err := c.call(ctx, SQLiteHelperRequest{Operation: "discover", DiscoveryRoots: roots})
	return response.SQLiteCandidates, err
}

func (c *sqliteRuntimeClient) Ping(ctx context.Context) error {
	_, err := c.call(ctx, SQLiteHelperRequest{Operation: "ping"})
	return err
}

func (c *sqliteRuntimeClient) Schemas(ctx context.Context) ([]Schema, error) {
	response, err := c.call(ctx, SQLiteHelperRequest{Operation: "schemas"})
	return response.Schemas, err
}

func (c *sqliteRuntimeClient) Tables(ctx context.Context, schema string) ([]Table, error) {
	response, err := c.call(ctx, SQLiteHelperRequest{Operation: "tables", Schema: schema})
	return response.Tables, err
}

func (c *sqliteRuntimeClient) Columns(ctx context.Context, schema, table string) ([]Column, error) {
	response, err := c.call(ctx, SQLiteHelperRequest{Operation: "columns", Schema: schema, Table: table})
	return response.Columns, err
}

func (c *sqliteRuntimeClient) TableDetails(ctx context.Context, schema, table string) (TableDetails, error) {
	response, err := c.call(ctx, SQLiteHelperRequest{Operation: "table_details", Schema: schema, Table: table})
	if err != nil {
		return TableDetails{}, err
	}
	if response.TableDetails == nil {
		return TableDetails{}, fmt.Errorf("SQLite helper returned no table details")
	}
	return *response.TableDetails, nil
}

func (c *sqliteRuntimeClient) Rows(ctx context.Context, query RowQuery) (RowPage, error) {
	response, err := c.call(ctx, SQLiteHelperRequest{Operation: "rows", Query: &query})
	if err != nil {
		return RowPage{}, err
	}
	if response.Rows == nil {
		return RowPage{}, fmt.Errorf("SQLite helper returned no rows")
	}
	return *response.Rows, nil
}

func (c *sqliteRuntimeClient) Insert(ctx context.Context, mutation Mutation) error {
	_, err := c.call(ctx, SQLiteHelperRequest{Operation: "insert", Mutation: &mutation})
	return err
}

func (c *sqliteRuntimeClient) Update(ctx context.Context, mutation Mutation) error {
	_, err := c.call(ctx, SQLiteHelperRequest{Operation: "update", Mutation: &mutation})
	return err
}

func (c *sqliteRuntimeClient) Delete(ctx context.Context, mutation Mutation) error {
	_, err := c.call(ctx, SQLiteHelperRequest{Operation: "delete", Mutation: &mutation})
	return err
}
