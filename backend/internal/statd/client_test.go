package statd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"
)

type fakeServer struct {
	path string
	done <-chan error
}

func startFakeServer(t *testing.T, handle func(*bufio.Reader, net.Conn) error) fakeServer {
	t.Helper()
	path := filepath.Join(t.TempDir(), "statd.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		done <- handle(bufio.NewReader(conn), conn)
	}()
	return fakeServer{path: path, done: done}
}

func expectLine(reader *bufio.Reader, want string) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if line != want {
		return fmt.Errorf("line mismatch: got %q want %q", line, want)
	}
	return nil
}

func hello(reader *bufio.Reader, conn net.Conn) error {
	if err := expectLine(reader, "{\"op\":\"hello\",\"protocol\":1}\n"); err != nil {
		return err
	}
	_, err := conn.Write([]byte("{\"ok\":true,\"protocol\":1,\"agent\":\"mypaas-statd\",\"version\":\"0.1.0-dev\"}\n"))
	return err
}

func waitServer(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("fake server: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fake server did not finish")
	}
}

func TestSnapshot(t *testing.T) {
	server := startFakeServer(t, func(reader *bufio.Reader, conn net.Conn) error {
		if err := hello(reader, conn); err != nil {
			return err
		}
		if err := expectLine(reader, "{\"op\":\"snapshot\",\"id\":\"runtime-1\"}\n"); err != nil {
			return err
		}
		_, err := conn.Write([]byte("{\"ok\":true,\"protocol\":1,\"valid\":true,\"stale\":false,\"cpu\":{\"usage_usec\":1200,\"percent\":12.5,\"quota_usec\":null,\"period_usec\":100000},\"memory\":{\"current_bytes\":4096,\"max_bytes\":null,\"oom\":2,\"oom_kill\":1},\"pids\":{\"current\":7,\"max\":64}}\n"))
		return err
	})

	snapshot, err := NewClient(server.path).Snapshot(context.Background(), "runtime-1")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !snapshot.Valid || snapshot.Stale {
		t.Fatalf("unexpected validity: %+v", snapshot)
	}
	if snapshot.CPU.Percent == nil || *snapshot.CPU.Percent != 12.5 {
		t.Fatalf("unexpected cpu percent: %+v", snapshot.CPU.Percent)
	}
	if snapshot.CPU.QuotaUsec != nil || snapshot.Memory.MaxBytes != nil {
		t.Fatal("unlimited values must decode as nil")
	}
	if snapshot.Memory.CurrentBytes != 4096 || snapshot.PIDs.Current != 7 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	waitServer(t, server.done)
}

func TestRegisterByPIDProtocolError(t *testing.T) {
	server := startFakeServer(t, func(reader *bufio.Reader, conn net.Conn) error {
		if err := hello(reader, conn); err != nil {
			return err
		}
		if err := expectLine(reader, "{\"op\":\"register\",\"id\":\"runtime-1\",\"pid\":4321}\n"); err != nil {
			return err
		}
		_, err := conn.Write([]byte("{\"ok\":false,\"error\":{\"code\":\"REGISTRATION_LIMIT\"}}\n"))
		return err
	})

	err := NewClient(server.path).Register(context.Background(), "runtime-1", 4321)
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) || protocolErr.Code != "REGISTRATION_LIMIT" {
		t.Fatalf("expected registration protocol error, got %v", err)
	}
	waitServer(t, server.done)
}

func TestStatus(t *testing.T) {
	server := startFakeServer(t, func(reader *bufio.Reader, conn net.Conn) error {
		if err := hello(reader, conn); err != nil {
			return err
		}
		if err := expectLine(reader, "{\"op\":\"status\"}\n"); err != nil {
			return err
		}
		_, err := conn.Write([]byte("{\"ok\":true,\"protocol\":1,\"registrations\":3}\n"))
		return err
	})

	status, err := NewClient(server.path).Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Registrations != 3 {
		t.Fatalf("registrations = %d, want 3", status.Registrations)
	}
	waitServer(t, server.done)
}

func TestRegisterRejectsInvalidInputBeforeDial(t *testing.T) {
	client := NewClient(filepath.Join(t.TempDir(), "missing.sock"))
	if err := client.Register(context.Background(), "bad\"id", 123); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid id, got %v", err)
	}
	if err := client.Register(context.Background(), "runtime-1", 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid pid, got %v", err)
	}
	if err := client.Register(context.Background(), "runtime-1", -1); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid pid, got %v", err)
	}
}

func TestProtocolMismatch(t *testing.T) {
	server := startFakeServer(t, func(reader *bufio.Reader, conn net.Conn) error {
		if err := expectLine(reader, "{\"op\":\"hello\",\"protocol\":1}\n"); err != nil {
			return err
		}
		_, err := conn.Write([]byte("{\"ok\":true,\"protocol\":2}\n"))
		return err
	})

	_, err := NewClient(server.path).Status(context.Background())
	if err == nil {
		t.Fatal("expected protocol mismatch")
	}
	waitServer(t, server.done)
}
