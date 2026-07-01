package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"ref":"refs/heads/main"}`)
	secret := "webhook-secret"

	if !verifySignature(secret, body, signatureHeader(secret, body)) {
		t.Fatal("expected valid signature")
	}
	if verifySignature(secret, body, signatureHeader("different-secret", body)) {
		t.Fatal("expected signature with different secret to be invalid")
	}
	if verifySignature(secret, body, "sha1=legacy") {
		t.Fatal("expected non-sha256 signature to be invalid")
	}
	if verifySignature(secret, body, "sha256=not-hex") {
		t.Fatal("expected malformed hex signature to be invalid")
	}
}

func TestBranchFromRef(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{ref: "refs/heads/main", want: "main"},
		{ref: "refs/heads/feature/deploy", want: "feature/deploy"},
		{ref: "refs/tags/v1.0.0", want: "refs/tags/v1.0.0"},
		{ref: "", want: ""},
	}

	for _, tt := range tests {
		if got := branchFromRef(tt.ref); got != tt.want {
			t.Fatalf("branchFromRef(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	limiter := newRateLimiter(2, time.Minute, func() time.Time { return now })
	projectID := uuid.New()

	if !limiter.allow(projectID) {
		t.Fatal("first request should be allowed")
	}
	if !limiter.allow(projectID) {
		t.Fatal("second request should be allowed")
	}
	if limiter.allow(projectID) {
		t.Fatal("third request in same window should be rejected")
	}

	now = now.Add(time.Minute)
	if !limiter.allow(projectID) {
		t.Fatal("request after window reset should be allowed")
	}
}

func signatureHeader(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
