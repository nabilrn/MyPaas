package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDailyTime(t *testing.T) {
	hour, minute, err := parseDailyTime("02:30")
	if err != nil {
		t.Fatalf("parseDailyTime() error = %v", err)
	}
	if hour != 2 || minute != 30 {
		t.Fatalf("parseDailyTime() = %d:%d, want 2:30", hour, minute)
	}

	if _, _, err := parseDailyTime("25:00"); err == nil {
		t.Fatal("parseDailyTime() error = nil, want range error")
	}
}

func TestNextDaily(t *testing.T) {
	loc := time.FixedZone("test", 7*60*60)
	now := time.Date(2026, 7, 2, 1, 0, 0, 0, loc)
	got := nextDaily(now, 2, 0)
	want := time.Date(2026, 7, 2, 2, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("nextDaily() = %s, want %s", got, want)
	}

	now = time.Date(2026, 7, 2, 3, 0, 0, 0, loc)
	got = nextDaily(now, 2, 0)
	want = time.Date(2026, 7, 3, 2, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("nextDaily() = %s, want %s", got, want)
	}
}

func TestApplyRetentionKeepsNewestMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	names := []string{
		"mypaas-daily-old.dump",
		"mypaas-daily-mid.dump",
		"mypaas-daily-new.dump",
		"notes.txt",
		"mypaas-weekly-old.dump",
	}
	for i, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, base, base.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	if err := applyRetention(dir, dailyPrefix, 2); err != nil {
		t.Fatalf("applyRetention() error = %v", err)
	}

	assertExists(t, filepath.Join(dir, "mypaas-daily-new.dump"))
	assertExists(t, filepath.Join(dir, "mypaas-daily-mid.dump"))
	assertMissing(t, filepath.Join(dir, "mypaas-daily-old.dump"))
	assertExists(t, filepath.Join(dir, "notes.txt"))
	assertExists(t, filepath.Join(dir, "mypaas-weekly-old.dump"))
}

func TestPGDumpEnvUsesPGVariables(t *testing.T) {
	env, err := pgDumpEnv("postgres://user:secret@localhost:15432/mypaas?sslmode=require", []string{"PATH=/bin"})
	if err != nil {
		t.Fatalf("pgDumpEnv() error = %v", err)
	}

	want := map[string]string{
		"PGHOST":     "localhost",
		"PGPORT":     "15432",
		"PGDATABASE": "mypaas",
		"PGUSER":     "user",
		"PGPASSWORD": "secret",
		"PGSSLMODE":  "require",
	}
	for key, value := range want {
		if !containsEnv(env, key+"="+value) {
			t.Fatalf("pgDumpEnv() missing %s=%s in %v", key, value, env)
		}
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s to be removed", path)
	}
}

func containsEnv(env []string, target string) bool {
	for _, item := range env {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
