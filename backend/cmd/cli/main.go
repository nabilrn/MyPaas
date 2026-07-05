package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/joho/godotenv"

	"mypaas/internal/backup"
	"mypaas/internal/config"
	"mypaas/internal/container"
)

const (
	defaultAPIURL  = "http://localhost:8080"
	configDirName  = ".mypaas"
	configFileName = "config.yml"
)

type cliConfig struct {
	APIURL string
	Token  string
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

type projectSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Subdomain     string `json:"subdomain"`
	DeployMode    string `json:"deployMode"`
	Status        string `json:"status"`
	Branch        string `json:"branch"`
	AllocatedPort *int32 `json:"allocatedPort"`
}

type deploymentSummary struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"projectId"`
	Status      string  `json:"status"`
	CommitSha   *string `json:"commitSha"`
	TriggeredBy string  `json:"triggeredBy"`
	StartedAt   string  `json:"startedAt"`
}

type userSummary struct {
	ID             string  `json:"id"`
	Email          string  `json:"email"`
	GithubUsername *string `json:"githubUsername"`
	Role           string  `json:"role"`
	LastLoginAt    *string `json:"lastLoginAt"`
}

type logsResponse struct {
	Lines []string `json:"lines"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	case "config":
		return runConfig(args[1:])
	case "user":
		client, err := newAPIClient()
		if err != nil {
			return err
		}
		return runUser(client, args[1:])
	case "project":
		client, err := newAPIClient()
		if err != nil {
			return err
		}
		return runProject(client, args[1:])
	case "backup":
		return runBackup(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runConfig(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: mypaas config set <api-url|token> <value>")
	}
	switch args[0] {
	case "set":
		if len(args) != 3 {
			return errors.New("usage: mypaas config set <api-url|token> <value>")
		}
		cfg, err := loadCLIConfig()
		if err != nil {
			return err
		}
		switch args[1] {
		case "api-url":
			cfg.APIURL = strings.TrimRight(args[2], "/")
		case "token":
			cfg.Token = strings.TrimSpace(args[2])
		default:
			return fmt.Errorf("unknown config key %q", args[1])
		}
		if err := saveCLIConfig(cfg); err != nil {
			return err
		}
		fmt.Println("config updated")
		return nil
	case "show":
		cfg, err := loadCLIConfig()
		if err != nil {
			return err
		}
		fmt.Printf("api-url: %s\n", cfg.APIURL)
		if cfg.Token == "" {
			fmt.Println("token: <not set>")
		} else {
			fmt.Println("token: <set>")
		}
		return nil
	default:
		return fmt.Errorf("unknown config command %q", args[0])
	}
}

func runUser(client *apiClient, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: mypaas user <list|add|remove>")
	}
	switch args[0] {
	case "list":
		data, err := client.do(context.Background(), http.MethodGet, "/api/admin/users", nil)
		if err != nil {
			return err
		}
		var users []userSummary
		if err := json.Unmarshal(data, &users); err != nil {
			return err
		}
		printUsers(users)
		return nil
	case "add":
		fs := flag.NewFlagSet("user add", flag.ContinueOnError)
		role := fs.String("role", "collaborator", "user role: owner or collaborator")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("usage: mypaas user add [--role collaborator] <email>")
		}
		body, err := json.Marshal(struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}{Email: fs.Arg(0), Role: *role})
		if err != nil {
			return err
		}
		data, err := client.do(context.Background(), http.MethodPost, "/api/admin/users", body)
		if err != nil {
			return err
		}
		var user userSummary
		if err := json.Unmarshal(data, &user); err != nil {
			return err
		}
		fmt.Printf("added %s (%s)\n", user.Email, user.Role)
		return nil
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: mypaas user remove <id>")
		}
		_, err := client.do(context.Background(), http.MethodDelete, "/api/admin/users/"+args[1], nil)
		if err != nil {
			return err
		}
		fmt.Println("user removed")
		return nil
	default:
		return fmt.Errorf("unknown user command %q", args[0])
	}
}

