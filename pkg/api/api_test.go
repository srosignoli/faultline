package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/srosignoli/faultline/pkg/api"
	"github.com/srosignoli/faultline/pkg/k8s"
)

// mockClient is a configurable test double for api.SimulatorClient.
type mockClient struct {
	createFn func(ctx context.Context, name, dump, rules string) error
	listFn   func(ctx context.Context) ([]k8s.SimulatorInfo, error)
	deleteFn func(ctx context.Context, name string) error
}

func (m *mockClient) CreateSimulator(ctx context.Context, name, dump, rules string) error {
	return m.createFn(ctx, name, dump, rules)
}

func (m *mockClient) ListSimulators(ctx context.Context) ([]k8s.SimulatorInfo, error) {
	return m.listFn(ctx)
}

func (m *mockClient) DeleteSimulator(ctx context.Context, name string) error {
	return m.deleteFn(ctx, name)
}

// --- test helpers ---

func newRouter(mc *mockClient) *http.ServeMux {
	return api.NewRouter(api.NewHandler(mc))
}

func doRequest(mux *http.ServeMux, method, path string, body []byte) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return b
}

func decodeErrorBody(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error body: %v (raw: %s)", err, rr.Body.String())
	}
	return resp.Error
}

func decodeSimulators(t *testing.T, rr *httptest.ResponseRecorder) []api.Simulator {
	t.Helper()
	var sims []api.Simulator
	if err := json.NewDecoder(rr.Body).Decode(&sims); err != nil {
		t.Fatalf("decode simulators: %v (raw: %s)", err, rr.Body.String())
	}
	return sims
}

// --- TestListSimulators ---

func TestListSimulators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		listResult []k8s.SimulatorInfo
		listErr    error
		wantStatus int
		wantNames  []string
	}{
		{
			name:       "empty",
			listResult: []k8s.SimulatorInfo{},
			wantStatus: http.StatusOK,
			wantNames:  []string{},
		},
		{
			name: "two simulators",
			listResult: []k8s.SimulatorInfo{
				{Name: "alpha", RulesYAML: ""},
				{Name: "beta", RulesYAML: ""},
			},
			wantStatus: http.StatusOK,
			wantNames:  []string{"alpha", "beta"},
		},
		{
			name:       "k8s error",
			listErr:    errors.New("cluster unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "rules attached",
			listResult: []k8s.SimulatorInfo{
				{
					Name: "sim1",
					RulesYAML: `rules:
  - name: jitter-rule
    match:
      metric_name: http_requests_total
    mutator:
      type: jitter
      params:
        variance: 0.1
`,
				},
			},
			wantStatus: http.StatusOK,
			wantNames:  []string{"sim1"},
		},
		{
			name: "invalid rules yaml returns empty active_rules",
			listResult: []k8s.SimulatorInfo{
				{Name: "bad-sim", RulesYAML: "!!invalid"},
			},
			wantStatus: http.StatusOK,
			wantNames:  []string{"bad-sim"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mc := &mockClient{
				listFn: func(_ context.Context) ([]k8s.SimulatorInfo, error) {
					return tc.listResult, tc.listErr
				},
			}
			rr := doRequest(newRouter(mc), http.MethodGet, "/api/simulators", nil)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			got := decodeSimulators(t, rr)
			if got == nil {
				got = []api.Simulator{}
			}
			sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
			sort.Strings(tc.wantNames)

			if len(got) != len(tc.wantNames) {
				t.Fatalf("names: got %v, want %v", got, tc.wantNames)
			}
			for i := range got {
				if got[i].Name != tc.wantNames[i] {
					t.Errorf("names[%d]: got %s, want %s", i, got[i].Name, tc.wantNames[i])
				}
			}

			// Subtest-specific assertions.
			switch tc.name {
			case "rules attached":
				if len(got) != 1 {
					t.Fatalf("expected 1 simulator, got %d", len(got))
				}
				rules := got[0].ActiveRules
				if len(rules) != 1 {
					t.Fatalf("expected 1 active rule, got %d", len(rules))
				}
				r := rules[0]
				if r.Name != "jitter-rule" {
					t.Errorf("rule name: got %q, want %q", r.Name, "jitter-rule")
				}
				if r.Match.MetricName != "http_requests_total" {
					t.Errorf("metric name: got %q, want %q", r.Match.MetricName, "http_requests_total")
				}
				if r.Mutator.Type != "jitter" {
					t.Errorf("mutator type: got %q, want %q", r.Mutator.Type, "jitter")
				}
				if r.Mutator.Params["variance"] == nil {
					t.Error("expected variance param to be set")
				}

			case "invalid rules yaml returns empty active_rules":
				if len(got) != 1 {
					t.Fatalf("expected 1 simulator, got %d", len(got))
				}
				if len(got[0].ActiveRules) != 0 {
					t.Errorf("expected empty active_rules, got %v", got[0].ActiveRules)
				}
			}
		})
	}
}

