package dbstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mypaas/internal/db"
)

func TestSQLitePathFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		envs   map[string]string
		path   string
		source string
		ok     bool
	}{
		{name: "prisma file url", envs: map[string]string{"DATABASE_URL": "file:/data/app.db"}, path: "/data/app.db", source: "DATABASE_URL", ok: true},
		{name: "sqlite url", envs: map[string]string{"DATABASE_URL": "sqlite:///var/lib/app/app.sqlite3"}, path: "/var/lib/app/app.sqlite3", source: "DATABASE_URL", ok: true},
		{name: "explicit path", envs: map[string]string{"SQLITE_PATH": "/data/app.db"}, path: "/data/app.db", source: "SQLITE_PATH", ok: true},
		{name: "driver parts", envs: map[string]string{"DB_CONNECTION": "sqlite", "DB_DATABASE": "./data/app.db"}, path: "./data/app.db", source: "env-parts", ok: true},
		{name: "driver generic path", envs: map[string]string{"DB_CONNECTION": "sqlite", "DATABASE_PATH": "/data/app.db"}, path: "/data/app.db", source: "DATABASE_PATH", ok: true},
		{name: "generic path without sqlite hint ignored", envs: map[string]string{"DATABASE_PATH": "/data/app.db"}, ok: false},
		{name: "postgres ignored", envs: map[string]string{"DATABASE_URL": "postgres://db/app"}, ok: false},
		{name: "server url wins over sqlite path", envs: map[string]string{"DATABASE_URL": "postgres://db/app", "SQLITE_PATH": "/data/app.db"}, ok: false},
		{name: "postgres hint does not claim generic path", envs: map[string]string{"DB_CONNECTION": "postgres", "DATABASE_PATH": "/data/app.db"}, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, source, ok := sqlitePathFromEnv(tt.envs)
			if ok != tt.ok || path != tt.path || source != tt.source {
				t.Fatalf("sqlitePathFromEnv() = %q, %q, %v; want %q, %q, %v", path, source, ok, tt.path, tt.source, tt.ok)
			}
		})
	}
}

func TestResolveSQLiteContainerPath(t *testing.T) {
	path, err := resolveSQLiteContainerPath("./data/app.db", "/app")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/app/data/app.db" {
		t.Fatalf("path = %q, want /app/data/app.db", path)
	}
	if _, err := resolveSQLiteContainerPath("relative.db", ""); err == nil {
		t.Fatal("expected relative path without working directory to fail")
	}
}

func TestSQLitePathOnPersistentMount(t *testing.T) {
	inspect := sqliteRuntimeInspect{}
	inspect.Mounts = append(inspect.Mounts, struct {
		Type        string `json:"Type"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	}{Type: "volume", Destination: "/app/data", RW: true})
	inspect.Mounts = append(inspect.Mounts, struct {
		Type        string `json:"Type"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	}{Type: "tmpfs", Destination: "/app/cache", RW: true})
	if !sqlitePathOnPersistentMount("/app/data/app.db", inspect) {
		t.Fatal("expected database inside persistent volume")
	}
	if sqlitePathOnPersistentMount("/app/cache/app.db", inspect) {
		t.Fatal("did not expect tmpfs database to be accepted as persistent")
	}
	if sqlitePathOnPersistentMount("/app/app.db", inspect) {
		t.Fatal("did not expect writable-layer database to be accepted")
	}
	if sqlitePathOnPersistentMount("/app/data-other/app.db", inspect) {
		t.Fatal("did not expect a sibling path outside the persistent mount to be accepted")
	}
}

