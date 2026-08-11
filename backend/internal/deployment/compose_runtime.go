package deployment

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// injectComposeEnvFile adds the MyPaas-managed project env file to the generated
// main-service override. docker compose --env-file only supplies interpolation
// values; it does not automatically populate a container's runtime environment.
// Keeping this in the MyPaas override makes project env behavior consistent with
// registry/image deployments without requiring repositories to repeat env_file.
func injectComposeEnvFile(ctx context.Context, overridePath, envFile string) error {
	envFile = strings.TrimSpace(envFile)
	if envFile == "" {
		return fmt.Errorf("compose runtime env file is required")
	}

	if err := ctx.Err(); err != nil {
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
