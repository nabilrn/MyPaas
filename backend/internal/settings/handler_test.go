package settings

import (
	"errors"
	"testing"

	"mypaas/internal/statd"
)

func TestHostTelemetryErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "protocol", err: &statd.ProtocolError{Code: "INVALID_REQUEST"}, want: "INVALID_REQUEST"},
		{name: "protocol without code", err: &statd.ProtocolError{}, want: "PROTOCOL_ERROR"},
		{name: "invalid config", err: statd.ErrInvalidInput, want: "INVALID_CONFIG"},
		{name: "wrapped invalid config", err: errors.Join(errors.New("outer"), statd.ErrInvalidInput), want: "INVALID_CONFIG"},
		{name: "io", err: errors.New("dial unix: permission denied"), want: "CONNECT_OR_IO_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostTelemetryErrorCode(tt.err); got != tt.want {
				t.Fatalf("hostTelemetryErrorCode() = %q, want %q", got, tt.want)
			}
		})
	}
}
