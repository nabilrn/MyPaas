package gitremote

import (
	"context"
	"encoding/base64"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// CommandContext supplies a one-process GitHub credential without writing it
// to the repository's .git/config or exposing it in command arguments.
func CommandContext(ctx context.Context, repoURL, accessToken string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	host, ok := githubHTTPSHost(repoURL)
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" || !ok {
		return cmd
	}
	cmd.Env = setEnv(os.Environ(), map[string]string{
		"GIT_CONFIG_COUNT":    "1",
		"GIT_CONFIG_KEY_0":    "http.https://" + host + "/.extraheader",
		"GIT_CONFIG_VALUE_0":  githubBasicAuthorizationHeader(accessToken),
		"GIT_TERMINAL_PROMPT": "0",
	})
	return cmd
}

func githubBasicAuthorizationHeader(accessToken string) string {
	credential := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + accessToken))
	return "AUTHORIZATION: basic " + credential
}

func setEnv(env []string, values map[string]string) []string {
	filtered := make([]string, 0, len(env)+len(values))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replace := values[key]; replace {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	for key, value := range values {
		filtered = append(filtered, key+"="+value)
	}
	return filtered
}

func IsGitHubURL(rawURL string) bool {
	_, ok := githubHTTPSHost(rawURL)
	return ok
}

func githubHTTPSHost(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" {
		return "", false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return "", false
	}
	return host, true
}
