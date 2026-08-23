package replica

import (
	"context"
	"fmt"
	"strings"
)

// CleanupInactive removes secondary runtime replicas whose project is no longer
// routable. This covers stopped and deleted projects without making the project
// lifecycle handlers aware of replica internals.
func (s *Service) CleanupInactive(ctx context.Context) error {
	projects, err := s.queries.ListRoutableProjects(ctx)
	if err != nil {
		return fmt.Errorf("list routable projects for replica cleanup: %w", err)
	}
	active := make(map[string]struct{}, len(projects))
	for _, project := range projects {
		active[project.ID.String()] = struct{}{}
	}

	items, err := s.docker.AllReplicaInfos(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		projectID := strings.TrimSpace(item.Project)
		if projectID != "" {
			if _, ok := active[projectID]; ok {
				continue
			}
		}
		if err := s.docker.Remove(ctx, item.Name); err != nil {
			return fmt.Errorf("remove inactive replica %s: %w", item.Name, err)
		}
	}
	return nil
}
