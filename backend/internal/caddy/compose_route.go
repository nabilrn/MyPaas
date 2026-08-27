package caddy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type composeRouteInspectRow struct {
	ID              string `json:"Id"`
	NetworkSettings struct {
		Networks map[string]struct {
			Aliases []string `json:"Aliases"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func composeRouteAlias(projectName, service string) string {
	sum := sha256.Sum256([]byte(projectName + ":" + service))
	return "mypaas-http-" + hex.EncodeToString(sum[:5])
}

func (c *Client) AddComposeRoute(ctx context.Context, host, projectName, service, routeName string, containerPort int32) error {
	if strings.TrimSpace(c.upstreamHost) != runtimeUpstreamMode {
		return fmt.Errorf("additional compose HTTP routes require CADDY_UPSTREAM_HOST=runtime")
	}
	if containerPort < 1 || containerPort > 65535 {
		return fmt.Errorf("invalid compose route container port %d", containerPort)
	}
	dial, err := composeRouteDial(ctx, projectName, service, containerPort)
	if err != nil {
		return fmt.Errorf("resolve additional route %q: %w", routeName, err)
	}
	route, err := json.Marshal(map[string]any{
		"match": []map[string]any{{"host": []string{host}}},
		"handle": []map[string]any{{
			"handler": "reverse_proxy",
			"upstreams": []map[string]any{{"dial": dial}},
			"load_balancing": map[string]any{
				"try_duration": "10s",
				"try_interval": "250ms",
			},
		}},
		"terminal": true,
	})
	if err != nil {
		return err
	}
	return c.replaceHostRoute(ctx, host, route)
}

func composeRouteDial(ctx context.Context, projectName, service string, containerPort int32) (string, error) {
	projectName = strings.TrimSpace(projectName)
	service = strings.TrimSpace(service)
	if projectName == "" || service == "" {
		return "", fmt.Errorf("compose route requires project and service")
	}

	idsRaw, err := exec.CommandContext(
		ctx,
		"docker", "ps", "-q",
		"--filter", "label=com.docker.compose.project="+projectName,
		"--filter", "label=com.docker.compose.service="+service,
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("find compose route service %q: %w: %s", service, err, strings.TrimSpace(string(idsRaw)))
	}
	ids := strings.Fields(string(idsRaw))
	if len(ids) != 1 {
		return "", fmt.Errorf("resolve compose route service %q: expected one running container, found %d", service, len(ids))
	}

	inspectRaw, err := exec.CommandContext(ctx, "docker", "inspect", ids[0]).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect compose route service %q: %w: %s", service, err, strings.TrimSpace(string(inspectRaw)))
	}
	var rows []composeRouteInspectRow
	if err := json.Unmarshal(inspectRaw, &rows); err != nil || len(rows) != 1 {
		if err == nil {
			err = fmt.Errorf("expected one inspect row")
		}
		return "", fmt.Errorf("decode compose route service %q: %w", service, err)
	}
	row := rows[0]

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
	if _, ok := row.NetworkSettings.Networks[projectNetwork]; !ok {
		return "", fmt.Errorf("compose route service %q is not attached to project network %q", service, projectNetwork)
	}

	if routing, attached := row.NetworkSettings.Networks[routingNetwork]; attached {
		if alias := existingManagedRoutingAlias(routing.Aliases); alias != "" {
			return alias + ":" + strconv.Itoa(int(containerPort)), nil
		}
		return "", fmt.Errorf("compose route service %q is attached to routing network %q without a MyPaaS-managed alias", service, routingNetwork)
	}

	alias := composeRouteAlias(projectName, service)
	out, connectErr := exec.CommandContext(ctx, "docker", "network", "connect", "--alias", alias, routingNetwork, row.ID).CombinedOutput()
	if connectErr != nil {
		return "", fmt.Errorf("attach compose route service to routing network: %w: %s", connectErr, strings.TrimSpace(string(out)))
	}
	return alias + ":" + strconv.Itoa(int(containerPort)), nil
}

func existingManagedRoutingAlias(aliases []string) string {
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if strings.HasPrefix(alias, "mypaas-port-") || strings.HasPrefix(alias, "mypaas-http-") {
			return alias
		}
	}
	return ""
}
