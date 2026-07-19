package envdiscover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverEnvFilesAndComposeVariables(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".env.example"), []byte(`
APP_NAME=demo
SECRET_KEY=sample
export PUBLIC_URL="https://example.test" # comment
`), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "compose.yml"), []byte(`
services:
  app:
    environment:
      DATABASE_URL: ${DATABASE_URL}
      PORT: ${PORT:-3000}
      CACHE_DSN: ${CACHE_DSN:?required}
      HEALTHCHECK_PASSWORD: $${MARIADB_ROOT_PASSWORD}
`), 0640); err != nil {
		t.Fatal(err)
	}

	vars, err := Discover(workspace, "compose.yml")
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	byKey := make(map[string]Var, len(vars))
	for _, item := range vars {
		byKey[item.Key] = item
	}

	if got := byKey["APP_NAME"].DefaultValue; got == nil || *got != "demo" {
		t.Fatalf("APP_NAME default = %v, want demo", got)
	}
	if !byKey["SECRET_KEY"].Sensitive || byKey["SECRET_KEY"].DefaultValue != nil {
		t.Fatalf("SECRET_KEY should be sensitive without copied default: %#v", byKey["SECRET_KEY"])
	}
	if got := byKey["PORT"].DefaultValue; got == nil || *got != "3000" {
		t.Fatalf("PORT default = %v, want 3000", got)
	}
	if !byKey["DATABASE_URL"].Sensitive {
		t.Fatalf("DATABASE_URL should be sensitive")
	}
	if !byKey["CACHE_DSN"].Sensitive {
		t.Fatalf("CACHE_DSN should be sensitive")
	}
	if _, ok := byKey["MARIADB_ROOT_PASSWORD"]; ok {
		t.Fatalf("escaped Compose variable should not be discovered: %#v", byKey["MARIADB_ROOT_PASSWORD"])
	}
}

func TestDiscoverNestedEnvFiles(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "server"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "server", ".env.example"), []byte("\ufeffDB_NAME=sop_arsip\nPUBLIC_URL=https://example.test/#/login\nSECRET_KEY=sample\n"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "server", "Dockerfile.prod"), []byte(`
FROM node:20
ARG NODE_VERSION=20
ENV PORT=8080 NODE_ENV=production
ENV LEGACY_MODE true
`), 0640); err != nil {
		t.Fatal(err)
	}

	vars, err := Discover(workspace, "")
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	byKey := make(map[string]Var, len(vars))
	for _, item := range vars {
		byKey[item.Key] = item
	}

	if got := byKey["DB_NAME"].DefaultValue; got == nil || *got != "sop_arsip" {
		t.Fatalf("DB_NAME default = %v, want sop_arsip", got)
	}
	if got := byKey["PUBLIC_URL"].DefaultValue; got == nil || *got != "https://example.test/#/login" {
		t.Fatalf("PUBLIC_URL default = %v, want URL with anchor", got)
	}
	if !byKey["SECRET_KEY"].Sensitive || byKey["SECRET_KEY"].DefaultValue != nil {
		t.Fatalf("SECRET_KEY should be sensitive without copied default: %#v", byKey["SECRET_KEY"])
	}
	for _, key := range []string{"PORT", "NODE_VERSION", "NODE_ENV", "LEGACY_MODE"} {
		if _, ok := byKey[key]; ok {
			t.Fatalf("Dockerfile key %s should not be discovered as project env: %#v", key, byKey[key])
		}
	}
}

