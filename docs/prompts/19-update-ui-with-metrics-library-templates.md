Let's enhance the frontend UI by adding a "Synthetic Metric Pattern Library". This will be a dropdown selector that allows users to instantly load pre-configured, deterministic test scenarios into the Metric Dump and Mutation Rule text areas to validate PromQL alerts and Grafana dashboards.

Please implement this in two steps:

### Step 1: Create the Data Library
Create a new file (e.g., `src/data/syntheticPatterns.js` or `.ts`). Export a constant array named `syntheticPatterns` containing the following 20 scenarios. Note that `metricDump` and `mutationRule` are multiline strings.

```javascript
export const syntheticPatterns = [
  // --- SPIKE SCENARIOS ---
  {
    id: "spike-traffic", group: "Spikes (Sudden Surges)", name: "Viral Traffic Surge",
    metricDump: `# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{method="POST", status="500", service="checkout"} 42`,
    mutationRule: `rules:\n  - name: "Checkout 500 Error Surge"\n    match:\n      metric_name: "http_requests_total"\n      labels: { service: "checkout", status: "500" }\n    mutator:\n      type: "spike"\n      params:\n        multiplier: 50.0\n        initial_delay: "1m"\n        duration: "2m"\n        interval: "1h"`
  },
  {
    id: "spike-cpu", group: "Spikes (Sudden Surges)", name: "CPU Steal Time Spike",
    metricDump: `# HELP node_cpu_seconds_total CPU time in seconds\n# TYPE node_cpu_seconds_total counter\nnode_cpu_seconds_total{mode="steal", cpu="0"} 120.5`,
    mutationRule: `rules:\n  - name: "CPU Steal Time Spike"\n    match:\n      metric_name: "node_cpu_seconds_total"\n      labels: { mode: "steal" }\n    mutator:\n      type: "spike"\n      params:\n        multiplier: 15.0\n        duration: "5m"\n        interval: "4h"`
  },
  {
    id: "spike-db", group: "Spikes (Sudden Surges)", name: "Slow DB Queries",
    metricDump: `# HELP db_query_duration_seconds_sum Total time spent in DB queries\n# TYPE db_query_duration_seconds_sum counter\ndb_query_duration_seconds_sum{db="orders"} 350.2`,
    mutationRule: `rules:\n  - name: "Slow DB Queries"\n    match:\n      metric_name: "db_query_duration_seconds_sum"\n      labels: { db: "orders" }\n    mutator:\n      type: "spike"\n      params:\n        multiplier: 8.0\n        duration: "3m"\n        interval: "30m"`
  },
  {
    id: "spike-restart", group: "Spikes (Sudden Surges)", name: "CrashLoopBackOff Spike",
    metricDump: `# HELP kube_pod_container_status_restarts_total Number of container restarts\n# TYPE kube_pod_container_status_restarts_total counter\nkube_pod_container_status_restarts_total{pod="payment-worker-xyz"} 2`,
    mutationRule: `rules:\n  - name: "CrashLoopBackOff Spike"\n    match:\n      metric_name: "kube_pod_container_status_restarts_total"\n    mutator:\n      type: "spike"\n      params:\n        multiplier: 10.0\n        duration: "10m"\n        interval: "0s"`
  },

  // --- TREND SCENARIOS ---
  {
    id: "trend-memory", group: "Trends (Gradual Degradation)", name: "API Server Memory Leak",
    metricDump: `# HELP container_memory_working_set_bytes Current working set\n# TYPE container_memory_working_set_bytes gauge\ncontainer_memory_working_set_bytes{container="api-server"} 256000000`,
    mutationRule: `rules:\n  - name: "API Server Memory Leak"\n    match:\n      metric_name: "container_memory_working_set_bytes"\n    mutator:\n      type: "trend"\n      params:\n        slope: 1048576.0\n        interval: "1h"`
  },
  {
    id: "trend-goroutine", group: "Trends (Gradual Degradation)", name: "Goroutine Leak",
    metricDump: `# HELP go_goroutines Number of goroutines that currently exist.\n# TYPE go_goroutines gauge\ngo_goroutines{app="data-processor"} 150`,
    mutationRule: `rules:\n  - name: "Goroutine Leak"\n    match:\n      metric_name: "go_goroutines"\n    mutator:\n      type: "trend"\n      params:\n        slope: 5.0\n        interval: "24h"`
  },
  {
    id: "trend-disk", group: "Trends (Gradual Degradation)", name: "Log Spam Disk Fill",
    metricDump: `# HELP node_filesystem_avail_bytes Available space\n# TYPE node_filesystem_avail_bytes gauge\nnode_filesystem_avail_bytes{mountpoint="/var/lib/docker"} 50000000000`,
    mutationRule: `rules:\n  - name: "Log Spam Disk Fill"\n    match:\n      metric_name: "node_filesystem_avail_bytes"\n    mutator:\n      type: "trend"\n      params:\n        slope: -50000000.0\n        interval: "0s"`
  },
  {
    id: "trend-queue", group: "Trends (Gradual Degradation)", name: "Email Queue Backup",
    metricDump: `# HELP rabbitmq_queue_messages Number of messages ready to be delivered\n# TYPE rabbitmq_queue_messages gauge\nrabbitmq_queue_messages{queue="email_outbound"} 12`,
    mutationRule: `rules:\n  - name: "Email Queue Backup"\n    match:\n      metric_name: "rabbitmq_queue_messages"\n    mutator:\n      type: "trend"\n      params:\n        slope: 10.0\n        interval: "45m"`
  },

  // --- JITTER SCENARIOS ---
  {
    id: "jitter-network", group: "Jitter (Instability & Noise)", name: "Unstable Network Interface",
    metricDump: `# HELP node_network_transmit_drop_total Network drops\n# TYPE node_network_transmit_drop_total counter\nnode_network_transmit_drop_total{device="eth0"} 500`,
    mutationRule: `rules:\n  - name: "Unstable Network Interface"\n    match:\n      metric_name: "node_network_transmit_drop_total"\n    mutator:\n      type: "jitter"\n      params:\n        variance: 0.80\n        duration: "5m"\n        interval: "30m"`
  },
  {
    id: "jitter-cache", group: "Jitter (Instability & Noise)", name: "Redis Cache Thrashing",
    metricDump: `# HELP redis_cache_hits_total Number of successful cache lookups\n# TYPE redis_cache_hits_total counter\nredis_cache_hits_total{instance="redis-primary"} 850000`,
    mutationRule: `rules:\n  - name: "Redis Cache Thrashing"\n    match:\n      metric_name: "redis_cache_hits_total"\n    mutator:\n      type: "jitter"\n      params:\n        variance: 0.50\n        duration: "10m"\n        interval: "1h"`
  },
  {
    id: "jitter-cpu", group: "Jitter (Instability & Noise)", name: "Sporadic CPU Throttling",
    metricDump: `# HELP container_cpu_cfs_throttled_seconds_total Total time throttled\n# TYPE container_cpu_cfs_throttled_seconds_total counter\ncontainer_cpu_cfs_throttled_seconds_total{container="worker"} 45.2`,
    mutationRule: `rules:\n  - name: "Sporadic CPU Throttling"\n    match:\n      metric_name: "container_cpu_cfs_throttled_seconds_total"\n    mutator:\n      type: "jitter"\n      params:\n        variance: 0.35\n        duration: "2m"\n        interval: "10m"\n        interval_jitter: "5m"`
  },
  {
    id: "jitter-db", group: "Jitter (Instability & Noise)", name: "DB Connection Storm",
    metricDump: `# HELP pg_stat_activity_count Number of active connections\n# TYPE pg_stat_activity_count gauge\npg_stat_activity_count{datname="production"} 40`,
    mutationRule: `rules:\n  - name: "DB Connection Storm"\n    match:\n      metric_name: "pg_stat_activity_count"\n    mutator:\n      type: "jitter"\n      params:\n        variance: 0.90\n        duration: "3m"\n        interval: "15m"`
  },

  // --- OUTAGE SCENARIOS ---
  {
    id: "outage-crash", group: "Outages (Complete Failures)", name: "Auth Service Crash",
    metricDump: `# HELP up Target is scrapeable\n# TYPE up gauge\nup{job="auth-service", instance="10.0.0.5"} 1`,
    mutationRule: `rules:\n  - name: "Auth Service Crash"\n    match:\n      metric_name: "up"\n    mutator:\n      type: "outage"\n      params:\n        action: "drop_to_zero"\n        duration: "4m"\n        interval: "2h"`
  },
  {
    id: "outage-heartbeat", group: "Outages (Complete Failures)", name: "Silent Backup Job Failure",
    metricDump: `# HELP worker_last_seen_timestamp_seconds Epoch time of last heartbeat\n# TYPE worker_last_seen_timestamp_seconds gauge\nworker_last_seen_timestamp_seconds{worker_id="backup-job"} 1709320000`,
    mutationRule: `rules:\n  - name: "Silent Backup Job Failure"\n    match:\n      metric_name: "worker_last_seen_timestamp_seconds"\n    mutator:\n      type: "outage"\n      params:\n        action: "drop_to_zero"\n        duration: "30m"\n        interval: "24h"`
  },
  {
    id: "outage-sqs", group: "Outages (Complete Failures)", name: "AWS SQS Outage",
    metricDump: `# HELP aws_sqs_messages_received_total Messages pulled from SQS\n# TYPE aws_sqs_messages_received_total counter\naws_sqs_messages_received_total{queue="orders"} 14502`,
    mutationRule: `rules:\n  - name: "AWS SQS Outage"\n    match:\n      metric_name: "aws_sqs_messages_received_total"\n    mutator:\n      type: "outage"\n      params:\n        action: "drop_to_zero"\n        duration: "15m"\n        interval: "0s"`
  },
  {
    id: "outage-ebs", group: "Outages (Complete Failures)", name: "EBS Volume Detached",
    metricDump: `# HELP node_disk_io_now Number of I/O operations currently in progress\n# TYPE node_disk_io_now gauge\nnode_disk_io_now{device="nvme0n1"} 24`,
    mutationRule: `rules:\n  - name: "EBS Volume Detached"\n    match:\n      metric_name: "node_disk_io_now"\n      labels: { device: "nvme0n1" }\n    mutator:\n      type: "outage"\n      params:\n        action: "drop_to_zero"\n        duration: "8m"\n        interval: "0s"`
  },

  // --- WAVE SCENARIOS ---
  {
    id: "wave-daily", group: "Waves (Cyclical Patterns)", name: "Daily Traffic Wave",
    metricDump: `# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{method="GET", service="frontend"} 1000`,
    mutationRule: `rules:\n  - name: "Daily Traffic Wave"\n    match:\n      metric_name: "http_requests_total"\n    mutator:\n      type: "wave"\n      params:\n        amplitude: 500.0\n        frequency: 0.0000115741`
  },
  {
    id: "wave-latency", group: "Waves (Cyclical Patterns)", name: "Intraday Latency Oscillation",
    metricDump: `# HELP http_request_duration_seconds Average request duration\n# TYPE http_request_duration_seconds gauge\nhttp_request_duration_seconds{endpoint="/api/v1/users"} 0.15`,
    mutationRule: `rules:\n  - name: "Intraday Latency Oscillation"\n    match:\n      metric_name: "http_request_duration_seconds"\n    mutator:\n      type: "wave"\n      params:\n        amplitude: 0.05\n        frequency: 0.0000347222\n        initial_delay: "5m"`
  },
  {
    id: "wave-noise", group: "Waves (Cyclical Patterns)", name: "Periodic Scrape Noise",
    metricDump: `# HELP up Target is scrapeable\n# TYPE up gauge\nup{job="temperature_sensor"} 1.0`,
    mutationRule: `rules:\n  - name: "Periodic Scrape Noise"\n    match:\n      metric_name: "up"\n    mutator:\n      type: "wave"\n      params:\n        amplitude: 0.1\n        frequency: 0.00166667\n        duration: "10m"\n        interval: "1h"`
  },
  {
    id: "wave-batch", group: "Waves (Cyclical Patterns)", name: "Weekly Batch Volume Pattern",
    metricDump: `# HELP batch_jobs_total Total batch jobs processed\n# TYPE batch_jobs_total counter\nbatch_jobs_total{queue="nightly_reports"} 5000`,
    mutationRule: `rules:\n  - name: "Weekly Batch Volume Pattern"\n    match:\n      metric_name: "batch_jobs_total"\n    mutator:\n      type: "wave"\n      params:\n        amplitude: 200.0\n        frequency: 0.00000165`
  }
];
