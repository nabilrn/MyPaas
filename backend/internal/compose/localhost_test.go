package compose

import (
	"reflect"
	"testing"
)

func TestDetectLocalhostInEnvFindsPortMatch(t *testing.T) {
	envs := map[string]string{
		"DATABASE_URL": "postgres://user:pass@localhost:5432/mydb",
		"REDIS_URL":    "redis://localhost:6379",
		"APP_NAME":     "myapp",
	}
	portToService := map[int32]string{
		5432: "db",
		6379: "redis",
	}

	warnings := DetectLocalhostInEnv(envs, portToService)
	if len(warnings) != 2 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 2", len(warnings))
	}

	byKey := make(map[string]LocalhostWarning, len(warnings))
	for _, w := range warnings {
		byKey[w.Key] = w
	}

	db := byKey["DATABASE_URL"]
	if db.Port != 5432 {
		t.Fatalf("DATABASE_URL port = %d, want 5432", db.Port)
	}
	if db.Service != "db" {
		t.Fatalf("DATABASE_URL service = %q, want db", db.Service)
	}
	wantSuggested := "postgres://user:pass@db:5432/mydb"
	if db.Suggested != wantSuggested {
		t.Fatalf("DATABASE_URL suggested = %q, want %q", db.Suggested, wantSuggested)
	}

	redis := byKey["REDIS_URL"]
	if redis.Port != 6379 {
		t.Fatalf("REDIS_URL port = %d, want 6379", redis.Port)
	}
	if redis.Service != "redis" {
		t.Fatalf("REDIS_URL service = %q, want redis", redis.Service)
	}
}

func TestDetectLocalhostInEnvMatches127001(t *testing.T) {
	envs := map[string]string{
		"NATS_URL": "nats://127.0.0.1:4222",
	}
	portToService := map[int32]string{
		4222: "nats",
	}

	warnings := DetectLocalhostInEnv(envs, portToService)
	if len(warnings) != 1 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 1", len(warnings))
	}
	if warnings[0].Service != "nats" {
		t.Fatalf("NATS_URL service = %q, want nats", warnings[0].Service)
	}
}

func TestDetectLocalhostInEnvWithoutPortMatch(t *testing.T) {
	envs := map[string]string{
		"API_URL": "http://localhost:3000",
	}

	warnings := DetectLocalhostInEnv(envs, map[int32]string{})
	if len(warnings) != 1 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 1", len(warnings))
	}
	if warnings[0].Port != 3000 {
		t.Fatalf("API_URL port = %d, want 3000", warnings[0].Port)
	}
	if warnings[0].Service != "" {
		t.Fatalf("API_URL service = %q, want empty (no match)", warnings[0].Service)
	}
	if warnings[0].Suggested != "" {
		t.Fatalf("API_URL suggested = %q, want empty (no match)", warnings[0].Suggested)
	}
}

func TestDetectLocalhostInEnvBareLocalhostNoPort(t *testing.T) {
	envs := map[string]string{
		"WEBAPI_URL": "http://localhost/api",
	}

	warnings := DetectLocalhostInEnv(envs, nil)
	if len(warnings) != 1 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 1", len(warnings))
	}
	if warnings[0].Port != 0 {
		t.Fatalf("Port = %d, want 0 (no port in URL)", warnings[0].Port)
	}
}

func TestDetectLocalhostInEnvIgnoresCleanValues(t *testing.T) {
	envs := map[string]string{
		"DATABASE_URL": "postgres://user:pass@db:5432/mydb",
		"REDIS_URL":    "redis://redis:6379",
		"APP_NAME":     "myapp",
		"PORT":         "3000",
	}

	warnings := DetectLocalhostInEnv(envs, map[int32]string{5432: "db", 6379: "redis"})
	if len(warnings) != 0 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 0 for clean values", len(warnings))
	}
}

func TestDetectLocalhostInEnvSortedByKey(t *testing.T) {
	envs := map[string]string{
		"ZOOKEEPER_URL": "zk://localhost:2181",
		"DATABASE_URL":  "postgres://localhost:5432",
		"AMQP_URL":      "amqp://localhost:5672",
	}

	warnings := DetectLocalhostInEnv(envs, nil)
	if len(warnings) != 3 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 3", len(warnings))
	}
	keys := make([]string, 0, len(warnings))
	for _, w := range warnings {
		keys = append(keys, w.Key)
	}
	want := []string{"AMQP_URL", "DATABASE_URL", "ZOOKEEPER_URL"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("Warnings not sorted by key: got %v, want %v", keys, want)
	}
}

func TestDetectLocalhostInEnvHandlesMultipleLocalhostInOneValue(t *testing.T) {
	envs := map[string]string{
		"COMPOSITE_URL": "http://localhost:3000 and redis://localhost:6379",
	}
	portToService := map[int32]string{
		3000: "app",
		6379: "redis",
	}

	warnings := DetectLocalhostInEnv(envs, portToService)
	if len(warnings) != 2 {
		t.Fatalf("DetectLocalhostInEnv() returned %d warnings, want 2 for two localhost refs in one value", len(warnings))
	}
}