func TestDiscoverAttributesVarsToComposeServices(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "compose.yml"), []byte(`
services:
  api:
    image: myapi
    environment:
      DATABASE_URL: ${DATABASE_URL}
      JWT_SECRET: ${JWT_SECRET}
  worker:
    image: myworker
    environment:
      DATABASE_URL: ${DATABASE_URL}
      QUEUE_NAME: ${QUEUE_NAME}
`), 0640); err != nil {
		t.Fatal(err)
	}

	vars, err := Discover(workspace, "compose.yml")
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	// Simulate the JSON config that `docker compose config --format json` would
	// output for this compose file.
	configJSON := []byte(`{
		"services": {
			"api": {
				"environment": {
					"DATABASE_URL": "",
					"JWT_SECRET": ""
				}
			},
			"worker": {
				"environment": {
					"DATABASE_URL": "",
					"QUEUE_NAME": ""
				}
			}
		}
	}`)
	vars = AttributeServicesFromConfig(vars, configJSON)

	byKey := make(map[string]Var, len(vars))
	for _, item := range vars {
		byKey[item.Key] = item
	}

	db := byKey["DATABASE_URL"]
	if len(db.Services) != 2 {
		t.Fatalf("DATABASE_URL services = %v, want [api, worker]", db.Services)
	}
	if db.Services[0] != "api" || db.Services[1] != "worker" {
		t.Fatalf("DATABASE_URL services = %v, want [api, worker]", db.Services)
	}

	jwt := byKey["JWT_SECRET"]
	if len(jwt.Services) != 1 || jwt.Services[0] != "api" {
		t.Fatalf("JWT_SECRET services = %v, want [api]", jwt.Services)
	}

	queue := byKey["QUEUE_NAME"]
	if len(queue.Services) != 1 || queue.Services[0] != "worker" {
		t.Fatalf("QUEUE_NAME services = %v, want [worker]", queue.Services)
	}
}

func TestGenerateEnvFromTemplateSubstitutesOverrides(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, ".env.example")
	if err := os.WriteFile(templatePath, []byte(`# Database config
DATABASE_URL=postgres://localhost:5432/mydb
REDIS_URL=redis://localhost:6379
# App config
APP_NAME=myapp
`), 0640); err != nil {
		t.Fatal(err)
	}

	result, err := GenerateEnvFromTemplate(templatePath, map[string]string{
		"DATABASE_URL": "postgres://user:pass@db:5432/mydb",
		"NEW_VAR":      "injected",
	})
	if err != nil {
		t.Fatalf("GenerateEnvFromTemplate() error = %v", err)
	}

	if !strings.Contains(result, "DATABASE_URL=postgres://user:pass@db:5432/mydb") {
		t.Fatalf("expected DATABASE_URL override in result:\n%s", result)
	}
	if !strings.Contains(result, "REDIS_URL=redis://localhost:6379") {
		t.Fatalf("expected REDIS_URL to keep template default:\n%s", result)
	}
	if !strings.Contains(result, "NEW_VAR=injected") {
		t.Fatalf("expected NEW_VAR to be appended:\n%s", result)
	}
	if !strings.Contains(result, "# Database config") {
		t.Fatalf("expected comments to be preserved:\n%s", result)
	}
}

func TestFormatEnvFileSortsByKey(t *testing.T) {
	result := FormatEnvFile(map[string]string{
		"ZOO": "3",
		"ALPHA": "1",
		"MID": "2",
	})
	want := "ALPHA=1\nMID=2\nZOO=3\n"
	if result != want {
		t.Fatalf("FormatEnvFile() = %q, want %q", result, want)
	}
}

func TestParseEnvFileEntriesHandlesAllShapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []ServiceEnvFile
	}{
		{
			name: "string",
			raw:  `"apps/api/.env"`,
			want: []ServiceEnvFile{{Path: "apps/api/.env", Required: true}},
		},
		{
			name: "list of strings",
			raw:  `["apps/api/.env", "apps/api/.env.prod"]`,
			want: []ServiceEnvFile{
				{Path: "apps/api/.env", Required: true},
				{Path: "apps/api/.env.prod", Required: true},
			},
		},
		{
			name: "list of objects",
			raw:  `[{"path": "apps/api/.env", "required": true}, {"path": "apps/api/.env.optional", "required": false}]`,
			want: []ServiceEnvFile{
				{Path: "apps/api/.env", Required: true},
				{Path: "apps/api/.env.optional", Required: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEnvFileEntries(jsonRawMessage(tt.raw))
			if len(got) != len(tt.want) {
				t.Fatalf("parseEnvFileEntries() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i].Path != tt.want[i].Path || got[i].Required != tt.want[i].Required {
					t.Fatalf("parseEnvFileEntries()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func jsonRawMessage(s string) []byte {
	return []byte(s)
}
