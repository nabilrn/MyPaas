package container

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DockerCLI struct {
	bindHost       string
	projectNetwork string
}

type Metrics struct {
	Service       string
	CPUPercent    float64
	MemoryMB      float64
	MemoryLimitMB float64
	Uptime        string
	CollectedAt   time.Time
}

type ComposeResourceSummary struct {
	ProjectName string `json:"projectName"`
	Containers  int    `json:"containers"`
	Volumes     int    `json:"volumes"`
	Networks    int    `json:"networks"`
}

type ComposeLogLine struct {
	Service string
	Line    string
}

type RunOptions struct {
	Name          string
	Image         string
	HostPort      int32
	ContainerPort int32
	MemoryMB      int32
	CPULimit      float64
	EnvFile       string
}

type ComposeUpOptions struct {
	ProjectName  string
	WorkDir      string
	ComposeFile  string
	OverrideFile string
	EnvFile      string
	NoBuild      bool
}

const ManagedImageLabel = "mypaas.managed=true"

func NewDockerCLI(bindHost string, projectNetwork ...string) *DockerCLI {
	bindHost = strings.TrimSpace(bindHost)
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	network := ""
	if len(projectNetwork) > 0 {
		network = strings.TrimSpace(projectNetwork[0])
	}
	return &DockerCLI{bindHost: bindHost, projectNetwork: network}
}

func (d *DockerCLI) Build(ctx context.Context, dir, image string, log func(string)) error {
	return runLogged(ctx, dir, log, "docker", "build", "--label", ManagedImageLabel, "-t", image, ".")
}

func (d *DockerCLI) Run(ctx context.Context, opts RunOptions, log func(string)) error {
	args := []string{
		"run", "-d",
		"--name", opts.Name,
		"-p", d.portMapping(opts),
		"--memory", fmt.Sprintf("%dm", opts.MemoryMB),
		"--cpus", fmt.Sprintf("%.2f", opts.CPULimit),
		"--restart", "unless-stopped",
	}
	if d.projectNetwork != "" {
		args = append(args, "--network", d.projectNetwork)
	} else {
		args = append(args, "--add-host", "host.docker.internal:host-gateway")
	}
	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}
	args = append(args, opts.Image)
	return runLogged(ctx, "", log, "docker", args...)
}

func (d *DockerCLI) portMapping(opts RunOptions) string {
	return fmt.Sprintf("%s:%d:%d", d.bindHost, opts.HostPort, opts.ContainerPort)
}

func (d *DockerCLI) ComposePortMapping(hostPort, containerPort int32) string {
	return fmt.Sprintf("%s:%d:%d", d.bindHost, hostPort, containerPort)
}

func (d *DockerCLI) Stop(ctx context.Context, name string) error {
	return runIgnoreNotFound(ctx, "docker", "stop", "--timeout", "30", name)
}

func (d *DockerCLI) Start(ctx context.Context, name string) error {
	return runSimple(ctx, "docker", "start", name)
}

func (d *DockerCLI) Restart(ctx context.Context, name string) error {
	return runSimple(ctx, "docker", "restart", name)
}

func (d *DockerCLI) Rename(ctx context.Context, oldName, newName string) error {
	return runSimple(ctx, "docker", "rename", oldName, newName)
}

func (d *DockerCLI) Remove(ctx context.Context, name string) error {
	return runIgnoreNotFound(ctx, "docker", "rm", "-f", name)
}

func (d *DockerCLI) CleanupUnusedManagedImages(ctx context.Context, until string) error {
	until = strings.TrimSpace(until)
	if until == "" {
		until = "168h"
	}
	return runSimple(ctx, "docker", "image", "prune", "-a", "-f", "--filter", "label="+ManagedImageLabel, "--filter", "until="+until)
}

func (d *DockerCLI) ComposeServices(ctx context.Context, dir, envFile string) ([]string, error) {
	args := composeBaseArgs(envFile)
	args = append(args, "config", "--services")
	out, err := withDir(commandContext(ctx, "docker", args...), dir).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker compose config --services: %w: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	services := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			services = append(services, line)
		}
	}
	return services, nil
}

