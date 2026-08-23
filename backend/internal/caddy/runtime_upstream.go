package caddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const runtimeUpstreamMode = "runtime"

const (
	defaultProjectNetwork = "mypaas-projects"
	defaultRoutingNetwork = "mypaas-routing"
)

type runtimeInspectRow struct {
	ID              string `json:"Id"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			Aliases []string `json:"Aliases"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type runtimeRouteTarget struct {
	ContainerID         string
	ContainerPort       string
	RoutingAttached     bool
	RoutingAliasPresent bool
}

type runtimeHealthState struct {
	Status string `json:"Status"`
}

type runtimeContainerState struct {
	Status  string              `json:"Status"`
	Running bool                `json:"Running"`
	Health  *runtimeHealthState `json:"Health"`
}

func runtimeRouteAlias(hostPort int32) string {
	return fmt.Sprintf("mypaas-port-%d", hostPort)
}

func evaluateRuntimeReadiness(state runtimeContainerState) (bool, error) {
	status := strings.ToLower(strings.TrimSpace(state.Status))
	if !state.Running {
		switch status {
		case "exited", "dead", "removing":
			return false, fmt.Errorf("runtime container status=%s", status)
		default:
			return false, nil
		}
	}

	healthStatus := ""
	if state.Health != nil {
		healthStatus = strings.ToLower(strings.TrimSpace(state.Health.Status))
	}
	if healthStatus == "" {
		return true, nil
	}

	switch healthStatus {
	case "healthy":
		return true, nil
	case "unhealthy":
		return false, fmt.Errorf("runtime container health status=unhealthy")
	default:
		return false, nil
	}
}

func runtimeState(ctx context.Context, containerID string) (runtimeContainerState, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{json .State}}", containerID).CombinedOutput()
	if err != nil {
		return runtimeContainerState{}, fmt.Errorf("inspect runtime readiness: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var state runtimeContainerState
	if err := json.Unmarshal(out, &state); err != nil {
		return runtimeContainerState{}, fmt.Errorf("decode runtime readiness: %w", err)
	}
	return state, nil
}

func waitRuntimeReady(ctx context.Context, containerID string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var last runtimeContainerState
	for {
		state, err := runtimeState(waitCtx, containerID)
		if err == nil {
			last = state
			ready, readinessErr := evaluateRuntimeReadiness(state)
			if readinessErr != nil {
				return readinessErr
			}
			if ready {
				return nil
			}
		} else if waitCtx.Err() == nil {
			return err
		}

		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				health := "none"
				if last.Health != nil && strings.TrimSpace(last.Health.Status) != "" {
					health = last.Health.Status
				}
				return fmt.Errorf("runtime readiness timeout after %s (status=%s health=%s)", timeout, last.Status, health)
			}
			return waitCtx.Err()
		case <-ticker.C:
		}
	}
}

