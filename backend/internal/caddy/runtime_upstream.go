package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const runtimeUpstreamMode = "runtime"
const runtimeDNSIDLength = 12

type runtimeInspectRow struct {
	ID              string `json:"Id"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

// upstreamDial keeps the existing fixed-host behavior for development and
// compatibility deployments. Production can set CADDY_UPSTREAM_HOST=runtime
// to resolve the container that owns the MyPaaS allocated host port and route
// Caddy directly to that runtime on PROJECT_NETWORK. The allocated port is a
// stable lookup key; application traffic does not hairpin through a published
// host port.
func (c *Client) upstreamDial(ctx context.Context, hostPort int32) (string, error) {
	if strings.TrimSpace(c.upstreamHost) != runtimeUpstreamMode {
		return fmt.Sprintf("%s:%d", c.upstreamHost, hostPort), nil
	}

	projectNetwork := strings.TrimSpace(os.Getenv("PROJECT_NETWORK"))
	if projectNetwork == "" {
		return "", fmt.Errorf("PROJECT_NETWORK is required when CADDY_UPSTREAM_HOST=%s", runtimeUpstreamMode)
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
	return runtimeDialFromInspect(inspectRaw, projectNetwork, hostPort)
}

func runtimeDialFromInspect(raw []byte, projectNetwork string, hostPort int32) (string, error) {
	var rows []runtimeInspectRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return "", fmt.Errorf("decode runtime inspect for Caddy upstream: %w", err)
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
		if len(containerID) < runtimeDNSIDLength {
			if candidateErr == nil {
				candidateErr = fmt.Errorf("resolve Caddy upstream for host port %d: container has invalid runtime ID %q", hostPort, containerID)
			}
			continue
		}

		// Podman with netavark/aardvark-dns registers the first 12 characters of
		// every container ID as a network-scoped DNS alias. Docker compatibility
		// is continuously gated in CI. The short ID survives container rename,
		// which is required by MyPaaS rolling Dockerfile/image deployments.
		return fmt.Sprintf("%s:%s", containerID[:runtimeDNSIDLength], containerPort), nil
	}

	if candidateErr != nil {
		return "", candidateErr
	}
	return "", fmt.Errorf("resolve Caddy upstream for host port %d: no running container owns that published port", hostPort)
}
