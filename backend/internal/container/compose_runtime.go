package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var ErrComposeServiceUnhealthy = errors.New("compose service unhealthy")

type composeHealthState struct {
	Status string `json:"Status"`
}

type composeContainerState struct {
	Status  string              `json:"Status"`
	Running bool                `json:"Running"`
	Health  *composeHealthState `json:"Health"`
}

func composePullArgs(opts ComposeUpOptions) []string {
	args := composeBaseArgs(opts.EnvFile)
	args = append(args, "-p", opts.ProjectName)
	for _, file := range composeUpFiles(opts) {
		args = append(args, "-f", file)
	}
	return append(args, "pull", "--ignore-buildable")
}

// ComposePull refreshes remote image-only services while leaving buildable
// services alone. This gives repository/Compose deployments the same explicit
// registry refresh behavior as image deployments without trying to pull local
// MyPaas build tags.
func (d *DockerCLI) ComposePull(ctx context.Context, opts ComposeUpOptions, log func(string)) error {
	cmd := commandContext(ctx, "docker", composePullArgs(opts)...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	if len(opts.Profiles) > 0 {
		cmd.Env = append(os.Environ(), "COMPOSE_PROFILES="+strings.Join(opts.Profiles, ","))
	}
	return runLoggedCmd(ctx, cmd, log)
}

func evaluateComposeReadiness(state composeContainerState) (bool, error) {
	status := strings.ToLower(strings.TrimSpace(state.Status))
	if !state.Running {
		switch status {
		case "exited", "dead", "removing":
			return false, fmt.Errorf("%w: container status=%s", ErrComposeServiceUnhealthy, status)
		default:
			return false, nil
		}
	}

	if state.Health == nil {
		return true, nil
	}
	switch strings.ToLower(strings.TrimSpace(state.Health.Status)) {
	case "healthy":
		return true, nil
	case "unhealthy":
		return false, fmt.Errorf("%w: health status=unhealthy", ErrComposeServiceUnhealthy)
	default:
		return false, nil
	}
}

func (d *DockerCLI) composeServiceState(ctx context.Context, projectName, service string) (composeContainerState, error) {
	id, err := d.composeServiceContainer(ctx, projectName, service)
	if err != nil {
		return composeContainerState{}, err
	}
	out, err := commandContext(ctx, "docker", "inspect", "--format", "{{json .State}}", id).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if isNoContainerMessage(message) {
			return composeContainerState{}, ErrNoContainer
		}
		return composeContainerState{}, fmt.Errorf("docker inspect compose service: %w: %s", err, message)
	}

	var state composeContainerState
	if err := json.Unmarshal(out, &state); err != nil {
		return composeContainerState{}, fmt.Errorf("decode compose service state: %w", err)
	}
	return state, nil
}

// WaitComposeServiceReady blocks route activation until the selected public
// service is actually usable. Containers without a healthcheck are ready once
// Docker reports them running; containers with a healthcheck must be healthy.
func (d *DockerCLI) WaitComposeServiceReady(ctx context.Context, projectName, service string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var last composeContainerState
	for {
		state, err := d.composeServiceState(waitCtx, projectName, service)
		if err == nil {
			last = state
			ready, readinessErr := evaluateComposeReadiness(state)
			if readinessErr != nil {
				return fmt.Errorf("compose service %q readiness failed: %w", service, readinessErr)
			}
			if ready {
				return nil
			}
		} else if !errors.Is(err, ErrNoContainer) {
			return err
		}

		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				health := "none"
				if last.Health != nil {
					health = last.Health.Status
				}
				return fmt.Errorf("compose service %q readiness timeout after %s (status=%s health=%s)", service, timeout, last.Status, health)
			}
			return waitCtx.Err()
		case <-ticker.C:
		}
	}
}
