package mutator_test

import (
	"math"
	"testing"
	"time"

	"github.com/srosignoli/faultline/pkg/mutator"
)

// sec converts a float64 number of seconds to a time.Duration.
func sec(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

func assertFloat(t *testing.T, got, want float64, label string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", label, got, want)
	}
}

func assertApprox(t *testing.T, got, want, tol float64, label string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", label, got, want, tol)
	}
}

// alwaysActive returns a state and schedule that make any mutator always fire.
func alwaysActive(now time.Time) (*mutator.RuleState, mutator.ScheduleConfig) {
	state := mutator.NewRuleState(now.Add(-time.Hour))
	return state, mutator.ScheduleConfig{} // Duration==0 = always active
}

// ---- Jitter ----------------------------------------------------------------

func TestJitter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		variance float64
		base     float64
	}{
		{"5pct variance", 0.05, 100.0},
		{"20pct variance", 0.20, 50.0},
		{"zero variance", 0.0, 42.0},
		{"large value", 0.10, 1e9},
	}

	const iterations = 1000

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			j := mutator.Jitter{Variance: tc.variance}
			maxDelta := tc.variance * math.Abs(tc.base)
			now := time.Unix(1_000_000, 0)
			state, sched := alwaysActive(now)
			for i := 0; i < iterations; i++ {
				got := j.Apply(tc.base, state, sched, now)
				if math.Abs(got-tc.base) > maxDelta+1e-12 {
					t.Fatalf("iteration %d: got %v outside [%v, %v]",
						i, got, tc.base-maxDelta, tc.base+maxDelta)
				}
			}
		})
	}
}

// ---- Trend -----------------------------------------------------------------

func TestTrend(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		rate    float64
		base    float64
		elapsed time.Duration
		want    float64
	}{
		{"positive rate 1s", 10.0, 100.0, time.Second, 110.0},
		{"positive rate 5s", 10.0, 100.0, 5 * time.Second, 150.0},
		{"negative rate 4s", -5.0, 200.0, 4 * time.Second, 180.0},
		{"zero elapsed", 10.0, 100.0, 0, 100.0},
		{"zero rate", 0.0, 100.0, 10 * time.Second, 100.0},
		{"half second", 8.0, 0.0, sec(0.5), 4.0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := mutator.Trend{RatePerSecond: tc.rate}
			now := time.Unix(1_000_000, 0)
			state := mutator.NewRuleState(now.Add(-time.Hour))
			state.ActiveSince = now.Add(-tc.elapsed)
			sched := mutator.ScheduleConfig{} // Duration==0 = always active
			got := tr.Apply(tc.base, state, sched, now)
			assertFloat(t, got, tc.want, tc.name)
		})
	}
}

// ---- Spike -----------------------------------------------------------------

func TestSpike(t *testing.T) {
	t.Parallel()

	base := 100.0
	multiplier := 3.0
	s := mutator.Spike{Multiplier: multiplier}
	now := time.Unix(1_000_000, 0)
	sched := mutator.ScheduleConfig{Duration: 5 * time.Second}

	t.Run("active window", func(t *testing.T) {
		t.Parallel()
		state := mutator.NewRuleState(now.Add(-time.Hour))
		state.ActiveUntil = now.Add(time.Hour)
		got := s.Apply(base, state, sched, now)
		assertFloat(t, got, base*multiplier, "active")
	})

	t.Run("inactive not yet triggered", func(t *testing.T) {
		t.Parallel()
		state := mutator.NewRuleState(now.Add(-time.Hour))
		state.ActiveUntil = now.Add(-time.Second)      // window expired
		state.NextTriggerTime = now.Add(time.Hour)     // next trigger in future
		got := s.Apply(base, state, sched, now)
		assertFloat(t, got, base, "inactive")
	})
}

// ---- Wave ------------------------------------------------------------------

