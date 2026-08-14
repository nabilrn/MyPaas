package container

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const persistentVolumesLabel = "io.mypaas.persistent-volumes"

type BindMount struct {
	Source string
	Target string
}

type imageInspectConfig struct {
	Volumes map[string]json.RawMessage `json:"Volumes"`
	Labels  map[string]string          `json:"Labels"`
}

func parseImageVolumeTargets(raw []byte) ([]string, error) {
	var rows []struct {
		Config imageInspectConfig `json:"Config"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode docker image volumes: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	targetSet := make(map[string]struct{}, len(rows[0].Config.Volumes))
	addTarget := func(target string) {
		target = strings.TrimSpace(target)
		if target != "" {
			targetSet[target] = struct{}{}
		}
	}

	for target := range rows[0].Config.Volumes {
		addTarget(target)
	}

	if labelValue := rows[0].Config.Labels[persistentVolumesLabel]; labelValue != "" {
		for _, target := range strings.Split(labelValue, ",") {
			addTarget(target)
		}
	}

	if len(targetSet) == 0 {
		return nil, nil
	}

	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets, nil
}
