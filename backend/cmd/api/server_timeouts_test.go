package main

import (
	"testing"
	"time"
)

func TestAPIWriteTimeoutExceedsRequestAndStopWindows(t *testing.T) {
	if apiWriteTimeout <= apiRequestTimeout {
		t.Fatalf("apiWriteTimeout=%s must exceed request timeout=%s so timeout responses can be written cleanly", apiWriteTimeout, apiRequestTimeout)
	}
	if apiWriteTimeout <= 30*time.Second {
		t.Fatalf("apiWriteTimeout=%s must exceed docker stop grace window", apiWriteTimeout)
	}
}
