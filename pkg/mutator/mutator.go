package mutator

import (
	"math"
	"math/rand"
	"time"
)

// Mutator applies a transformation to a metric value.
// state carries per-rule scheduling state; sched controls when the mutator fires;
// now is the current clock snapshot for the scrape.
type Mutator interface {
	Apply(value float64, state *RuleState, sched ScheduleConfig, now time.Time) float64
}

// Jitter adds random noise proportional to the current value.
// Variance is the fractional variance, e.g. 0.05 = ±5%.
type Jitter struct {
	Variance float64
}

func (j Jitter) Apply(value float64, state *RuleState, sched ScheduleConfig, now time.Time) float64 {
	if !state.IsActive(sched, now) {
		return value
	}
	noise := (rand.Float64()*2 - 1) * j.Variance * value
	return value + noise
}

// Trend adds a linear drift to the current value, computed from the start of the active window.
// RatePerSecond is the units added per second (negative = decreasing).
type Trend struct {
	RatePerSecond float64
}

func (t Trend) Apply(value float64, state *RuleState, sched ScheduleConfig, now time.Time) float64 {
	if !state.IsActive(sched, now) {
		return value
	}
	elapsed := now.Sub(state.GetActiveSince())
	return value + t.RatePerSecond*elapsed.Seconds()
}

// Spike multiplies the current value during the active window.
type Spike struct {
	Multiplier float64
}

func (s Spike) Apply(value float64, state *RuleState, sched ScheduleConfig, now time.Time) float64 {
	if !state.IsActive(sched, now) {
		return value
	}
	return value * s.Multiplier
}

// Wave adds a sinusoidal oscillation using lifetime elapsed (from state.StartTime).
// Amplitude is the peak deviation; Frequency is in Hz.
type Wave struct {
	Amplitude float64
	Frequency float64
}

func (w Wave) Apply(value float64, state *RuleState, sched ScheduleConfig, now time.Time) float64 {
	if !state.IsActive(sched, now) {
		return value
	}
	elapsed := now.Sub(state.StartTime)
	return value + w.Amplitude*math.Sin(2*math.Pi*w.Frequency*elapsed.Seconds())
}