func (d *DockerCLI) ComposeBuildServices(ctx context.Context, dir, envFile string) ([]string, error) {
	args := composeBaseArgs(envFile)
	args = append(args, "config", "--format", "json")
	out, err := withDir(commandContext(ctx, "docker", args...), dir).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker compose config --format json: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return parseComposeBuildServicesJSON(out)
}

func (d *DockerCLI) WriteSanitizedComposeConfig(ctx context.Context, dir, envFile, composeFile, outputPath string) error {
	args := composeBaseArgs(envFile)
	args = append(args, "-f", composeFile, "config", "--format", "json")
	out, err := withDir(commandContext(ctx, "docker", args...), dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose config --format json: %w: %s", err, strings.TrimSpace(string(out)))
	}

	sanitized, err := removeComposeHostPorts(out)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, sanitized, 0600)
}

func (d *DockerCLI) ComposeUp(ctx context.Context, opts ComposeUpOptions, log func(string)) error {
	args := composeBaseArgs(opts.EnvFile)
	args = append(args,
		"-p", opts.ProjectName,
		"-f", opts.ComposeFile,
		"-f", opts.OverrideFile,
		"up", "-d",
	)
	if opts.NoBuild {
		args = append(args, "--no-build")
	} else {
		args = append(args, "--build")
	}
	args = append(args, "--remove-orphans")
	return runLogged(ctx, opts.WorkDir, log, "docker", args...)
}

func (d *DockerCLI) ImageExists(ctx context.Context, image string) (bool, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return false, nil
	}
	out, err := commandContext(ctx, "docker", "image", "inspect", image).CombinedOutput()
	if err == nil {
		return true, nil
	}
	msg := strings.TrimSpace(string(out))
	if strings.Contains(msg, "No such image") || strings.Contains(msg, "not found") {
		return false, nil
	}
	return false, fmt.Errorf("docker image inspect: %w: %s", err, msg)
}

func (d *DockerCLI) StopComposeProject(ctx context.Context, projectName string) error {
	ids, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil {
		return err
	}
	return runSimple(ctx, "docker", append([]string{"stop", "--timeout", "30"}, ids...)...)
}

func (d *DockerCLI) StartComposeProject(ctx context.Context, projectName string) error {
	ids, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil {
		return err
	}
	return runSimple(ctx, "docker", append([]string{"start"}, ids...)...)
}

func (d *DockerCLI) RestartComposeProject(ctx context.Context, projectName string) error {
	ids, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil {
		return err
	}
	return runSimple(ctx, "docker", append([]string{"restart"}, ids...)...)
}

func (d *DockerCLI) RemoveComposeProject(ctx context.Context, projectName string) error {
	containers, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil && !errors.Is(err, ErrNoContainer) {
		return err
	}
	if len(containers) > 0 {
		if err := runSimple(ctx, "docker", append([]string{"rm", "-f"}, containers...)...); err != nil {
			return err
		}
	}
	if err := removeLabeledResources(ctx, "volume", projectName); err != nil {
		return err
	}
	return removeLabeledResources(ctx, "network", projectName)
}

func (d *DockerCLI) ComposeResources(ctx context.Context, projectName string) (ComposeResourceSummary, error) {
	summary := ComposeResourceSummary{ProjectName: projectName}

	containers, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil && !errors.Is(err, ErrNoContainer) {
		return summary, err
	}
	summary.Containers = len(containers)

	volumes, err := labeledResourceIDs(ctx, "volume", projectName)
	if err != nil {
		return summary, err
	}
	summary.Volumes = len(volumes)

	networks, err := labeledResourceIDs(ctx, "network", projectName)
	if err != nil {
		return summary, err
	}
	summary.Networks = len(networks)

	return summary, nil
}

func (d *DockerCLI) ComposeServiceNames(ctx context.Context, projectName string) ([]string, error) {
	ids, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil {
		return nil, err
	}

	args := append([]string{"inspect", "--format", `{{ index .Config.Labels "com.docker.compose.service" }}`}, ids...)
	out, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isNoContainerMessage(msg) {
			return nil, ErrNoContainer
		}
		return nil, fmt.Errorf("docker inspect compose services: %w: %s", err, msg)
	}

	services := parseComposeServiceNames(string(out))
	if len(services) == 0 {
		return nil, ErrNoContainer
	}
	return services, nil
}

