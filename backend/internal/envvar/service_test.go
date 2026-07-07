package envvar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/crypto"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

func TestBulkUpdateNormalizesKeysAndRevealDecrypts(t *testing.T) {
	cipher, err := crypto.NewAESGCM("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	store := newFakeEnvStore()
	service := NewService(db.New(store), cipher)
	projectID := uuid.New()

	if err := service.BulkUpdate(context.Background(), projectID, []Value{{Key: " api_token ", Value: "secret-value"}}); err != nil {
		t.Fatalf("bulk update: %v", err)
	}
	if store.lastUpsert.Key != "API_TOKEN" {
		t.Fatalf("expected normalized key API_TOKEN, got %q", store.lastUpsert.Key)
	}

	got, err := service.Reveal(context.Background(), projectID, "api_token")
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("expected decrypted value, got %q", got)
	}
}

func TestRevealMissingKeyReturnsNotFound(t *testing.T) {
	cipher, err := crypto.NewAESGCM("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	service := NewService(db.New(newFakeEnvStore()), cipher)
	_, err = service.Reveal(context.Background(), uuid.New(), "missing")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

type fakeEnvStore struct {
	byKey      map[string]db.EnvVar
	lastUpsert db.UpsertEnvVarParams
}

func newFakeEnvStore() *fakeEnvStore {
	return &fakeEnvStore{byKey: make(map[string]db.EnvVar)}
}

func (s *fakeEnvStore) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s *fakeEnvStore) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (s *fakeEnvStore) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "INSERT INTO env_vars"):
		arg := db.UpsertEnvVarParams{
			ProjectID:      args[0].(uuid.UUID),
			Key:            args[1].(string),
			ValueEncrypted: args[2].(string),
			ValueNonce:     args[3].(string),
		}
		s.lastUpsert = arg
		row := db.EnvVar{
			ID:             uuid.New(),
			ProjectID:      arg.ProjectID,
			Key:            arg.Key,
			ValueEncrypted: arg.ValueEncrypted,
			ValueNonce:     arg.ValueNonce,
			CreatedAt:      pgtype.Timestamp{Valid: true},
			UpdatedAt:      pgtype.Timestamp{Valid: true},
		}
		s.byKey[arg.Key] = row
		return fakeEnvRow{row: row}
	case strings.Contains(query, "WHERE project_id = $1 AND key = $2"):
		row, ok := s.byKey[args[1].(string)]
		if !ok {
			return fakeEnvRow{err: pgx.ErrNoRows}
		}
		return fakeEnvRow{row: row}
	default:
		return fakeEnvRow{err: fmt.Errorf("unexpected query: %s", query)}
	}
}

type fakeEnvRow struct {
	row db.EnvVar
	err error
}

func (r fakeEnvRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 7 {
		return fmt.Errorf("unexpected env var scan target count %d", len(dest))
	}
	*(dest[0].(*uuid.UUID)) = r.row.ID
	*(dest[1].(*uuid.UUID)) = r.row.ProjectID
	*(dest[2].(*string)) = r.row.Key
	*(dest[3].(*string)) = r.row.ValueEncrypted
	*(dest[4].(*string)) = r.row.ValueNonce
	*(dest[5].(*pgtype.Timestamp)) = r.row.CreatedAt
	*(dest[6].(*pgtype.Timestamp)) = r.row.UpdatedAt
	return nil
}
