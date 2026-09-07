package mcpserver

import (
	"testing"
	"time"
)

func TestFixedWindowLimiterIsPerToken(t *testing.T) {
	limiter := newLimiter(2, time.Minute)
	if !limiter.allow("myp_first") || !limiter.allow("myp_first") {
		t.Fatal("first token should be allowed up to the limit")
	}
	if limiter.allow("myp_first") {
		t.Fatal("first token should be rejected after reaching the limit")
	}
	if !limiter.allow("myp_second") {
		t.Fatal("second token should have an independent rate bucket")
	}
}

func TestFixedWindowLimiterResets(t *testing.T) {
	limiter := newLimiter(1, time.Millisecond)
	if !limiter.allow("myp_token") {
		t.Fatal("first request should be allowed")
	}
	time.Sleep(2 * time.Millisecond)
	if !limiter.allow("myp_token") {
		t.Fatal("request should be allowed after the window resets")
	}
}
