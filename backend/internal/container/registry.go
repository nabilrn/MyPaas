package container

import (
	"context"
	"fmt"
	"os"
	"strings"
)

const (
	registryHostEnv     = "MYPAAS_REGISTRY_HOST"
	registryUsernameEnv = "MYPAAS_REGISTRY_USERNAME"
	registryPasswordEnv = "MYPAAS_REGISTRY_PASSWORD"
)

type registryCredentials struct {
	host     string
	username string
	password string
}

func (d *DockerCLI) pullImage(ctx context.Context, image string, log func(string)) error {
	credentials, configured, err := registryCredentialsForImage(image)
	if err != nil {
		return err
	}
	if !configured {
		return runRegistryPull(ctx, image, "", log)
	}

	dockerConfig, cleanup, err := prepareRegistryDockerConfig(ctx, credentials, log)
	if err != nil {
		return err
	}
	defer cleanup()

	return runRegistryPull(ctx, image, dockerConfig, log)
}

func configuredRegistryCredentials() (registryCredentials, bool, error) {
	configuredHost := normalizeRegistryHost(os.Getenv(registryHostEnv))
	username := strings.TrimSpace(os.Getenv(registryUsernameEnv))
	password := os.Getenv(registryPasswordEnv)
	if configuredHost == "" && username == "" && password == "" {
		return registryCredentials{}, false, nil
	}
	if configuredHost == "" || username == "" || password == "" {
		return registryCredentials{}, false, fmt.Errorf("private registry configuration is incomplete: %s, %s, and %s must be set together", registryHostEnv, registryUsernameEnv, registryPasswordEnv)
	}
	return registryCredentials{host: configuredHost, username: username, password: password}, true, nil
}

func registryCredentialsForImage(image string) (registryCredentials, bool, error) {
	credentials, configured, err := configuredRegistryCredentials()
	if err != nil || !configured {
		return registryCredentials{}, configured, err
	}
	if credentials.host != registryHost(image) {
		return registryCredentials{}, false, nil
	}
	return credentials, true, nil
}

func registryCredentialsForImages(images []string) (registryCredentials, bool, error) {
	credentials, configured, err := configuredRegistryCredentials()
	if err != nil || !configured {
		return registryCredentials{}, configured, err
	}
	for _, image := range images {
		if credentials.host == registryHost(image) {
			return credentials, true, nil
		}
	}
	return registryCredentials{}, false, nil
}

func prepareRegistryDockerConfig(ctx context.Context, credentials registryCredentials, log func(string)) (string, func(), error) {
	dockerConfig, err := os.MkdirTemp("", "mypaas-registry-auth-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create isolated registry auth directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dockerConfig) }
	if err := os.Chmod(dockerConfig, 0700); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("secure isolated registry auth directory: %w", err)
	}

	login := commandContext(ctx, "docker", "login", credentials.host, "--username", credentials.username, "--password-stdin")
	login.Env = dockerEnvWithConfig(dockerConfig)
	login.Stdin = strings.NewReader(credentials.password + "\n")
	loginOutput, loginErr := login.CombinedOutput()
	logRegistryOutput(loginOutput, log)
	if loginErr != nil {
		cleanup()
		return "", func() {}, classifyRegistryCommandError("login", credentials.host, credentials.host, loginErr, loginOutput)
	}
	if log != nil {
		log("Using configured registry credentials for " + credentials.host)
	}
	return dockerConfig, cleanup, nil
}

func registryHost(image string) string {
	image = strings.TrimSpace(image)
	first, _, hasSlash := strings.Cut(image, "/")
	if !hasSlash {
		return "docker.io"
	}
	first = strings.ToLower(strings.TrimSpace(first))
	if first == "" || (!strings.Contains(first, ".") && !strings.Contains(first, ":") && first != "localhost") {
		return "docker.io"
	}
	return normalizeRegistryHost(first)
}

func normalizeRegistryHost(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimSuffix(value, "/")
	if slash := strings.IndexByte(value, '/'); slash >= 0 {
		value = value[:slash]
	}
	switch value {
	case "index.docker.io", "registry-1.docker.io":
		return "docker.io"
	default:
		return value
	}
}

func runRegistryPull(ctx context.Context, image, dockerConfig string, log func(string)) error {
	cmd := commandContext(ctx, "docker", "pull", image)
	if dockerConfig != "" {
		cmd.Env = dockerEnvWithConfig(dockerConfig)
	}
	output, err := cmd.CombinedOutput()
	logRegistryOutput(output, log)
	if err != nil {
		return classifyRegistryCommandError("pull", image, registryHost(image), err, output)
	}
	return nil
}

func dockerEnvWithConfig(path string) []string {
	return envWithValue(dockerEnv(), "DOCKER_CONFIG", path)
}

func envWithValue(base []string, wantedKey, value string) []string {
	out := make([]string, 0, len(base)+1)
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(key, wantedKey) {
			continue
		}
		out = append(out, item)
	}
	return append(out, wantedKey+"="+value)
}

func logRegistryOutput(output []byte, log func(string)) {
	if log == nil {
		return
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			log(line)
		}
	}
}

func classifyRegistryCommandError(operation, image, host string, commandErr error, output []byte) error {
	message := strings.ToLower(string(output))
	switch {
	case strings.Contains(message, "toomanyrequests"), strings.Contains(message, "rate limit"), strings.Contains(message, "429 too many requests"):
		return fmt.Errorf("registry rate limit from %s while pulling %s; configure registry credentials or retry later: %w", host, image, commandErr)
	case strings.Contains(message, "unauthorized"), strings.Contains(message, "authentication required"), strings.Contains(message, "incorrect username or password"):
		return fmt.Errorf("registry authentication failed for %s while pulling %s: %w", host, image, commandErr)
	case strings.Contains(message, "denied"), strings.Contains(message, "insufficient_scope"), strings.Contains(message, "requested access to the resource is denied"):
		return fmt.Errorf("registry permission denied for %s while pulling %s: %w", host, image, commandErr)
	case strings.Contains(message, "manifest unknown"), strings.Contains(message, "manifest for") && strings.Contains(message, "not found"), strings.Contains(message, "no such manifest"):
		return fmt.Errorf("container image or tag %s was not found in %s: %w", image, host, commandErr)
	default:
		detail := firstRegistryErrorLine(output)
		if detail == "" {
			return fmt.Errorf("docker registry %s failed for %s: %w", operation, image, commandErr)
		}
		return fmt.Errorf("docker registry %s failed for %s: %w: %s", operation, image, commandErr, detail)
	}
}

func firstRegistryErrorLine(output []byte) string {
	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
