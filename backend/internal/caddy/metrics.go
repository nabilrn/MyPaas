package caddy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const caddyHTTPServerName = "srv0"

type HistogramBucket struct {
	UpperBound string  `json:"le"`
	Count      float64 `json:"count"`
}

type DeliveryStats struct {
	SampledAtUnixMs        int64              `json:"sampled_at_unix_ms"`
	RequestsTotal          float64            `json:"requests_total"`
	RequestErrorsTotal     float64            `json:"request_errors_total"`
	RequestsInFlight       float64            `json:"requests_in_flight"`
	ResponseBodyBytesTotal float64            `json:"response_body_bytes_total"`
	ResponsesByStatusClass map[string]float64 `json:"responses_by_status_class"`
	RequestDurationBuckets []HistogramBucket  `json:"request_duration_buckets"`
	ResponseTTFBBuckets    []HistogramBucket  `json:"response_ttfb_buckets"`
	UpstreamsHealthy       int                `json:"upstreams_healthy"`
	UpstreamsTotal         int                `json:"upstreams_total"`
}

// DeliveryStats returns a compact snapshot from Caddy's native Prometheus
// metrics endpoint. The admin API is normally exposed only through the shared
// Unix socket, so these metrics do not need another public listener.
func (c *Client) DeliveryStats(ctx context.Context) (DeliveryStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/metrics", nil)
	if err != nil {
		return DeliveryStats{}, err
	}
	req.Header.Set("Accept", "text/plain; version=0.0.4")

	resp, err := c.http.Do(req)
	if err != nil {
		return DeliveryStats{}, fmt.Errorf("caddy metrics request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DeliveryStats{}, fmt.Errorf("caddy metrics returned %s", resp.Status)
	}

	stats := DeliveryStats{
		SampledAtUnixMs:        time.Now().UnixMilli(),
		ResponsesByStatusClass: map[string]float64{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0},
	}
	requestBuckets := make(map[string]float64)
	ttfbBuckets := make(map[string]float64)
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 2<<20))
	scanner.Buffer(make([]byte, 64*1024), 512*1024)
	for scanner.Scan() {
		name, labels, value, ok := parsePrometheusSample(scanner.Text())
		if !ok {
			continue
		}

		if strings.HasPrefix(name, "caddy_http_") && labels["server"] != caddyHTTPServerName {
			continue
		}

		switch name {
		case "caddy_http_requests_total":
			stats.RequestsTotal += value
		case "caddy_http_request_errors_total":
			stats.RequestErrorsTotal += value
		case "caddy_http_requests_in_flight":
			stats.RequestsInFlight += value
		case "caddy_http_response_size_bytes_sum":
			stats.ResponseBodyBytesTotal += value
		case "caddy_http_request_duration_seconds_count":
			if class := statusClass(labels["code"]); class != "" {
				stats.ResponsesByStatusClass[class] += value
			}
		case "caddy_http_request_duration_seconds_bucket":
			if upperBound := labels["le"]; upperBound != "" {
				requestBuckets[upperBound] += value
			}
		case "caddy_http_response_duration_seconds_bucket":
			if upperBound := labels["le"]; upperBound != "" {
				ttfbBuckets[upperBound] += value
			}
		case "caddy_reverse_proxy_upstreams_healthy":
			stats.UpstreamsTotal++
			if value >= 0.5 {
				stats.UpstreamsHealthy++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return DeliveryStats{}, fmt.Errorf("scan caddy metrics: %w", err)
	}

	stats.RequestDurationBuckets = sortedHistogramBuckets(requestBuckets)
	stats.ResponseTTFBBuckets = sortedHistogramBuckets(ttfbBuckets)
	return stats, nil
}

func sortedHistogramBuckets(values map[string]float64) []HistogramBucket {
	buckets := make([]HistogramBucket, 0, len(values))
	for upperBound, count := range values {
		buckets = append(buckets, HistogramBucket{UpperBound: upperBound, Count: count})
	}
	sort.Slice(buckets, func(i, j int) bool {
		return prometheusUpperBound(buckets[i].UpperBound) < prometheusUpperBound(buckets[j].UpperBound)
	})
	return buckets
}

func statusClass(code string) string {
	if len(code) != 3 {
		return ""
	}
	switch code[0] {
	case '2', '3', '4', '5':
		return string(code[0]) + "xx"
	default:
		return ""
	}
}

func parsePrometheusSample(line string) (string, map[string]string, float64, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", nil, 0, false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", nil, 0, false
	}
	value, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return "", nil, 0, false
	}

	spec := fields[0]
	labels := map[string]string{}
	name := spec
	if open := strings.IndexByte(spec, '{'); open >= 0 {
		close := strings.LastIndexByte(spec, '}')
		if close <= open {
			return "", nil, 0, false
		}
		name = spec[:open]
		for _, raw := range strings.Split(spec[open+1:close], ",") {
			parts := strings.SplitN(raw, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			labelValue := strings.TrimSpace(parts[1])
			if unquoted, err := strconv.Unquote(labelValue); err == nil {
				labels[key] = unquoted
			} else {
				labels[key] = strings.Trim(labelValue, `"`)
			}
		}
	}
	if name == "" {
		return "", nil, 0, false
	}
	return name, labels, value, true
}

func prometheusUpperBound(value string) float64 {
	if value == "+Inf" || value == "Inf" {
		return 1e308
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 1e308
	}
	return parsed
}
