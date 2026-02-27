package mutator

import (
	"math"
	"math/rand"
	"time"
)

// Mutator applies a transformation to a metric value based on elapsed time.
// elapsed is time since the rule was first applied to a metric.
// Mutators are stateless w.r.t. the clock; the caller owns startTime.
type Mutator interface {
	Apply(currentValue float64, elapsed time.Duration) float64
}

// Jitter adds random noise proportional to the current value.
// Variance is the fractional variance, e.g. 0.05 = ±5%.
type Jitter struct {
	Variance float64
}

func (j Jitter) Apply(currentValue float64, _ time.Duration) float64 {
	noise := (rand.Float64()*2 - 1) * j.Variance * currentValue
	return currentValue + noise
}

// Trend adds a linear drift to the current value over time.
// RatePerSecond is the units added per second (negative = decreasing).
type Trend struct {
	RatePerSecond float64
}

func (t Trend) Apply(currentValue float64, elapsed time.Duration) float64 {
	return currentValue + t.RatePerSecond*elapsed.Seconds()
}

// Spike multiplies the current value by Multiplier for the given Duration,
// then returns the original value. At elapsed == Duration the spike is off.
type Spike struct {
	Multiplier float64
	Duration   time.Duration
}

func (s Spike) Apply(currentValue float64, elapsed time.Duration) float64 {
	if elapsed < s.Duration {
		return currentValue * s.Multiplier
	}
	return currentValue
}

// Wave adds a sinusoidal oscillation to the current value.
// Amplitude is the peak deviation; Frequency is in Hz.
type Wave struct {
	Amplitude float64
	Frequency float64
}

func (w Wave) Apply(currentValue float64, elapsed time.Duration) float64 {
	return currentValue + w.Amplitude*math.Sin(2*math.Pi*w.Frequency*elapsed.Seconds())
}
