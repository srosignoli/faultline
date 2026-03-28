Let's completely refactor the K8s mutator engine in the Go worker application (likely in `/pkg/worker/` or `/pkg/mutator/`) to support stateful, time-based scheduling and precise mathematical execution for all mutators (`spike`, `jitter`, `trend`, `outage`). 

Previously, mutator-specific params (like `multiplier` or `slope`) were lost or unhandled when scheduling params (like `duration`) were introduced, and the mathematical implementation was ambiguous. 

Please implement the following four requirements:

### 1. Architectural Fix: Flexible Params
Ensure the YAML unmarshaler can read BOTH scheduling keys AND mutator-specific keys from the `params` block. 
- You can either use `map[string]interface{}` for `params` and parse values dynamically, OR create a superset `MutatorParams` struct that contains every possible field (`duration`, `interval`, `initial_delay`, `interval_jitter`, `multiplier`, `slope`, `variance`, `action`) with `yaml:",omitempty"`.

### 2. The Stateful Engine
Implement a thread-safe `RuleState` struct to track scheduling across Prometheus HTTP scrapes:
- **Fields:** `StartTime`, `NextTriggerTime`, `ActiveUntil`, `CurrentWindowStart` (specifically for trend tracking), and a `sync.Mutex`.
- **Method `IsActive(time.Now())`:**
  - If `Now() < StartTime + initial_delay`: return false.
  - If `duration` is empty/0: return true (always active).
  - If `Now() < ActiveUntil`: return true (currently in an active window).
  - If `Now() >= NextTriggerTime`: Calculate next `ActiveUntil` (Now + duration) and next `NextTriggerTime` (Now + interval + random jitter). Update `CurrentWindowStart = Now()`. Return true.

### 3. Explicit Math & Execution Rules
Do not guess the implementation. Apply the mutations exactly as follows when `IsActive()` is true:

- **SPIKE (`type: "spike"`):**
  - Required Param: `multiplier` (float64)
  - Execution: `newValue = originalValue * multiplier`

- **TREND (`type: "trend"`):**
  - Required Param: `slope` (float64)
  - Execution: The slope is the rate of change **per second**. 
  - Calculate elapsed time: `elapsed := time.Since(CurrentWindowStart).Seconds()`
  - Execution: `newValue = originalValue + (slope * elapsed)`
  - *Crucial:* Because `CurrentWindowStart` resets in `IsActive()` when the interval triggers, this will correctly reset the trend to baseline, simulating a pod restart.

- **JITTER (`type: "jitter"`):**
  - Required Param: `variance` (float64, representing a percentage, e.g., 0.20 for 20%)
  - Execution: Calculate a random multiplier between `(1.0 - variance)` and `(1.0 + variance)`. Ensure `math/rand` is properly seeded.
  - Execution: `newValue = originalValue * random_multiplier`

- **OUTAGE (`type: "outage"`):**
  - Required Param: `action` (string)
  - Execution: If `action == "drop_to_zero"`, then `newValue = 0.0`

### 4. The Unit Test Requirement
Create a new test file (e.g., `mutator_test.go`). You MUST use the following 16 scenarios as a table-driven test. Your test must unmarshal this exact YAML payload, initialize the state engine, and verify that NO parameters (neither scheduling nor mutator-specific) are dropped or ignored.

```yaml
rules:
  # --- SPIKE SCENARIOS ---
  - name: "Viral Traffic Surge"
    match: { metric_name: "http_requests_total" }
    mutator:
      type: "spike"
      params: { multiplier: 50.0, initial_delay: "1m", duration: "2m", interval: "1h" }
  - name: "CPU Steal Time Spike"
    match: { metric_name: "node_cpu_seconds_total" }
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
      params: { multiplier: 10.0, duration: "10m", interval: "0s" }

  # --- TREND SCENARIOS ---
  - name: "API Server Memory Leak"
    match: { metric_name: "container_memory_working_set_bytes" }
    mutator:
      type: "trend"
      params: { slope: 1048576.0, interval: "1h" }
  - name: "Goroutine Leak"
    match: { metric_name: "go_goroutines" }
    mutator:
      type: "trend"
      params: { slope: 5.0, interval: "24h" }
  - name: "Log Spam Disk Fill"
    match: { metric_name: "node_filesystem_avail_bytes" }
    mutator:
      type: "trend"
      params: { slope: -50000000.0, interval: "0s" }
  - name: "Email Queue Backup"
    match: { metric_name: "rabbitmq_queue_messages" }
    mutator:
      type: "trend"
      params: { slope: 10.0, interval: "45m" }

  # --- JITTER SCENARIOS ---
  - name: "Unstable Network Interface"
    match: { metric_name: "node_network_transmit_drop_total" }
    mutator:
      type: "jitter"
      params: { variance: 0.80, duration: "5m", interval: "30m" }
  - name: "Redis Cache Thrashing"
    match: { metric_name: "redis_cache_hits_total" }
    mutator:
      type: "jitter"
      params: { variance: 0.50, duration: "10m", interval: "1h" }
  - name: "Sporadic CPU Throttling"
    match: { metric_name: "container_cpu_cfs_throttled_seconds_total" }
    mutator:
      type: "jitter"
      params: { variance: 0.35, duration: "2m", interval: "10m", interval_jitter: "5m" }
  - name: "DB Connection Storm"
    match: { metric_name: "pg_stat_activity_count" }
    mutator:
      type: "jitter"
      params: { variance: 0.90, duration: "3m", interval: "15m" }

  # --- OUTAGE SCENARIOS ---
  - name: "Auth Service Crash"
    match: { metric_name: "up" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "4m", interval: "2h" }
  - name: "Silent Backup Job Failure"
    match: { metric_name: "worker_last_seen_timestamp_seconds" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "30m", interval: "24h" }
  - name: "AWS SQS Outage"
    match: { metric_name: "aws_sqs_messages_received_total" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "15m", interval: "0s" }
  - name: "EBS Volume Detached"
    match: { metric_name: "node_disk_io_now" }
    mutator:
      type: "outage"
      params: { action: "drop_to_zero", duration: "8m", interval: "0s" }

For the new `wave` mutator, please use the following explicit mathematical execution:
- Required Params: `amplitude` (float64) and `frequency` (float64, in Hertz/waves per second).
- Execution: Calculate the elapsed time in seconds since `StartTime` (or `CurrentWindowStart` if you want it to reset on intervals). 
  `elapsed := time.Since(StartTime).Seconds()`
- The new value is calculated using a standard Sine wave:
  `newValue = originalValue + (amplitude * math.Sin(2 * math.Pi * frequency * elapsed))`
