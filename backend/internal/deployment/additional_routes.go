package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"mypaas/internal/db"
	projectpkg "mypaas/internal/project"
)

func (s *Service) ReconcileAllRoutes(ctx context.Context) error {
	return errors.Join(s.ReconcileRoutes(ctx), s.ReconcileAdditionalRoutes(ctx))
}

func (s *Service) ReconcileAdditionalRoutes(ctx context.Context) error {
	configs, err := s.queries.ListProjectAdditionalRouteConfigs(ctx)
	if err != nil {
		return fmt.Errorf("list additional project routes: %w", err)
	}

	var firstErr error
	for _, config := range configs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.reconcileAdditionalRouteConfig(ctx, config); err != nil {
			slog.Warn("reconcile additional caddy routes failed", "projectId", config.ID, "name", config.Name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *Service) reconcileAdditionalRouteConfig(ctx context.Context, config db.ProjectAdditionalRouteConfig) error {
	var routes []projectpkg.AdditionalRoute
	if err := json.Unmarshal(config.Routes, &routes); err != nil {
		return fmt.Errorf("decode additional routes for project %s: %w", config.Name, err)
	}

	shouldServe := !config.Deleted && config.Status == "running" && config.DeployMode == "compose"
	var firstErr error
	for _, route := range routes {
		host := s.additionalRouteHost(config.Name, route.Name)
		if !shouldServe {
			if err := s.caddy.RemoveRoute(ctx, host); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if route.Name == "" || strings.TrimSpace(route.Service) == "" || route.ContainerPort < 1 || route.ContainerPort > 65535 {
			if firstErr == nil {
				firstErr = fmt.Errorf("invalid persisted additional route for project %s", config.Name)
			}
			continue
		}
		if err := s.caddy.AddComposeRoute(ctx, host, composeProjectName(config.Name), route.Service, route.Name, route.ContainerPort); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if config.Deleted && firstErr == nil {
		if err := s.queries.ClearProjectAdditionalRoutes(ctx, config.ID); err != nil {
			return err
		}
	}
	return firstErr
}

func (s *Service) additionalRouteHost(projectName, routeName string) string {
	domain := strings.TrimSpace(s.cfg.PublicDomain)
	if domain == "" {
		domain = "localhost"
	}
	return projectName + "-" + routeName + "." + domain
}