// --- TestCreateSimulator ---

func TestCreateSimulator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       func(t *testing.T) []byte
		createErr  error
		wantStatus int
		wantErrMsg string // non-empty → exact match on {"error": "..."}
	}{
		{
			name: "success",
			body: func(t *testing.T) []byte {
				return mustMarshal(t, map[string]string{
					"name": "my-sim", "dump_payload": "dump", "rules_payload": "rules",
				})
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid JSON",
			body:       func(t *testing.T) []byte { return []byte(`{bad`) },
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing name",
			body: func(t *testing.T) []byte {
				return mustMarshal(t, map[string]string{"dump_payload": "d", "rules_payload": "r"})
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "name is required",
		},
		{
			name: "missing dump_payload",
			body: func(t *testing.T) []byte {
				return mustMarshal(t, map[string]string{"name": "x", "rules_payload": "r"})
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "dump_payload is required",
		},
		{
			name: "missing rules_payload",
			body: func(t *testing.T) []byte {
				return mustMarshal(t, map[string]string{"name": "x", "dump_payload": "d"})
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "rules_payload is required",
		},
		{
			name: "k8s error",
			body: func(t *testing.T) []byte {
				return mustMarshal(t, map[string]string{
					"name": "x", "dump_payload": "d", "rules_payload": "r",
				})
			},
			createErr:  errors.New("k8s down"),
			wantStatus: http.StatusInternalServerError,
			wantErrMsg: "k8s down",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mc := &mockClient{
				createFn: func(_ context.Context, _, _, _ string) error {
					return tc.createErr
				},
			}
			rr := doRequest(newRouter(mc), http.MethodPost, "/api/simulators", tc.body(t))

			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if tc.wantErrMsg != "" {
				if got := decodeErrorBody(t, rr); got != tc.wantErrMsg {
					t.Errorf("error message: got %q, want %q", got, tc.wantErrMsg)
				}
			}
		})
	}
}

// --- TestDeleteSimulator ---

func TestDeleteSimulator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		simName    string
		deleteErr  error
		wantStatus int
		wantErrMsg string
	}{
		{
			name:       "success",
			simName:    "my-sim",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "k8s error",
			simName:    "bad-sim",
			deleteErr:  errors.New("delete failed"),
			wantStatus: http.StatusInternalServerError,
			wantErrMsg: "delete failed",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var deletedName string
			mc := &mockClient{
				deleteFn: func(_ context.Context, name string) error {
					deletedName = name
					return tc.deleteErr
				},
			}
			rr := doRequest(newRouter(mc), http.MethodDelete, "/api/simulators/"+tc.simName, nil)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if tc.wantStatus == http.StatusNoContent && deletedName != tc.simName {
				t.Errorf("deleted name: got %q, want %q", deletedName, tc.simName)
			}
			if tc.wantErrMsg != "" {
				if got := decodeErrorBody(t, rr); got != tc.wantErrMsg {
					t.Errorf("error message: got %q, want %q", got, tc.wantErrMsg)
				}
			}
		})
	}
}