func (d *DockerCLI) ComposeLogs(ctx context.Context, projectName, service string, tail int) ([]string, error) {
	id, err := d.composeServiceContainer(ctx, projectName, service)
	if err != nil {
		return nil, err
	}
	return d.Logs(ctx, id, tail)
}

func (d *DockerCLI) ComposeLogsAll(ctx context.Context, projectName string, tail int) ([]ComposeLogLine, error) {
	services, err := d.ComposeServiceNames(ctx, projectName)
	if err != nil {
		return nil, err
	}

	items := make([]ComposeLogLine, 0)
	for _, service := range services {
		lines, err := d.ComposeLogs(ctx, projectName, service, tail)
		if errors.Is(err, ErrNoContainer) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			items = append(items, ComposeLogLine{Service: service, Line: line})
		}
	}
	if len(items) == 0 {
		return []ComposeLogLine{}, nil
	}
	return items, nil
}

func (d *DockerCLI) ComposeStats(ctx context.Context, projectName, service string) (Metrics, error) {
	id, err := d.composeServiceContainer(ctx, projectName, service)
	if err != nil {
		return Metrics{}, err
	}
	metrics, err := d.Stats(ctx, id)
	if err != nil {
		return Metrics{}, err
	}
	metrics.Service = service
	return metrics, nil
}

func (d *DockerCLI) ComposeStatsAll(ctx context.Context, projectName string) ([]Metrics, error) {
	services, err := d.ComposeServiceNames(ctx, projectName)
	if err != nil {
		return nil, err
	}

	items := make([]Metrics, 0, len(services))
	for _, service := range services {
		metrics, err := d.ComposeStats(ctx, projectName, service)
		if errors.Is(err, ErrNoContainer) {
			continue
		}
		if err != nil {
			return nil, err
		}
		items = append(items, metrics)
	}
	if len(items) == 0 {
		return nil, ErrNoContainer
	}
	return items, nil
}

func (d *DockerCLI) composeServiceContainer(ctx context.Context, projectName, service string) (string, error) {
	ids, err := d.composeContainerIDs(ctx, projectName, service)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

func (d *DockerCLI) composeContainerIDs(ctx context.Context, projectName, service string) ([]string, error) {
	args := []string{
		"ps", "-aq",
		"--filter", "label=com.docker.compose.project=" + projectName,
	}
	if service != "" {
		args = append(args, "--filter", "label=com.docker.compose.service="+service)
	}
	out, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker ps compose project: %w: %s", err, strings.TrimSpace(string(out)))
	}
	ids := fieldsByLine(string(out))
	if len(ids) == 0 {
		return nil, ErrNoContainer
	}
	return ids, nil
}

func (d *DockerCLI) Logs(ctx context.Context, name string, tail int) ([]string, error) {
	out, err := commandContext(ctx, "docker", "logs", "--tail", fmt.Sprint(tail), name).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isNoContainerMessage(msg) {
			return nil, ErrNoContainer
		}
		return nil, fmt.Errorf("docker logs: %w: %s", err, msg)
	}
	if len(out) == 0 {
		return nil, nil
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

func (d *DockerCLI) Stats(ctx context.Context, name string) (Metrics, error) {
	out, err := commandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}", name).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isNoContainerMessage(msg) {
			return Metrics{}, ErrNoContainer
		}
		return Metrics{}, fmt.Errorf("docker stats: %w: %s", err, msg)
	}

	metrics, err := parseStatsLine(firstLine(string(out)))
	if err != nil {
		return Metrics{}, err
	}
	if metrics.Service == "" {
		metrics.Service = name
	}
	metrics.CollectedAt = time.Now().UTC()

	if startedAt, err := d.startedAt(ctx, name); err == nil && !startedAt.IsZero() {
		metrics.Uptime = formatUptime(time.Since(startedAt))
	}
	if metrics.Uptime == "" {
		metrics.Uptime = "unknown"
	}
	return metrics, nil
}

