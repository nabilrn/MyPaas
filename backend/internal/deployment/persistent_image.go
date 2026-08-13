package deployment

import (
	"crypto/sha256"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/google/uuid"
)

func persistentImageVolumeName(projectID uuid.UUID, target string) string {
	sum := sha256.Sum256([]byte(target))
	return fmt.Sprintf("mypaas-%s-%x", projectID.String(), sum[:6])
}

func normalizeImageVolumeTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" || !pathpkg.IsAbs(target) {
		return "", fmt.Errorf("image volume target must be an absolute path")
	}
	cleaned := pathpkg.Clean(target)
	if cleaned == "/" {
		return "", fmt.Errorf("image volume target cannot be the container root")
	}
	return cleaned, nil
}
