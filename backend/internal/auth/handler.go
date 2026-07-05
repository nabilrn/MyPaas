package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

const stateCookieName = "mypaas_oauth_state"

type Handler struct {
	cfg     *config.Config
	oauth   *oauth2.Config
	queries *db.Queries
	tokens  *TokenService
}

func NewHandler(cfg *config.Config, queries *db.Queries, tokens *TokenService) *Handler {
	return &Handler{
		cfg: cfg,
		oauth: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.GitHubCallbackURL,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		},
		queries: queries,
		tokens:  tokens,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		httpx.DomainError(w, fmt.Errorf("oauth state: %w", err))
		return
	}

	http.SetCookie(w, h.cookie(stateCookieName, state, 10*time.Minute, false))
	http.Redirect(w, r, h.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline), http.StatusFound)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		httpx.Error(w, http.StatusBadRequest, "INVALID_OAUTH_STATE", "OAuth state is invalid or expired.", nil)
		return
	}
	clearCookie(w, stateCookieName, h.cfg.IsDevelopment())

	code := r.URL.Query().Get("code")
	if code == "" {
		httpx.Error(w, http.StatusBadRequest, "MISSING_OAUTH_CODE", "OAuth callback did not include a code.", nil)
		return
	}

	token, err := h.oauth.Exchange(r.Context(), code)
	if err != nil {
		httpx.DomainError(w, fmt.Errorf("github token exchange: %w", err))
		return
	}

	profile, err := fetchGitHubProfile(r.Context(), h.oauth.Client(r.Context(), token))
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), profile.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.DomainError(w, errs.ErrEmailNotWhitelisted)
			return
		}
		httpx.DomainError(w, err)
		return
	}

	githubID := strconv.FormatInt(profile.ID, 10)
	user, err = h.queries.UpdateUserGithubProfile(r.Context(), db.UpdateUserGithubProfileParams{
		ID:             user.ID,
		GithubID:       &githubID,
		GithubUsername: &profile.Login,
		AvatarUrl:      &profile.AvatarURL,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	tokens, err := h.tokens.Issue(user)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	http.SetCookie(w, h.cookie(AccessCookieName, tokens.AccessToken, time.Until(tokens.ExpiresAt), true))
	http.SetCookie(w, h.cookie(RefreshCookieName, tokens.RefreshToken, time.Until(tokens.RefreshExpiresAt), true))
	http.Redirect(w, r, mustJoinURL(h.cfg.FrontendURL, "/projects"), http.StatusFound)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userCtx, err := CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userCtx.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.DomainError(w, errs.ErrUnauthorized)
			return
		}
		httpx.DomainError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, UserResponseFromDB(user))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshCookie, err := r.Cookie(RefreshCookieName)
	if err != nil || refreshCookie.Value == "" {
		httpx.DomainError(w, errs.ErrUnauthorized)
		return
	}
	claims, err := h.tokens.ParseRefresh(refreshCookie.Value)
	if err != nil {
		httpx.DomainError(w, errs.ErrUnauthorized)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.DomainError(w, errs.ErrUnauthorized)
			return
		}
		httpx.DomainError(w, err)
		return
	}

	tokens, err := h.tokens.Issue(user)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	http.SetCookie(w, h.cookie(AccessCookieName, tokens.AccessToken, time.Until(tokens.ExpiresAt), true))
	http.SetCookie(w, h.cookie(RefreshCookieName, tokens.RefreshToken, time.Until(tokens.RefreshExpiresAt), true))
	httpx.JSON(w, http.StatusOK, map[string]any{
		"expiresAt":        tokens.ExpiresAt,
		"refreshExpiresAt": tokens.RefreshExpiresAt,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	clearCookie(w, AccessCookieName, h.cfg.IsDevelopment())
	clearCookie(w, RefreshCookieName, h.cfg.IsDevelopment())
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) cookie(name, value string, maxAge time.Duration, httpOnly bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
		Secure:   !h.cfg.IsDevelopment(),
	}
}

type githubProfile struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func fetchGitHubProfile(ctx context.Context, client *http.Client) (githubProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return githubProfile{}, fmt.Errorf("github user request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return githubProfile{}, fmt.Errorf("github user request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubProfile{}, fmt.Errorf("github user request returned %s", resp.Status)
	}

	var profile githubProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return githubProfile{}, fmt.Errorf("decode github user: %w", err)
	}
	if profile.Email != "" {
		return profile, nil
	}

	email, err := fetchPrimaryEmail(ctx, client)
	if err != nil {
		return githubProfile{}, err
	}
	profile.Email = email
	return profile, nil
}

func fetchPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("github email request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github email request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github email request returned %s", resp.Status)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode github emails: %w", err)
	}
	for _, item := range emails {
		if item.Primary && item.Verified {
			return item.Email, nil
		}
	}
	return "", errs.ErrEmailNotWhitelisted
}

func randomState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func clearCookie(w http.ResponseWriter, name string, development bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: name == AccessCookieName || name == RefreshCookieName,
		SameSite: http.SameSiteLaxMode,
		Secure:   !development,
	})
}

func mustJoinURL(base, path string) string {
	u, err := url.Parse(base)
	if err != nil {
		return path
	}
	u.Path = path
	return u.String()
}
