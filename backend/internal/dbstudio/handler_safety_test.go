package dbstudio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeleteIsDisabled(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/projects/test/db/rows", strings.NewReader(`{"schema":"public","table":"users"}`))
	recorder := httptest.NewRecorder()

	handler.Delete(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
	if got := recorder.Header().Get("Allow"); got != "GET, POST, PATCH" {
		t.Fatalf("expected safe mutation methods in Allow header, got %q", got)
	}
	if !strings.Contains(recorder.Body.String(), "DBSTUDIO_DELETE_DISABLED") {
		t.Fatalf("expected delete-disabled error code, got %s", recorder.Body.String())
	}
}
