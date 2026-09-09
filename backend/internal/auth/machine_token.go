package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"mypaas/internal/apitoken"
)

const machineTokenContextKey contextKey = "machine-token"

type MachineToken struct {
	ID     uuid.UUID
	Name   string
	Prefix string
	Scopes []string
}

func withMachineToken(ctx context.Context, token apitoken.Token) context.Context {
	return context.WithValue(ctx, machineTokenContextKey, MachineToken{
		ID:     token.ID,
		Name:   token.Name,
		Prefix: token.Prefix,
		Scopes: append([]string(nil), token.Scopes...),
	})
}

func MachineTokenFromRequest(r *http.Request) (MachineToken, bool) {
	value, ok := r.Context().Value(machineTokenContextKey).(MachineToken)
	return value, ok
}

func machineTokenAllows(r *http.Request, scopes []string) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/") {
		path = strings.TrimPrefix(path, "/api")
	} else if path == "/api" {
		path = "/"
	}
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}

	if r.Method == http.MethodGet && path == "/auth/me" {
		return true
	}
	if r.Method == http.MethodGet && path == "/me/quota" {
		return apitoken.HasScope(scopes, "projects:read")
	}
	if r.Method == http.MethodGet && path == "/admin/host-stats" {
		return apitoken.HasScope(scopes, "admin:read")
	}

	if path == "/projects" {
		switch r.Method {
		case http.MethodGet:
			return apitoken.HasScope(scopes, "projects:read")
		case http.MethodPost:
			return apitoken.HasScope(scopes, "projects:write")
		default:
			return false
		}
	}
	if (path == "/projects/detect-mode" || path == "/projects/detect-compose") && r.Method == http.MethodPost {
		return apitoken.HasScope(scopes, "projects:read")
	}
	if !strings.HasPrefix(path, "/projects/") && !strings.HasPrefix(path, "/deployments/") {
		return false
	}

	if strings.HasPrefix(path, "/deployments/") {
		if strings.HasSuffix(path, "/rollback") && r.Method == http.MethodPost {
			return apitoken.HasScope(scopes, "deployments:write")
		}
		return r.Method == http.MethodGet && apitoken.HasScope(scopes, "deployments:read")
	}

	// Never expose DB Studio, secret reveal, route/firewall mutation, project deletion,
	// webhook secret management, shell, backup, update, or other owner surfaces to
	// machine tokens. New API routes therefore fail closed until explicitly mapped.
	if strings.Contains(path, "/db/") || strings.HasSuffix(path, "/db") ||
		strings.Contains(path, "/env/") && strings.HasSuffix(path, "/reveal") ||
		strings.Contains(path, "/routes") ||
		strings.Contains(path, "/webhook") ||
		strings.Contains(path, "/compose-resources") ||
		strings.Contains(path, "/analytics") ||
		strings.Contains(path, "/stream") {
		return false
	}

	if strings.HasSuffix(path, "/deploy") || strings.HasSuffix(path, "/start") || strings.HasSuffix(path, "/stop") || strings.HasSuffix(path, "/restart") {
		return r.Method == http.MethodPost && apitoken.HasScope(scopes, "deployments:write")
	}
	if strings.HasSuffix(path, "/deployments") {
		return r.Method == http.MethodGet && apitoken.HasScope(scopes, "deployments:read")
	}
	if strings.HasSuffix(path, "/logs") {
		return r.Method == http.MethodGet && apitoken.HasScope(scopes, "logs:read")
	}
	if strings.HasSuffix(path, "/metrics") {
		return r.Method == http.MethodGet && apitoken.HasScope(scopes, "metrics:read")
	}
	if strings.HasSuffix(path, "/env") {
		switch r.Method {
		case http.MethodGet:
			return apitoken.HasScope(scopes, "env:read")
		case http.MethodPut:
			return apitoken.HasScope(scopes, "env:write")
		default:
			return false
		}
	}
	if strings.Contains(path, "/env/") && r.Method == http.MethodDelete {
		return apitoken.HasScope(scopes, "env:write")
	}

	// Remaining /projects/{id} requests are limited to read/update only.
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 2 && segments[0] == "projects" {
		switch r.Method {
		case http.MethodGet:
			return apitoken.HasScope(scopes, "projects:read")
		case http.MethodPatch:
			return apitoken.HasScope(scopes, "projects:write")
		default:
			return false
		}
	}
	return false
}