func runProject(client *apiClient, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: mypaas project <list|deploy|logs>")
	}
	switch args[0] {
	case "list":
		projects, err := client.projects(context.Background())
		if err != nil {
			return err
		}
		printProjects(projects)
		return nil
	case "deploy":
		if len(args) != 2 {
			return errors.New("usage: mypaas project deploy <name>")
		}
		project, err := client.projectByName(context.Background(), args[1])
		if err != nil {
			return err
		}
		data, err := client.do(context.Background(), http.MethodPost, "/api/projects/"+project.ID+"/deploy", []byte(`{}`))
		if err != nil {
			return err
		}
		var deployment deploymentSummary
		if err := json.Unmarshal(data, &deployment); err != nil {
			return err
		}
		fmt.Printf("deployment queued: %s (%s)\n", deployment.ID, deployment.Status)
		return nil
	case "logs":
		return runProjectLogs(client, args[1:])
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func runProjectLogs(client *apiClient, args []string) error {
	fs := flag.NewFlagSet("project logs", flag.ContinueOnError)
	tail := fs.Int("tail", 200, "number of recent log lines")
	follow := fs.Bool("follow", false, "poll for new log lines")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: mypaas project logs [--tail 200] [--follow] <name>")
	}

	project, err := client.projectByName(context.Background(), fs.Arg(0))
	if err != nil {
		return err
	}

	printed := 0
	for {
		lines, err := client.logs(context.Background(), project.ID, *tail)
		if err != nil {
			return err
		}
		if printed > len(lines) {
			printed = 0
		}
		for _, line := range lines[printed:] {
			fmt.Println(line)
		}
		printed = len(lines)
		if !*follow {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func runBackup(args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: mypaas backup")
	}

	loadDotenv()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	timeout := time.Duration(cfg.BackupTimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := backup.NewService(cfg, container.NewDockerCLI(cfg.DockerBindHost)).Run(ctx)
	if err != nil {
		return err
	}
	fmt.Println("backup written:", result.DailyPath)
	if result.WeeklyPath != "" {
		fmt.Println("weekly snapshot:", result.WeeklyPath)
	}
	return nil
}

func newAPIClient() (*apiClient, error) {
	cfg, err := loadCLIConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Token == "" {
		return nil, errors.New("JWT token is not configured; run: mypaas config set token <jwt>")
	}
	return &apiClient{
		baseURL: strings.TrimRight(defaultString(cfg.APIURL, defaultAPIURL), "/"),
		token:   cfg.Token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *apiClient) projects(ctx context.Context) ([]projectSummary, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/projects", nil)
	if err != nil {
		return nil, err
	}
	var projects []projectSummary
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *apiClient) projectByName(ctx context.Context, name string) (projectSummary, error) {
	projects, err := c.projects(ctx)
	if err != nil {
		return projectSummary{}, err
	}
	for _, project := range projects {
		if project.Name == name {
			return project, nil
		}
	}
	return projectSummary{}, fmt.Errorf("project %q was not found", name)
}

func (c *apiClient) logs(ctx context.Context, projectID string, tail int) ([]string, error) {
	if tail <= 0 {
		tail = 200
	}
	data, err := c.do(ctx, http.MethodGet, "/api/projects/"+projectID+"/logs?tail="+strconv.Itoa(tail), nil)
	if err != nil {
		return nil, err
	}
	var logs logsResponse
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, err
	}
	return logs.Lines, nil
}

func (c *apiClient) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("request failed: %s", resp.Status)
		}
		return nil, nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w: %s", err, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode >= 400 {
		if envelope.Error != nil {
			return nil, fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}
	return envelope.Data, nil
}

func loadCLIConfig() (cliConfig, error) {
	cfg := cliConfig{APIURL: defaultAPIURL}
	path, err := cliConfigPath()
	if err != nil {
		return cfg, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || strings.HasPrefix(key, "#") {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "api_url", "api-url":
			cfg.APIURL = value
		case "token":
			cfg.Token = value
		}
	}
	return cfg, nil
}

func saveCLIConfig(cfg cliConfig) error {
	path, err := cliConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	content := fmt.Sprintf("api_url: %s\ntoken: %s\n", defaultString(cfg.APIURL, defaultAPIURL), cfg.Token)
	return os.WriteFile(path, []byte(content), 0600)
}

func cliConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

func printProjects(projects []projectSummary) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tMODE\tBRANCH\tSUBDOMAIN\tPORT")
	for _, project := range projects {
		port := "-"
		if project.AllocatedPort != nil {
			port = strconv.Itoa(int(*project.AllocatedPort))
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", project.Name, project.Status, project.DeployMode, project.Branch, project.Subdomain, port)
	}
	_ = tw.Flush()
}

func printUsers(users []userSummary) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "EMAIL\tROLE\tGITHUB\tLAST LOGIN\tID")
	for _, user := range users {
		github := "-"
		if user.GithubUsername != nil && *user.GithubUsername != "" {
			github = *user.GithubUsername
		}
		lastLogin := "-"
		if user.LastLoginAt != nil && *user.LastLoginAt != "" {
			lastLogin = *user.LastLoginAt
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", user.Email, user.Role, github, lastLogin, user.ID)
	}
	_ = tw.Flush()
}

func loadDotenv() {
	for _, path := range []string{".env", "../.env", "../../.env", "../../../.env"} {
		if err := godotenv.Load(path); err == nil {
			return
		}
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "MyPaas CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  mypaas config set api-url <url>")
	fmt.Fprintln(w, "  mypaas config set token <jwt>")
	fmt.Fprintln(w, "  mypaas config show")
	fmt.Fprintln(w, "  mypaas user list")
	fmt.Fprintln(w, "  mypaas user add [--role collaborator] <email>")
	fmt.Fprintln(w, "  mypaas user remove <id>")
	fmt.Fprintln(w, "  mypaas project list")
	fmt.Fprintln(w, "  mypaas project deploy <name>")
	fmt.Fprintln(w, "  mypaas project logs [--tail 200] [--follow] <name>")
	fmt.Fprintln(w, "  mypaas backup")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