func TestDiscoverSQLiteFilesFindsWagoShapedDatabaseGenerically(t *testing.T) {
	ctx := context.Background()
	dataRoot := t.TempDir()
	databasePath := filepath.Join(dataRoot, "wago.db")
	if err := os.WriteFile(databasePath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	conn, err := openSQLiteDatabase(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	candidates, err := discoverSQLiteFiles(ctx, []string{dataRoot})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(databasePath)
	if len(candidates) != 1 || candidates[0] != want {
		t.Fatalf("candidates = %#v, want [%q]", candidates, want)
	}
}

func TestDiscoverSQLiteFilesRejectsNonSQLiteAndSymlinkEscape(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "not-a-database.db"), []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(t.TempDir(), "outside.db")
	if err := os.WriteFile(outside, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape.db")
	if err := os.Symlink(outside, link); err == nil {
		candidates, err := discoverSQLiteFiles(ctx, []string{root})
		if err != nil {
			t.Fatal(err)
		}
		if len(candidates) != 0 {
			t.Fatalf("symlink escape was discovered: %#v", candidates)
		}
	} else {
		t.Logf("symlink test unavailable on this host: %v", err)
	}

	if _, err := discoverSQLiteFiles(ctx, []string{"relative/data"}); err == nil {
		t.Fatal("expected relative discovery root to fail")
	}
	if _, err := discoverSQLiteFiles(ctx, []string{"/"}); err == nil {
		t.Fatal("expected filesystem root discovery to fail")
	}
}

func TestDiscoverSQLiteFilesHonorsDepthAndSizeBounds(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	deepRoot := filepath.Join(root, "one", "two", "three", "four")
	if err := os.MkdirAll(deepRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	deepPath := filepath.Join(deepRoot, "deep.db")
	if err := os.WriteFile(deepPath, append([]byte("SQLite format 3\x00"), make([]byte, 84)...), 0o600); err != nil {
		t.Fatal(err)
	}

	largePath := filepath.Join(root, "large.db")
	file, err := os.Create(largePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("SQLite format 3\x00")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Truncate(maxSQLiteCandidateBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	candidates, err := discoverSQLiteFiles(ctx, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("bounded discovery returned out-of-bound candidates: %#v", candidates)
	}
}

func TestSelectSQLiteRuntimeCandidateRejectsAmbiguity(t *testing.T) {
	candidates := []sqliteRuntimeCandidate{
		{container: "container-b", path: "/app/data/second.db"},
		{container: "container-a", path: "/app/data/first.db"},
	}
	selected, ambiguous := selectSQLiteRuntimeCandidate(candidates)
	if !ambiguous || selected.path != "" {
		t.Fatalf("selection = %#v, ambiguous = %v; want safe ambiguity", selected, ambiguous)
	}

	selected, ambiguous = selectSQLiteRuntimeCandidate([]sqliteRuntimeCandidate{{container: "container-a", path: "/app/data/wago.db"}})
	if ambiguous || selected.path != "/app/data/wago.db" {
		t.Fatalf("selection = %#v, ambiguous = %v; want the single candidate", selected, ambiguous)
	}
}

func TestResolveConnectionPreservesServerDatabasePrecedence(t *testing.T) {
	called := false
	previous := sqliteRuntimeCommand
	sqliteRuntimeCommand = func(context.Context, ...string) ([]byte, error) {
		called = true
		return nil, fmt.Errorf("runtime discovery should not run")
	}
	t.Cleanup(func() { sqliteRuntimeCommand = previous })

	project := db.Project{Name: "server-first", DeployMode: "dockerfile"}
	conn, err := resolveConnection(context.Background(), project, map[string]string{
		"DATABASE_URL": "postgres://app:secret@db:5432/appdb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn.Driver != DriverPostgres || called {
		t.Fatalf("connection = %#v, runtime command called = %v; server connection must win", conn, called)
	}
	if strings.Contains(conn.Source, "runtime") {
		t.Fatalf("server connection was replaced by runtime discovery: %#v", conn)
	}
}

func TestResolveSQLiteRuntimeConnectionUsesPersistentMountDiscovery(t *testing.T) {
	previousRuntimeCommand := sqliteRuntimeCommand
	previousHelperCommand := sqliteRuntimeHelperCommand
	sqliteRuntimeCommand = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "ps" {
			return []byte("mypaas-wago\n"), nil
		}
		if len(args) > 0 && args[0] == "inspect" {
			return []byte(`[{"Config":{"WorkingDir":"/app"},"Mounts":[{"Type":"volume","Destination":"/app/data","RW":true}]}]`), nil
		}
		if len(args) == 4 && args[0] == "exec" && args[1] == "mypaas-wago" && args[2] == "id" && (args[3] == "-u" || args[3] == "-g") {
			return []byte("1000\n"), nil
		}
		return nil, fmt.Errorf("unexpected runtime command: %v", args)
	}
	sqliteRuntimeHelperCommand = func(_ context.Context, args []string, payload []byte) ([]byte, error) {
		interactive := false
		for _, arg := range args {
			if arg == "-i" {
				interactive = true
				break
			}
		}
		if !interactive {
			return nil, fmt.Errorf("SQLite helper must keep stdin attached")
		}
		var request SQLiteHelperRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if request.Operation != "discover" || len(request.DiscoveryRoots) != 1 || request.DiscoveryRoots[0] != "/app/data" {
			return nil, fmt.Errorf("unexpected discovery request: %#v", request)
		}
		return []byte(`{"ok":true,"response":{"sqliteCandidates":["/app/data/wago.db"]}}`), nil
	}
	t.Setenv("MYPAAS_SQLITE_HELPER_IMAGE", "mypaas-api:test")
	t.Cleanup(func() {
		sqliteRuntimeCommand = previousRuntimeCommand
		sqliteRuntimeHelperCommand = previousHelperCommand
	})

	conn, ok, err := resolveSQLiteRuntimeConnection(context.Background(), db.Project{Name: "wago", DeployMode: "dockerfile"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || conn.Driver != DriverSQLite || conn.Database != "/app/data/wago.db" || conn.Source != "runtime-discovery" || conn.runtimeContainer != "mypaas-wago" {
		t.Fatalf("connection = %#v, ok = %v; want generic persistent runtime discovery", conn, ok)
	}
}

func TestResolveSQLiteRuntimeConnectionFindsComposeLabeledRuntimeWhenStableNameIsMissing(t *testing.T) {
	previousRuntimeCommand := sqliteRuntimeCommand
	previousHelperCommand := sqliteRuntimeHelperCommand
	sqliteRuntimeCommand = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "inspect" && len(args) == 2 && args[1] == "mypaas-wago" {
			return []byte("Error: no such object: mypaas-wago"), fmt.Errorf("exit status 1")
		}
		if len(args) > 0 && args[0] == "ps" {
			return []byte("mypaas-wago-wago-1\n"), nil
		}
		if len(args) > 0 && args[0] == "inspect" {
			return []byte(`[ {"Config":{"WorkingDir":"/app"},"Mounts":[{"Type":"volume","Destination":"/app/data","RW":true}] } ]`), nil
		}
		if len(args) == 4 && args[0] == "exec" && args[1] == "mypaas-wago-wago-1" && args[2] == "id" && (args[3] == "-u" || args[3] == "-g") {
			return []byte("1000\n"), nil
		}
		return nil, fmt.Errorf("unexpected runtime command: %v", args)
	}
	sqliteRuntimeHelperCommand = func(_ context.Context, _ []string, payload []byte) ([]byte, error) {
		var request SQLiteHelperRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if request.Operation != "discover" || len(request.DiscoveryRoots) != 1 || request.DiscoveryRoots[0] != "/app/data" {
			return nil, fmt.Errorf("unexpected discovery request: %#v", request)
		}
		return []byte(`{"ok":true,"response":{"sqliteCandidates":["/app/data/wago.db"]}}`), nil
	}
	t.Setenv("MYPAAS_SQLITE_HELPER_IMAGE", "mypaas-api:test")
	t.Cleanup(func() {
		sqliteRuntimeCommand = previousRuntimeCommand
		sqliteRuntimeHelperCommand = previousHelperCommand
	})

	conn, ok, err := resolveSQLiteRuntimeConnection(context.Background(), db.Project{Name: "wago", DeployMode: "dockerfile"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || conn.Driver != DriverSQLite || conn.Database != "/app/data/wago.db" || conn.runtimeContainer != "mypaas-wago-wago-1" {
		t.Fatalf("connection = %#v, ok = %v; want Compose-labeled Wago runtime discovery", conn, ok)
	}
}

func TestSQLiteRuntimeClientUsesRuntimeIdentityForDatabaseAccess(t *testing.T) {
	previousRuntimeCommand := sqliteRuntimeCommand
	previousHelperCommand := sqliteRuntimeHelperCommand
	sqliteRuntimeCommand = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) == 4 && args[0] == "exec" && args[1] == "mypaas-wago-wago-1" && args[2] == "id" {
			if args[3] == "-u" || args[3] == "-g" {
				return []byte("1000\n"), nil
			}
		}
		return nil, fmt.Errorf("unexpected runtime command: %v", args)
	}
	sqliteRuntimeHelperCommand = func(_ context.Context, args []string, payload []byte) ([]byte, error) {
		interactive := false
		identity := ""
		for index, arg := range args {
			if arg == "-i" {
				interactive = true
			}
			if arg == "--user" && index+1 < len(args) {
				identity = args[index+1]
			}
		}
		if !interactive || identity != "1000:1000" {
			return nil, fmt.Errorf("unexpected helper runtime options: %v", args)
		}
		var request SQLiteHelperRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if request.Operation != "ping" || request.DatabasePath != "/app/data/wago.db" {
			return nil, fmt.Errorf("unexpected helper request: %#v", request)
		}
		return []byte(`{"ok":true,"response":{}}`), nil
	}
	t.Setenv("MYPAAS_SQLITE_HELPER_IMAGE", "mypaas-api:test")
	t.Cleanup(func() {
		sqliteRuntimeCommand = previousRuntimeCommand
		sqliteRuntimeHelperCommand = previousHelperCommand
	})

	client := &sqliteRuntimeClient{container: "mypaas-wago-wago-1", databasePath: "/app/data/wago.db"}
	if err := client.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.runtimeUser != "1000:1000" {
		t.Fatalf("runtimeUser = %q, want 1000:1000", client.runtimeUser)
	}
}

func TestOpenSQLiteDatabaseDoesNotCreateMissingFile(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "missing.db")
	conn, err := openSQLiteDatabase(ctx, databasePath)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected missing SQLite database to fail")
	}
	if _, statErr := os.Stat(databasePath); !os.IsNotExist(statErr) {
		t.Fatalf("missing SQLite path was created or stat failed unexpectedly: %v", statErr)
	}
}

func TestSQLiteAdapterCRUDAndMetadata(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "app.db")
	if err := os.WriteFile(databasePath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	conn, err := openSQLiteDatabase(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `
CREATE TABLE teams (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT UNIQUE,
  team_id INTEGER,
  FOREIGN KEY(team_id) REFERENCES teams(id) ON DELETE SET NULL
);
CREATE INDEX users_lower_email_idx ON users(lower(email));`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO teams(name) VALUES ('infra')`); err != nil {
		t.Fatal(err)
	}

	adapter := sqliteAdapter{}
	schemas, err := adapter.Schemas(ctx, conn)
	if err != nil || len(schemas) != 1 || schemas[0].Name != "main" {
		t.Fatalf("schemas = %#v, err = %v", schemas, err)
	}
	tables, err := adapter.Tables(ctx, conn, "main")
	if err != nil || len(tables) != 2 {
		t.Fatalf("tables = %#v, err = %v", tables, err)
	}
	columns, err := adapter.Columns(ctx, conn, "main", "users")
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 4 || columns[0].Name != "id" || !columns[0].PrimaryKey || !columns[0].AutoGenerated {
		t.Fatalf("unexpected users columns: %#v", columns)
	}
	details, err := adapter.TableDetails(ctx, conn, "main", "users")
	if err != nil {
		t.Fatal(err)
	}
	if len(details.ForeignKeys) != 1 || details.ForeignKeys[0].ReferencedTable != "teams" {
		t.Fatalf("unexpected foreign keys: %#v", details.ForeignKeys)
	}
	foundExpressionIndex := false
	for _, index := range details.Indexes {
		if index.Name == "users_lower_email_idx" {
			foundExpressionIndex = true
			if index.Definition == "" {
				t.Fatalf("expression index definition is empty: %#v", index)
			}
		}
	}
	if !foundExpressionIndex {
		t.Fatalf("expression index missing from table details: %#v", details.Indexes)
	}

	if err := adapter.Insert(ctx, conn, Mutation{Schema: "main", Table: "users", Values: map[string]any{"name": "Nabil", "email": "nabil@example.test", "team_id": 1}}); err != nil {
		t.Fatal(err)
	}
	page, err := adapter.Rows(ctx, conn, RowQuery{Schema: "main", Table: "users", Limit: 20, Search: "Nabil"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0]["name"] != "Nabil" {
		t.Fatalf("unexpected rows: %#v", page.Rows)
	}
	id := page.Rows[0]["id"]
	if err := adapter.Update(ctx, conn, Mutation{Schema: "main", Table: "users", Values: map[string]any{"name": "Nabil R"}, PrimaryKey: map[string]any{"id": id}}); err != nil {
		t.Fatal(err)
	}
	page, err = adapter.Rows(ctx, conn, RowQuery{Schema: "main", Table: "users", Limit: 20, Search: "Nabil R"})
	if err != nil || len(page.Rows) != 1 {
		t.Fatalf("updated rows = %#v, err = %v", page.Rows, err)
	}
	if err := adapter.Delete(ctx, conn, Mutation{Schema: "main", Table: "users", PrimaryKey: map[string]any{"id": id}}); err != nil {
		t.Fatal(err)
	}
	page, err = adapter.Rows(ctx, conn, RowQuery{Schema: "main", Table: "users", Limit: 20})
	if err != nil || len(page.Rows) != 0 {
		t.Fatalf("rows after delete = %#v, err = %v", page.Rows, err)
	}
}
