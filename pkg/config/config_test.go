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
        slope: 1.0
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
						Params: map[string]interface{}{"slope": float64(10)},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-time.Hour))
				state.CurrentWindowStart = now.Add(-time.Second)
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
						Params: map[string]interface{}{"slope": int(10)},
					},
				},
			},
			check: func(t *testing.T, rules []mutator.Rule) {
				now := time.Unix(1_000_000, 0)
				state := mutator.NewRuleState(now.Add(-time.Hour))
				state.CurrentWindowStart = now.Add(-time.Second)
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
						Params: map[string]interface{}{"slope": float64(1)},
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

// TestBuildRules16Scenarios parses a 20-rule YAML payload and asserts each
// rule's mutator type, relevant fields, and schedule parameters.
func TestBuildRules16Scenarios(t *testing.T) {
	t.Parallel()

	const yamlPayload = `
rules:
  # --- SPIKE SCENARIOS ---
  - name: "Viral Traffic Surge"
    match: { metric_name: "http_requests_total" }
    mutator:
      type: "spike"
      params: { multiplier: 50.0, initial_delay: "1m", duration: "2m", interval: "1h" }
  - name: "CPU Steal Time Spike"
    match: { metric_name: "node_cpu_steal_seconds_total" }
    mutator:
      type: "spike"
      params: { multiplier: 15.0, duration: "5m", interval: "4h" }
  - name: "Slow DB Queries"
    match: { metric_name: "db_query_duration_seconds" }
    mutator:
      type: "spike"
      params: { multiplier: 8.0, duration: "3m", interval: "30m" }
  - name: "CrashLoopBackOff Spike"
    match: { metric_name: "kube_pod_container_status_restarts_total" }
    mutator:
      type: "spike"
      params: { multiplier: 10.0, duration: "10m" }
  # --- TREND SCENARIOS ---
  - name: "API Server Memory Leak"
    match: { metric_name: "process_resident_memory_bytes" }
    mutator:
      type: "trend"
      params: { slope: 1048576.0, interval: "1h" }
  - name: "Goroutine Leak"
    match: { metric_name: "go_goroutines" }
    mutator:
      type: "trend"
      params: { slope: 5.0, interval: "24h" }
  - name: "Log Spam Disk Fill"
    match: { metric_name: "node_filesystem_free_bytes" }
    mutator:
      type: "trend"
      params: { slope: -50000000.0 }
  - name: "Email Queue Backup"
    match: { metric_name: "email_queue_length" }
    mutator:
      type: "trend"
      params: { slope: 10.0, interval: "45m" }
  # --- JITTER SCENARIOS ---
  - name: "Unstable Network Interface"
    match: { metric_name: "node_network_transmit_bytes_total" }
    mutator:
      type: "jitter"
      params: { variance: 0.80, duration: "5m", interval: "30m" }
  - name: "Redis Cache Thrashing"
    match: { metric_name: "redis_keyspace_hits_total" }
    mutator:
      type: "jitter"
      params: { variance: 0.50, duration: "10m", interval: "1h" }
  - name: "Sporadic CPU Throttling"
    match: { metric_name: "container_cpu_cfs_throttled_seconds_total" }
    mutator:
      type: "jitter"
      params: { variance: 0.35, duration: "2m", interval: "10m", interval_jitter: "5m" }
  - name: "DB Connection Storm"
    match: { metric_name: "mysql_global_status_threads_connected" }
    mutator:
      type: "jitter"
      params: { variance: 0.90, duration: "3m", interval: "15m" }
  # --- OUTAGE SCENARIOS ---
  - name: "Auth Service Crash"
    match: { metric_name: "auth_service_up" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "4m", interval: "2h" }
  - name: "Silent Backup Job Failure"
    match: { metric_name: "backup_job_last_success_timestamp" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "30m", interval: "24h" }
  - name: "AWS SQS Outage"
    match: { metric_name: "sqs_messages_visible" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "15m" }
  - name: "EBS Volume Detached"
    match: { metric_name: "node_filesystem_avail_bytes" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "8m" }
  # --- WAVE SCENARIOS ---
  - name: "Daily Traffic Wave"
    match: { metric_name: "http_requests_total" }
    mutator:
      type: "wave"
      params: { amplitude: 500.0, frequency: 0.0000115741 }
  - name: "Intraday Latency Oscillation"
    match: { metric_name: "http_request_duration_seconds" }
    mutator:
      type: "wave"
      params: { amplitude: 0.05, frequency: 0.0000347222, initial_delay: "5m" }
  - name: "Periodic Scrape Noise"
    match: { metric_name: "up" }
    mutator:
      type: "wave"
      params: { amplitude: 0.1, frequency: 0.00166667, duration: "10m", interval: "1h" }
  - name: "Weekly Batch Volume Pattern"
    match: { metric_name: "batch_jobs_total" }
    mutator:
      type: "wave"
      params: { amplitude: 200.0, frequency: 0.00000165 }
`

	cfg, err := config.ParseConfig([]byte(yamlPayload))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	rules, err := config.BuildRules(cfg)
	if err != nil {
		t.Fatalf("BuildRules: %v", err)
	}
	if len(rules) != 20 {
		t.Fatalf("expected 20 rules, got %d", len(rules))
	}

	type ruleAssertion struct {
		name     string
		checkFn  func(t *testing.T, r mutator.Rule)
	}

	assertions := []ruleAssertion{
		// 0 — Viral Traffic Surge
		{"Viral Traffic Surge", func(t *testing.T, r mutator.Rule) {
			s, ok := r.Mutator.(mutator.Spike)
			if !ok {
				t.Fatal("expected Spike")
			}
			assertFloat(t, s.Multiplier, 50.0)
			assertEqual(t, r.Schedule.InitialDelay, 1*time.Minute)
			assertEqual(t, r.Schedule.Duration, 2*time.Minute)
			assertEqual(t, r.Schedule.Interval, 1*time.Hour)
		}},
		// 1 — CPU Steal Time Spike
		{"CPU Steal Time Spike", func(t *testing.T, r mutator.Rule) {
			s, ok := r.Mutator.(mutator.Spike)
			if !ok {
				t.Fatal("expected Spike")
			}
			assertFloat(t, s.Multiplier, 15.0)
			assertEqual(t, r.Schedule.Duration, 5*time.Minute)
			assertEqual(t, r.Schedule.Interval, 4*time.Hour)
		}},
		// 2 — Slow DB Queries
		{"Slow DB Queries", func(t *testing.T, r mutator.Rule) {
			s, ok := r.Mutator.(mutator.Spike)
			if !ok {
				t.Fatal("expected Spike")
			}
			assertFloat(t, s.Multiplier, 8.0)
			assertEqual(t, r.Schedule.Duration, 3*time.Minute)
			assertEqual(t, r.Schedule.Interval, 30*time.Minute)
		}},
		// 3 — CrashLoopBackOff Spike
		{"CrashLoopBackOff Spike", func(t *testing.T, r mutator.Rule) {
			s, ok := r.Mutator.(mutator.Spike)
			if !ok {
				t.Fatal("expected Spike")
			}
			assertFloat(t, s.Multiplier, 10.0)
			assertEqual(t, r.Schedule.Duration, 10*time.Minute)
			assertEqual(t, r.Schedule.Interval, 0)
		}},
		// 4 — API Server Memory Leak
		{"API Server Memory Leak", func(t *testing.T, r mutator.Rule) {
			tr, ok := r.Mutator.(mutator.Trend)
			if !ok {
				t.Fatal("expected Trend")
			}
			assertFloat(t, tr.Slope, 1048576.0)
			assertEqual(t, r.Schedule.Interval, 1*time.Hour)
		}},
		// 5 — Goroutine Leak
		{"Goroutine Leak", func(t *testing.T, r mutator.Rule) {
			tr, ok := r.Mutator.(mutator.Trend)
			if !ok {
				t.Fatal("expected Trend")
			}
			assertFloat(t, tr.Slope, 5.0)
			assertEqual(t, r.Schedule.Interval, 24*time.Hour)
		}},
		// 6 — Log Spam Disk Fill
		{"Log Spam Disk Fill", func(t *testing.T, r mutator.Rule) {
			tr, ok := r.Mutator.(mutator.Trend)
			if !ok {
				t.Fatal("expected Trend")
			}
			assertFloat(t, tr.Slope, -50000000.0)
			assertEqual(t, r.Schedule.Interval, 0)
		}},
		// 7 — Email Queue Backup
		{"Email Queue Backup", func(t *testing.T, r mutator.Rule) {
			tr, ok := r.Mutator.(mutator.Trend)
			if !ok {
				t.Fatal("expected Trend")
			}
			assertFloat(t, tr.Slope, 10.0)
			assertEqual(t, r.Schedule.Interval, 45*time.Minute)
		}},
		// 8 — Unstable Network Interface
		{"Unstable Network Interface", func(t *testing.T, r mutator.Rule) {
			j, ok := r.Mutator.(mutator.Jitter)
			if !ok {
				t.Fatal("expected Jitter")
			}
			assertFloat(t, j.Variance, 0.80)
			assertEqual(t, r.Schedule.Duration, 5*time.Minute)
			assertEqual(t, r.Schedule.Interval, 30*time.Minute)
		}},
		// 9 — Redis Cache Thrashing
		{"Redis Cache Thrashing", func(t *testing.T, r mutator.Rule) {
			j, ok := r.Mutator.(mutator.Jitter)
			if !ok {
				t.Fatal("expected Jitter")
			}
			assertFloat(t, j.Variance, 0.50)
			assertEqual(t, r.Schedule.Duration, 10*time.Minute)
			assertEqual(t, r.Schedule.Interval, 1*time.Hour)
		}},
		// 10 — Sporadic CPU Throttling
		{"Sporadic CPU Throttling", func(t *testing.T, r mutator.Rule) {
			j, ok := r.Mutator.(mutator.Jitter)
			if !ok {
				t.Fatal("expected Jitter")
			}
			assertFloat(t, j.Variance, 0.35)
			assertEqual(t, r.Schedule.Duration, 2*time.Minute)
			assertEqual(t, r.Schedule.Interval, 10*time.Minute)
			assertEqual(t, r.Schedule.IntervalJitter, 5*time.Minute)
		}},
		// 11 — DB Connection Storm
		{"DB Connection Storm", func(t *testing.T, r mutator.Rule) {
			j, ok := r.Mutator.(mutator.Jitter)
			if !ok {
				t.Fatal("expected Jitter")
			}
			assertFloat(t, j.Variance, 0.90)
			assertEqual(t, r.Schedule.Duration, 3*time.Minute)
			assertEqual(t, r.Schedule.Interval, 15*time.Minute)
		}},
		// 12 — Auth Service Crash
		{"Auth Service Crash", func(t *testing.T, r mutator.Rule) {
			o, ok := r.Mutator.(mutator.Outage)
			if !ok {
				t.Fatal("expected Outage")
			}
			assertEqual(t, o.Action, "drop_to_zero")
			assertEqual(t, r.Schedule.Duration, 4*time.Minute)
			assertEqual(t, r.Schedule.Interval, 2*time.Hour)
		}},
		// 13 — Silent Backup Job Failure
		{"Silent Backup Job Failure", func(t *testing.T, r mutator.Rule) {
			o, ok := r.Mutator.(mutator.Outage)
			if !ok {
				t.Fatal("expected Outage")
			}
			assertEqual(t, o.Action, "drop_to_zero")
			assertEqual(t, r.Schedule.Duration, 30*time.Minute)
			assertEqual(t, r.Schedule.Interval, 24*time.Hour)
		}},
		// 14 — AWS SQS Outage
		{"AWS SQS Outage", func(t *testing.T, r mutator.Rule) {
			o, ok := r.Mutator.(mutator.Outage)
			if !ok {
				t.Fatal("expected Outage")
			}
			assertEqual(t, o.Action, "drop_to_zero")
			assertEqual(t, r.Schedule.Duration, 15*time.Minute)
			assertEqual(t, r.Schedule.Interval, 0)
		}},
		// 15 — EBS Volume Detached
		{"EBS Volume Detached", func(t *testing.T, r mutator.Rule) {
			o, ok := r.Mutator.(mutator.Outage)
			if !ok {
				t.Fatal("expected Outage")
			}
			assertEqual(t, o.Action, "drop_to_zero")
			assertEqual(t, r.Schedule.Duration, 8*time.Minute)
			assertEqual(t, r.Schedule.Interval, 0)
		}},
		// 16 — Daily Traffic Wave
		{"Daily Traffic Wave", func(t *testing.T, r mutator.Rule) {
			w, ok := r.Mutator.(mutator.Wave)
			if !ok {
				t.Fatal("expected Wave")
			}
			assertFloat(t, w.Amplitude, 500.0)
			assertApprox(t, w.Frequency, 0.0000115741, 1e-10)
			assertEqual(t, r.Schedule.Duration, 0)
			assertEqual(t, r.Schedule.Interval, 0)
		}},
		// 17 — Intraday Latency Oscillation
		{"Intraday Latency Oscillation", func(t *testing.T, r mutator.Rule) {
			w, ok := r.Mutator.(mutator.Wave)
			if !ok {
				t.Fatal("expected Wave")
			}
			assertFloat(t, w.Amplitude, 0.05)
			assertApprox(t, w.Frequency, 0.0000347222, 1e-10)
			assertEqual(t, r.Schedule.InitialDelay, 5*time.Minute)
		}},
		// 18 — Periodic Scrape Noise
		{"Periodic Scrape Noise", func(t *testing.T, r mutator.Rule) {
			w, ok := r.Mutator.(mutator.Wave)
			if !ok {
				t.Fatal("expected Wave")
			}
			assertFloat(t, w.Amplitude, 0.1)
			assertApprox(t, w.Frequency, 0.00166667, 1e-8)
			assertEqual(t, r.Schedule.Duration, 10*time.Minute)
			assertEqual(t, r.Schedule.Interval, 1*time.Hour)
		}},
		// 19 — Weekly Batch Volume Pattern
		{"Weekly Batch Volume Pattern", func(t *testing.T, r mutator.Rule) {
			w, ok := r.Mutator.(mutator.Wave)
			if !ok {
				t.Fatal("expected Wave")
			}
			assertFloat(t, w.Amplitude, 200.0)
			assertApprox(t, w.Frequency, 0.00000165, 1e-10)
			assertEqual(t, r.Schedule.Duration, 0)
			assertEqual(t, r.Schedule.Interval, 0)
		}},
	}

	for i, a := range assertions {
		i, a := i, a
		t.Run(a.name, func(t *testing.T) {
			t.Parallel()
			a.checkFn(t, rules[i])
		})
	}
}
