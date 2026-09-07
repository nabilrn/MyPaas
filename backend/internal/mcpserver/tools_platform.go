package mcpserver

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	mcptransport "github.com/mark3labs/mcp-go/server"
)

func registerPlatformTools(mcpServer *mcptransport.MCPServer, s *server) {
	addTypedTool(mcpServer, mcp.NewTool("get_quota",
		mcp.WithDescription("Get quota usage for the token owner."),
	), s.getQuota)

	addTypedTool(mcpServer, mcp.NewTool("get_host_stats",
		mcp.WithDescription("Get host resource capacity and allocation stats. Requires the admin:read scope."),
	), s.getHostStats)
}

func (s *server) getQuota(ctx context.Context, _ struct{}) (*mcp.CallToolResult, error) {
	return s.call(ctx, "get_quota", http.MethodGet, "/me/quota", nil, "")
}

func (s *server) getHostStats(ctx context.Context, _ struct{}) (*mcp.CallToolResult, error) {
	return s.call(ctx, "get_host_stats", http.MethodGet, "/admin/host-stats", nil, "")
}
