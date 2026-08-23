package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	ReplicaProjectLabel = "mypaas.replica.project"
	ReplicaSlotLabel    = "mypaas.replica.slot"
	ReplicaImageLabel   = "mypaas.replica.image"
	ReplicaLogMaxSize   = "20m"
)

type ReplicaRunOptions struct {
	Name           string
	ProjectID      string
	Slot           int
	Image          string
	ContainerPort  int32
	MemoryMB       int32
	CPULimit       float64
	EnvFile        string
	RoutingNetwork string
	RoutingAlias   string
}

type ReplicaInfo struct {
	ID      string
	Name    string
	Project string
	Slot    int
	Image   string
	Running bool
	Health  string
}

func (d *DockerCLI) RunReplica(ctx context.Context, opts ReplicaRunOptions, log func(string)) error {
	if strings.TrimSpace(opts.Name) == "" || strings.TrimSpace(opts.ProjectID) == "" {
		return fmt.Errorf("replica name and project ID are required")
	}
	if opts.Slot < 2 {
		return fmt.Errorf("replica slot must be >= 2")
	}
	if strings.TrimSpace(opts.Image) == "" {
		return fmt.Errorf("replica image is required")
	}
	if opts.ContainerPort <= 0 || opts.ContainerPort > 65535 {
		return fmt.Errorf("replica container port is invalid")
	}
	if opts.MemoryMB <= 0 {
		opts.MemoryMB = 512
	}
	if opts.CPULimit <= 0 {
		opts.CPULimit = 0.5
	}
	if d.projectNetwork == "" {
		return fmt.Errorf("replicas require PROJECT_NETWORK")
	}
	if strings.TrimSpace(opts.RoutingNetwork) == "" || strings.TrimSpace(opts.RoutingAlias) == "" {
		return fmt.Errorf("replica routing network and alias are required")
	}

	args := []string{
		"run", "-d",
		"--name", opts.Name,
		"--label", ManagedImageLabel,
		"--label", ReplicaProjectLabel + "=" + opts.ProjectID,
		"--label", ReplicaSlotLabel + "=" + strconv.Itoa(opts.Slot),
		"--label", ReplicaImageLabel + "=" + opts.Image,
		"--memory", fmt.Sprintf("%dm", opts.MemoryMB),
		"--cpus", fmt.Sprintf("%.2f", opts.CPULimit),
		"--restart", "unless-stopped",
		"--log-opt", "max-size=" + ReplicaLogMaxSize,
		"--network", d.projectNetwork,
	}
	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}
	args = append(args, opts.Image)
	if err := runLogged(ctx, "", log, "docker", args...); err != nil {
		return err
	}

	out, err := commandContext(ctx, "docker", "network", "connect", "--alias", opts.RoutingAlias, opts.RoutingNetwork, opts.Name).CombinedOutput()
	if err != nil {
		_ = d.Remove(context.Background(), opts.Name)
		return fmt.Errorf("attach replica to routing network: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *DockerCLI) ReplicaInfos(ctx context.Context, projectID string) ([]ReplicaInfo, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	out, err := commandContext(ctx, "docker", "ps", "-a", "-q", "--filter", "label="+ReplicaProjectLabel+"="+projectID).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list project replicas: %w: %s", err, strings.TrimSpace(string(out)))
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return nil, nil
	}
	args := append([]string{"inspect"}, ids...)
	inspect, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("inspect project replicas: %w: %s", err, strings.TrimSpace(string(inspect)))
	}
	return parseReplicaInspect(inspect)
}

func parseReplicaInspect(raw []byte) ([]ReplicaInfo, error) {
	var rows []struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Config struct {
			Image  string            `json:"Image"`
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		State struct {
			Running bool   `json:"Running"`
			Status  string `json:"Status"`
			Health  *struct {
				Status string `json:"Status"`
			} `json:"Health"`
		} `json:"State"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode replica inspect: %w", err)
	}
	items := make([]ReplicaInfo, 0, len(rows))
	for _, row := range rows {
		slot, _ := strconv.Atoi(strings.TrimSpace(row.Config.Labels[ReplicaSlotLabel]))
		health := ""
		if row.State.Health != nil {
			health = strings.TrimSpace(row.State.Health.Status)
		}
		image := strings.TrimSpace(row.Config.Labels[ReplicaImageLabel])
		if image == "" {
			image = strings.TrimSpace(row.Config.Image)
		}
		items = append(items, ReplicaInfo{
			ID:      strings.TrimSpace(row.ID),
			Name:    strings.TrimPrefix(strings.TrimSpace(row.Name), "/"),
			Project: strings.TrimSpace(row.Config.Labels[ReplicaProjectLabel]),
			Slot:    slot,
			Image:   image,
			Running: row.State.Running,
			Health:  health,
		})
	}
	return items, nil
}

func (d *DockerCLI) RemoveReplicas(ctx context.Context, projectID string) error {
	items, err := d.ReplicaInfos(ctx, projectID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := d.Remove(ctx, item.Name); err != nil {
			return err
		}
	}
	return nil
}
