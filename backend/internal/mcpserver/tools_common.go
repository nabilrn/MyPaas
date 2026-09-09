package mcpserver

import (
	"net/url"
	"strings"
)

type projectIDInput struct {
	ProjectID string `json:"projectId" jsonschema:"MyPaaS project UUID"`
}

type deploymentIDInput struct {
	DeploymentID string `json:"deploymentId" jsonschema:"MyPaaS deployment UUID"`
}

type sourceInput struct {
	RepoURL       string `json:"repoUrl" jsonschema:"Git repository URL"`
	Branch        string `json:"branch,omitempty" jsonschema:"Git branch; empty uses the repository default"`
	BaseDirectory string `json:"baseDirectory,omitempty" jsonschema:"Repository-relative source directory"`
}

func cleanRequired(value, name string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", name + " is required"
	}
	return value, ""
}

func pathID(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}
