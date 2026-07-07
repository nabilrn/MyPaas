package quota

import (
	"net/http/httptest"
	"testing"
)

func TestIncludeRuntimeQuery(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "missing", raw: "", want: false},
		{name: "true", raw: "?includeRuntime=true", want: true},
		{name: "one", raw: "?includeRuntime=1", want: true},
		{name: "false", raw: "?includeRuntime=false", want: false},
		{name: "invalid", raw: "?includeRuntime=maybe", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/me/quota"+tt.raw, nil)
			if got := includeRuntime(req); got != tt.want {
				t.Fatalf("includeRuntime() = %v, want %v", got, tt.want)
			}
		})
	}
}