func TestWave(t *testing.T) {
	t.Parallel()

	freq := 1.0 // 1 Hz → period = 1 s
	period := 1.0
	amp := 20.0
	base := 100.0
	w := mutator.Wave{Amplitude: amp, Frequency: freq}

	cases := []struct {
		name    string
		elapsed time.Duration
		want    float64
		tol     float64
	}{
		{"t=0 sin=0", sec(0), base, 0},
		{"t=period/4 sin=1", sec(period / 4), base + amp, 1e-10},
		{"t=period/2 sin≈0", sec(period / 2), base, 1e-10},
		{"t=3period/4 sin=-1", sec(3 * period / 4), base - amp, 1e-10},
		{"t=period sin≈0", sec(period), base, 1e-10},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			now := time.Unix(1_000_000, 0)
			state := mutator.NewRuleState(now.Add(-tc.elapsed)) // StartTime = now - elapsed
			sched := mutator.ScheduleConfig{}                    // Duration==0 = always active
			got := w.Apply(base, state, sched, now)
			assertApprox(t, got, tc.want, tc.tol+1e-10, tc.name)
		})
	}
}

// ---- Rule ------------------------------------------------------------------

func TestRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		selector  mutator.LabelSelector
		metricN   string
		metricL   map[string]string
		wantMatch bool
	}{
		{
			name:      "empty selector matches anything",
			selector:  mutator.LabelSelector{},
			metricN:   "http_requests_total",
			metricL:   map[string]string{"method": "GET"},
			wantMatch: true,
		},
		{
			name:      "name-only match",
			selector:  mutator.LabelSelector{Name: "http_requests_total"},
			metricN:   "http_requests_total",
			metricL:   nil,
			wantMatch: true,
		},
		{
			name:      "name-only reject",
			selector:  mutator.LabelSelector{Name: "http_requests_total"},
			metricN:   "rpc_calls_total",
			metricL:   nil,
			wantMatch: false,
		},
		{
			name:      "label-only match",
			selector:  mutator.LabelSelector{Labels: map[string]string{"method": "GET"}},
			metricN:   "anything",
			metricL:   map[string]string{"method": "GET", "status": "200"},
			wantMatch: true,
		},
		{
			name:      "label-only reject",
			selector:  mutator.LabelSelector{Labels: map[string]string{"method": "POST"}},
			metricN:   "anything",
			metricL:   map[string]string{"method": "GET"},
			wantMatch: false,
		},
		{
			name:      "name+labels both match",
			selector:  mutator.LabelSelector{Name: "http_requests_total", Labels: map[string]string{"method": "GET"}},
			metricN:   "http_requests_total",
			metricL:   map[string]string{"method": "GET"},
			wantMatch: true,
		},
		{
			name:      "name match labels reject",
			selector:  mutator.LabelSelector{Name: "http_requests_total", Labels: map[string]string{"method": "POST"}},
			metricN:   "http_requests_total",
			metricL:   map[string]string{"method": "GET"},
			wantMatch: false,
		},
		{
			name:      "name reject labels match",
			selector:  mutator.LabelSelector{Name: "other_metric", Labels: map[string]string{"method": "GET"}},
			metricN:   "http_requests_total",
			metricL:   map[string]string{"method": "GET"},
			wantMatch: false,
		},
		{
			name:      "nil metric labels with required label",
			selector:  mutator.LabelSelector{Labels: map[string]string{"method": "GET"}},
			metricN:   "something",
			metricL:   nil,
			wantMatch: false,
		},
		{
			name:      "multiple labels AND all match",
			selector:  mutator.LabelSelector{Labels: map[string]string{"method": "GET", "status": "200"}},
			metricN:   "http_requests_total",
			metricL:   map[string]string{"method": "GET", "status": "200", "handler": "/api"},
			wantMatch: true,
		},
		{
			name:      "multiple labels AND partial match",
			selector:  mutator.LabelSelector{Labels: map[string]string{"method": "GET", "status": "500"}},
			metricN:   "http_requests_total",
			metricL:   map[string]string{"method": "GET", "status": "200"},
			wantMatch: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := mutator.Rule{Selector: tc.selector, Mutator: mutator.Jitter{}}
			got := r.Matches(tc.metricN, tc.metricL)
			if got != tc.wantMatch {
				t.Errorf("Matches(%q, %v) = %v, want %v", tc.metricN, tc.metricL, got, tc.wantMatch)
			}
		})
	}
}
