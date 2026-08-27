package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"mypaas/internal/compose"
	"mypaas/internal/db"
	"mypaas/internal/envdiscover"
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
	project, err := s.Get(ctx, projectID)
	if err != nil {
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
	return normalizeAdditionalRoutes(project.DeployMode, routes)
}

func (s *Service) SetAdditionalRoutes(ctx context.Context, projectID uuid.UUID, routes []AdditionalRoute) ([]AdditionalRoute, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.DeployMode != "compose" {
		return nil, fmt.Errorf("%w: additional HTTP routes are only supported for compose projects", errs.ErrValidation)
	}
	if project.ActiveDeploymentID.Valid || project.Status == "running" || project.Status == "building" {
		return nil, fmt.Errorf("%w: additional HTTP routes are immutable after the first deployment; recreate the project to change the route contract", errs.ErrValidation)
	}
	normalized, err := normalizeAdditionalRoutes(project.DeployMode, routes)
	if err != nil {
		return nil, err
	}
	for _, route := range normalized {
		if project.MainService != nil && route.Service == strings.TrimSpace(*project.MainService) && route.ContainerPort == project.AppPort {
			return nil, fmt.Errorf("%w: route %q duplicates the primary project route", errs.ErrValidation, route.Name)
		}
	}
	if err := validateAdditionalRoutesForProjectSource(ctx, project, normalized); err != nil {
		return nil, err
	}
	raw, err := encodeAdditionalRoutes(normalized)
	if err != nil {
		return nil, err
	}
	if err := s.queries.SetProjectAdditionalRoutes(ctx, projectID, raw); err != nil {
		return nil, err
	}
	return normalized, nil
}

func validateAdditionalRoutesForProjectSource(ctx context.Context, project db.Project, routes []AdditionalRoute) error {
	if len(routes) == 0 {
		return nil
	}
	validateCtx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()

	root, err := os.MkdirTemp("", "mypaas-route-preflight-*")
	if err != nil {
		return fmt.Errorf("create route preflight workspace: %w", err)
	}
	defer os.RemoveAll(root)

	if err := cloneForDetect(validateCtx, root, project.RepoUrl, project.Branch); err != nil {
		return err
	}
	workspace, err := resolveLifecycleDirectory(root, valueOrEmpty(project.BaseDirectory), "base directory")
	if err != nil {
		return err
	}
	composeFile := strings.TrimSpace(valueOrEmpty(project.ComposeFilePath))
	if composeFile == "" {
		candidates, discoverErr := compose.Discover(workspace)
		if discoverErr != nil {
			return discoverErr
		}
		if len(candidates) == 0 {
			return fmt.Errorf("%w: compose file was not found for additional route validation", errs.ErrValidation)
		}
		composeFile = candidates[0].Path
	}

	envVars, err := envdiscover.Discover(workspace, composeFile)
	if err != nil {
		return fmt.Errorf("discover compose env for additional routes: %w", err)
	}
	if err := prepareComposePreviewEnv(workspace, composeFile, envVars); err != nil {
		return err
	}
	rawConfig, err := composeConfigJSON(validateCtx, workspace, composeFile)
	if err != nil {
		return err
	}
	var doc composeConfigDoc
	if err := json.Unmarshal(rawConfig, &doc); err != nil {
		return fmt.Errorf("%w: parse compose config for additional routes: %v", errs.ErrValidation, err)
	}
	return validateAdditionalRouteTargets(project, doc, routes)
}

func validateAdditionalRouteTargets(project db.Project, doc composeConfigDoc, routes []AdditionalRoute) error {
	for _, route := range routes {
		spec, ok := doc.Services[route.Service]
		if !ok {
			return fmt.Errorf("%w: route %q targets compose service %q which does not exist", errs.ErrValidation, route.Name, route.Service)
		}
		if project.MainService != nil && strings.TrimSpace(*project.MainService) == route.Service && project.AppPort == route.ContainerPort {
			return fmt.Errorf("%w: route %q duplicates the primary project route", errs.ErrValidation, route.Name)
		}
		if !composeServiceDeclaresTCPPort(spec, route.ContainerPort) {
			return fmt.Errorf("%w: route %q targets %s:%d but that TCP port is not declared by the compose service via ports or expose", errs.ErrValidation, route.Name, route.Service, route.ContainerPort)
		}
	}
	return nil
}

func composeServiceDeclaresTCPPort(spec composeServiceConfig, wanted int32) bool {
	for _, port := range composePortPlans(spec.Ports) {
		if port.Target == wanted && (port.Protocol == "" || strings.EqualFold(port.Protocol, "tcp")) {
			return true
		}
	}
	for _, port := range composeExposePorts(spec.Expose) {
		if port == wanted {
			return true
		}
	}
	return false
}
