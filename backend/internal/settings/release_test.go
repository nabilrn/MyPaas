package settings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	oldReleaseSHA = "1111111111111111111111111111111111111111"
	newReleaseSHA = "2222222222222222222222222222222222222222"
)

func releaseServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"tag_name":"v9.9.9-draft","target_commitish":"3333333333333333333333333333333333333333","html_url":"https://example.test/draft","draft":true,"published_at":"2026-08-30T00:00:00Z"},
			{"tag_name":"v0.5.0-beta.3","target_commitish":"` + newReleaseSHA + `","html_url":"https://example.test/beta3","draft":false,"published_at":"2026-08-29T00:00:00Z"},
			{"tag_name":"v0.5.0-beta.2","target_commitish":"` + oldReleaseSHA + `","html_url":"https://example.test/beta2","draft":false,"published_at":"2026-08-18T00:00:00Z"}
		]`))
	}))
}

func TestReleaseStatusReportsAvailableRelease(t *testing.T) {
	server := releaseServer(t)
	defer server.Close()
	t.Setenv("MYPAAS_UPDATE_CHANNEL", "release")
	t.Setenv("MYPAAS_RELEASES_API_URL", server.URL)

	status, err := releaseStatus(context.Background(), oldReleaseSHA)
	if err != nil {
		t.Fatalf("releaseStatus: %v", err)
	}
	if status.State != "available" || !status.UpdateAvailable {
		t.Fatalf("expected available update, got %#v", status)
	}
	if status.CurrentTag != "v0.5.0-beta.2" {
		t.Fatalf("expected current beta.2 tag, got %q", status.CurrentTag)
	}
	if status.LatestTag != "v0.5.0-beta.3" || status.LatestSHA != newReleaseSHA {
		t.Fatalf("unexpected latest release: %#v", status)
	}
}

func TestReleaseStatusReportsCurrentRelease(t *testing.T) {
	server := releaseServer(t)
	defer server.Close()
	t.Setenv("MYPAAS_UPDATE_CHANNEL", "release")
	t.Setenv("MYPAAS_RELEASES_API_URL", server.URL)

	status, err := releaseStatus(context.Background(), newReleaseSHA)
	if err != nil {
		t.Fatalf("releaseStatus: %v", err)
	}
	if status.State != "current" || status.UpdateAvailable {
		t.Fatalf("expected current release, got %#v", status)
	}
	if status.CurrentTag != "v0.5.0-beta.3" {
		t.Fatalf("expected current beta.3 tag, got %q", status.CurrentTag)
	}
}

func TestReleaseStatusRefChannelDoesNotCallGitHub(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Setenv("MYPAAS_UPDATE_CHANNEL", "ref")
	t.Setenv("MYPAAS_REF", "main")
	t.Setenv("MYPAAS_RELEASES_API_URL", server.URL)

	status, err := releaseStatus(context.Background(), newReleaseSHA)
	if err != nil {
		t.Fatalf("releaseStatus: %v", err)
	}
	if status.State != "tracking_ref" || status.TrackingRef != "main" {
		t.Fatalf("unexpected ref status: %#v", status)
	}
	if called {
		t.Fatal("ref channel must not query GitHub Releases")
	}
}

func TestReleaseStatusRejectsMutableLatestTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"tag_name":"v0.5.0-beta.3","target_commitish":"main","draft":false,"published_at":"2026-08-29T00:00:00Z"}]`))
	}))
	defer server.Close()
	t.Setenv("MYPAAS_UPDATE_CHANNEL", "release")
	t.Setenv("MYPAAS_RELEASES_API_URL", server.URL)

	status, err := releaseStatus(context.Background(), oldReleaseSHA)
	if err == nil {
		t.Fatal("expected mutable release target to be rejected")
	}
	if status.State != "unavailable" || status.UpdateAvailable {
		t.Fatalf("unexpected status: %#v", status)
	}
}
