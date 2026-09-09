package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type apiErrorBody struct {
	Error struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Details json.RawMessage `json:"details,omitempty"`
	} `json:"error"`
}

type apiHTTPError struct {
	StatusCode int
	Message    string
}

func (e *apiHTTPError) Error() string {
	return e.Message
}

type apiClient struct {
	baseURL string
	token   string
	tool    string
	http    *http.Client
}

func newAPIClient(baseURL string) *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *apiClient) withToken(token string) *apiClient {
	return &apiClient{baseURL: c.baseURL, token: token, http: c.http}
}

func (c *apiClient) withTool(tool string) *apiClient {
	return &apiClient{baseURL: c.baseURL, token: c.token, tool: strings.TrimSpace(tool), http: c.http}
}

func (c *apiClient) request(ctx context.Context, method, path string, payload any) (string, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return "", err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.tool != "" {
		// Audit metadata only. Authorization remains derived entirely from the
		// scoped bearer token at the MyPaaS REST boundary.
		req.Header.Set("X-MyPaaS-Source", "mcp")
		req.Header.Set("X-MyPaaS-MCP-Tool", c.tool)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", formatAPIError(resp.StatusCode, data)
	}
	return string(data), nil
}

func formatAPIError(status int, data []byte) error {
	message := ""
	var apiErr apiErrorBody
	if json.Unmarshal(data, &apiErr) == nil && apiErr.Error.Message != "" {
		if apiErr.Error.Code != "" {
			message = fmt.Sprintf("MyPaaS API %d %s: %s", status, apiErr.Error.Code, apiErr.Error.Message)
		} else {
			message = fmt.Sprintf("MyPaaS API %d: %s", status, apiErr.Error.Message)
		}
	} else if len(data) == 0 {
		message = fmt.Sprintf("MyPaaS API %d", status)
	} else {
		message = fmt.Sprintf("MyPaaS API %d: %s", status, strings.TrimSpace(string(data)))
	}
	return &apiHTTPError{StatusCode: status, Message: message}
}
