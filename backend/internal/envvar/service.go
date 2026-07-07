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

func WriteEnvFile(path string, values map[string]string) error {
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
	return os.WriteFile(path, []byte(b.String()), 0600)
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
