package firewall

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const defaultTimeout = 3 * time.Second

type Rule struct {
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

type Status struct {
	Available bool   `json:"available"`
	Active    bool   `json:"active"`
	Rules     []Rule `json:"rules"`
}

type Client struct {
	socketPath string
	timeout    time.Duration
}

type response struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error"`
	Available bool   `json:"available"`
	Active    bool   `json:"active"`
	Rules     []Rule `json:"rules"`
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: strings.TrimSpace(socketPath), timeout: defaultTimeout}
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var out response
	if err := c.exchange(ctx, map[string]any{"op": "status"}, &out); err != nil {
		return Status{}, err
	}
	return Status{Available: out.Available, Active: out.Active, Rules: out.Rules}, nil
}

func (c *Client) Allow(ctx context.Context, port int32, protocol string) error {
	protocol, err := validateRule(port, protocol)
	if err != nil {
		return err
	}
	return c.exchange(ctx, map[string]any{"op": "allow", "port": port, "protocol": protocol}, nil)
}

func (c *Client) Delete(ctx context.Context, port int32, protocol string) error {
	protocol, err := validateRule(port, protocol)
	if err != nil {
		return err
	}
	return c.exchange(ctx, map[string]any{"op": "delete", "port": port, "protocol": protocol}, nil)
}

func validateRule(port int32, protocol string) (string, error) {
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("port must be between 1 and 65535")
	}
	if port == 22 || port == 80 || port == 443 {
		return "", fmt.Errorf("port %d is protected by MyPaaS", port)
	}
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol != "tcp" && protocol != "udp" {
		return "", fmt.Errorf("protocol must be tcp or udp")
	}
	return protocol, nil
}

func (c *Client) exchange(ctx context.Context, request any, target *response) error {
	if c == nil || c.socketPath == "" {
		return fmt.Errorf("firewall helper socket is not configured")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("firewall helper connect: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	if _, err := conn.Write(payload); err != nil {
		return fmt.Errorf("firewall helper write: %w", err)
	}
	line, err := bufio.NewReaderSize(conn, 16*1024).ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("firewall helper read: %w", err)
	}
	var out response
	if err := json.Unmarshal(line, &out); err != nil {
		return fmt.Errorf("firewall helper response: %w", err)
	}
	if !out.OK {
		if strings.TrimSpace(out.Error) == "" {
			out.Error = "operation failed"
		}
		return fmt.Errorf("firewall helper: %s", out.Error)
	}
	if target != nil {
		*target = out
	}
	return nil
}
