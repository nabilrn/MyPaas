package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const runtimeUpstreamMode = "runtime"

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
// Caddy directly to that container on PROJECT_NETWORK. The allocated port is
// therefore only a stable lookup key; traffic does not hairpin through a host
// published port.
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
					return "", fmt.Errorf("resolve Caddy upstream for host port %d: invalid container port %q", hostPort, portSpec)
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

		network, ok := row.NetworkSettings.Networks[projectNetwork]
		if !ok {
			return "", fmt.Errorf("resolve Caddy upstream for host port %d: container %s is not attached to project network %q", hostPort, strings.TrimSpace(row.ID), projectNetwork)
		}
		ip := strings.TrimSpace(network.IPAddress)
		if net.ParseIP(ip) == nil {
			return "", fmt.Errorf("resolve Caddy upstream for host port %d: container %s has no valid IP on project network %q", hostPort, strings.TrimSpace(row.ID), projectNetwork)
		}
		return net.JoinHostPort(ip, containerPort), nil
	}

	return "", fmt.Errorf("resolve Caddy upstream for host port %d: no running container owns that published port", hostPort)
}
