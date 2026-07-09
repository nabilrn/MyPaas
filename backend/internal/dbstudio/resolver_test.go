package dbstudio

import (
	"testing"

	"github.com/google/uuid"

	"mypaas/internal/db"
)

func TestResolvePartsInfersMariaDBComposeDatabase(t *testing.T) {
	project := db.Project{ID: uuid.New(), Name: "sop-arsip", DeployMode: "compose"}
	envs := map[string]string{
		"DB_ROOT_PASSWORD": "root-secret",
		"DB_NAME":          "sop_biro_organisasi",
		"DB_USER":          "sop_app",
		"DB_PASSWORD":      "app-secret",
	}

	conn, ok := resolveParts(project, envs)
	if !ok {
		t.Fatal("resolveParts() did not detect MariaDB compose env")
	}
	if conn.Driver != DriverMariaDB {
		t.Fatalf("Driver = %q, want %q", conn.Driver, DriverMariaDB)
	}
	if conn.Host != "db" || conn.Port != 3306 || conn.Database != "sop_biro_organisasi" || conn.User != "sop_app" {
		t.Fatalf("unexpected connection: %#v", conn)
	}
	if conn.DSN == "" || conn.Source != "env-parts" {
		t.Fatalf("connection DSN/source not set: %#v", conn)
	}
}

func TestConnectionFromPostgresURL(t *testing.T) {
	conn, err := connectionFromURL("postgres://app:secret@postgres:5432/appdb?sslmode=disable", "DATABASE_URL")
	if err != nil {
		t.Fatalf("connectionFromURL() error = %v", err)
	}
	if conn.Driver != DriverPostgres || conn.Host != "postgres" || conn.Database != "appdb" || conn.User != "app" {
		t.Fatalf("unexpected connection: %#v", conn)
	}
}
