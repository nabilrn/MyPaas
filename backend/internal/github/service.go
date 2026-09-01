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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/crypto"
	"mypaas/internal/db"
	"mypaas/internal/errs"
)

const (
	githubAPIBaseURL       = "https://api.github.com"
	githubClientTimeout    = 15 * time.Second
	maxRepositoriesPerPage = 100
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

type Service struct {
	queries *db.Queries
	cipher  *crypto.AESGCM
	client  *http.Client
}

func NewService(queries *db.Queries, cipher *crypto.AESGCM) *Service {
	return &Service{queries: queries, cipher: cipher, client: &http.Client{Timeout: githubClientTimeout}}
}

func (s *Service) SaveAccessToken(ctx context.Context, userID uuid.UUID, accessToken string) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return fmt.Errorf("%w: GitHub access token is empty", errs.ErrValidation)
	}
	if strings.ContainsAny(accessToken, "\r\n") {
		return fmt.Errorf("%w: GitHub access token contains invalid characters", errs.ErrValidation)
	}
	ciphertext, nonce, err := s.cipher.Encrypt(accessToken)
	if err != nil {
		return fmt.Errorf("encrypt GitHub access token: %w", err)
	}
	if err := s.queries.SetGithubAccessToken(ctx, db.SetGithubAccessTokenParams{
		ID:                         userID,
		GithubAccessTokenEncrypted: &ciphertext,
		GithubAccessTokenNonce:     &nonce,
	}); err != nil {
		return fmt.Errorf("store GitHub access token: %w", err)
	}
	return nil
}

func (s *Service) ClearAccessToken(ctx context.Context, userID uuid.UUID) error {
	if err := s.queries.ClearGithubAccessToken(ctx, userID); err != nil {
		return fmt.Errorf("clear GitHub access token: %w", err)
	}
	return nil
}

func (s *Service) AccessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	row, err := s.queries.GetGithubAccessToken(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrNotFound
		}
		return "", fmt.Errorf("load GitHub access token: %w", err)
	}
	if row.GithubAccessTokenEncrypted == nil || row.GithubAccessTokenNonce == nil {
		return "", errs.ErrNotFound
	}
	accessToken, err := s.cipher.Decrypt(*row.GithubAccessTokenEncrypted, *row.GithubAccessTokenNonce)
	if err != nil {
		return "", fmt.Errorf("decrypt GitHub access token: %w", err)
	}
	if strings.TrimSpace(accessToken) == "" {
		return "", errs.ErrNotFound
	}
	return accessToken, nil
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
	return s.listRepositories(ctx, userID, page, accessToken)
}

func (s *Service) listRepositories(ctx context.Context, userID uuid.UUID, page int, accessToken string) (RepositoryPage, error) {
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
		_ = s.ClearAccessToken(ctx, userID)
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
