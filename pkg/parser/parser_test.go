package parser_test

import (
	"math"
	"strings"
	"testing"

	"github.com/srosignoli/faultline/pkg/parser"
)

// fixture is a realistic multi-family dump used by several sub-tests.
const fixture = `# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="GET",code="200"} 1027 1395066363000
http_requests_total{method="POST",code="500"} 3 1395066363000

# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 23552

# HELP rpc_duration_seconds A summary of RPC durations.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.5"} 4.9351e-05
rpc_duration_seconds_sum 1.7560473e+04
rpc_duration_seconds_count 2693

bare_metric 42
`

func TestParseDump(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errSub  string // substring expected in error message
		check   func(t *testing.T, metrics []*parser.Metric)
	}{
		{
			name:  "counter no labels",
			input: "# TYPE hits counter\nhits 7\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1 metric, got %d", len(ms))
				}
				m := ms[0]
				assertEqual(t, "Name", "hits", m.Name)
				assertEqual(t, "Type", parser.Counter, m.Type)
				assertFloat(t, "Value", 7, m.Value)
				if m.Labels != nil {
					t.Errorf("Labels: want nil, got %v", m.Labels)
				}
			},
		},
		{
			name:  "gauge multiple samples with labels",
			input: "# TYPE temp gauge\ntemp{host=\"a\"} 1\ntemp{host=\"b\"} 2\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 2 {
					t.Fatalf("want 2 metrics, got %d", len(ms))
				}
				assertEqual(t, "ms[0].Labels[host]", "a", ms[0].Labels["host"])
				assertEqual(t, "ms[1].Labels[host]", "b", ms[1].Labels["host"])
			},
		},
		{
			name:  "multiple labels on one line",
			input: `reqs{method="GET",code="200"} 100` + "\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1 metric, got %d", len(ms))
				}
				assertEqual(t, "method", "GET", ms[0].Labels["method"])
				assertEqual(t, "code", "200", ms[0].Labels["code"])
			},
		},
		{
			name:  "HELP and TYPE both propagated",
			input: "# HELP foo the foo metric\n# TYPE foo gauge\nfoo 3\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1 metric, got %d", len(ms))
				}
				assertEqual(t, "Help", "the foo metric", ms[0].Help)
				assertEqual(t, "Type", parser.Gauge, ms[0].Type)
			},
		},
		{
			name:  "TYPE only no HELP",
			input: "# TYPE bar counter\nbar 0\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				assertEqual(t, "Help", "", ms[0].Help)
				assertEqual(t, "Type", parser.Counter, ms[0].Type)
			},
		},
		{
			name:  "HELP only no TYPE",
			input: "# HELP baz just a baz\nbaz 1\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				assertEqual(t, "Help", "just a baz", ms[0].Help)
				assertEqual(t, "Type", parser.Untyped, ms[0].Type)
			},
		},
		{
			name:  "timestamp ignored value parsed",
			input: "up 1 1234567890000\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				assertFloat(t, "Value", 1, ms[0].Value)
			},
		},
		{
			name:  "escaped label value backslash",
			input: `disk_used{path="C:\\DIR"} 500` + "\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				assertEqual(t, "path", `C:\DIR`, ms[0].Labels["path"])
			},
		},
		{
			name:  "escaped HELP newline",
			input: "# HELP multi line one\\ntwo\nmulti 0\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				want := "line one\ntwo"
				assertEqual(t, "Help", want, ms[0].Help)
			},
		},
		{
			name:  "bare metric no labels no type",
			input: "bare_metric 42\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				assertFloat(t, "Value", 42, ms[0].Value)
				assertEqual(t, "Type", parser.Untyped, ms[0].Type)
				if ms[0].Labels != nil {
					t.Errorf("Labels: want nil, got %v", ms[0].Labels)
				}
			},
		},
		{
			name:  "empty input",
			input: "",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 0 {
					t.Errorf("want empty slice, got %d metrics", len(ms))
				}
			},
		},
		{
			name:  "NaN value",
			input: "x NaN\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				if !math.IsNaN(ms[0].Value) {
					t.Errorf("Value: want NaN, got %v", ms[0].Value)
				}
			},
		},
		{
			name:  "+Inf value",
			input: "x +Inf\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				if !math.IsInf(ms[0].Value, 1) {
					t.Errorf("Value: want +Inf, got %v", ms[0].Value)
				}
			},
		},
		{
			name:  "-Inf value",
			input: "x -Inf\n",
			check: func(t *testing.T, ms []*parser.Metric) {
				if len(ms) != 1 {
					t.Fatalf("want 1, got %d", len(ms))
				}
				if !math.IsInf(ms[0].Value, -1) {
					t.Errorf("Value: want -Inf, got %v", ms[0].Value)
				}
			},
		},
		{
			name:    "malformed no value",
			input:   "mymetric\n",
			wantErr: true,
			errSub:  "line 1",
		},
		{
			name:    "malformed non-numeric value",
			input:   "mymetric notanumber\n",
			wantErr: true,
		},
		{
			name:    "malformed unterminated label string",
			input:   `mymetric{label="unterminated` + "\n",
			wantErr: true,
		},
		{
			name:    "unknown TYPE keyword",
			input:   "# TYPE foo unknownkind\nfoo 1\n",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ms, err := parser.ParseDump(strings.NewReader(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errSub != "" && !strings.Contains(err.Error(), tc.errSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, ms)
			}
		})
	}
}

// TestParseDumpFixture exercises the realistic multi-family fixture.
func TestParseDumpFixture(t *testing.T) {
	ms, err := parser.ParseDump(strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("ParseDump error: %v", err)
	}

	// Expected: 2 counter samples + 1 gauge + 3 summary lines + 1 bare = 7.
	if len(ms) != 7 {
		t.Fatalf("want 7 metrics, got %d", len(ms))
	}

	// First counter sample.
	c := ms[0]
	assertEqual(t, "counter Name", "http_requests_total", c.Name)
	assertEqual(t, "counter Type", parser.Counter, c.Type)
	assertEqual(t, "counter method", "GET", c.Labels["method"])
	assertEqual(t, "counter code", "200", c.Labels["code"])
	assertFloat(t, "counter Value", 1027, c.Value)

	// Gauge sample.
	g := ms[2]
	assertEqual(t, "gauge Name", "process_resident_memory_bytes", g.Name)
	assertEqual(t, "gauge Type", parser.Gauge, g.Type)
	if g.Labels != nil {
		t.Errorf("gauge Labels: want nil, got %v", g.Labels)
	}
	assertFloat(t, "gauge Value", 23552, g.Value)

	// Summary quantile sample.
	s := ms[3]
	assertEqual(t, "summary Type", parser.Summary, s.Type)
	assertEqual(t, "summary quantile", "0.5", s.Labels["quantile"])

	// Bare metric.
	b := ms[6]
	assertEqual(t, "bare Name", "bare_metric", b.Name)
	assertEqual(t, "bare Type", parser.Untyped, b.Type)
	assertFloat(t, "bare Value", 42, b.Value)
}

// ---- helpers ----

func assertEqual[T comparable](t *testing.T, field string, want, got T) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", field, want, got)
	}
}

func assertFloat(t *testing.T, field string, want, got float64) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", field, want, got)
	}
}
