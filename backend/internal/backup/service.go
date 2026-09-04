package backup

import (
	"bytes"
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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	mpconfig "mypaas/internal/config"
	"mypaas/internal/container"
)

const (
	dailyPrefix  = "mypaas-daily-"
	weeklyPrefix = "mypaas-weekly-"
	dumpSuffix   = ".tar.gz"
)

type Service struct {
	cfg    *mpconfig.Config
	docker *container.DockerCLI
	now    func() time.Time
}

type Result struct {
	DailyPath  string
	WeeklyPath string
}

type S3Config struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
}

func NewService(cfg *mpconfig.Config, docker *container.DockerCLI) *Service {
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

	if s.cfg.S3Endpoint != "" && s.cfg.S3Bucket != "" && s.cfg.S3AccessKey != "" && s.cfg.S3SecretKey != "" {
		if err := s.uploadToS3(ctx, result.DailyPath); err != nil {
			slog.Error("failed to upload backup to S3", "error", err)
		} else {
			slog.Info("backup uploaded to S3", "path", result.DailyPath)
		}
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
		if err := s.docker.CleanupBuildCache(ctx, s.cfg.ImageCleanupUntil); err != nil {
			slog.Warn("BuildKit cache cleanup failed", "error", err)
		} else {
			slog.Info("BuildKit cache cleanup completed", "until", s.cfg.ImageCleanupUntil)
		}
	}
}

func (s *Service) pgDump(ctx context.Context, outputPath string) error {
	env, err := pgDumpEnv(s.cfg.DatabaseURL, os.Environ())
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "mypaas-backup-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "database.sql")
	envPath := filepath.Join(tempDir, ".env")

	envData, err := os.ReadFile("/etc/mypaas/.env")
	if err != nil {
		slog.Warn("could not read /etc/mypaas/.env, falling back to os.Environ", "error", err)
		envContent := strings.Join(os.Environ(), "\n")
		envData = []byte(envContent)
	}

	if err := os.WriteFile(envPath, envData, 0600); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("pg_dump --no-owner --no-privileges > %s", dbPath))
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, strings.TrimSpace(string(out)))
	}

	tarCmd := exec.CommandContext(ctx, "tar", "-czf", outputPath, "-C", tempDir, "database.sql", ".env")
	out, err = tarCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar: %w: %s", err, strings.TrimSpace(string(out)))
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

func (s *Service) uploadToS3(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer file.Close()

	client, normalized, err := newS3Client(ctx, S3Config{
		Endpoint:  s.cfg.S3Endpoint,
		Bucket:    s.cfg.S3Bucket,
		Region:    s.cfg.S3Region,
		AccessKey: s.cfg.S3AccessKey,
		SecretKey: s.cfg.S3SecretKey,
	})
	if err != nil {
		return err
	}

	key := filepath.Base(filePath)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(normalized.Bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

func ValidateS3Connection(ctx context.Context, candidate S3Config) error {
	client, normalized, err := newS3Client(ctx, candidate)
	if err != nil {
		return err
	}

	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(normalized.Bucket)}); err != nil {
		return fmt.Errorf("access bucket: %w", err)
	}

	probeKey := fmt.Sprintf(".mypaas/connection-check-%d", time.Now().UnixNano())
	probeBody := []byte("mypaas-backup-storage-check")
	if _, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(normalized.Bucket),
		Key:    aws.String(probeKey),
		Body:   bytes.NewReader(probeBody),
	}); err != nil {
		return fmt.Errorf("write probe: %w", err)
	}

	if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(normalized.Bucket),
		Key:    aws.String(probeKey),
	}); err != nil {
		return fmt.Errorf("remove probe: %w", err)
	}

	return nil
}

func newS3Client(ctx context.Context, candidate S3Config) (*s3.Client, S3Config, error) {
	normalized := S3Config{
		Endpoint:  strings.TrimRight(strings.TrimSpace(candidate.Endpoint), "/"),
		Bucket:    strings.TrimSpace(candidate.Bucket),
		Region:    strings.TrimSpace(candidate.Region),
		AccessKey: strings.TrimSpace(candidate.AccessKey),
		SecretKey: candidate.SecretKey,
	}
	if normalized.Region == "" {
		normalized.Region = "auto"
	}
	if normalized.Endpoint == "" || normalized.Bucket == "" || normalized.AccessKey == "" || strings.TrimSpace(normalized.SecretKey) == "" {
		return nil, normalized, fmt.Errorf("endpoint, bucket, access key, and secret key are required")
	}
	parsedEndpoint, err := url.Parse(normalized.Endpoint)
	if err != nil || parsedEndpoint.Host == "" || (parsedEndpoint.Scheme != "https" && parsedEndpoint.Scheme != "http") {
		return nil, normalized, fmt.Errorf("endpoint must be a valid HTTP or HTTPS URL")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{URL: normalized.Endpoint}, nil
	})

	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(normalized.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(normalized.AccessKey, normalized.SecretKey, "")),
	)
	if err != nil {
		return nil, normalized, fmt.Errorf("load S3 config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return client, normalized, nil
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
	if keep >= len(files) {
		return nil
	}
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