func (d *DockerCLI) startedAt(ctx context.Context, name string) (time.Time, error) {
	out, err := commandContext(ctx, "docker", "inspect", "--format", "{{.State.StartedAt}}", name).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isNoContainerMessage(msg) {
			return time.Time{}, ErrNoContainer
		}
		return time.Time{}, fmt.Errorf("docker inspect: %w: %s", err, msg)
	}
	value := strings.TrimSpace(string(out))
	if value == "" || value == "<no value>" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

type dockerStatsLine struct {
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
}

func parseStatsLine(line string) (Metrics, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Metrics{}, ErrNoContainer
	}

	var raw dockerStatsLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Metrics{}, fmt.Errorf("parse docker stats: %w", err)
	}

	cpu, err := parsePercent(raw.CPUPerc)
	if err != nil {
		return Metrics{}, err
	}
	used, limit, err := parseMemoryUsage(raw.MemUsage)
	if err != nil {
		return Metrics{}, err
	}

	return Metrics{
		Service:       raw.Name,
		CPUPercent:    cpu,
		MemoryMB:      used,
		MemoryLimitMB: limit,
	}, nil
}

func parsePercent(value string) (float64, error) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse percent %q: %w", value, err)
	}
	return parsed, nil
}

func parseMemoryUsage(value string) (float64, float64, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("parse memory usage %q", value)
	}
	used, err := parseMemoryMB(parts[0])
	if err != nil {
		return 0, 0, err
	}
	limit, err := parseMemoryMB(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return used, limit, nil
}

func parseMemoryMB(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	end := 0
	for end < len(value) {
		ch := value[end]
		if (ch < '0' || ch > '9') && ch != '.' {
			break
		}
		end++
	}
	if end == 0 {
		return 0, fmt.Errorf("parse memory value %q", value)
	}

	amount, err := strconv.ParseFloat(value[:end], 64)
	if err != nil {
		return 0, fmt.Errorf("parse memory value %q: %w", value, err)
	}

	unit := strings.ToLower(strings.TrimSpace(value[end:]))
	switch unit {
	case "b":
		return amount / 1024 / 1024, nil
	case "kb", "kib":
		return amount / 1024, nil
	case "mb", "mib", "":
		return amount, nil
	case "gb", "gib":
		return amount * 1024, nil
	case "tb", "tib":
		return amount * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unsupported memory unit %q", unit)
	}
}

func firstLine(value string) string {
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func formatUptime(duration time.Duration) string {
	if duration < time.Minute {
		return "<1m"
	}
	minutes := int(duration.Minutes())
	days := minutes / (24 * 60)
	hours := (minutes / 60) % 24
	remainingMinutes := minutes % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, remainingMinutes)
	}
	return fmt.Sprintf("%dm", remainingMinutes)
}

func runLogged(ctx context.Context, dir string, log func(string), name string, args ...string) error {
	cmd := commandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s start: %w", name, err)
	}

	done := make(chan struct{}, 2)
	go scanPipe(stdout, log, done)
	go scanPipe(stderr, log, done)
	<-done
	<-done

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func scanPipe(r io.Reader, log func(string), done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log(scanner.Text())
	}
}

func runSimple(ctx context.Context, name string, args ...string) error {
	out, err := commandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runIgnoreNotFound(ctx context.Context, name string, args ...string) error {
	err := runSimple(ctx, name, args...)
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "No such container") || strings.Contains(msg, "not found") {
		return nil
	}
	return err
}

func isNoContainerMessage(msg string) bool {
	return strings.Contains(msg, "No such container") || strings.Contains(msg, "not found")
}

func fieldsByLine(value string) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func parseComposeServiceNames(value string) []string {
	seen := make(map[string]struct{})
	services := make([]string, 0)
	for _, line := range fieldsByLine(value) {
		if line == "<no value>" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		services = append(services, line)
	}
	sort.Strings(services)
	return services
}

