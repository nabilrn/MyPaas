package sharedpostgres

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"mypaas/internal/config"
)

func TestDatabaseAndRoleNamesAreDeterministic(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	if got, want := databaseName(id), "mypaas_p_11111111222233334444555555555555"; got != want {
		t.Fatalf("databaseName = %q, want %q", got, want)
	}
	if got := roleName(id); !strings.HasSuffix(got, "_user") {
		t.Fatalf("roleName = %q, want _user suffix", got)
	}
}

func TestQuoteLiteralEscapesSingleQuotes(t *testing.T) {
	if got, want := quoteLiteral("a'b"), "'a''b'"; got != want {
		t.Fatalf("quoteLiteral = %q, want %q", got, want)
	}
}

func TestProjectConnectionLimitUsesConfiguredValue(t *testing.T) {
	service := &Service{cfg: &config.Config{SharedPostgresProjectConnectionLimit: 17}}
	if got := service.projectConnectionLimit(); got != 17 {
		t.Fatalf("projectConnectionLimit() = %d, want 17", got)
	}
}

func TestProjectConnectionLimitHasSafeCompatibilityDefault(t *testing.T) {
	service := &Service{cfg: &config.Config{}}
	if got := service.projectConnectionLimit(); got != 10 {
		t.Fatalf("projectConnectionLimit() = %d, want 10", got)
	}
}
