package apitoken

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
)

const tokenPrefix = "myp_"

var ErrInvalidToken = errors.New("invalid API token")

var allowedScopes = map[string]struct{}{
	"projects:read":     {},
	"projects:write":    {},
	"deployments:read":  {},
	"deployments:write": {},
	"logs:read":         {},
	"metrics:read":      {},
	"env:read":          {},
	"env:write":         {},
	"admin:read":        {},
}

var DefaultMCPScopes = []string{
	"projects:read",
	"projects:write",
	"deployments:read",
	"deployments:write",
	"logs:read",
	"metrics:read",
	"env:read",
	"env:write",
}

type Token struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"userId"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type CreatedToken struct {
	Token
	Secret string `json:"token"`
}

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (CreatedToken, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return CreatedToken{}, fmt.Errorf("token name must be 1-100 characters")
	}
	normalized, err := NormalizeScopes(scopes)
	if err != nil {
		return CreatedToken{}, err
	}
	if len(normalized) == 0 {
		return CreatedToken{}, fmt.Errorf("at least one scope is required")
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return CreatedToken{}, fmt.Errorf("token expiration must be in the future")
	}

	secret, hash, displayPrefix, err := newSecret()
	if err != nil {
		return CreatedToken{}, err
	}

	expires := pgtype.Timestamp{}
	if expiresAt != nil {
		expires = pgtype.Timestamp{Time: *expiresAt, Valid: true}
	}

	row, err := r.queries.CreateAPIToken(ctx, db.CreateAPITokenParams{
		UserID:      userID,
		Name:        name,
		TokenHash:   hash[:],
		TokenPrefix: displayPrefix,
		Scopes:      normalized,
		ExpiresAt:   expires,
	})
	if err != nil {
		return CreatedToken{}, err
	}
	return CreatedToken{Token: tokenFromRow(row), Secret: secret}, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Token, error) {
	rows, err := r.queries.ListAPITokens(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Token, 0, len(rows))
	for _, row := range rows {
		result = append(result, tokenFromRow(row))
	}
	return result, nil
}

func (r *Repository) Revoke(ctx context.Context, userID, tokenID uuid.UUID) error {
	count, err := r.queries.RevokeAPIToken(ctx, db.RevokeAPITokenParams{
		ID:     tokenID,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) RevokeByName(ctx context.Context, userID uuid.UUID, name string) error {
	return r.queries.RevokeActiveAPITokensByName(ctx, db.RevokeActiveAPITokensByNameParams{
		UserID: userID,
		Name:   name,
	})
}

func (r *Repository) Authenticate(ctx context.Context, raw string) (Token, error) {
	if !strings.HasPrefix(raw, tokenPrefix) || len(raw) < len(tokenPrefix)+20 {
		return Token{}, ErrInvalidToken
	}
	hash := sha256.Sum256([]byte(raw))
	row, err := r.queries.AuthenticateAPIToken(ctx, hash[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, ErrInvalidToken
	}
	if err != nil {
		return Token{}, err
	}
	return tokenFromRow(row), nil
}

func AllowedScopes() []string {
	result := make([]string, 0, len(allowedScopes))
	for scope := range allowedScopes {
		result = append(result, scope)
	}
	sort.Strings(result)
	return result
}

func NormalizeScopes(scopes []string) ([]string, error) {
	seen := make(map[string]struct{}, len(scopes))
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := allowedScopes[scope]; !ok {
			return nil, fmt.Errorf("unsupported scope %q", scope)
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	sort.Strings(result)
	return result, nil
}

func HasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}
	return false
}

func newSecret() (string, [32]byte, string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", [32]byte{}, "", fmt.Errorf("generate API token: %w", err)
	}
	secret := tokenPrefix + base64.RawURLEncoding.EncodeToString(random[:])
	hash := sha256.Sum256([]byte(secret))
	displayPrefix := secret
	if len(displayPrefix) > 16 {
		displayPrefix = displayPrefix[:16]
	}
	return secret, hash, displayPrefix, nil
}

func tokenFromRow(row db.ApiToken) Token {
	token := Token{
		ID:     row.ID,
		UserID: row.UserID,
		Name:   row.Name,
		Prefix: row.TokenPrefix,
		Scopes: append([]string(nil), row.Scopes...),
	}
	if row.CreatedAt.Valid {
		token.CreatedAt = row.CreatedAt.Time
	}
	if row.ExpiresAt.Valid {
		value := row.ExpiresAt.Time
		token.ExpiresAt = &value
	}
	if row.LastUsedAt.Valid {
		value := row.LastUsedAt.Time
		token.LastUsedAt = &value
	}
	if row.RevokedAt.Valid {
		value := row.RevokedAt.Time
		token.RevokedAt = &value
	}
	return token
}
