package mcpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcptransport "github.com/mark3labs/mcp-go/server"
)

type envValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type setEnvInput struct {
	ProjectID string     `json:"projectId"`
	Vars      []envValue `json:"vars"`
}

type deleteEnvInput struct {
	ProjectID  string `json:"projectId"`
	Key        string `json:"key"`
	ConfirmKey string `json:"confirmKey"`
}

func registerEnvTools(mcpServer *mcptransport.MCPServer, s *server) {
	addTypedTool(mcpServer, mcp.NewTool("list_env_vars",
		mcp.WithDescription("List environment variable keys for a project. Secret values are not revealed."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
	), s.listEnv)

	addTypedTool(mcpServer, mcp.NewTool("set_env_vars",
		mcp.WithDescription("Set one or more project environment variables. Values are sent only to the scoped MyPaaS REST API and are not logged by MCP."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
		mcp.WithArray("vars", mcp.Required(), mcp.Description("Environment variable objects with key and value fields.")),
	), s.setEnv)

	addTypedTool(mcpServer, mcp.NewTool("delete_env_var",
		mcp.WithDescription("Delete one project environment variable. confirmKey must exactly match key."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Environment variable key.")),
		mcp.WithString("confirmKey", mcp.Required(), mcp.Description("Must exactly match key.")),
	), s.deleteEnv)
}

func (s *server) listEnv(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	return s.call(ctx, "list_env_vars", http.MethodGet, "/projects/"+pathID(projectID)+"/env", nil, "")
}

func (s *server) setEnv(ctx context.Context, in setEnvInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	if len(in.Vars) == 0 {
		return toolError("vars must contain at least one environment variable"), nil
	}
	for _, item := range in.Vars {
		if strings.TrimSpace(item.Key) == "" {
			return toolError("environment variable keys must not be empty"), nil
		}
	}
	return s.mutate(ctx, "set_env_vars", http.MethodPut, "/projects/"+pathID(projectID)+"/env", map[string]any{"vars": in.Vars}, "Environment updated")
}

func (s *server) deleteEnv(ctx context.Context, in deleteEnvInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	key, message := cleanRequired(in.Key, "key")
	if message != "" {
		return toolError(message), nil
	}
	if in.ConfirmKey != key {
		return toolError("confirmKey must exactly match key"), nil
	}
	return s.mutate(ctx, "delete_env_var", http.MethodDelete, "/projects/"+pathID(projectID)+"/env/"+pathID(key), nil, "Environment variable deleted")
}
