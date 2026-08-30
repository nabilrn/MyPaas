package container

import (
	"errors"
	"strings"
	"testing"
)

func TestRegistryHost(t *testing.T) {
	tests := map[string]string{
		"nginx:latest":                         "docker.io",
		"library/postgres:16":                  "docker.io",
		"ghcr.io/nabilrn/private-app:latest":   "ghcr.io",
		"registry.example.com:5000/team/app:v1": "registry.example.com:5000",
		"localhost:5000/app:latest":            "localhost:5000",
	}
	for image, want := range tests {
		if got := registryHost(image); got != want {
			t.Fatalf("registryHost(%q) = %q, want %q", image, got, want)
		}
	}
}

func TestRegistryCredentialsOnlyApplyToMatchingHost(t *testing.T) {
	t.Setenv(registryHostEnv, "https://ghcr.io/")
	t.Setenv(registryUsernameEnv, "octocat")
	t.Setenv(registryPasswordEnv, "secret")

	credentials, configured, err := registryCredentialsForImage("ghcr.io/acme/private:latest")
	if err != nil {
		t.Fatal(err)
	}
	if !configured {
		t.Fatal("expected matching registry credentials to be configured")
	}
	if credentials.host != "ghcr.io" || credentials.username != "octocat" || credentials.password != "secret" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}

	_, configured, err = registryCredentialsForImage("docker.io/library/postgres:16")
	if err != nil {
		t.Fatal(err)
	}
	if configured {
		t.Fatal("credentials must not leak to a different registry host")
	}
}

func TestRegistryCredentialsForImagesOnlyMatchReferencedRegistry(t *testing.T) {
	t.Setenv(registryHostEnv, "docker.io")
	t.Setenv(registryUsernameEnv, "mypaas-ci")
	t.Setenv(registryPasswordEnv, "secret")

	credentials, configured, err := registryCredentialsForImages([]string{
		"ghcr.io/acme/app:latest",
		"postgres:16-alpine",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !configured {
		t.Fatal("expected Docker Hub credentials to match the postgres image")
	}
	if credentials.host != "docker.io" {
		t.Fatalf("credentials host = %q, want docker.io", credentials.host)
	}

	_, configured, err = registryCredentialsForImages([]string{"ghcr.io/acme/app:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if configured {
		t.Fatal("Docker Hub credentials must not be applied to GHCR-only compose projects")
	}
}

func TestRegistryCredentialsRejectIncompleteConfiguration(t *testing.T) {
	t.Setenv(registryHostEnv, "ghcr.io")
	t.Setenv(registryUsernameEnv, "octocat")
	t.Setenv(registryPasswordEnv, "")

	_, _, err := registryCredentialsForImage("ghcr.io/acme/private:latest")
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete configuration error, got %v", err)
	}
}

func TestClassifyRegistryCommandError(t *testing.T) {
	commandErr := errors.New("exit status 1")
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "auth", output: "unauthorized: authentication required", want: "authentication failed"},
		{name: "permission", output: "denied: requested access to the resource is denied", want: "permission denied"},
		{name: "rate", output: "toomanyrequests: You have reached your pull rate limit", want: "rate limit"},
		{name: "missing", output: "manifest unknown: manifest unknown", want: "was not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyRegistryCommandError("pull", "ghcr.io/acme/app:latest", "ghcr.io", commandErr, []byte(tt.output))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("expected %q in error, got %v", tt.want, err)
			}
		})
	}
}
