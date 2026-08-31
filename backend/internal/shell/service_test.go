package shell

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"mypaas/internal/errs"
)

func TestServiceLifecycle(t *testing.T) {
	service := NewService()
	defer service.Close()

	info, err := service.Start(context.Background())
	if err != nil {
		t.Skipf("host shell is unavailable: %v", err)
	}

	events, done, _, unsubscribe, err := service.Subscribe(info.ID)
	if err != nil {
		t.Fatalf("subscribe to shell session: %v", err)
	}
	defer unsubscribe()

	if err := service.Write(info.ID, "echo mypaas-shell-test\n"); err != nil {
		t.Fatalf("write shell input: %v", err)
	}

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case event := <-events:
			if event.Type == "output" && strings.Contains(event.Data, "mypaas-shell-test") {
				if err := service.Stop(info.ID); err != nil {
					t.Fatalf("stop shell session: %v", err)
				}
				waitForSessionDone(t, done)
				return
			}
		case <-done:
			t.Fatal("shell session ended before producing output")
		case <-deadline.C:
			t.Fatal("timed out waiting for shell output")
		}
	}
}

func TestServiceRejectsOversizedInput(t *testing.T) {
	service := NewService()

	err := service.Write(uuid.New(), strings.Repeat("x", maxInputBytes+1))
	if err != errs.ErrShellInputTooLarge {
		t.Fatalf("expected oversized input error, got %v", err)
	}
}

func waitForSessionDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for shell session to stop")
	}
}
