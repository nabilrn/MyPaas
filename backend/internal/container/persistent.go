package container

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type BindMount struct {
	Source string
	Target string
}

func parseImageVolumeTargets(raw []byte) ([]string, error) {
	var rows []struct {
		Config struct {
			Volumes map[string]json.RawMessage `json:"Volumes"`
		} `json:"Config"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode docker image volumes: %w", err)
	}
	if len(rows) == 0 || len(rows[0].Config.Volumes) == 0 {
		return nil, nil
	}
	targets := make([]string, 0, len(rows[0].Config.Volumes))
	for target := range rows[0].Config.Volumes {
		target = strings.TrimSpace(target)
		if target != "" {
			targets = append(targets, target)
		}
	}
	sort.Strings(targets)
	return targets, nil
}
