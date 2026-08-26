package project

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"mypaas/internal/errs"
)

const maxAdditionalRoutes = 4

var additionalRouteNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,18}[a-z0-9])?$`)

type AdditionalRoute struct {
	Name          string `json:"name"`
	Service       string `json:"service"`
	ContainerPort int32  `json:"containerPort"`
}

func normalizeAdditionalRoutes(deployMode string, routes []AdditionalRoute) ([]AdditionalRoute, error) {
	if len(routes) == 0 {
		return []AdditionalRoute{}, nil
	}
	if deployMode != "compose" {
		return nil, fmt.Errorf("%w: additional HTTP routes are only supported for compose projects", errs.ErrValidation)
	}
	if len(routes) > maxAdditionalRoutes {
		return nil, fmt.Errorf("%w: at most %d additional HTTP routes are allowed", errs.ErrValidation, maxAdditionalRoutes)
	}

	seen := make(map[string]struct{}, len(routes))
	normalized := make([]AdditionalRoute, 0, len(routes))
	for _, route := range routes {
		name := strings.ToLower(strings.TrimSpace(route.Name))
		service := strings.TrimSpace(route.Service)
		if !additionalRouteNamePattern.MatchString(name) {
			return nil, fmt.Errorf("%w: route name %q must use 1-20 lowercase letters, numbers, or dashes", errs.ErrValidation, route.Name)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("%w: duplicate additional route name %q", errs.ErrValidation, name)
		}
		if service == "" {
			return nil, fmt.Errorf("%w: route %q requires a compose service", errs.ErrValidation, name)
		}
		if route.ContainerPort < 1 || route.ContainerPort > 65535 {
			return nil, fmt.Errorf("%w: route %q container port must be between 1 and 65535", errs.ErrValidation, name)
		}
		seen[name] = struct{}{}
		normalized = append(normalized, AdditionalRoute{Name: name, Service: service, ContainerPort: route.ContainerPort})
	}
	return normalized, nil
}

func encodeAdditionalRoutes(routes []AdditionalRoute) (json.RawMessage, error) {
	if routes == nil {
		routes = []AdditionalRoute{}
	}
	raw, err := json.Marshal(routes)
	if err != nil {
		return nil, fmt.Errorf("encode additional HTTP routes: %w", err)
	}
	return raw, nil
}

func decodeAdditionalRoutes(raw json.RawMessage) ([]AdditionalRoute, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return []AdditionalRoute{}, nil
	}
	var routes []AdditionalRoute
	if err := json.Unmarshal(raw, &routes); err != nil {
		return nil, fmt.Errorf("decode additional HTTP routes: %w", err)
	}
	return routes, nil
}

func (s *Service) AdditionalRoutes(ctx context.Context, projectID uuid.UUID) ([]AdditionalRoute, error) {
	if _, err := s.Get(ctx, projectID); err != nil {
		return nil, err
	}
	raw, err := s.queries.GetProjectAdditionalRoutes(ctx, projectID)
	if err != nil {
		return nil, err
	}
	routes, err := decodeAdditionalRoutes(raw)
	if err != nil {
		return nil, err
	}
	return normalizeAdditionalRoutes("compose", routes)
}
