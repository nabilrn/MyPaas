package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RuntimeProcess is the small set of Docker runtime metadata needed to register
// a running container with mypaas-statd. It is collected on lifecycle/reconcile
// paths, not on the high-frequency metrics path.
type RuntimeProcess struct {
	ID        string
	Service   string
	PID       int
	StartedAt time.Time
}

type runtimeInspectRow struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
	State struct {
		PID       int    `json:"Pid"`
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

// RuntimeProcess returns runtime metadata for one running container in one
// docker inspect invocation.
func (d *DockerCLI) RuntimeProcess(ctx context.Context, name string) (RuntimeProcess, error) {
	rows, err := d.inspectRuntimeProcesses(ctx, []string{name})
	if err != nil {
		return RuntimeProcess{}, err
	}
	if len(rows) == 0 || rows[0].PID <= 0 {
		return RuntimeProcess{}, ErrNoContainer
	}
	if rows[0].Service == "" {
		rows[0].Service = "app"
	}
	return rows[0], nil
}

// ComposeRuntimeProcesses discovers the project's containers once, then reads
// all runtime metadata in one batched docker inspect. Stopped containers are
// omitted because they do not have a host PID to register with statd.
func (d *DockerCLI) ComposeRuntimeProcesses(ctx context.Context, projectName string) ([]RuntimeProcess, error) {
	ids, err := d.composeContainerIDs(ctx, projectName, "")
	if err != nil {
		return nil, err
	}
	rows, err := d.inspectRuntimeProcesses(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]RuntimeProcess, 0, len(rows))
	for _, row := range rows {
		if row.PID <= 0 {
			continue
		}
		if strings.TrimSpace(row.Service) == "" {
			return nil, fmt.Errorf("docker inspect compose runtime %s: missing service label", row.ID)
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		return nil, ErrNoContainer
	}
	return out, nil
}

func (d *DockerCLI) inspectRuntimeProcesses(ctx context.Context, names []string) ([]RuntimeProcess, error) {
	if len(names) == 0 {
		return nil, ErrNoContainer
	}
	args := append([]string{"inspect"}, names...)
	out, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isNoContainerMessage(msg) {
			return nil, ErrNoContainer
		}
		return nil, fmt.Errorf("docker inspect runtime metadata: %w: %s", err, msg)
	}
	return parseRuntimeProcesses(out)
}

func parseRuntimeProcesses(raw []byte) ([]RuntimeProcess, error) {
	var rows []runtimeInspectRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode docker runtime metadata: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrNoContainer
	}

	out := make([]RuntimeProcess, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ID) == "" {
			return nil, fmt.Errorf("decode docker runtime metadata: missing container id")
		}
		startedAt := time.Time{}
		value := strings.TrimSpace(row.State.StartedAt)
		if value != "" && value != "0001-01-01T00:00:00Z" {
			parsed, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				return nil, fmt.Errorf("decode docker runtime startedAt for %s: %w", row.ID, err)
			}
			startedAt = parsed
		}
		service := ""
		if row.Config.Labels != nil {
			service = strings.TrimSpace(row.Config.Labels["com.docker.compose.service"])
		}
		if service == "" {
			service = strings.TrimPrefix(strings.TrimSpace(row.Name), "/")
		}
		out = append(out, RuntimeProcess{
			ID:        strings.TrimSpace(row.ID),
			Service:   service,
			PID:       row.State.PID,
			StartedAt: startedAt,
		})
	}
	return out, nil
}
