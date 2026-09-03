package container

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRuntimeInventory struct {
	containers []RuntimeContainer
	metadata   []RuntimeContainer
	listErr    error
}

func (f *fakeRuntimeInventory) RuntimeContainers(context.Context) ([]RuntimeContainer, error) {
	return f.containers, f.listErr
}

func (f *fakeRuntimeInventory) RuntimeContainerMetadata(context.Context) ([]RuntimeContainer, error) {
	return f.metadata, nil
}

func TestHandlerListReturnsInventory(t *testing.T) {
	runtime := &fakeRuntimeInventory{containers: []RuntimeContainer{{ID: "abc", Name: "demo"}}}
	recorder := httptest.NewRecorder()
	NewHandler(runtime).List(recorder, httptest.NewRequest(http.MethodGet, "/admin/containers", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"id":"abc"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestHandlerDeleteKeepsInventoryReadOnly(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewHandler(&fakeRuntimeInventory{}).Delete(recorder, httptest.NewRequest(http.MethodDelete, "/admin/containers/abc", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"CONTAINER_INVENTORY_READ_ONLY"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestHandlerListReturnsMetadataWithoutTelemetry(t *testing.T) {
	runtime := &fakeRuntimeInventory{
		containers: []RuntimeContainer{{ID: "telemetry", Name: "slow"}},
		metadata:   []RuntimeContainer{{ID: "metadata", Name: "fast"}},
	}
	recorder := httptest.NewRecorder()
	NewHandler(runtime).List(recorder, httptest.NewRequest(http.MethodGet, "/admin/containers?telemetry=false", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"id":"metadata"`) || strings.Contains(recorder.Body.String(), `"id":"telemetry"`) {
		t.Fatalf("body = %s; want metadata-only response", recorder.Body.String())
	}
}
