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
	"strconv"
	"strings"
	"time"
)

type DockerCLI struct {
	bindHost string
}

type Metrics struct {
	Service       string
	CPUPercent    float64
	MemoryMB      float64
	MemoryLimitMB float64
	Uptime        string
	CollectedAt   time.Time
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

func NewDockerCLI(bindHost string) *DockerCLI {
	bindHost = strings.TrimSpace(bindHost)
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	return &DockerCLI{bindHost: bindHost}
}

func (d *DockerCLI) Build(ctx context.Context, dir, image string, log func(string)) error {
	return runLogged(ctx, dir, log, "docker", "build", "-t", image, ".")
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
	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}
	args = append(args, opts.Image)
	return runLogged(ctx, "", log, "docker", args...)
}

func (d *DockerCLI) portMapping(opts RunOptions) string {
	return fmt.Sprintf("%s:%d:%d", d.bindHost, opts.HostPort, opts.ContainerPort)
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

var ErrNoContainer = errors.New("container not found")

func commandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	if name == "docker" {
		cmd.Env = dockerEnv()
	}
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
