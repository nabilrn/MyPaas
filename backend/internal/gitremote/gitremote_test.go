package gitremote

import (
	"context"
	"strings"
	"testing"
)

func TestCommandContextUsesEphemeralGitHubCredential(t *testing.T) {
	const token = "github-token-for-test"
	cmd := CommandContext(context.Background(), "https://github.com/acme/app.git", token, "clone", "https://github.com/acme/app.git", ".")

	if strings.Contains(strings.Join(cmd.Args, " "), token) {
		t.Fatal("GitHub access token must not be present in command arguments")
	}

	joinedEnv := strings.Join(cmd.Env, "\n")
	for _, want := range []string{
		"GIT_CONFIG_KEY_0=http.https://github.com/.extraheader",
		"GIT_CONFIG_VALUE_0=AUTHORIZATION: bearer " + token,
		"GIT_TERMINAL_PROMPT=0",
	} {
		if !strings.Contains(joinedEnv, want) {
			t.Fatalf("command environment does not contain %q", want)
		}
	}
}

func TestCommandContextUsesWWWGitHubCredentialScope(t *testing.T) {
	const token = "github-token-for-test"
	cmd := CommandContext(context.Background(), "https://www.github.com/acme/app.git", token, "ls-remote", "https://www.github.com/acme/app.git")

	joinedEnv := strings.Join(cmd.Env, "\n")
	if !strings.Contains(joinedEnv, "GIT_CONFIG_KEY_0=http.https://www.github.com/.extraheader") {
		t.Fatalf("command environment does not scope the credential to www.github.com: %v", cmd.Env)
	}
	if strings.Contains(strings.Join(cmd.Args, " "), token) {
		t.Fatal("GitHub access token must not be present in command arguments")
	}
}

func TestCommandContextDoesNotAttachGitHubCredentialToOtherHosts(t *testing.T) {
	cmd := CommandContext(context.Background(), "https://gitlab.com/acme/app.git", "github-token-for-test", "clone")

	if len(cmd.Env) != 0 {
		t.Fatalf("expected no credential environment for non-GitHub repository, got %v", cmd.Env)
	}
}

func TestSetEnvReplacesExistingKeys(t *testing.T) {
	got := setEnv([]string{"GIT_CONFIG_COUNT=9", "OTHER=value"}, map[string]string{"GIT_CONFIG_COUNT": "1"})
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "GIT_CONFIG_COUNT=9") || !strings.Contains(joined, "GIT_CONFIG_COUNT=1") {
		t.Fatalf("setEnv() = %v, want replaced GIT_CONFIG_COUNT", got)
	}
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "github", url: "https://github.com/acme/app.git", want: true},
		{name: "www github", url: "https://www.github.com/acme/app.git", want: true},
		{name: "ssh", url: "git@github.com:acme/app.git", want: false},
		{name: "other host", url: "https://gitlab.com/acme/app.git", want: false},
		{name: "insecure", url: "http://github.com/acme/app.git", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGitHubURL(tt.url); got != tt.want {
				t.Fatalf("IsGitHubURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
