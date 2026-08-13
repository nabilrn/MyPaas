package statd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	protocolVersion = 1
	responseMax     = 4096
	runtimeIDMax    = 127
	defaultTimeout  = 2 * time.Second
)

var ErrInvalidInput = errors.New("statd invalid input")

type ProtocolError struct {
	Code string
}

func (e *ProtocolError) Error() string {
	if e == nil || e.Code == "" {
		return "statd protocol error"
	}
	return "statd protocol error: " + e.Code
}

type Client struct {
	socketPath string
	timeout    time.Duration
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath, timeout: defaultTimeout}
}

type Status struct {
	Registrations int `json:"registrations"`
}

type Snapshot struct {
	Valid  bool           `json:"valid"`
	Stale  bool           `json:"stale"`
	CPU    CPUSnapshot    `json:"cpu"`
	Memory MemorySnapshot `json:"memory"`
	PIDs   PIDSnapshot    `json:"pids"`
}

type CPUSnapshot struct {
	UsageUsec  uint64   `json:"usage_usec"`
	Percent    *float64 `json:"percent"`
	QuotaUsec  *uint64  `json:"quota_usec"`
	PeriodUsec uint64   `json:"period_usec"`
}

type MemorySnapshot struct {
	CurrentBytes uint64  `json:"current_bytes"`
	MaxBytes     *uint64 `json:"max_bytes"`
	OOM          uint64  `json:"oom"`
	OOMKill      uint64  `json:"oom_kill"`
}

type PIDSnapshot struct {
	Current uint64  `json:"current"`
	Max     *uint64 `json:"max"`
}

type HostSnapshot struct {
	Memory  *HostMemorySnapshot  `json:"memory"`
	CPU     *HostCPUSnapshot     `json:"cpu"`
	Storage *HostStorageSnapshot `json:"storage"`
	Network *HostNetworkSnapshot `json:"network"`
}

type HostMemorySnapshot struct {
	TotalBytes     uint64 `json:"total_bytes"`
	AvailableBytes uint64 `json:"available_bytes"`
}

type HostCPUSnapshot struct {
	TotalTicks uint64 `json:"total_ticks"`
	IdleTicks  uint64 `json:"idle_ticks"`
}

type HostStorageSnapshot struct {
	TotalBytes     uint64 `json:"total_bytes"`
	AvailableBytes uint64 `json:"available_bytes"`
}

type HostNetworkSnapshot struct {
	Interface string `json:"interface"`
	RXBytes   uint64 `json:"rx_bytes"`
	TXBytes   uint64 `json:"tx_bytes"`
}

type wireResponse struct {
	OK       bool `json:"ok"`
	Protocol int  `json:"protocol"`
	Error    struct {
		Code string `json:"code"`
	} `json:"error"`
}

// Register associates a MyPaaS runtime ID with the host PID reported by Docker.
// statd resolves cgroup v2 membership from /proc/<pid>/cgroup on the host.
func (c *Client) Register(ctx context.Context, id string, pid int) error {
	if err := validateASCII("id", id, runtimeIDMax); err != nil {
		return err
	}
	if pid <= 0 {
		return fmt.Errorf("%w: pid must be positive", ErrInvalidInput)
	}
	request := fmt.Sprintf("{\"op\":\"register\",\"id\":\"%s\",\"pid\":%d}\n", id, pid)
	return c.exchange(ctx, request, nil)
}

func (c *Client) Unregister(ctx context.Context, id string) error {
	if err := validateASCII("id", id, runtimeIDMax); err != nil {
		return err
	}
	request := fmt.Sprintf("{\"op\":\"unregister\",\"id\":\"%s\"}\n", id)
	return c.exchange(ctx, request, nil)
}

func (c *Client) Snapshot(ctx context.Context, id string) (Snapshot, error) {
	var snapshot Snapshot
	if err := validateASCII("id", id, runtimeIDMax); err != nil {
		return snapshot, err
	}
	request := fmt.Sprintf("{\"op\":\"snapshot\",\"id\":\"%s\"}\n", id)
	if err := c.exchange(ctx, request, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (c *Client) HostSnapshot(ctx context.Context) (HostSnapshot, error) {
	var snapshot HostSnapshot
	if err := c.exchange(ctx, "{\"op\":\"host_snapshot\"}\n", &snapshot); err != nil {
		return HostSnapshot{}, err
	}
	return snapshot, nil
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var status Status
	if err := c.exchange(ctx, "{\"op\":\"status\"}\n", &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (c *Client) exchange(ctx context.Context, request string, target any) error {
	if c == nil || c.socketPath == "" {
		return fmt.Errorf("%w: socket path is required", ErrInvalidInput)
	}

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("statd connect: %w", err)
	}
	defer conn.Close()

	if deadline, ok := operationDeadline(ctx, c.timeout); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("statd deadline: %w", err)
		}
	}

	reader := bufio.NewReaderSize(conn, responseMax)
	if err := writeFull(conn, "{\"op\":\"hello\",\"protocol\":1}\n"); err != nil {
		return fmt.Errorf("statd hello write: %w", err)
	}
	hello, err := readWireResponse(reader)
	if err != nil {
		return fmt.Errorf("statd hello read: %w", err)
	}
	if !hello.OK {
		return &ProtocolError{Code: hello.Error.Code}
	}
	if hello.Protocol != protocolVersion {
		return fmt.Errorf("statd protocol mismatch: got %d want %d", hello.Protocol, protocolVersion)
	}

	if err := writeFull(conn, request); err != nil {
		return fmt.Errorf("statd request write: %w", err)
	}
	payload, response, err := readResponse(reader)
	if err != nil {
		return fmt.Errorf("statd response read: %w", err)
	}
	if !response.OK {
		return &ProtocolError{Code: response.Error.Code}
	}
	if response.Protocol != protocolVersion {
		return fmt.Errorf("statd protocol mismatch: got %d want %d", response.Protocol, protocolVersion)
	}
	if target != nil {
		if err := json.Unmarshal(payload, target); err != nil {
			return fmt.Errorf("statd decode response: %w", err)
		}
	}
	return nil
}

func validateASCII(name, value string, max int) error {
	if value == "" || len(value) > max {
		return fmt.Errorf("%w: %s length", ErrInvalidInput, name)
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < 0x20 || ch > 0x7e || ch == '"' || ch == '\\' {
			return fmt.Errorf("%w: %s contains unsupported character", ErrInvalidInput, name)
		}
	}
	return nil
}

func operationDeadline(ctx context.Context, timeout time.Duration) (time.Time, bool) {
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		return ctxDeadline, true
	}
	if timeout > 0 {
		return deadline, true
	}
	return time.Time{}, false
}

func writeFull(w io.Writer, value string) error {
	for len(value) > 0 {
		n, err := io.WriteString(w, value)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrUnexpectedEOF
		}
		value = value[n:]
	}
	return nil
}

func readWireResponse(reader *bufio.Reader) (wireResponse, error) {
	_, response, err := readResponse(reader)
	return response, err
}

func readResponse(reader *bufio.Reader) ([]byte, wireResponse, error) {
	var response wireResponse
	line, err := reader.ReadSlice('\n')
	if err != nil {
		return nil, response, err
	}
	if len(line) == 0 || len(line) > responseMax {
		return nil, response, fmt.Errorf("response exceeds %d bytes", responseMax)
	}
	payload := line[:len(line)-1]
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, response, err
	}
	return payload, response, nil
}
