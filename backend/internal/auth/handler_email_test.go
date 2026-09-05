package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func githubTestClient(t *testing.T, profileJSON, emailsJSON string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/user":
			body = profileJSON
		case "/user/emails":
			body = emailsJSON
		default:
			t.Fatalf("unexpected GitHub path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
}

func TestFetchGitHubProfileAlwaysUsesVerifiedPrimaryEmail(t *testing.T) {
	client := githubTestClient(
		t,
		`{"id":42,"login":"owner","avatar_url":"https://example.test/avatar","email":"public@example.com"}`,
		`[{"email":"secondary@example.com","primary":false,"verified":true},{"email":"primary@example.com","primary":true,"verified":true}]`,
	)

	profile, err := fetchGitHubProfile(context.Background(), client)
	if err != nil {
		t.Fatalf("fetchGitHubProfile returned error: %v", err)
	}
	if profile.Email != "primary@example.com" {
		t.Fatalf("profile email = %q, want verified primary email", profile.Email)
	}
	if profile.ID != 42 || profile.Login != "owner" {
		t.Fatalf("profile identity changed unexpectedly: %#v", profile)
	}
}

func TestFetchGitHubProfileRejectsMissingVerifiedPrimaryEmail(t *testing.T) {
	client := githubTestClient(
		t,
		`{"id":42,"login":"owner","avatar_url":"https://example.test/avatar","email":"public@example.com"}`,
		`[{"email":"unverified@example.com","primary":true,"verified":false},{"email":"secondary@example.com","primary":false,"verified":true}]`,
	)

	if _, err := fetchGitHubProfile(context.Background(), client); err == nil {
		t.Fatal("fetchGitHubProfile succeeded without a verified primary email")
	}
}
