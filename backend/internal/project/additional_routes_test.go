package project

import (
	"encoding/json"
	"errors"
	"testing"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

func TestAdditionalRouteHostLabel(t *testing.T) {
	if got := additionalRouteHostLabel("minio-prod", "console"); got != "minio-prod-console" {
		t.Fatalf("additionalRouteHostLabel() = %q", got)
	}
}

func TestNormalizeAdditionalRoutes(t *testing.T) {
	routes, err := normalizeAdditionalRoutes("compose", []AdditionalRoute{
		{Name: " Console ", Service: "minio", ContainerPort: 9001},
	})
	if err != nil {
		t.Fatalf("normalize routes: %v", err)
	}
	if len(routes) != 1 || routes[0].Name != "console" || routes[0].Service != "minio" || routes[0].ContainerPort != 9001 {
		t.Fatalf("unexpected normalized route: %#v", routes)
	}
}

func TestNormalizeAdditionalRoutesRejectsUnsafeContracts(t *testing.T) {
	tests := []struct {
		name       string
		deployMode string
		routes     []AdditionalRoute
	}{
		{name: "non compose", deployMode: "dockerfile", routes: []AdditionalRoute{{Name: "admin", Service: "app", ContainerPort: 3001}}},
		{name: "duplicate names", deployMode: "compose", routes: []AdditionalRoute{{Name: "admin", Service: "app", ContainerPort: 3001}, {Name: "admin", Service: "other", ContainerPort: 3002}}},
		{name: "unsafe name", deployMode: "compose", routes: []AdditionalRoute{{Name: "Admin UI", Service: "app", ContainerPort: 3001}}},
		{name: "missing service", deployMode: "compose", routes: []AdditionalRoute{{Name: "admin", Service: "", ContainerPort: 3001}}},
		{name: "invalid port", deployMode: "compose", routes: []AdditionalRoute{{Name: "admin", Service: "app", ContainerPort: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeAdditionalRoutes(tt.deployMode, tt.routes)
			if !errors.Is(err, errs.ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestAdditionalRoutesRoundTrip(t *testing.T) {
	want := []AdditionalRoute{{Name: "console", Service: "minio", ContainerPort: 9001}}
	raw, err := encodeAdditionalRoutes(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeAdditionalRoutes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("round trip mismatch: got %#v want %#v", got, want)
	}
}

func TestValidateAdditionalRouteTargetsRequiresDeclaredComposePort(t *testing.T) {
	main := "minio"
	project := db.Project{DeployMode: "compose", MainService: &main, AppPort: 9000}
	doc := composeConfigDoc{Services: map[string]composeServiceConfig{
		"minio": {
			Ports:  json.RawMessage(`[{"target":9000,"published":"9000","protocol":"tcp"}]`),
			Expose: json.RawMessage(`["9001"]`),
		},
	}}

	if err := validateAdditionalRouteTargets(project, doc, []AdditionalRoute{{Name: "console", Service: "minio", ContainerPort: 9001}}); err != nil {
		t.Fatalf("expected declared console port to pass: %v", err)
	}

	for _, route := range []AdditionalRoute{
		{Name: "missing-service", Service: "console", ContainerPort: 9001},
		{Name: "undeclared", Service: "minio", ContainerPort: 9443},
		{Name: "primary", Service: "minio", ContainerPort: 9000},
	} {
		if err := validateAdditionalRouteTargets(project, doc, []AdditionalRoute{route}); !errors.Is(err, errs.ErrValidation) {
			t.Fatalf("expected validation error for %#v, got %v", route, err)
		}
	}
}