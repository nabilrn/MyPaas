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

func TestResolvePartsUsesMergedComposeDatabaseServiceEnv(t *testing.T) {
	project := db.Project{ID: uuid.New(), Name: "sop-generate-app", DeployMode: "compose"}
	projectEnv := map[string]string{
		"DB_PASSWORD": "project-secret",
	}
	dbServiceEnv := envListMap([]string{
		"MARIADB_DATABASE=sop_biro_organisasi",
		"MARIADB_USER=sop_app",
		"MARIADB_PASSWORD=compose-secret",
	})

	conn, ok := resolveParts(project, mergeMissingEnv(projectEnv, dbServiceEnv))
	if !ok {
		t.Fatal("resolveParts() did not detect merged compose DB service env")
	}
	if conn.Driver != DriverMariaDB || conn.Host != "db" || conn.Database != "sop_biro_organisasi" || conn.User != "sop_app" {
		t.Fatalf("unexpected connection: %#v", conn)
	}
	if conn.password != "compose-secret" {
		t.Fatal("expected MariaDB service password to be used for MARIADB_USER")
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

func TestComposeProjectCandidatesIncludeMyPaasPrefix(t *testing.T) {
	got := composeProjectCandidates("sop-arsip")
	want := []string{"mypaas-sop-arsip", "sop-arsip"}
	if len(got) != len(want) {
		t.Fatalf("candidates length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestComposeProjectCandidatesIncludeDeploymentPrefixForPrefixedProject(t *testing.T) {
	got := composeProjectCandidates("mypaas-sample-crud-db")
	want := []string{"mypaas-mypaas-sample-crud-db", "mypaas-sample-crud-db"}
	if len(got) != len(want) {
		t.Fatalf("candidates length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseComposeNetworkInspect(t *testing.T) {
	endpoints, err := parseComposeNetworkInspect(`{
		"mypaas-prod": {"IPAddress": "172.20.0.9"},
		"mypaas-sop-arsip_sop-network": {"IPAddress": "172.21.0.3"}
	}`)
	if err != nil {
		t.Fatalf("parseComposeNetworkInspect() error = %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("endpoints length = %d, want 2: %#v", len(endpoints), endpoints)
	}
}

func TestParseContainerEnvInspect(t *testing.T) {
	envs, err := parseContainerEnvInspect(`["MARIADB_DATABASE=sop_biro_organisasi","MARIADB_USER=sop_app","MARIADB_PASSWORD=secret","MALFORMED"]`)
	if err != nil {
		t.Fatalf("parseContainerEnvInspect() error = %v", err)
	}
	if envs["MARIADB_DATABASE"] != "sop_biro_organisasi" || envs["MARIADB_USER"] != "sop_app" || envs["MARIADB_PASSWORD"] != "secret" {
		t.Fatalf("unexpected env map: %#v", envs)
	}
	if _, ok := envs["MALFORMED"]; ok {
		t.Fatalf("malformed env entry should be skipped: %#v", envs)
	}
}

func TestMergeMissingEnvKeepsProjectValues(t *testing.T) {
	merged := mergeMissingEnv(
		map[string]string{"DATABASE_HOST": "custom-db", "DB_PASSWORD": "project-secret"},
		map[string]string{"DATABASE_HOST": "db", "MARIADB_DATABASE": "app"},
	)
	if merged["DATABASE_HOST"] != "custom-db" || merged["DB_PASSWORD"] != "project-secret" || merged["MARIADB_DATABASE"] != "app" {
		t.Fatalf("unexpected merged env: %#v", merged)
	}
}

func TestPreferredEndpointSkipsMyPaasPlatformNetwork(t *testing.T) {
	endpoint, ok := preferredEndpoint([]composeServiceEndpoint{
		{Network: "mypaas_platform", IPAddress: "172.20.0.9"},
		{Network: "mypaas-sop-arsip_sop-network", IPAddress: "172.21.0.3"},
	})
	if !ok {
		t.Fatal("preferredEndpoint() did not return an endpoint")
	}
	if endpoint.Network != "mypaas-sop-arsip_sop-network" || endpoint.IPAddress != "172.21.0.3" {
		t.Fatalf("endpoint = %#v, want sop-network endpoint", endpoint)
	}
}

func TestRetargetConnectionRebuildsDSN(t *testing.T) {
	conn := connectionWithDSN(Connection{
		Driver: DriverMariaDB, Host: "db", Port: 3306, Database: "sop_biro_organisasi", User: "sop_app", Source: "env-parts",
	}, "secret", nil)

	retargeted := retargetConnection(conn, "172.21.0.3")
	if retargeted.Host != "172.21.0.3" {
		t.Fatalf("Host = %q, want container IP", retargeted.Host)
	}
	if retargeted.DSN == conn.DSN {
		t.Fatal("DSN did not change after retarget")
	}
	if retargeted.Source != "env-parts+compose-ip" {
		t.Fatalf("Source = %q, want compose-ip marker", retargeted.Source)
	}
}