// upstreamDial keeps the existing fixed-host behavior for development and
// compatibility deployments. Production sets CADDY_UPSTREAM_HOST=runtime. In
// that mode the allocated host port is only a runtime lookup key: the selected
// container is attached to a dedicated routing network with an explicit,
// Docker/Podman-portable DNS alias and Caddy dials that alias directly.
func (c *Client) upstreamDial(ctx context.Context, hostPort int32) (string, error) {
	if strings.TrimSpace(c.upstreamHost) != runtimeUpstreamMode {
		return fmt.Sprintf("%s:%d", c.upstreamHost, hostPort), nil
	}

	projectNetwork := strings.TrimSpace(os.Getenv("PROJECT_NETWORK"))
	if projectNetwork == "" {
		projectNetwork = defaultProjectNetwork
	}
	routingNetwork := strings.TrimSpace(os.Getenv("ROUTING_NETWORK"))
	if routingNetwork == "" {
		routingNetwork = defaultRoutingNetwork
	}
	if routingNetwork == projectNetwork {
		return "", fmt.Errorf("ROUTING_NETWORK must be distinct from PROJECT_NETWORK")
	}

	idsRaw, err := exec.CommandContext(ctx, "docker", "ps", "-q").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list running containers for Caddy upstream: %w: %s", err, strings.TrimSpace(string(idsRaw)))
	}
	ids := strings.Fields(string(idsRaw))
	if len(ids) == 0 {
		return "", fmt.Errorf("resolve Caddy upstream for host port %d: no running containers", hostPort)
	}

	args := append([]string{"inspect"}, ids...)
	inspectRaw, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect running containers for Caddy upstream: %w: %s", err, strings.TrimSpace(string(inspectRaw)))
	}

	alias := runtimeRouteAlias(hostPort)
	target, err := runtimeTargetFromInspect(inspectRaw, projectNetwork, routingNetwork, alias, hostPort)
	if err != nil {
		return "", err
	}

	// Route activation is the cutover boundary for Dockerfile/image runtimes.
	// Respect an image-defined HEALTHCHECK when present; without one, a running
	// container keeps the existing generic readiness semantics. This mirrors the
	// Compose path and avoids guessing application-specific HTTP health routes.
	if err := waitRuntimeReady(ctx, target.ContainerID, 60*time.Second); err != nil {
		return "", fmt.Errorf("runtime for host port %d is not ready: %w", hostPort, err)
	}

	if target.RoutingAttached && !target.RoutingAliasPresent {
		out, disconnectErr := exec.CommandContext(
			ctx,
			"docker",
			"network",
			"disconnect",
			routingNetwork,
			target.ContainerID,
		).CombinedOutput()
		if disconnectErr != nil {
			return "", fmt.Errorf(
				"refresh routing alias for runtime %s on network %q: disconnect: %w: %s",
				strings.TrimSpace(target.ContainerID),
				routingNetwork,
				disconnectErr,
				strings.TrimSpace(string(out)),
			)
		}
		target.RoutingAttached = false
	}

	if !target.RoutingAttached {
		out, connectErr := exec.CommandContext(
			ctx,
			"docker",
			"network",
			"connect",
			"--alias",
			alias,
			routingNetwork,
			target.ContainerID,
		).CombinedOutput()
		if connectErr != nil {
			return "", fmt.Errorf(
				"attach runtime %s to routing network %q: %w: %s",
				strings.TrimSpace(target.ContainerID),
				routingNetwork,
				connectErr,
				strings.TrimSpace(string(out)),
			)
		}
	}

	return fmt.Sprintf("%s:%s", alias, target.ContainerPort), nil
}

func runtimeTargetFromInspect(raw []byte, projectNetwork, routingNetwork, alias string, hostPort int32) (runtimeRouteTarget, error) {
	var rows []runtimeInspectRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return runtimeRouteTarget{}, fmt.Errorf("decode runtime inspect for Caddy upstream: %w", err)
	}

	wantedHostPort := strconv.Itoa(int(hostPort))
	var candidateErr error
	for _, row := range rows {
		containerPort := ""
		for portSpec, bindings := range row.NetworkSettings.Ports {
			for _, binding := range bindings {
				if strings.TrimSpace(binding.HostPort) != wantedHostPort {
					continue
				}
				portText := strings.SplitN(strings.TrimSpace(portSpec), "/", 2)[0]
				parsed, err := strconv.Atoi(portText)
				if err != nil || parsed <= 0 || parsed > 65535 {
					candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: invalid container port %q", hostPort, portSpec)
					continue
				}
				containerPort = strconv.Itoa(parsed)
				break
			}
			if containerPort != "" {
				break
			}
		}
		if containerPort == "" {
			continue
		}

		if _, ok := row.NetworkSettings.Networks[projectNetwork]; !ok {
			if candidateErr == nil {
				candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: container %s is not attached to project network %q", hostPort, strings.TrimSpace(row.ID), projectNetwork)
			}
			continue
		}

		containerID := strings.TrimSpace(row.ID)
		if containerID == "" {
			if candidateErr == nil {
				candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: matched container has empty runtime ID", hostPort)
			}
			continue
		}

		routing, routingAttached := row.NetworkSettings.Networks[routingNetwork]
		return runtimeRouteTarget{
			ContainerID:         containerID,
			ContainerPort:       containerPort,
			RoutingAttached:     routingAttached,
			RoutingAliasPresent: routingAttached && stringSliceContains(routing.Aliases, alias),
		}, nil
	}

	if candidateErr != nil {
		return runtimeRouteTarget{}, candidateErr
	}
	return runtimeRouteTarget{}, fmt.Errorf("resolve Caddy upstream for host port %d: no running container owns that published port", hostPort)
}

func stringSliceContains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
