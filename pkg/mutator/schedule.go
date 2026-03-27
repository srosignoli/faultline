package mutator

import (
	"math/rand"
	"sync"
	"time"
)

// ScheduleConfig controls when a mutator fires.
// Duration == 0 means always active (backward compatible).
type ScheduleConfig struct {
	InitialDelay   time.Duration
	Duration       time.Duration
	Interval       time.Duration
	IntervalJitter time.Duration
}

// RuleState holds per-rule scheduling state that persists across scrapes.
// StartTime is immutable after creation and safe to read without the lock.
// CurrentWindowStart, ActiveUntil, and NextTriggerTime are protected by mu.
type RuleState struct {
	mu                 sync.Mutex
	StartTime          time.Time
	CurrentWindowStart time.Time
	ActiveUntil        time.Time
	NextTriggerTime    time.Time
}

// NewRuleState creates a RuleState with the given start time.
func NewRuleState(startTime time.Time) *RuleState {
	return &RuleState{StartTime: startTime}
}

// IsActive reports whether the rule should fire at now.
// When a new window triggers, it updates state under mu.
func (rs *RuleState) IsActive(sched ScheduleConfig, now time.Time) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if now.Before(rs.StartTime.Add(sched.InitialDelay)) {
		return false
	}

	if sched.Duration == 0 {
		return true
	}

	if now.Before(rs.ActiveUntil) {
		return true
	}

	if rs.NextTriggerTime.IsZero() || !now.Before(rs.NextTriggerTime) {
		rs.CurrentWindowStart = now
		rs.ActiveUntil = now.Add(sched.Duration)
		jitter := time.Duration(0)
		if sched.IntervalJitter > 0 {
			jitter = time.Duration(rand.Int63n(int64(2*sched.IntervalJitter))) - sched.IntervalJitter
		}
		rs.NextTriggerTime = now.Add(sched.Interval + jitter)
		return true
	}

	return false
}

// GetCurrentWindowStart safely returns CurrentWindowStart under the lock.
func (rs *RuleState) GetCurrentWindowStart() time.Time {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.CurrentWindowStart
}
