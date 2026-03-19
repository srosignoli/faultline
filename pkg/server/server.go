package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/srosignoli/faultline/pkg/mutator"
	"github.com/srosignoli/faultline/pkg/parser"
)

// SimulatorServer holds the pre-parsed metrics, mutation rules, and per-rule state.
type SimulatorServer struct {
	Metrics []*parser.Metric
	Rules   []mutator.Rule
	States  map[int]*mutator.RuleState // keyed by rule index
}

// New creates a SimulatorServer with per-rule states initialized to now.
func New(metrics []*parser.Metric, rules []mutator.Rule) *SimulatorServer {
	now := time.Now()
	states := make(map[int]*mutator.RuleState, len(rules))
	for i := range rules {
		states[i] = mutator.NewRuleState(now)
	}
	return &SimulatorServer{
		Metrics: metrics,
		Rules:   rules,
		States:  states,
	}
}

// MetricsHandler serves the mutated metrics in Prometheus text exposition format.
func (s *SimulatorServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	now := time.Now()
	seen := make(map[string]bool)

	for _, m := range s.Metrics {
		if !seen[m.Name] {
			if m.Help != "" {
				fmt.Fprintf(w, "# HELP %s %s\n", m.Name, m.Help)
			}
			fmt.Fprintf(w, "# TYPE %s %s\n", m.Name, string(m.Type))
			seen[m.Name] = true
		}

		value := m.Value
		for i, rule := range s.Rules {
			if rule.Matches(m.Name, m.Labels) {
				value = rule.Mutator.Apply(value, s.States[i], rule.Schedule, now)
				break // first matching rule wins
			}
		}

		if len(m.Labels) > 0 {
			fmt.Fprintf(w, "%s{%s} %s\n", m.Name, formatLabels(m.Labels), formatValue(value))
		} else {
			fmt.Fprintf(w, "%s %s\n", m.Name, formatValue(value))
		}
	}
}

// formatLabels returns a sorted, comma-joined label string: key="value",...
func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + `="` + labels[k] + `"`
	}
	return strings.Join(parts, ",")
}

// formatValue formats a float64 using the shortest Prometheus-compatible representation.
func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
