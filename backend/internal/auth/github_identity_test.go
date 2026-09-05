package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

type fakeAuthQueries struct {
	getByEmail    func(context.Context, string) (db.User, error)
	getByGithubID func(context.Context, string) (db.User, error)
	getByID       func(context.Context, uuid.UUID) (db.User, error)
	updateProfile func(context.Context, db.UpdateUserGithubProfileParams) (db.User, error)
}

func (f *fakeAuthQueries) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if f.getByEmail == nil {
		return db.User{}, pgx.ErrNoRows
	}
	return f.getByEmail(ctx, email)
}

func (f *fakeAuthQueries) GetUserByGithubID(ctx context.Context, githubID string) (db.User, error) {
	if f.getByGithubID == nil {
		return db.User{}, pgx.ErrNoRows
	}
	return f.getByGithubID(ctx, githubID)
}

func (f *fakeAuthQueries) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	if f.getByID == nil {
		return db.User{}, pgx.ErrNoRows
	}
	return f.getByID(ctx, id)
}

func (f *fakeAuthQueries) UpdateUserGithubProfile(ctx context.Context, params db.UpdateUserGithubProfileParams) (db.User, error) {
	if f.updateProfile == nil {
		return db.User{}, pgx.ErrNoRows
	}
	return f.updateProfile(ctx, params)
}

func TestResolveGitHubUserPrefersBoundNumericIdentity(t *testing.T) {
	userID := uuid.New()
	githubID := "42"
	stored := db.User{ID: userID, Email: "old@example.com", GithubID: &githubID, Role: "owner"}
	emailLookups := 0

	queries := &fakeAuthQueries{
		getByGithubID: func(_ context.Context, got string) (db.User, error) {
			if got != githubID {
				t.Fatalf("unexpected GitHub ID lookup: %s", got)
			}
			return stored, nil
		},
		getByEmail: func(_ context.Context, _ string) (db.User, error) {
			emailLookups++
			return db.User{}, pgx.ErrNoRows
		},
		updateProfile: func(_ context.Context, params db.UpdateUserGithubProfileParams) (db.User, error) {
			if params.ID != userID || params.GithubID == nil || *params.GithubID != githubID {
				t.Fatalf("unexpected profile update: %+v", params)
			}
			return stored, nil
		},
	}

	handler := &Handler{queries: queries}
	resolved, err := handler.resolveGitHubUser(context.Background(), githubProfile{
		ID:    42,
		Login: "owner-renamed",
		Email: "new-primary@example.com",
	})
	if err != nil {
		t.Fatalf("resolveGitHubUser returned error: %v", err)
	}
	if resolved.ID != userID {
		t.Fatalf("resolved wrong user: %s", resolved.ID)
	}
	if emailLookups != 0 {
		t.Fatalf("durable GitHub ID should bypass email fallback, got %d email lookups", emailLookups)
	}
}

func TestResolveGitHubUserBindsWhitelistedEmailOnce(t *testing.T) {
	userID := uuid.New()
	stored := db.User{ID: userID, Email: "owner@example.com", Role: "owner"}
	updated := false

	queries := &fakeAuthQueries{
		getByGithubID: func(_ context.Context, _ string) (db.User, error) {
			return db.User{}, pgx.ErrNoRows
		},
		getByEmail: func(_ context.Context, email string) (db.User, error) {
			if email != "owner@example.com" {
				t.Fatalf("unexpected email lookup: %s", email)
			}
			return stored, nil
		},
		updateProfile: func(_ context.Context, params db.UpdateUserGithubProfileParams) (db.User, error) {
			updated = true
			if params.GithubID == nil || *params.GithubID != "42" {
				t.Fatalf("GitHub ID was not bound: %+v", params)
			}
			bound := stored
			bound.GithubID = params.GithubID
			return bound, nil
		},
	}

	handler := &Handler{queries: queries}
	resolved, err := handler.resolveGitHubUser(context.Background(), githubProfile{
		ID:    42,
		Login: "owner",
		Email: " OWNER@EXAMPLE.COM ",
	})
	if err != nil {
		t.Fatalf("resolveGitHubUser returned error: %v", err)
	}
	if !updated || resolved.GithubID == nil || *resolved.GithubID != "42" {
		t.Fatalf("expected durable GitHub binding, got %+v", resolved)
	}
}

func TestResolveGitHubUserRejectsRebindingWhitelistedEmail(t *testing.T) {
	userID := uuid.New()
	boundID := "99"
	stored := db.User{ID: userID, Email: "owner@example.com", GithubID: &boundID, Role: "owner"}
	updated := false

	queries := &fakeAuthQueries{
		getByGithubID: func(_ context.Context, _ string) (db.User, error) {
			return db.User{}, pgx.ErrNoRows
		},
		getByEmail: func(_ context.Context, _ string) (db.User, error) {
			return stored, nil
		},
		updateProfile: func(_ context.Context, _ db.UpdateUserGithubProfileParams) (db.User, error) {
			updated = true
			return db.User{}, nil
		},
	}

	handler := &Handler{queries: queries}
	_, err := handler.resolveGitHubUser(context.Background(), githubProfile{
		ID:    42,
		Login: "different-account",
		Email: "owner@example.com",
	})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("expected forbidden identity rebind, got %v", err)
	}
	if updated {
		t.Fatal("profile must not be updated when the whitelist row is already bound to another GitHub ID")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetchGitHubProfileAlwaysUsesVerifiedPrimaryEmail(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body string
		switch request.URL.Path {
		case "/user":
			body = `{"id":42,"login":"owner","avatar_url":"https://example.com/avatar.png","email":"public-but-not-proven@example.com"}`
		case "/user/emails":
			body = `[{"email":"secondary@example.com","primary":false,"verified":true},{"email":"Verified.Primary@Example.com","primary":true,"verified":true}]`
		default:
			t.Fatalf("unexpected GitHub API path: %s", request.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}

	profile, err := fetchGitHubProfile(context.Background(), client)
	if err != nil {
		t.Fatalf("fetchGitHubProfile returned error: %v", err)
	}
	if profile.Email != "verified.primary@example.com" {
		t.Fatalf("expected verified primary email, got %q", profile.Email)
	}
}

func TestFetchGitHubProfileRejectsMissingVerifiedPrimaryEmail(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{"id":42,"login":"owner","email":"public@example.com"}`
		if request.URL.Path == "/user/emails" {
			body = `[{"email":"primary@example.com","primary":true,"verified":false}]`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}

	_, err := fetchGitHubProfile(context.Background(), client)
	if !errors.Is(err, errs.ErrGitHubVerifiedPrimaryEmailRequired) {
		t.Fatalf("expected verified-primary-email error, got %v", err)
	}
}
