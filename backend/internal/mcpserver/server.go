package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcptransport "github.com/mark3labs/mcp-go/server"
)

const (
	serverName        = "MyPaaS"
	serverVersion     = "1.2.0"
	maxMCPRequestBody = 1 << 20
)

type requestTokenKey struct{}

type Handler struct {
	base      *apiClient
	transport http.Handler
	requests  *fixedWindowLimiter
	mutations *fixedWindowLimiter
}

func NewHandler(apiURL string) http.Handler {
	h := &Handler{
		base:      newAPIClient(apiURL),
		requests:  newLimiter(240, time.Minute),
		mutations: newLimiter(30, time.Minute),
	}

	toolServer := newServer(h.base, h.mutations)
	h.transport = mcptransport.NewStreamableHTTPServer(
		toolServer,
		mcptransport.WithStateLess(true),
		mcptransport.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return context.WithValue(ctx, requestTokenKey{}, bearerToken(r))
		}),
	)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if !strings.HasPrefix(token, "myp_") {
		unauthorized(w)
		return
	}
	if !h.requests.allow(token) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "MCP request rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Validate the machine token at the REST authorization boundary before the
	// MCP transport handles initialize/tools/list. Revoked and expired keys fail
	// immediately rather than appearing connected until the first tool call.
	if _, err := h.base.withToken(token).request(r.Context(), http.MethodGet, "/auth/me", nil); err != nil {
		var apiErr *apiHTTPError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden) {
			unauthorized(w)
			return
		}
		http.Error(w, "MyPaaS API is unavailable", http.StatusBadGateway)
		return
	}

	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, maxMCPRequestBody)
	}
	h.transport.ServeHTTP(w, r)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="MyPaaS MCP"`)
	http.Error(w, "valid MyPaaS API token required", http.StatusUnauthorized)
}

func bearerToken(r *http.Request) string {
	parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func newServer(base *apiClient, mutations *fixedWindowLimiter) *mcptransport.MCPServer {
	s := &server{base: base, mutations: mutations}
	mcpServer := mcptransport.NewMCPServer(
		serverName,
		serverVersion,
		mcptransport.WithToolCapabilities(true),
	)
	registerProjectTools(mcpServer, s)
	registerDeploymentTools(mcpServer, s)
	registerEnvTools(mcpServer, s)
	registerPlatformTools(mcpServer, s)
	return mcpServer
}

type server struct {
	base      *apiClient
	mutations *fixedWindowLimiter
}

type typedToolHandler[T any] func(context.Context, T) (*mcp.CallToolResult, error)

func addTypedTool[T any](mcpServer *mcptransport.MCPServer, tool mcp.Tool, handler typedToolHandler[T]) {
	mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input T
		data, err := json.Marshal(request.GetArguments())
		if err != nil {
			return toolError("invalid tool arguments"), nil
		}
		if err := json.Unmarshal(data, &input); err != nil {
			return toolError("invalid tool arguments: " + err.Error()), nil
		}
		return handler(ctx, input)
	})
}

func tokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(requestTokenKey{}).(string)
	return strings.TrimSpace(token)
}

func (s *server) call(ctx context.Context, tool, method, path string, payload any, prefix string) (*mcp.CallToolResult, error) {
	token := tokenFromContext(ctx)
	if token == "" {
		return toolError("missing MCP bearer token"), nil
	}
	response, err := s.base.withToken(token).withTool(tool).request(ctx, method, path, payload)
	if err != nil {
		return toolError(err.Error()), nil
	}
	if prefix != "" {
		response = prefix + ": " + response
	}
	return toolText(response), nil
}

func (s *server) mutate(ctx context.Context, tool, method, path string, payload any, prefix string) (*mcp.CallToolResult, error) {
	token := tokenFromContext(ctx)
	if token == "" {
		return toolError("missing MCP bearer token"), nil
	}
	if !s.mutations.allow(token) {
		return toolError("MCP mutation rate limit exceeded; retry after the current minute window."), nil
	}
	return s.call(ctx, tool, method, path, payload, prefix)
}

func toolText(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

func toolError(text string) *mcp.CallToolResult {
	return mcp.NewToolResultError(text)
}