func parseComposeBuildServicesJSON(raw []byte) ([]string, error) {
	var config struct {
		Services map[string]struct {
			Build json.RawMessage `json:"build"`
		} `json:"services"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("parse docker compose config json: %w", err)
	}

	services := make([]string, 0, len(config.Services))
	for service, spec := range config.Services {
		if len(spec.Build) == 0 || string(spec.Build) == "null" {
			continue
		}
		services = append(services, service)
	}
	sort.Strings(services)
	return services, nil
}

func removeComposeHostPorts(raw []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse compose config json: %w", err)
	}

	services, ok := doc["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return nil, fmt.Errorf("compose config does not define services")
	}
	for _, rawService := range services {
		service, ok := rawService.(map[string]any)
		if !ok {
			continue
		}
		delete(service, "ports")
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("write sanitized compose config: %w", err)
	}
	return append(out, '\n'), nil
}

func labeledResourceIDs(ctx context.Context, resource, projectName string) ([]string, error) {
	out, err := commandContext(ctx, "docker", resource, "ls", "-q", "--filter", "label=com.docker.compose.project="+projectName).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker %s ls compose project: %w: %s", resource, err, strings.TrimSpace(string(out)))
	}
	return fieldsByLine(string(out)), nil
}

func removeLabeledResources(ctx context.Context, resource, projectName string) error {
	ids, err := labeledResourceIDs(ctx, resource, projectName)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return runSimple(ctx, "docker", append([]string{resource, "rm"}, ids...)...)
}

var ErrNoContainer = errors.New("container not found")

func commandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	if name == "docker" {
		if len(args) > 0 && args[0] == "compose" {
			cmd.Env = composeEnv()
		} else {
			cmd.Env = dockerEnv()
		}
	}
	return cmd
}

func withDir(cmd *exec.Cmd, dir string) *exec.Cmd {
	cmd.Dir = dir
	return cmd
}

func dockerEnv() []string {
	env := os.Environ()
	if runtime.GOOS != "windows" {
		return env
	}

	out := make([]string, 0, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(key, "DOCKER_HOST") && value == "unix:///var/run/docker.sock" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func composeBaseArgs(envFile string) []string {
	args := []string{"compose"}
	envFile = strings.TrimSpace(envFile)
	if envFile != "" {
		args = append(args, "--env-file", envFile)
	}
	return args
}

func composeEnv() []string {
	base := dockerEnv()
	out := make([]string, 0, len(base))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if ok && isMypaasInternalEnv(key) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func isMypaasInternalEnv(key string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	if strings.HasPrefix(key, "GITHUB_") || strings.HasPrefix(key, "CLOUDFLARE_") || strings.HasPrefix(key, "CADDY_") {
		return true
	}
	switch key {
	case "DATABASE_URL",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"ENVIRONMENT",
		"API_HOST",
		"API_PORT",
		"FRONTEND_URL",
		"JWT_SECRET",
		"ENCRYPTION_KEY",
		"DOCKER_SOCKET",
		"DOCKER_BIND_HOST",
		"PROJECT_NETWORK",
		"PUBLIC_DOMAIN",
		"OWNER_EMAIL",
		"USER_RAM_QUOTA_GB",
		"USER_CPU_QUOTA",
		"MAX_PROJECTS",
		"PROJECT_DEFAULT_RAM_MB",
		"PROJECT_DEFAULT_CPU",
		"ENABLE_WEBHOOKS",
		"ENABLE_METRICS",
		"METRICS_USERNAME",
		"METRICS_PASSWORD",
		"MAX_CONCURRENT_DEPLOYS",
		"BUILD_TIMEOUT_MINUTES",
		"STATIC_ROOT",
		"SHARED_POSTGRES_ENABLED",
		"SHARED_POSTGRES_HOST",
		"SHARED_POSTGRES_PORT",
		"SHARED_POSTGRES_SSLMODE",
		"BACKUP_ENABLED",
		"BACKUP_DIR",
		"BACKUP_DAILY_AT",
		"BACKUP_TIMEOUT_MINUTES",
		"BACKUP_KEEP_DAILY",
		"BACKUP_KEEP_WEEKLY",
		"BACKUP_WEEKLY_DAY",
		"IMAGE_CLEANUP_ENABLED",
		"IMAGE_CLEANUP_UNTIL",
		"IMAGE_CLEANUP_WEEKDAY",
		"LOG_LEVEL",
		"LOG_FORMAT":
		return true
	default:
		return false
	}
}
