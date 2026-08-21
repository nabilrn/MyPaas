package caddy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeliveryStatsAggregatesSrv0Metrics(t *testing.T) {
	body := `# HELP caddy_http_requests_total test
caddy_http_requests_total{handler="reverse_proxy",server="srv0"} 120
caddy_http_requests_total{handler="file_server",server="srv0"} 30
caddy_http_requests_total{handler="metrics",server="other"} 999
caddy_http_request_errors_total{handler="reverse_proxy",server="srv0"} 3
caddy_http_requests_in_flight{handler="reverse_proxy",server="srv0"} 4
caddy_http_requests_in_flight{handler="file_server",server="srv0"} 2
caddy_http_response_size_bytes_sum{code="200",handler="reverse_proxy",method="GET",server="srv0"} 1048576
caddy_http_response_size_bytes_sum{code="200",handler="file_server",method="GET",server="srv0"} 524288
caddy_http_request_duration_seconds_count{code="200",handler="reverse_proxy",method="GET",server="srv0"} 110
caddy_http_request_duration_seconds_count{code="200",handler="file_server",method="GET",server="srv0"} 30
caddy_http_request_duration_seconds_count{code="500",handler="reverse_proxy",method="GET",server="srv0"} 10
caddy_http_request_duration_seconds_count{code="503",handler="reverse_proxy",method="GET",server="other"} 999
caddy_http_request_duration_seconds_bucket{code="200",handler="reverse_proxy",method="GET",server="srv0",le="0.1"} 80
caddy_http_request_duration_seconds_bucket{code="200",handler="file_server",method="GET",server="srv0",le="0.1"} 20
caddy_http_request_duration_seconds_bucket{code="200",handler="reverse_proxy",method="GET",server="srv0",le="0.5"} 110
caddy_http_request_duration_seconds_bucket{code="200",handler="file_server",method="GET",server="srv0",le="0.5"} 28
caddy_http_request_duration_seconds_bucket{code="200",handler="reverse_proxy",method="GET",server="srv0",le="+Inf"} 120
caddy_http_request_duration_seconds_bucket{code="200",handler="file_server",method="GET",server="srv0",le="+Inf"} 30
caddy_http_response_duration_seconds_bucket{code="200",handler="reverse_proxy",method="GET",server="srv0",le="0.05"} 90
caddy_http_response_duration_seconds_bucket{code="200",handler="file_server",method="GET",server="srv0",le="0.05"} 20
caddy_http_response_duration_seconds_bucket{code="200",handler="reverse_proxy",method="GET",server="srv0",le="0.25"} 118
caddy_http_response_duration_seconds_bucket{code="200",handler="file_server",method="GET",server="srv0",le="0.25"} 29
caddy_http_response_duration_seconds_bucket{code="200",handler="reverse_proxy",method="GET",server="srv0",le="+Inf"} 120
caddy_http_response_duration_seconds_bucket{code="200",handler="file_server",method="GET",server="srv0",le="+Inf"} 30
caddy_reverse_proxy_upstreams_healthy{upstream="runtime:3000"} 1
caddy_reverse_proxy_upstreams_healthy{upstream="runtime:3001"} 0
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Fatalf("path = %q, want /metrics", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := &Client{baseURL: server.URL, http: server.Client()}
	stats, err := client.DeliveryStats(context.Background())
	if err != nil {
		t.Fatalf("DeliveryStats() error = %v", err)
	}
	if stats.RequestsTotal != 150 {
		t.Fatalf("requests total = %v, want 150", stats.RequestsTotal)
	}
	if stats.RequestErrorsTotal != 3 {
		t.Fatalf("request errors total = %v, want 3", stats.RequestErrorsTotal)
	}
	if stats.RequestsInFlight != 6 {
		t.Fatalf("requests in flight = %v, want 6", stats.RequestsInFlight)
	}
	if stats.ResponseBodyBytesTotal != 1572864 {
		t.Fatalf("response bytes total = %v, want 1572864", stats.ResponseBodyBytesTotal)
	}
	if stats.ResponsesByStatusClass["2xx"] != 140 || stats.ResponsesByStatusClass["5xx"] != 10 {
		t.Fatalf("status classes = %#v, want 2xx=140 and 5xx=10", stats.ResponsesByStatusClass)
	}
	if stats.UpstreamsHealthy != 1 || stats.UpstreamsTotal != 2 {
		t.Fatalf("upstreams = %d/%d healthy, want 1/2", stats.UpstreamsHealthy, stats.UpstreamsTotal)
	}
	if len(stats.RequestDurationBuckets) != 3 {
		t.Fatalf("duration bucket count = %d, want 3", len(stats.RequestDurationBuckets))
	}
	if stats.RequestDurationBuckets[0].UpperBound != "0.1" || stats.RequestDurationBuckets[0].Count != 100 {
		t.Fatalf("first bucket = %#v, want le=0.1 count=100", stats.RequestDurationBuckets[0])
	}
	if stats.RequestDurationBuckets[1].UpperBound != "0.5" || stats.RequestDurationBuckets[1].Count != 138 {
		t.Fatalf("second bucket = %#v, want le=0.5 count=138", stats.RequestDurationBuckets[1])
	}
	if stats.RequestDurationBuckets[2].UpperBound != "+Inf" || stats.RequestDurationBuckets[2].Count != 150 {
		t.Fatalf("last bucket = %#v, want le=+Inf count=150", stats.RequestDurationBuckets[2])
	}
	if len(stats.ResponseTTFBBuckets) != 3 {
		t.Fatalf("TTFB bucket count = %d, want 3", len(stats.ResponseTTFBBuckets))
	}
	if stats.ResponseTTFBBuckets[0].UpperBound != "0.05" || stats.ResponseTTFBBuckets[0].Count != 110 {
		t.Fatalf("first TTFB bucket = %#v, want le=0.05 count=110", stats.ResponseTTFBBuckets[0])
	}
	if stats.ResponseTTFBBuckets[1].UpperBound != "0.25" || stats.ResponseTTFBBuckets[1].Count != 147 {
		t.Fatalf("second TTFB bucket = %#v, want le=0.25 count=147", stats.ResponseTTFBBuckets[1])
	}
	if stats.ResponseTTFBBuckets[2].UpperBound != "+Inf" || stats.ResponseTTFBBuckets[2].Count != 150 {
		t.Fatalf("last TTFB bucket = %#v, want le=+Inf count=150", stats.ResponseTTFBBuckets[2])
	}
}

func TestDeliveryStatsRejectsNonSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "metrics disabled", http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{baseURL: server.URL, http: server.Client()}
	if _, err := client.DeliveryStats(context.Background()); err == nil {
		t.Fatal("DeliveryStats() error = nil, want error")
	}
}

func TestParsePrometheusSample(t *testing.T) {
	name, labels, value, ok := parsePrometheusSample(`caddy_http_requests_total{handler="reverse_proxy",server="srv0"} 42`)
	if !ok {
		t.Fatal("parsePrometheusSample() ok = false")
	}
	if name != "caddy_http_requests_total" || labels["handler"] != "reverse_proxy" || labels["server"] != "srv0" || value != 42 {
		t.Fatalf("unexpected parsed sample: name=%q labels=%v value=%v", name, labels, value)
	}
}
