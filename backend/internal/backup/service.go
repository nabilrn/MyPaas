package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"mypaas/internal/config"
	"mypaas/internal/container"
)

const (
	dailyPrefix  = "mypaas-daily-"
	weeklyPrefix = "mypaas-weekly-"
	dumpSuffix   = ".dump"
)

type Service struct {
	cfg    *config.Config
	docker *container.DockerCLI
	now    func() time.Time
}

type Result struct {
	DailyPath  string
	WeeklyPath string
}

func NewService(cfg *config.Config, docker *container.DockerCLI) *Service {
	return &Service{
		cfg:    cfg,
		docker: docker,
		now:    time.Now,
	}
}

func (s *Service) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if !s.cfg.BackupEnabled && !s.cfg.ImageCleanupEnabled {
			return
		}
		s.loop(ctx)
	}()
	return done
}

func (s *Service) Run(ctx context.Context) (Result, error) {
	dir := strings.TrimSpace(s.cfg.BackupDir)
	if dir == "" {
		dir = "/var/lib/mypaas/backups"
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return Result{}, fmt.Errorf("create backup dir: %w", err)
	}

	now := s.now()
	stamp := now.Format("20060102-150405")
	result := Result{
		DailyPath: filepath.Join(dir, dailyPrefix+stamp+dumpSuffix),
	}
	if err := s.pgDump(ctx, result.DailyPath); err != nil {
		return Result{}, err
	}

	if now.Weekday() == s.cfg.BackupWeeklyDay {
		result.WeeklyPath = filepath.Join(dir, weeklyPrefix+stamp+dumpSuffix)
		if err := copyFile(result.DailyPath, result.WeeklyPath); err != nil {
			return Result{}, fmt.Errorf("write weekly backup: %w", err)
		}
	}

	if err := applyRetention(dir, dailyPrefix, s.cfg.BackupKeepDaily); err != nil {
		return Result{}, err
	}
	if err := applyRetention(dir, weeklyPrefix, s.cfg.BackupKeepWeekly); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) loop(ctx context.Context) {
	hour, minute, err := parseDailyTime(s.cfg.BackupDailyAt)
	if err != nil {
		slog.Error("backup scheduler disabled", "error", err)
		return
	}

	for {
		next := nextDaily(s.now(), hour, minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.runScheduled(ctx)
		}
	}
}

func (s *Service) runScheduled(parent context.Context) {
	timeout := time.Duration(s.cfg.BackupTimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	runAt := s.now()
	if s.cfg.BackupEnabled {
		result, err := s.Run(ctx)
		if err != nil {
			slog.Error("database backup failed", "error", err)
		} else {
			slog.Info("database backup completed", "path", result.DailyPath, "weeklyPath", result.WeeklyPath)
		}
	}

	if s.cfg.ImageCleanupEnabled && runAt.Weekday() == s.cfg.ImageCleanupWeekday {
		if err := s.docker.CleanupUnusedManagedImages(ctx, s.cfg.ImageCleanupUntil); err != nil {
			slog.Warn("managed image cleanup failed", "error", err)
		} else {
			slog.Info("managed image cleanup completed", "until", s.cfg.ImageCleanupUntil)
		}
	}
}

func (s *Service) pgDump(ctx context.Context, outputPath string) error {
	env, err := pgDumpEnv(s.cfg.DatabaseURL, os.Environ())
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

func pgDumpEnv(databaseURL string, base []string) ([]string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported database url scheme %q", parsed.Scheme)
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		return nil, fmt.Errorf("database url is missing database name")
	}

	env := append([]string{}, base...)
	appendKV := func(key, value string) {
		if value != "" {
			env = append(env, key+"="+value)
		}
	}
	appendKV("PGHOST", parsed.Hostname())
	appendKV("PGPORT", defaultString(parsed.Port(), "5432"))
	appendKV("PGDATABASE", dbName)
	appendKV("PGUSER", parsed.User.Username())
	if password, ok := parsed.User.Password(); ok {
		appendKV("PGPASSWORD", password)
	}
	appendKV("PGSSLMODE", defaultString(parsed.Query().Get("sslmode"), "disable"))
	return env, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

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

func applyRetention(dir, prefix string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read backup dir: %w", err)
	}

	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || !strings.HasSuffix(entry.Name(), dumpSuffix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, fileInfo{name: entry.Name(), modTime: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	for _, file := range files[keep:] {
		if err := os.Remove(filepath.Join(dir, file.name)); err != nil {
			return fmt.Errorf("remove old backup %s: %w", file.name, err)
		}
	}
	return nil
}

type fileInfo struct {
	name    string
	modTime time.Time
}

func parseDailyTime(value string) (int, int, error) {
	hourRaw, minuteRaw, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok {
		return 0, 0, fmt.Errorf("backup time must use HH:MM")
	}
	hour, err := strconv.Atoi(hourRaw)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid backup hour: %w", err)
	}
	minute, err := strconv.Atoi(minuteRaw)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid backup minute: %w", err)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("backup time out of range: %s", value)
	}
	return hour, minute, nil
}

func nextDaily(now time.Time, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
