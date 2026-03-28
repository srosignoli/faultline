Let's refactor the Go worker's mutator engine to support stateful, time-based scheduling for all mutators (`spike`, `jitter`, `trend`, `outage`).

Since Prometheus scrapes the `/metrics` endpoint repeatedly, the worker must maintain state between HTTP requests to know when a mutation window is active.

Please update the worker code (likely in `/pkg/worker/` or `/pkg/mutator/`) with the following requirements:

1. **Create a `ScheduleConfig` struct:**
   - Parse these optional fields from the rule's `params` map: `initial_delay`, `duration`, `interval`, `interval_jitter`.
   - Use `time.ParseDuration` to parse these strings into `time.Duration`.

2. **Create a Thread-Safe `RuleState` struct:**
   - This will track the time state for a specific rule.
   - Fields: `StartTime` (time.Time), `NextTriggerTime` (time.Time), `ActiveUntil` (time.Time), and a `sync.Mutex` for thread safety.

3. **Implement the `IsActive` logic:**
   - Create a method on `RuleState` that takes the `ScheduleConfig` and `time.Now()`.
   - **Logic flow:**
     - If `Now() < StartTime + initial_delay`, return `false` (nominal value).
     - If `duration` is empty/0, assume it's always active (return `true`).
     - If `Now() < ActiveUntil`, return `true` (we are inside an active spike/jitter window).
     - If `Now() >= NextTriggerTime`:
       - Calculate the next window: `ActiveUntil = Now() + duration`.
       - Calculate the next trigger: `NextTriggerTime = Now() + interval + random(-jitter, +jitter)`.
       - Return `true`.
     - Otherwise, return `false`.

4. **Update the Worker's Main Server/Parser:**
   - The worker struct handling the HTTP requests must hold a map of `map[string]*RuleState` (keyed by rule name) so state persists across HTTP scrapes.
   - Initialize the `StartTime` for all rules when the worker boots up.
   - When iterating over the matched metrics, pass the rule's `RuleState` and `ScheduleConfig` to the mutator.

5. **Update the Mutators (`spike`, `jitter`, `trend`):**
   - Before applying their math, they should call `IsActive()`.
   - If `false`, return the original `value` untouched.
   - If `true`, apply the mutation.
   - *Special case for `trend`:* It needs to track its current accumulated value in `RuleState` and reset it when the interval triggers.
