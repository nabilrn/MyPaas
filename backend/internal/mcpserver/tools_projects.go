package mcpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcptransport "github.com/mark3labs/mcp-go/server"
)

type createProjectInput struct {
	Name                 string   `json:"name"`
	SourceType           string   `json:"sourceType,omitempty"`
	RepoURL              string   `json:"repoUrl,omitempty"`
	ImageRef             string   `json:"imageRef,omitempty"`
	Branch               string   `json:"branch,omitempty"`
	DeployMode           string   `json:"deployMode,omitempty"`
	ResourceProfile      string   `json:"resourceProfile,omitempty"`
	AppPort              *int     `json:"appPort,omitempty"`
	MemoryLimitMB        *int     `json:"memoryLimitMb,omitempty"`
	CPULimit             *float64 `json:"cpuLimit,omitempty"`
	SharedPostgres       *bool    `json:"sharedPostgres,omitempty"`
	MainService          string   `json:"mainService,omitempty"`
	BaseDirectory        string   `json:"baseDirectory,omitempty"`
	StaticFrontendPath   string   `json:"staticFrontendPath,omitempty"`
	ComposeFilePath      string   `json:"composeFilePath,omitempty"`
	ComposeOverridePaths []string `json:"composeOverridePaths,omitempty"`
	ComposeProfiles      []string `json:"composeProfiles,omitempty"`
	ComposeWorkdir       string   `json:"composeWorkdir,omitempty"`
}

type updateProjectInput struct {
	ProjectID            string    `json:"projectId"`
	Branch               *string   `json:"branch,omitempty"`
	ImageRef             *string   `json:"imageRef,omitempty"`
	ResourceProfile      *string   `json:"resourceProfile,omitempty"`
	AppPort              *int      `json:"appPort,omitempty"`
	MemoryLimitMB        *int      `json:"memoryLimitMb,omitempty"`
	CPULimit             *float64  `json:"cpuLimit,omitempty"`
	MainService          *string   `json:"mainService,omitempty"`
	BaseDirectory        *string   `json:"baseDirectory,omitempty"`
	StaticFrontendPath   *string   `json:"staticFrontendPath,omitempty"`
	ComposeFilePath      *string   `json:"composeFilePath,omitempty"`
	ComposeOverridePaths *[]string `json:"composeOverridePaths,omitempty"`
	ComposeProfiles      *[]string `json:"composeProfiles,omitempty"`
	ComposeWorkdir       *string   `json:"composeWorkdir,omitempty"`
	ClearBaseDirectory  bool      `json:"clearBaseDirectory,omitempty"`
	ClearStaticPath     bool      `json:"clearStaticFrontendPath,omitempty"`
}

