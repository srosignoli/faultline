package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/srosignoli/faultline/pkg/mutator"
)

// Config is the top-level configuration structure.
type Config struct {
	Rules []RuleConfig `yaml:"rules"`
}

// RuleConfig describes a single mutation rule.
type RuleConfig struct {
	Name    string        `yaml:"name"    json:"name"`
	Match   MatchConfig   `yaml:"match"   json:"match"`
	Mutator MutatorConfig `yaml:"mutator" json:"mutator"`
}

// MatchConfig specifies which metrics a rule applies to.
type MatchConfig struct {
	MetricName string            `yaml:"metric_name" json:"metric_name"`
	Labels     map[string]string `yaml:"labels"      json:"labels,omitempty"`
}

// MutatorConfig specifies the mutator type and its parameters.
// Params is map[string]interface{} because yaml.v3 decodes whole-number
// scalars as int and decimals as float64 — handled by the toFloat64 helper.
type MutatorConfig struct {
	Type   string                 `yaml:"type"   json:"type"`
	Params map[string]interface{} `yaml:"params" json:"params"`
}

// ParseConfig parses raw YAML bytes into a Config.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}

// LoadConfig reads and parses a YAML config file at path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}

// BuildRules converts a Config into a slice of mutator.Rule ready for use.
func BuildRules(cfg *Config) ([]mutator.Rule, error) {
	rules := make([]mutator.Rule, 0, len(cfg.Rules))
	for i, rc := range cfg.Rules {
		m, err := buildMutator(rc.Mutator)
		if err != nil {
			return nil, fmt.Errorf("config: rule[%d] %q: %w", i, rc.Name, err)
		}
		rules = append(rules, mutator.Rule{
			Selector: mutator.LabelSelector{
				Name:   rc.Match.MetricName,
				Labels: rc.Match.Labels,
			},
			Mutator: m,
		})
	}
	return rules, nil
}

func buildMutator(mc MutatorConfig) (mutator.Mutator, error) {
	switch mc.Type {
	case "jitter":
		v, err := toFloat64(mc.Params["variance"])
		if err != nil {
			return nil, fmt.Errorf("jitter: variance: %w", err)
		}
		return mutator.Jitter{Variance: v}, nil

	case "trend":
		v, err := toFloat64(mc.Params["rate_per_second"])
		if err != nil {
			return nil, fmt.Errorf("trend: rate_per_second: %w", err)
		}
		return mutator.Trend{RatePerSecond: v}, nil

	case "spike":
		mult, err := toFloat64(mc.Params["multiplier"])
		if err != nil {
			return nil, fmt.Errorf("spike: multiplier: %w", err)
		}
		dur, err := toDuration(mc.Params["duration"])
		if err != nil {
			return nil, fmt.Errorf("spike: duration: %w", err)
		}
		return mutator.Spike{Multiplier: mult, Duration: dur}, nil

	case "wave":
		amp, err := toFloat64(mc.Params["amplitude"])
		if err != nil {
			return nil, fmt.Errorf("wave: amplitude: %w", err)
		}
		freq, err := toFloat64(mc.Params["frequency"])
		if err != nil {
			return nil, fmt.Errorf("wave: frequency: %w", err)
		}
		return mutator.Wave{Amplitude: amp, Frequency: freq}, nil

	default:
		return nil, fmt.Errorf("unknown mutator type %q", mc.Type)
	}
}

func toFloat64(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case nil:
		return 0, fmt.Errorf("missing required parameter")
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

func toDuration(v interface{}) (time.Duration, error) {
	switch x := v.(type) {
	case string:
		d, err := time.ParseDuration(x)
		if err != nil {
			return 0, err
		}
		return d, nil
	case nil:
		return 0, fmt.Errorf("missing required parameter")
	default:
		return 0, fmt.Errorf("expected duration string, got %T", v)
	}
}
