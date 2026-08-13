package httpx

import (
	"fmt"
	"strings"
	"testing"

	"mypaas/internal/errs"
)

func TestValidationMessageHidesRepositoryInspectionImplementationDetail(t *testing.T) {
	err := fmt.Errorf("%w: failed to inspect remote branches: command failed", errs.ErrValidation)
	message := validationMessage(err)

	if strings.Contains(message, "remote branches") || strings.Contains(message, "command failed") {
		t.Fatalf("expected task-oriented repository error, got %q", message)
	}
	if !strings.Contains(message, "Repository could not be inspected") {
		t.Fatalf("expected repository guidance, got %q", message)
	}
}

func TestValidationMessagePreservesOtherValidationErrors(t *testing.T) {
	err := fmt.Errorf("%w: branch is required", errs.ErrValidation)
	if got := validationMessage(err); got != err.Error() {
		t.Fatalf("expected unrelated validation message to remain unchanged, got %q", got)
	}
}