func registerProjectTools(mcpServer *mcptransport.MCPServer, s *server) {
	addTypedTool(mcpServer, mcp.NewTool("list_projects",
		mcp.WithDescription("List projects visible to this scoped MyPaaS token."),
	), s.listProjects)

	addTypedTool(mcpServer, mcp.NewTool("get_project",
		mcp.WithDescription("Get one MyPaaS project by UUID."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
	), s.getProject)

	addTypedTool(mcpServer, mcp.NewTool("create_project",
		mcp.WithDescription("Create a MyPaaS project from a Git repository or public OCI image. Inspect Git sources first when deploy mode is not already known."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project slug using lowercase letters, numbers, and dashes.")),
		mcp.WithString("sourceType", mcp.Description("git or registry; defaults to git.")),
		mcp.WithString("repoUrl", mcp.Description("Git repository URL.")),
		mcp.WithString("imageRef", mcp.Description("Public OCI image reference.")),
		mcp.WithString("branch", mcp.Description("Git branch; empty uses the repository default.")),
		mcp.WithString("deployMode", mcp.Description("dockerfile, compose, static, or image.")),
		mcp.WithString("resourceProfile", mcp.Description("static, go-small, node-python, compose-main, or custom.")),
		mcp.WithNumber("appPort", mcp.Description("Internal application port.")),
		mcp.WithNumber("memoryLimitMb", mcp.Description("Main service memory limit in MB.")),
		mcp.WithNumber("cpuLimit", mcp.Description("Main service CPU limit.")),
		mcp.WithBoolean("sharedPostgres", mcp.Description("Provision shared PostgreSQL when the application explicitly needs it.")),
		mcp.WithString("mainService", mcp.Description("Compose service receiving public traffic.")),
		mcp.WithString("baseDirectory", mcp.Description("Repository-relative source directory.")),
		mcp.WithString("staticFrontendPath", mcp.Description("Repository-relative static frontend directory.")),
		mcp.WithString("composeFilePath", mcp.Description("Repository-relative Compose file.")),
		mcp.WithArray("composeOverridePaths", mcp.Description("Repository-relative Compose override files.")),
		mcp.WithArray("composeProfiles", mcp.Description("Compose profile names.")),
		mcp.WithString("composeWorkdir", mcp.Description("Repository-relative Compose working directory.")),
	), s.createProject)

	addTypedTool(mcpServer, mcp.NewTool("update_project_settings",
		mcp.WithDescription("Patch mutable project settings. Unspecified fields are preserved."),
		mcp.WithString("projectId", mcp.Required(), mcp.Description("MyPaaS project UUID.")),
		mcp.WithString("branch", mcp.Description("Git branch.")),
		mcp.WithString("imageRef", mcp.Description("Registry image reference.")),
		mcp.WithString("resourceProfile", mcp.Description("Resource profile.")),
		mcp.WithNumber("appPort", mcp.Description("Internal application port.")),
		mcp.WithNumber("memoryLimitMb", mcp.Description("Main service memory limit in MB.")),
		mcp.WithNumber("cpuLimit", mcp.Description("Main service CPU limit.")),
		mcp.WithString("mainService", mcp.Description("Compose public service.")),
		mcp.WithString("baseDirectory", mcp.Description("Repository-relative source directory.")),
		mcp.WithString("staticFrontendPath", mcp.Description("Repository-relative static frontend path.")),
		mcp.WithString("composeFilePath", mcp.Description("Repository-relative Compose file.")),
		mcp.WithArray("composeOverridePaths", mcp.Description("Compose override file paths.")),
		mcp.WithArray("composeProfiles", mcp.Description("Compose profile names.")),
		mcp.WithString("composeWorkdir", mcp.Description("Repository-relative Compose working directory.")),
		mcp.WithBoolean("clearBaseDirectory", mcp.Description("Set true to clear the configured base directory.")),
		mcp.WithBoolean("clearStaticFrontendPath", mcp.Description("Set true to clear the configured static frontend path.")),
	), s.updateProject)

	addTypedTool(mcpServer, mcp.NewTool("inspect_repository",
		mcp.WithDescription("Inspect a Git repository branch/tree before project creation or settings changes."),
		mcp.WithString("repoUrl", mcp.Required(), mcp.Description("Git repository URL.")),
		mcp.WithString("branch", mcp.Description("Git branch; empty uses the repository default.")),
		mcp.WithString("baseDirectory", mcp.Description("Repository-relative source directory.")),
	), s.inspectRepository)

	addTypedTool(mcpServer, mcp.NewTool("detect_compose",
		mcp.WithDescription("Detect and analyze Compose candidates for a Git repository."),
		mcp.WithString("repoUrl", mcp.Required(), mcp.Description("Git repository URL.")),
		mcp.WithString("branch", mcp.Description("Git branch; empty uses the repository default.")),
		mcp.WithString("baseDirectory", mcp.Description("Repository-relative source directory.")),
	), s.detectCompose)
}

func (s *server) listProjects(ctx context.Context, _ struct{}) (*mcp.CallToolResult, error) {
	return s.call(ctx, "list_projects", http.MethodGet, "/projects", nil, "")
}

func (s *server) getProject(ctx context.Context, in projectIDInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	return s.call(ctx, "get_project", http.MethodGet, "/projects/"+pathID(projectID), nil, "")
}

func (s *server) createProject(ctx context.Context, in createProjectInput) (*mcp.CallToolResult, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return toolError("name is required"), nil
	}
	in.SourceType = strings.TrimSpace(in.SourceType)
	if in.SourceType == "" {
		in.SourceType = "git"
	}
	switch in.SourceType {
	case "git":
		if strings.TrimSpace(in.RepoURL) == "" {
			return toolError("repoUrl is required for sourceType=git"), nil
		}
	case "registry":
		if strings.TrimSpace(in.ImageRef) == "" {
			return toolError("imageRef is required for sourceType=registry"), nil
		}
		if strings.TrimSpace(in.DeployMode) == "" {
			in.DeployMode = "image"
		}
	default:
		return toolError("sourceType must be git or registry"), nil
	}
	return s.mutate(ctx, "create_project", http.MethodPost, "/projects", in, "Project created")
}

func (s *server) updateProject(ctx context.Context, in updateProjectInput) (*mcp.CallToolResult, error) {
	projectID, message := cleanRequired(in.ProjectID, "projectId")
	if message != "" {
		return toolError(message), nil
	}
	payload := make(map[string]any)
	if in.Branch != nil {
		payload["branch"] = *in.Branch
	}
	if in.ImageRef != nil {
		payload["imageRef"] = *in.ImageRef
	}
	if in.ResourceProfile != nil {
		payload["resourceProfile"] = *in.ResourceProfile
	}
	if in.AppPort != nil {
		payload["appPort"] = *in.AppPort
	}
	if in.MemoryLimitMB != nil {
		payload["memoryLimitMb"] = *in.MemoryLimitMB
	}
	if in.CPULimit != nil {
		payload["cpuLimit"] = *in.CPULimit
	}
	if in.MainService != nil {
		payload["mainService"] = *in.MainService
	}
	if in.BaseDirectory != nil {
		payload["baseDirectory"] = *in.BaseDirectory
	}
	if in.StaticFrontendPath != nil {
		payload["staticFrontendPath"] = *in.StaticFrontendPath
	}
	if in.ComposeFilePath != nil {
		payload["composeFilePath"] = *in.ComposeFilePath
	}
	if in.ComposeOverridePaths != nil {
		payload["composeOverridePaths"] = *in.ComposeOverridePaths
	}
	if in.ComposeProfiles != nil {
		payload["composeProfiles"] = *in.ComposeProfiles
	}
	if in.ComposeWorkdir != nil {
		payload["composeWorkdir"] = *in.ComposeWorkdir
	}
	if in.ClearBaseDirectory {
		payload["baseDirectory"] = nil
	}
	if in.ClearStaticPath {
		payload["staticFrontendPath"] = nil
	}
	if len(payload) == 0 {
		return toolError("at least one project setting must be supplied"), nil
	}
	return s.mutate(ctx, "update_project_settings", http.MethodPatch, "/projects/"+pathID(projectID), payload, "Project updated")
}

func (s *server) inspectRepository(ctx context.Context, in sourceInput) (*mcp.CallToolResult, error) {
	if strings.TrimSpace(in.RepoURL) == "" {
		return toolError("repoUrl is required"), nil
	}
	payload := map[string]any{"repoUrl": strings.TrimSpace(in.RepoURL), "inspectOnly": true}
	if strings.TrimSpace(in.Branch) != "" {
		payload["branch"] = strings.TrimSpace(in.Branch)
	}
	if strings.TrimSpace(in.BaseDirectory) != "" {
		payload["baseDirectory"] = strings.TrimSpace(in.BaseDirectory)
	}
	return s.call(ctx, "inspect_repository", http.MethodPost, "/projects/detect-mode", payload, "")
}

func (s *server) detectCompose(ctx context.Context, in sourceInput) (*mcp.CallToolResult, error) {
	if strings.TrimSpace(in.RepoURL) == "" {
		return toolError("repoUrl is required"), nil
	}
	payload := map[string]any{"repoUrl": strings.TrimSpace(in.RepoURL)}
	if strings.TrimSpace(in.Branch) != "" {
		payload["branch"] = strings.TrimSpace(in.Branch)
	}
	if strings.TrimSpace(in.BaseDirectory) != "" {
		payload["baseDirectory"] = strings.TrimSpace(in.BaseDirectory)
	}
	return s.call(ctx, "detect_compose", http.MethodPost, "/projects/detect-compose", payload, "")
}
