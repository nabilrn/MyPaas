package migration

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mypaas/internal/config"
)

const (
	archiveDir      = "/tmp/mypaas/migrations"
	archiveMaxAge   = 24 * time.Hour
	StatusPreparing = "preparing"
	StatusReady     = "ready"
	StatusFailed    = "failed"
	StatusExpired   = "expired"
)

// Migration holds the state of a single export.
type Migration struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	DownloadToken string `json:"downloadToken,omitempty"`
	SizeBytes     int64  `json:"sizeBytes,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	Error         string `json:"error,omitempty"`
	archivePath   string
}

type manifest struct {
	Version    int      `json:"version"`
	ExportedAt string   `json:"exported_at"`
	Hostname   string   `json:"hostname"`
	MypaasDB   string   `json:"mypaas_db"`
	SharedDBs  []string `json:"shared_dbs"`
}

// Service manages migration exports. Only one export is allowed at a time.
type Service struct {
	cfg     *config.Config
	runtime RuntimeQuiescer

	mu      sync.Mutex
	current *Migration
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg, runtime: newRuntimeQuiescer(cfg)}
}

// Prepare starts a migration export in a background goroutine.
func (s *Service) Prepare(ctx context.Context) (*Migration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If there's an existing ready migration that hasn't expired, return it.
	if s.current != nil && s.current.Status == StatusReady {
		if exp, err := time.Parse(time.RFC3339, s.current.ExpiresAt); err == nil && time.Now().Before(exp) {
			return s.current, nil
		}
		// Expired — clean up.
		s.cleanup()
	}

	if s.current != nil && s.current.Status == StatusPreparing {
		return s.current, nil
	}

	id := randomHex(8)
	token := randomHex(16)

	m := &Migration{
		ID:            id,
		Status:        StatusPreparing,
		DownloadToken: token,
	}
	s.current = m

	go s.runExport(m)

	return m, nil
}

// Status returns the current migration state.
func (s *Service) Status(_ context.Context, id string) *Migration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil || s.current.ID != id {
		return nil
	}

	// Check expiry.
	if s.current.Status == StatusReady {
		if exp, err := time.Parse(time.RFC3339, s.current.ExpiresAt); err == nil && time.Now().After(exp) {
			s.cleanup()
			return &Migration{ID: id, Status: StatusExpired}
		}
	}

	cp := *s.current
	return &cp
}

// ArchivePath returns the file path for a download if the token matches.
func (s *Service) ArchivePath(id, token string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil || s.current.ID != id {
		return "", fmt.Errorf("migration not found")
	}
	if s.current.Status != StatusReady {
		return "", fmt.Errorf("migration not ready")
	}
	if s.current.DownloadToken != token {
		return "", fmt.Errorf("invalid download token")
	}
	if exp, err := time.Parse(time.RFC3339, s.current.ExpiresAt); err == nil && time.Now().After(exp) {
		s.cleanup()
		return "", fmt.Errorf("migration expired")
	}
	return s.current.archivePath, nil
}

func (s *Service) runExport(m *Migration) {
	if err := os.MkdirAll(archiveDir, 0700); err != nil {
		s.fail(m, fmt.Errorf("create migration dir: %w", err))
		return
	}

	workDir := filepath.Join(archiveDir, m.ID)
	if err := os.MkdirAll(workDir, 0700); err != nil {
		s.fail(m, fmt.Errorf("create work dir: %w", err))
		return
	}
	defer os.RemoveAll(workDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	resumeRuntime := ResumeFunc(func(context.Context) error { return nil })
	runtimeQuiesced := false
	runtimeResumed := false
	if s.runtime != nil {
		slog.Info("migration: quiescing running project runtimes")
		resume, err := s.runtime.Quiesce(ctx)
		if err != nil {
			s.fail(m, fmt.Errorf("quiesce project runtimes: %w", err))
			return
		}
		if resume != nil {
			resumeRuntime = resume
		}
		runtimeQuiesced = true
	}
	defer func() {
		if !runtimeQuiesced || runtimeResumed {
			return
		}
		resumeCtx, resumeCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer resumeCancel()
		if err := resumeRuntime(resumeCtx); err != nil {
			slog.Error("migration: failed to resume project runtimes after export failure", "error", err)
		}
	}()

	dbDir := filepath.Join(workDir, "databases")
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		s.fail(m, fmt.Errorf("create databases dir: %w", err))
		return
	}

	// 1. Dump system database.
	slog.Info("migration: dumping system database")
	if err := s.pgDump(ctx, s.cfg.DatabaseURL, filepath.Join(dbDir, "system.dump")); err != nil {
		s.fail(m, fmt.Errorf("dump system db: %w", err))
		return
	}

	// 2. Find and dump shared project databases.
	sharedDBs := s.findSharedDatabases(ctx)
	for _, dbName := range sharedDBs {
		slog.Info("migration: dumping shared database", "db", dbName)
		dbURL := s.replaceDBName(s.cfg.DatabaseURL, dbName)
		if err := s.pgDump(ctx, dbURL, filepath.Join(dbDir, dbName+".dump")); err != nil {
			slog.Warn("migration: failed to dump shared db", "db", dbName, "error", err)
			// Continue with other DBs — don't fail the whole export.
		}
	}

	// 3. Dump roles.
	slog.Info("migration: dumping database roles")
	if err := s.pgDumpRoles(ctx, filepath.Join(dbDir, "roles.sql")); err != nil {
		slog.Warn("migration: failed to dump roles", "error", err)
	}

	// 4. Copy .env if accessible.
	s.copyDotEnv(workDir)

	// 5. Write manifest.
	hostname, _ := os.Hostname()
	parsed, _ := url.Parse(s.cfg.DatabaseURL)
	dbName := strings.TrimPrefix(parsed.Path, "/")
	mf := manifest{
		Version:    1,
		ExportedAt: time.Now().Format(time.RFC3339),
		Hostname:   hostname,
		MypaasDB:   dbName,
		SharedDBs:  sharedDBs,
	}
	mfData, _ := json.MarshalIndent(mf, "", "  ")
	_ = os.WriteFile(filepath.Join(workDir, "manifest.json"), mfData, 0600)

	// 6. Create tar.gz archive directly from multiple sources to avoid disk duplication.
	slog.Info("migration: creating archive")
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("mypaas-export-%s.tar.gz", m.ID))

	sources := map[string]string{
		"": workDir, // Root of archive comes from workDir (databases, dot-env, manifest).
	}
	persistentDirs := []string{"volumes", "compose", "static"}
	for _, dir := range persistentDirs {
		fullPath := filepath.Join("/var/lib/mypaas", dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			sources[dir] = fullPath
		}
	}

	if err := createMultiTarGz(archivePath, sources); err != nil {
		s.fail(m, fmt.Errorf("create multi tar: %w", err))
		return
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		s.fail(m, fmt.Errorf("stat archive: %w", err))
		return
	}

	// Do not advertise a downloadable archive until every runtime that was
	// quiesced has been started again. A resume failure is an export failure.
	if runtimeQuiesced {
		resumeCtx, resumeCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		if err := resumeRuntime(resumeCtx); err != nil {
			resumeCancel()
			_ = os.Remove(archivePath)
			s.fail(m, fmt.Errorf("resume project runtimes: %w", err))
			return
		}
		resumeCancel()
		runtimeResumed = true
		slog.Info("migration: project runtimes resumed")
	}

	s.mu.Lock()
	m.Status = StatusReady
	m.SizeBytes = info.Size()
	m.ExpiresAt = time.Now().Add(archiveMaxAge).Format(time.RFC3339)
	m.archivePath = archivePath
	s.mu.Unlock()

	slog.Info("migration: export complete", "path", archivePath, "sizeBytes", info.Size())
}

func (s *Service) fail(m *Migration, err error) {
	slog.Error("migration export failed", "error", err)
	s.mu.Lock()
	m.Status = StatusFailed
	m.Error = err.Error()
	s.mu.Unlock()
}

func (s *Service) cleanup() {
	if s.current != nil && s.current.archivePath != "" {
		_ = os.Remove(s.current.archivePath)
	}
	s.current = nil
}

func (s *Service) pgDump(ctx context.Context, databaseURL, outputPath string) error {
	env, err := pgEnv(databaseURL, os.Environ())
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--file", outputPath)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Service) pgDumpRoles(ctx context.Context, outputPath string) error {
	env, err := pgEnv(s.cfg.DatabaseURL, os.Environ())
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "pg_dumpall", "--roles-only", "--file", outputPath)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dumpall: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Service) findSharedDatabases(ctx context.Context) []string {
	env, err := pgEnv(s.cfg.DatabaseURL, os.Environ())
	if err != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, "psql", "-t", "-A", "-c",
		"SELECT datname FROM pg_database WHERE datname LIKE 'mypaas_p_%'")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("migration: failed to list shared databases", "error", err)
		return nil
	}
	var dbs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			dbs = append(dbs, line)
		}
	}
	return dbs
}

func (s *Service) replaceDBName(databaseURL, newDB string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return databaseURL
	}
	parsed.Path = "/" + newDB
	return parsed.String()
}

func (s *Service) copyDotEnv(workDir string) {
	// Try several common locations relative to the running process.
	candidates := []string{"/mypaas/.env", ".env", "../.env", "../../.env", "../../../.env"}
	configDir := strings.TrimSpace(os.Getenv("MYPAAS_CONFIG_DIR"))
	if configDir != "" {
		candidates = append([]string{filepath.Join(configDir, ".env")}, candidates...)
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(workDir, "dot-env"), data, 0600)
		slog.Info("migration: copied .env", "from", path)
		return
	}
	slog.Warn("migration: .env not found")
}

func pgEnv(databaseURL string, base []string) ([]string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	env := append([]string{}, base...)
	set := func(key, value string) {
		if value != "" {
			env = append(env, key+"="+value)
		}
	}
	set("PGHOST", parsed.Hostname())
	port := parsed.Port()
	if port == "" {
		port = "5432"
	}
	set("PGPORT", port)
	dbName := strings.TrimPrefix(parsed.Path, "/")
	set("PGDATABASE", dbName)
	set("PGUSER", parsed.User.Username())
	if password, ok := parsed.User.Password(); ok {
		set("PGPASSWORD", password)
	}
	sslmode := parsed.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}
	set("PGSSLMODE", sslmode)
	return env, nil
}

func createMultiTarGz(archivePath string, sources map[string]string) error {
	out, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Use BestSpeed to compress huge files (like Minecraft worlds) much faster.
	gw, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for prefix, sourceDir := range sources {
		err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}
			if rel == "." && prefix == "" {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// Handle symlinks.
			if info.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err == nil {
					header.Linkname = link
				}
			}

			headerName := rel
			if prefix != "" {
				if rel == "." {
					headerName = prefix
				} else {
					headerName = filepath.Join(prefix, rel)
				}
			}

			header.Name = filepath.ToSlash(headerName)
			if d.IsDir() {
				header.Name += "/"
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if d.IsDir() || !info.Mode().IsRegular() {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
