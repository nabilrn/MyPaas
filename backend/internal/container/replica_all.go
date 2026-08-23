package container

import (
	"context"
	"fmt"
	"strings"
)

// AllReplicaInfos returns every MyPaaS-managed secondary runtime replica.
// The primary runtime is intentionally excluded because it does not carry the
// replica project label.
func (d *DockerCLI) AllReplicaInfos(ctx context.Context) ([]ReplicaInfo, error) {
	out, err := commandContext(ctx, "docker", "ps", "-a", "-q", "--filter", "label="+ReplicaProjectLabel).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list managed replicas: %w: %s", err, strings.TrimSpace(string(out)))
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return nil, nil
	}
	args := append([]string{"inspect"}, ids...)
	inspect, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("inspect managed replicas: %w: %s", err, strings.TrimSpace(string(inspect)))
	}
	return parseReplicaInspect(inspect)
}
