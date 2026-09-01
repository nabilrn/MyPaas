package container

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"mypaas/internal/errs"
)

type fakeRuntimeInventory struct {
	containers []RuntimeContainer
	listErr    error
	deleteErr  error
	deletedID  string
}

func (f *fakeRuntimeInventory) RuntimeContainers(context.Context) ([]RuntimeContainer, error) {
	return f.containers, f.listErr
}

func (f *fakeRuntimeInventory) RemoveStopped(_ context.Context, id string) error {
	f.deletedID = id
	return f.deleteErr
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

func TestHandlerDeleteRemovesStoppedContainer(t *testing.T) {
	runtime := &fakeRuntimeInventory{}
	router := chi.NewRouter()
	router.Delete("/admin/containers/{id}", NewHandler(runtime).Delete)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/admin/containers/abc", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if runtime.deletedID != "abc" {
		t.Fatalf("deleted id = %q, want abc", runtime.deletedID)
	}
}

func TestHandlerDeleteRejectsRunningContainer(t *testing.T) {
	runtime := &fakeRuntimeInventory{deleteErr: errors.Join(errs.ErrContainerRunning, errors.New("running"))}
	router := chi.NewRouter()
	router.Delete("/admin/containers/{id}", NewHandler(runtime).Delete)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/admin/containers/abc", nil))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"CONTAINER_RUNNING"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
