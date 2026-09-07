package mcpserver

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	mcptransport "github.com/mark3labs/mcp-go/server"
)

type listDeploymentsInput struct {
	ProjectID string `json:"projectId"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type logsInput struct {
	ProjectID string `json:"projectId"`
	Tail      int    `json:"tail,omitempty"`
}

func registerDeploymentTools(mcpServer *mcptransport.MCPServer, s *server) {
	for _, action := range []struct {
		name        string
		description string
		handler     typedToolHandler[projectIDInput]
	}{
		{"deploy_project", "Trigger a deployment for a project.", s.deployProject},
		{"start_project", "Start a stopped project.", s.startProject},
		{"stop_project", "Stop a running project.", s.stopProject},
		{"restart_project", "Restart a running project.", s.restartProject},
	} {
		addTypedTool(mcpServer, mcp.NewTool(action.name,
			mcp.WithDescription(action.description),
			mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
		), action.handler)
	}

	addTypedTool(mcpServer, mcp.NewTool("list_deployments",
		mcp.WithDescription("List deployments for a project."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
		mcp.WithNumber("limit", mcp.Description("Maximum rows, default 20 and max 100.")),
		mcp.WithNumber("offset", mcp.Description("Row offset.")),
	), s.listDeployments)

	addTypedTool(mcpServer, mcp.NewTool("get_deployment",
		mcp.WithDescription("Get one deployment by UUID."),
		mcp.WithString("deploymentId", mcp.Required(), mcp.Description("MyPaaS deployment UUID.")),
	), s.getDeployment)

	addTypedTool(mcpServer, mcp.NewTool("rollback_deployment",
		mcp.WithDescription("Rollback to a successful deployment."),
		mcp.WithString("deploymentId", mcp.Required(), mcp.Description("MyPaaS deployment UUID.")),
	), s.rollbackDeployment)

	addTypedTool(mcpServer, mcp.NewTool("get_logs",
		mcp.WithDescription("Get recent project logs."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
		mcp.WithNumber("tail", mcp.Description("Number of recent log lines, default 500 and max 5000.")),
	), s.getLogs)

	addTypedTool(mcpServer, mcp.NewTool("get_metrics_snapshot",
		mcp.WithDescription("Get the current project runtime metrics snapshot."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
	), s.getMetrics)
}

func (s *server) projectAction(ctx context.Context, tool, action, success string, in projectIDInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	return s.mutate(ctx, tool, http.MethodPost, "/projects/"+pathID(projectID)+"/"+action, nil, success)
}

func (s *server) deployProject(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	return s.projectAction(ctx, "deploy_project", "deploy", "Deployment triggered", in)
}

func (s *server) startProject(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	return s.projectAction(ctx, "start_project", "start", "Project started", in)
}

func (s *server) stopProject(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	return s.projectAction(ctx, "stop_project", "stop", "Project stopped", in)
}

func (s *server) restartProject(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	return s.projectAction(ctx, "restart_project", "restart", "Project restarted", in)
}

func (s *server) listDeployments(ctx context.Context, in listDeploymentsInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return toolError("limit must be between 1 and 100"), nil
	}
	if in.Offset < 0 {
		return toolError("offset must not be negative"), nil
	}
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(in.Offset))
	return s.call(ctx, "list_deployments", http.MethodGet, "/projects/"+pathID(projectID)+"/deployments?"+query.Encode(), nil, "")
}

func (s *server) getDeployment(ctx context.Context, in deploymentIDInput) (*mcp.CallToolResult, error) {
	deploymentID, message := cleanRequired(in.DeploymentID, "deploymentId")
	if message != "" {
		return toolError(message), nil
	}
	return s.call(ctx, "get_deployment", http.MethodGet, "/deployments/"+pathID(deploymentID), nil, "")
}

func (s *server) rollbackDeployment(ctx context.Context, in deploymentIDInput) (*mcp.CallToolResult, error) {
	deploymentID, message := cleanRequired(in.DeploymentID, "deploymentId")
	if message != "" {
		return toolError(message), nil
	}
	return s.mutate(ctx, "rollback_deployment", http.MethodPost, "/deployments/"+pathID(deploymentID)+"/rollback", nil, "Rollback triggered")
}

func (s *server) getLogs(ctx context.Context, in logsInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	tail := in.Tail
	if tail == 0 {
		tail = 500
	}
	if tail < 1 || tail > 5000 {
		return toolError("tail must be between 1 and 5000"), nil
	}
	query := url.Values{}
	query.Set("tail", strconv.Itoa(tail))
	return s.call(ctx, "get_logs", http.MethodGet, "/projects/"+pathID(projectID)+"/logs?"+query.Encode(), nil, "")
}

func (s *server) getMetrics(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	return s.call(ctx, "get_metrics_snapshot", http.MethodGet, "/projects/"+pathID(projectID)+"/metrics", nil, "")
}
