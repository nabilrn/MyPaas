package deployment

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const composeParallelLimitKey = "COMPOSE_PARALLEL_LIMIT"

// ensureComposeParallelLimit keeps repository-driven Compose deployments safe
// on small hosts. Docker Compose defaults to unlimited parallel engine calls,
// which can build multiple services concurrently and exhaust the VM's process
// or thread budget. Respect an explicit project value, otherwise default to 1.
func ensureComposeParallelLimit(envFile string) error {
	raw, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("read compose env file: %w", err)
	}
	content := string(raw)
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == composeParallelLimitKey {
			return nil
		}
	}

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += composeParallelLimitKey + "=1\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write compose env file: %w", err)
	}
	return nil
}

// injectComposeEnvFile adds the MyPaas-managed project env file to the generated
// main-service override. docker compose --env-file supplies interpolation values
// and Compose control variables such as COMPOSE_PARALLEL_LIMIT; it does not
// automatically populate a container's runtime environment. Keeping env_file in
// the MyPaas override makes project env behavior consistent with registry/image
// deployments without requiring repositories to repeat env_file.
func injectComposeEnvFile(ctx context.Context, overridePath, envFile string) error {
	envFile = strings.TrimSpace(envFile)
	if envFile == "" {
		return fmt.Errorf("compose runtime env file is required")
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ensureComposeParallelLimit(envFile); err != nil {
		return err
	}
	raw, err := os.ReadFile(overridePath)
	if err != nil {
		return fmt.Errorf("read compose override: %w", err)
	}
	content := string(raw)
	if strings.Contains(content, "\n    env_file:\n") {
		return nil
	}

	const marker = "\n    ports:\n"
	index := strings.Index(content, marker)
	if index < 0 {
		return fmt.Errorf("compose override does not contain main-service ports block")
	}

	block := "\n    env_file:\n      - " + strconv.Quote(envFile)
	content = content[:index] + block + content[index:]
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.WriteFile(overridePath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write compose override env file: %w", err)
	}
	return nil
}
