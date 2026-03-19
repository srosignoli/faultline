package config_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/srosignoli/faultline/pkg/config"
	"github.com/srosignoli/faultline/pkg/mutator"
)

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertApprox(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("got %v, want %v (tol %v)", got, want, tol)
	}
}

// mustApply calls Apply with an always-active state (Duration==0).
func mustApply(m mutator.Mutator, value float64) float64 {
	now := time.Now()
	state := mutator.NewRuleState(now.Add(-time.Hour))
	return m.Apply(value, state, mutator.ScheduleConfig{}, now)
}

// TestLoadConfig exercises file-level loading.
func TestLoadConfig(t *testing.T) {
	t.Parallel()

	const validYAML = `
rules:
  - name: jitter_rule
    match:
      metric_name: http_requests_total
    mutator:
      type: jitter
      params:
        variance: 0.05
  - name: trend_rule
    match:
      metric_name: cpu_usage
    mutator:
      type: trend
      params:
        rate_per_second: 1.0
  - name: spike_rule
    match:
      metric_name: error_rate
    mutator:
      type: spike
      params:
        multiplier: 3.0
        duration: 5s
  - name: wave_rule
    match:
      metric_name: latency
    mutator:
      type: wave
      params:
        amplitude: 50.0
        frequency: 0.1
`

	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantRules int
	}{
		{
			name:      "valid YAML with all 4 types",
			content:   validYAML,
			wantErr:   false,
			wantRules: 4,
		},
		{
			name:    "invalid YAML",
			content: "rules: [invalid: yaml: content",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "rules.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.LoadConfig(path)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqual(t, len(cfg.Rules), tc.wantRules)
		})
	}

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		_, err := config.LoadConfig("/nonexistent/path/rules.yaml")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

// TestBuildRules exercises in-memory rule construction.
func TestBuildRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rules   []config.RuleConfig
		wantErr bool
		check   func(t *testing.T, rules []mutator.Rule)
	}{
		{
			name: "jitter variance=0.0 base=100",
			rules: []config.RuleConfig{
				{
					Name:  "j",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type:   "jitter",
						Params: map[string]interface{}{"variance": float64(0)},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				assertFloat(t, mustApply(rules[0].Mutator, 100), 100)
			},
		},
		{
			name: "trend rate=10.0 elapsed=1s",
			rules: []config.RuleConfig{
				{
					Name:  "t",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type:   "trend",
						Params: map[string]interface{}{"rate_per_second": float64(10)},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-time.Hour))
				state.ActiveSince = now.Add(-time.Second)
				assertFloat(t, rules[0].Mutator.Apply(100, state, mutator.ScheduleConfig{}, now), 110)
			},
		},
		{
			name: "trend rate as int (yaml coercion)",
			rules: []config.RuleConfig{
				{
					Name:  "t",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type:   "trend",
						Params: map[string]interface{}{"rate_per_second": int(10)},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-time.Hour))
				state.ActiveSince = now.Add(-time.Second)
				assertFloat(t, rules[0].Mutator.Apply(100, state, mutator.ScheduleConfig{}, now), 110)
			},
		},
		{
			name: "spike multiplier=2 duration=10s active",
			rules: []config.RuleConfig{
				{
					Name:  "s",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type: "spike",
						Params: map[string]interface{}{
							"multiplier": float64(2),
							"duration":   "10s",
						},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-time.Hour))
				state.ActiveUntil = now.Add(time.Hour) // inside active window
				assertFloat(t, rules[0].Mutator.Apply(100, state, rules[0].Schedule, now), 200)
			},
		},
		{
			name: "spike elapsed>=duration inactive",
			rules: []config.RuleConfig{
				{
					Name:  "s",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type: "spike",
						Params: map[string]interface{}{
							"multiplier": float64(2),
							"duration":   "10s",
						},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-time.Hour))
				state.NextTriggerTime = now.Add(time.Hour) // next trigger not yet reached
				assertFloat(t, rules[0].Mutator.Apply(100, state, rules[0].Schedule, now), 100)
			},
		},
		{
			name: "spike interval now honoured",
			rules: []config.RuleConfig{
				{
					Name:  "s",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type: "spike",
						Params: map[string]interface{}{
							"multiplier": float64(2),
							"duration":   "10s",
							"interval":   "30s",
						},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				assertEqual(t, rules[0].Schedule.Interval, 30*time.Second)
			},
		},
		{
			name: "wave amplitude=10 freq=1Hz elapsed=0.25s",
			rules: []config.RuleConfig{
				{
					Name:  "w",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type: "wave",
						Params: map[string]interface{}{
							"amplitude": float64(10),
							"frequency": float64(1),
						},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				// At 0.25s with freq=1Hz: sin(2π*1*0.25) = sin(π/2) = 1 → 100+10=110
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-250 * time.Millisecond))
				assertApprox(t, rules[0].Mutator.Apply(100, state, mutator.ScheduleConfig{}, now), 110, 1e-9)
			},
		},
		{
			name: "unknown type",
			rules: []config.RuleConfig{
				{
					Name:  "x",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type: "unknown_type",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing required param",
			rules: []config.RuleConfig{
				{
					Name:  "t",
					Match: config.MatchConfig{MetricName: "m"},
					Mutator: config.MutatorConfig{
						Type:   "trend",
						Params: map[string]interface{}{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rule with labels",
			rules: []config.RuleConfig{
				{
					Name: "labeled",
					Match: config.MatchConfig{
						MetricName: "http_requests_total",
						Labels:     map[string]string{"job": "api", "env": "prod"},
					},
					Mutator: config.MutatorConfig{
						Type:   "trend",
						Params: map[string]interface{}{"rate_per_second": float64(1)},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				assertEqual(t, rules[0].Selector.Name, "http_requests_total")
				assertEqual(t, rules[0].Selector.Labels["job"], "api")
				assertEqual(t, rules[0].Selector.Labels["env"], "prod")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := &config.Config{Rules: tc.rules}
			rules, err := config.BuildRules(cfg)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, rules)
			}
		})
	}
}
