package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"mypaas/internal/crypto"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

const (
	githubAPIBaseURL        = "https://api.github.com"
	githubClientTimeout     = 15 * time.Second
	maxRepositoriesPerPage = 100
	storedOAuthTokenVersion = 1
)

// TokenReader is the narrow interface used by source and deployment services.
// GitHub credentials stay inside the control plane and are never passed to a
// project container.
type TokenReader interface {
	AccessToken(context.Context, uuid.UUID) (string, error)
}

type Repository struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	FullName      string  `json:"fullName"`
	Private       bool    `json:"private"`
	DefaultBranch string  `json:"defaultBranch"`
	CloneURL      string  `json:"cloneUrl"`
	HTMLURL       string  `json:"htmlUrl"`
	Description   *string `json:"description"`
	UpdatedAt     string  `json:"updatedAt"`
}

type RepositoryPage struct {
	Repositories []Repository `json:"repositories"`
	Page         int          `json:"page"`
	HasNextPage  bool         `json:"hasNextPage"`
}

type storedOAuthToken struct {
	Version      int       `json:"version"`
	AccessToken  string    `json:"accessToken"`
	TokenType    string    `json:"tokenType,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

type Service struct {
	queries *db.Queries
	cipher  *crypto.AESGCM
	client  *http.Client
	oauth   *oauth2.Config
	tokenMu sync.Mutex
}

func NewService(queries *db.Queries, cipher *crypto.AESGCM) *Service {
	return &Service{queries: queries, cipher: cipher, client: &http.Client{Timeout: githubClientTimeout}}
}

// ConfigureOAuth installs the same OAuth configuration used by the auth flow.
// It is called during process startup before requests are served and lets the
// repository/deployment path refresh expiring GitHub access tokens silently.
func (s *Service) ConfigureOAuth(config *oauth2.Config) {
	if config == nil {
		s.oauth = nil
		return
	}
	copy := *config
	s.oauth = &copy
}

// SaveAccessToken preserves compatibility with legacy rows that stored only a
// long-lived access token. New OAuth callbacks should call SaveOAuthToken so
// refresh metadata is retained when GitHub issues expiring credentials.
func (s *Service) SaveAccessToken(ctx context.Context, userID uuid.UUID, accessToken string) error {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	return s.saveCredentialLocked(ctx, userID, accessToken)
}

func (s *Service) SaveOAuthToken(ctx context.Context, userID uuid.UUID, token *oauth2.Token) error {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	return s.saveOAuthTokenLocked(ctx, userID, token)
}

func (s *Service) saveOAuthTokenLocked(ctx context.Context, userID uuid.UUID, token *oauth2.Token) error {
	payload, err := encodeOAuthToken(token)
	if err != nil {
		return err
	}
	return s.saveCredentialLocked(ctx, userID, payload)
}

func (s *Service) saveCredentialLocked(ctx context.Context, userID uuid.UUID, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: GitHub access token is empty", errs.ErrValidation)
	}
	if strings.ContainsAny(value, "\r\n") && !strings.HasPrefix(value, "{") {
		return fmt.Errorf("%w: GitHub access token contains invalid characters", errs.ErrValidation)
	}
	ciphertext, nonce, err := s.cipher.Encrypt(value)
	if err != nil {
		return fmt.Errorf("encrypt GitHub OAuth credential: %w", err)
	}
	if err := s.queries.SetGithubAccessToken(ctx, db.SetGithubAccessTokenParams{
		ID:                         userID,
		GithubAccessTokenEncrypted: &ciphertext,
		GithubAccessTokenNonce:     &nonce,
	}); err != nil {
		return fmt.Errorf("store GitHub OAuth credential: %w", err)
	}
	return nil
}

func (s *Service) ClearAccessToken(ctx context.Context, userID uuid.UUID) error {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	return s.clearAccessTokenLocked(ctx, userID)
}

func (s *Service) clearAccessTokenLocked(ctx context.Context, userID uuid.UUID) error {
	if err := s.queries.ClearGithubAccessToken(ctx, userID); err != nil {
		return fmt.Errorf("clear GitHub OAuth credential: %w", err)
	}
	return nil
}

func (s *Service) AccessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.accessToken(ctx, userID, false)
}

func (s *Service) accessToken(ctx context.Context, userID uuid.UUID, forceRefresh bool) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	value, err := s.loadCredentialLocked(ctx, userID)
	if err != nil {
		return "", err
	}
	token, stored, err := decodeOAuthToken(value)
	if err != nil {
		return "", err
	}
	if !stored {
		if forceRefresh {
			if err := s.clearAccessTokenLocked(ctx, userID); err != nil {
				return "", err
			}
			return "", errs.ErrGitHubAuthorizationRequired
		}
		return value, nil
	}

	if !forceRefresh && token.Valid() {
		return token.AccessToken, nil
	}
	if token.RefreshToken == "" {
		if err := s.clearAccessTokenLocked(ctx, userID); err != nil {
			return "", err
		}
		return "", errs.ErrGitHubAuthorizationRequired
	}
	if s.oauth == nil {
		return "", fmt.Errorf("GitHub OAuth refresh is not configured")
	}

	refreshToken := *token
	if forceRefresh {
		refreshToken.Expiry = time.Now().Add(-time.Minute)
	}
	refreshed, err := s.oauth.TokenSource(ctx, &refreshToken).Token()
	if err != nil {
		if invalidRefreshToken(err) {
			if clearErr := s.clearAccessTokenLocked(ctx, userID); clearErr != nil {
				return "", clearErr
			}
			return "", errs.ErrGitHubAuthorizationRequired
		}
		return "", fmt.Errorf("refresh GitHub access token: %w", err)
	}
	if err := s.saveOAuthTokenLocked(ctx, userID, refreshed); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

func (s *Service) loadCredentialLocked(ctx context.Context, userID uuid.UUID) (string, error) {
	row, err := s.queries.GetGithubAccessToken(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrNotFound
		}
		return "", fmt.Errorf("load GitHub OAuth credential: %w", err)
	}
	if row.GithubAccessTokenEncrypted == nil || row.GithubAccessTokenNonce == nil {
		return "", errs.ErrNotFound
	}
	value, err := s.cipher.Decrypt(*row.GithubAccessTokenEncrypted, *row.GithubAccessTokenNonce)
	if err != nil {
		return "", fmt.Errorf("decrypt GitHub OAuth credential: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errs.ErrNotFound
	}
	return value, nil
}

func (s *Service) ListRepositories(ctx context.Context, userID uuid.UUID, page int) (RepositoryPage, error) {
	if page < 1 || page > 1000 {
		return RepositoryPage{}, fmt.Errorf("%w: repository page must be between 1 and 1000", errs.ErrValidation)
	}
	accessToken, err := s.AccessToken(ctx, userID)
	if errors.Is(err, errs.ErrNotFound) {
		return RepositoryPage{}, errs.ErrGitHubAuthorizationRequired
	}
	if err != nil {
		return RepositoryPage{}, err
	}

	result, err := s.listRepositories(ctx, page, accessToken)
	if !errors.Is(err, errs.ErrGitHubAuthorizationRequired) {
		return result, err
	}

	// A token can be revoked or expire between local validity checks and the API
	// request. If a refresh token exists, rotate it once and retry transparently.
	accessToken, refreshErr := s.accessToken(ctx, userID, true)
	if refreshErr != nil {
		return RepositoryPage{}, refreshErr
	}
	result, err = s.listRepositories(ctx, page, accessToken)
	if errors.Is(err, errs.ErrGitHubAuthorizationRequired) {
		_ = s.ClearAccessToken(ctx, userID)
	}
	return result, err
}

func (s *Service) listRepositories(ctx context.Context, page int, accessToken string) (RepositoryPage, error) {
	query := url.Values{}
	query.Set("visibility", "all")
	query.Set("affiliation", "owner,collaborator,organization_member")
	query.Set("sort", "updated")
	query.Set("direction", "desc")
	query.Set("per_page", strconv.Itoa(maxRepositoriesPerPage))
	query.Set("page", strconv.Itoa(page))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIBaseURL+"/user/repos?"+query.Encode(), nil)
	if err != nil {
		return RepositoryPage{}, fmt.Errorf("create GitHub repository request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "MyPaaS")

	resp, err := s.client.Do(req)
	if err != nil {
		return RepositoryPage{}, fmt.Errorf("request GitHub repositories: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return RepositoryPage{}, errs.ErrGitHubAuthorizationRequired
	}
	if resp.StatusCode != http.StatusOK {
		return RepositoryPage{}, fmt.Errorf("GitHub repository request returned %s", resp.Status)
	}

	var payload []githubRepository
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return RepositoryPage{}, fmt.Errorf("decode GitHub repositories: %w", err)
	}
	repositories := make([]Repository, 0, len(payload))
	for _, item := range payload {
		repositories = append(repositories, Repository{
			ID:            item.ID,
			Name:          item.Name,
			FullName:      item.FullName,
			Private:       item.Private,
			DefaultBranch: item.DefaultBranch,
			CloneURL:      item.CloneURL,
			HTMLURL:       item.HTMLURL,
			Description:   item.Description,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return RepositoryPage{
		Repositories: repositories,
		Page:         page,
		HasNextPage:  strings.Contains(resp.Header.Get("Link"), `rel="next"`),
	}, nil
}

func encodeOAuthToken(token *oauth2.Token) (string, error) {
	if token == nil {
		return "", fmt.Errorf("%w: GitHub OAuth token is missing", errs.ErrValidation)
	}
	accessToken := strings.TrimSpace(token.AccessToken)
	if accessToken == "" {
		return "", fmt.Errorf("%w: GitHub access token is empty", errs.ErrValidation)
	}
	if strings.ContainsAny(accessToken, "\r\n") {
		return "", fmt.Errorf("%w: GitHub access token contains invalid characters", errs.ErrValidation)
	}
	payload, err := json.Marshal(storedOAuthToken{
		Version:      storedOAuthTokenVersion,
		AccessToken:  accessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	})
	if err != nil {
		return "", fmt.Errorf("encode GitHub OAuth credential: %w", err)
	}
	return string(payload), nil
}

func decodeOAuthToken(value string) (*oauth2.Token, bool, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "{") {
		if strings.ContainsAny(value, "\r\n") {
			return nil, false, fmt.Errorf("decode GitHub OAuth credential: legacy access token contains invalid characters")
		}
		return nil, false, nil
	}
	var stored storedOAuthToken
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		return nil, true, fmt.Errorf("decode GitHub OAuth credential: %w", err)
	}
	if stored.Version != storedOAuthTokenVersion {
		return nil, true, fmt.Errorf("decode GitHub OAuth credential: unsupported version %d", stored.Version)
	}
	stored.AccessToken = strings.TrimSpace(stored.AccessToken)
	if stored.AccessToken == "" || strings.ContainsAny(stored.AccessToken, "\r\n") {
		return nil, true, fmt.Errorf("decode GitHub OAuth credential: invalid access token")
	}
	return &oauth2.Token{
		AccessToken:  stored.AccessToken,
		TokenType:    stored.TokenType,
		RefreshToken: stored.RefreshToken,
		Expiry:       stored.Expiry,
	}, true, nil
}

func invalidRefreshToken(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(retrieveErr.ErrorCode))
	if code == "bad_refresh_token" || code == "invalid_grant" {
		return true
	}
	return strings.Contains(strings.ToLower(string(retrieveErr.Body)), "bad_refresh_token")
}

type githubRepository struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	FullName      string  `json:"full_name"`
	Private       bool    `json:"private"`
	DefaultBranch string  `json:"default_branch"`
	CloneURL      string  `json:"clone_url"`
	HTMLURL       string  `json:"html_url"`
	Description   *string `json:"description"`
	UpdatedAt     string  `json:"updated_at"`
}
