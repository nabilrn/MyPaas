package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
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

	result, err := service.listRepositories(context.Background(), uuid.New(), 1, token)
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
