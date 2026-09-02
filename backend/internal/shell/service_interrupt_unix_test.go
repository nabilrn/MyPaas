//go:build !windows

package shell

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestServiceControlCInterruptsForegroundCommandWithoutEndingSession(t *testing.T) {
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

	// Build the marker at execution time so PTY input echo cannot satisfy the
	// output assertion before the foreground command has actually started.
	if err := service.Write(info.ID, "printf 'mypaas-before-%s\\n' interrupt; sleep 30\n"); err != nil {
		t.Fatalf("start long-running shell command: %v", err)
	}
	waitForShellOutput(t, events, done, "mypaas-before-interrupt", 5*time.Second)

	if err := service.Write(info.ID, "\x03"); err != nil {
		t.Fatalf("interrupt shell command: %v", err)
	}
	// Likewise, require output produced by the shell after Ctrl+C instead of
	// accepting the terminal driver's echo of the command line itself.
	if err := service.Write(info.ID, "printf 'mypaas-shell-%s\\n' survived\n"); err != nil {
		t.Fatalf("write command after interrupt: %v", err)
	}
	waitForShellOutput(t, events, done, "mypaas-shell-survived", 5*time.Second)

	if err := service.Stop(info.ID); err != nil {
		t.Fatalf("stop shell session: %v", err)
	}
	waitForSessionDone(t, done)
}

func waitForShellOutput(t *testing.T, events <-chan Event, done <-chan struct{}, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	var output strings.Builder
	for {
		select {
		case event := <-events:
			if event.Type == "output" {
				output.WriteString(event.Data)
				if strings.Contains(output.String(), want) {
					return
				}
			}
		case <-done:
			t.Fatalf("shell session ended while waiting for %q; output=%q", want, output.String())
		case <-deadline.C:
			t.Fatalf("timed out waiting for %q; output=%q", want, output.String())
		}
	}
}
