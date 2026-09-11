package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestListRepositoriesMapsGitHubResponse(t *testing.T) {
	const token = "github-token-for-test"
	var gotRequest *http.Request

	service := &Service{
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotRequest = req
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header: http.Header{
					"Link": []string{`<https://api.github.com/user/repos?page=2>; rel="next"`},
				},
				Body: io.NopCloser(strings.NewReader(`[{"id":42,"name":"app","full_name":"acme/app","private":true,"default_branch":"main","clone_url":"https://github.com/acme/app.git","html_url":"https://github.com/acme/app","description":"A test app","updated_at":"2026-09-01T00:00:00Z"}]`)),
			}, nil
		})},
	}

	result, err := service.listRepositories(context.Background(), 1, token)
	if err != nil {
		t.Fatalf("listRepositories() error = %v", err)
	}
	if gotRequest == nil {
		t.Fatal("expected GitHub request")
	}
	if got := gotRequest.Header.Get("Authorization"); got != "Bearer "+token {
		t.Fatalf("Authorization header = %q, want bearer token", got)
	}
	if got := gotRequest.URL.Query().Get("per_page"); got != "100" {
		t.Fatalf("per_page = %q, want 100", got)
	}
	if !result.HasNextPage || len(result.Repositories) != 1 {
		t.Fatalf("result = %+v, want one repository with a next page", result)
	}
	if got := result.Repositories[0]; got.FullName != "acme/app" || !got.Private || got.DefaultBranch != "main" {
		t.Fatalf("repository mapping = %+v", got)
	}
}

func TestOAuthTokenStorageRoundTrip(t *testing.T) {
	expiry := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	encoded, err := encodeOAuthToken(&oauth2.Token{
		AccessToken:  "gho_access",
		TokenType:    "bearer",
		RefreshToken: "ghr_refresh",
		Expiry:       expiry,
	})
	if err != nil {
		t.Fatalf("encodeOAuthToken() error = %v", err)
	}

	decoded, stored, err := decodeOAuthToken(encoded)
	if err != nil {
		t.Fatalf("decodeOAuthToken() error = %v", err)
	}
	if !stored {
		t.Fatal("expected versioned OAuth token to be detected")
	}
	if decoded.AccessToken != "gho_access" || decoded.RefreshToken != "ghr_refresh" || decoded.TokenType != "bearer" {
		t.Fatalf("decoded token = %+v", decoded)
	}
	if !decoded.Expiry.Equal(expiry) {
		t.Fatalf("decoded expiry = %v, want %v", decoded.Expiry, expiry)
	}
}

func TestOAuthTokenStorageKeepsLegacyAccessTokensReadable(t *testing.T) {
	decoded, stored, err := decodeOAuthToken("github-token-for-test")
	if err != nil {
		t.Fatalf("decodeOAuthToken() error = %v", err)
	}
	if stored {
		t.Fatal("legacy access token must not be treated as a versioned OAuth token")
	}
	if decoded != nil {
		t.Fatalf("decoded legacy token = %+v, want nil", decoded)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
