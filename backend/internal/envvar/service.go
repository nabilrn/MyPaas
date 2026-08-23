package envvar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/crypto"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	PlatformPrefix          = "MYPAAS_PLATFORM_"
	ReleaseCommandKey       = "MYPAAS_PLATFORM_RELEASE_COMMAND"
	ReleaseCommandFileSuffix = ".mypaas-release-command"
)

type Service struct {
	queries *db.Queries
	cipher  *crypto.AESGCM
}

type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewService(queries *db.Queries, cipher *crypto.AESGCM) *Service {
	return &Service{queries: queries, cipher: cipher}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]db.EnvVar, error) {
	return s.queries.ListEnvVarsByProject(ctx, projectID)
}

func (s *Service) Reveal(ctx context.Context, projectID uuid.UUID, key string) (string, error) {
	key = normalizeKey(key)
	if key == "" {
		return "", fmt.Errorf("%w: env var key is required", errs.ErrValidation)
	}
	row, err := s.queries.GetEnvVar(ctx, db.GetEnvVarParams{ProjectID: projectID, Key: key})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrNotFound
		}
		return "", err
	}
	return s.cipher.Decrypt(row.ValueEncrypted, row.ValueNonce)
}

func (s *Service) BulkUpdate(ctx context.Context, projectID uuid.UUID, values []Value) error {
	for _, item := range values {
		key := normalizeKey(item.Key)
		if !keyPattern.MatchString(key) {
			return fmt.Errorf("%w: invalid env var key %q", errs.ErrValidation, item.Key)
		}
		encrypted, nonce, err := s.cipher.Encrypt(item.Value)
		if err != nil {
			return err
		}
		if _, err := s.queries.UpsertEnvVar(ctx, db.UpsertEnvVarParams{
			ProjectID:      projectID,
			Key:            key,
			ValueEncrypted: encrypted,
			ValueNonce:     nonce,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, projectID uuid.UUID, key string) error {
	key = normalizeKey(key)
	if key == "" {
		return fmt.Errorf("%w: env var key is required", errs.ErrValidation)
	}
	return s.queries.DeleteEnvVar(ctx, db.DeleteEnvVarParams{ProjectID: projectID, Key: key})
}

func (s *Service) DeleteAll(ctx context.Context, projectID uuid.UUID) error {
	return s.queries.DeleteAllEnvVars(ctx, projectID)
}

func (s *Service) DecryptedMap(ctx context.Context, projectID uuid.UUID) (map[string]string, error) {
	rows, err := s.queries.ListEnvVarsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		value, err := s.cipher.Decrypt(row.ValueEncrypted, row.ValueNonce)
		if err != nil {
			return nil, err
		}
		out[row.Key] = value
	}
	return out, nil
}

func IsPlatformKey(key string) bool {
	return strings.HasPrefix(normalizeKey(key), PlatformPrefix)
}

// RuntimeValues returns the project environment that may be passed to user
// application processes. MYPAAS_PLATFORM_* keys are control-plane settings
// stored with the project for backwards-compatible configuration, but they are
// never injected into the application runtime.
func RuntimeValues(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		if IsPlatformKey(key) {
			continue
		}
		out[key] = value
	}
	return out
}

// WriteEnvFile writes only application-visible variables to path. A release
// command, when configured, is persisted in a private sidecar file so the
// container lifecycle can execute it without exposing MYPAAS_PLATFORM_* values
// to the application process.
func WriteEnvFile(path string, values map[string]string) error {
	releaseCommand := strings.TrimSpace(values[ReleaseCommandKey])
	values = RuntimeValues(values)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(escapeEnvValue(values[key]))
		b.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		return err
	}

	releasePath := path + ReleaseCommandFileSuffix
	if releaseCommand == "" {
		if err := os.Remove(releasePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.WriteFile(releasePath, []byte(releaseCommand), 0600)
}

func escapeEnvValue(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t\n\r\"'\\#") {
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		return `"` + escaped + `"`
	}
	return value
}

func normalizeKey(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}
