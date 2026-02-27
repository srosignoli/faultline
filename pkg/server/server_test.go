package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/srosignoli/faultline/pkg/mutator"
	"github.com/srosignoli/faultline/pkg/parser"
	"github.com/srosignoli/faultline/pkg/server"
)

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("body does not contain %q\nbody:\n%s", substr, body)
	}
}

func assertNotContains(t *testing.T, body, substr string) {
	t.Helper()
	if strings.Contains(body, substr) {
		t.Errorf("body should not contain %q\nbody:\n%s", substr, body)
	}
}

func countOccurrences(s, sub string) int {
	return strings.Count(s, sub)
}

func doRequest(t *testing.T, srv *server.SimulatorServer) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	srv.MetricsHandler(rr, req)
	return rr.Body.String()
}

func TestMetricsHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		run   func(t *testing.T)
	}{
		{
			name: "gauge no labels no rules",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "metric_name", Help: "a gauge metric", Type: parser.Gauge, Value: 42},
				}
				srv := server.New(metrics, nil)
				body := doRequest(t, srv)
				assertContains(t, body, "# HELP metric_name a gauge metric\n")
				assertContains(t, body, "# TYPE metric_name gauge\n")
				assertContains(t, body, "metric_name 42\n")
			},
		},
		{
			name: "counter no labels no rules",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "requests_total", Type: parser.Counter, Value: 7},
				}
				srv := server.New(metrics, nil)
				body := doRequest(t, srv)
				assertContains(t, body, "# TYPE requests_total counter\n")
				assertNotContains(t, body, "# HELP requests_total")
			},
		},
		{
			name: "spike rule applied",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "metric_name", Type: parser.Gauge, Value: 100},
				}
				rules := []mutator.Rule{
					{
						Selector: mutator.LabelSelector{Name: "metric_name"},
						Mutator:  mutator.Spike{Multiplier: 2, Duration: 10 * time.Second},
					},
				}
				srv := server.New(metrics, rules)
				srv.StartTime = time.Now() // elapsed ≈ 0, within 10s spike window
				body := doRequest(t, srv)
				assertContains(t, body, "metric_name 200\n")
			},
		},
		{
			name: "rule not matched",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "metric_name", Type: parser.Gauge, Value: 55},
				}
				rules := []mutator.Rule{
					{
						Selector: mutator.LabelSelector{Name: "other_metric"},
						Mutator:  mutator.Spike{Multiplier: 10, Duration: 10 * time.Second},
					},
				}
				srv := server.New(metrics, rules)
				body := doRequest(t, srv)
				assertContains(t, body, "metric_name 55\n")
			},
		},
		{
			name: "metric with single label",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "metric_name", Type: parser.Gauge, Value: 1, Labels: map[string]string{"job": "api"}},
				}
				srv := server.New(metrics, nil)
				body := doRequest(t, srv)
				assertContains(t, body, `metric_name{job="api"} 1`+"\n")
			},
		},
		{
			name: "metric with multiple labels sorted",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "metric_name", Type: parser.Gauge, Value: 9, Labels: map[string]string{"job": "api", "env": "prod"}},
				}
				srv := server.New(metrics, nil)
				body := doRequest(t, srv)
				assertContains(t, body, `metric_name{env="prod",job="api"} 9`+"\n")
			},
		},
		{
			name: "two samples same metric name emit headers once",
			run: func(t *testing.T) {
				t.Parallel()
				metrics := []*parser.Metric{
					{Name: "http_requests", Help: "total requests", Type: parser.Counter, Value: 1, Labels: map[string]string{"method": "GET"}},
					{Name: "http_requests", Help: "total requests", Type: parser.Counter, Value: 2, Labels: map[string]string{"method": "POST"}},
				}
				srv := server.New(metrics, nil)
				body := doRequest(t, srv)
				assertEqual(t, countOccurrences(body, "# HELP http_requests"), 1)
				assertEqual(t, countOccurrences(body, "# TYPE http_requests"), 1)
				assertContains(t, body, `http_requests{method="GET"} 1`+"\n")
				assertContains(t, body, `http_requests{method="POST"} 2`+"\n")
			},
		},
		{
			name: "content-type header",
			run: func(t *testing.T) {
				t.Parallel()
				srv := server.New(nil, nil)
				req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
				rr := httptest.NewRecorder()
				srv.MetricsHandler(rr, req)
				assertEqual(t, rr.Header().Get("Content-Type"), "text/plain; version=0.0.4; charset=utf-8")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, tc.run)
	}
}
