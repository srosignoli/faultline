export interface SyntheticPattern {
  id: string;
  group: string;
  name: string;
  metricDump: string;
  mutationRule: string;
}

export const syntheticPatterns: SyntheticPattern[] = [
  // ── Spikes ──────────────────────────────────────────────────────────────
  {
    id: "spike-traffic",
    group: "Spikes (Sudden Surges)",
    name: "Viral Traffic Surge",
    metricDump:
      "# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{method=\"GET\",status=\"200\"} 1500\nhttp_requests_total{method=\"POST\",status=\"200\"} 300\nhttp_requests_total{method=\"GET\",status=\"500\"} 12",
    mutationRule:
      "rules:\n  - name: viral-traffic-spike\n    match:\n      metric_name: http_requests_total\n      labels:\n        method: GET\n        status: \"200\"\n    mutator:\n      type: spike\n      params:\n        multiplier: 25\n        duration: 45s\n        interval: 3m\n        interval_jitter: 30s",
  },
  {
    id: "spike-memory",
    group: "Spikes (Sudden Surges)",
    name: "Memory Pressure Spike",
    metricDump:
      "# HELP process_resident_memory_bytes Resident memory size in bytes\n# TYPE process_resident_memory_bytes gauge\nprocess_resident_memory_bytes 52428800\n# HELP go_memstats_heap_alloc_bytes Heap allocated bytes\n# TYPE go_memstats_heap_alloc_bytes gauge\ngo_memstats_heap_alloc_bytes 24576000",
    mutationRule:
      "rules:\n  - name: memory-spike\n    match:\n      metric_name: process_resident_memory_bytes\n    mutator:\n      type: spike\n      params:\n        multiplier: 8\n        duration: 30s\n        interval: 2m",
  },
  {
    id: "spike-errors",
    group: "Spikes (Sudden Surges)",
    name: "Error Rate Explosion",
    metricDump:
      "# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{method=\"GET\",status=\"200\"} 9800\nhttp_requests_total{method=\"GET\",status=\"500\"} 20\nhttp_requests_total{method=\"POST\",status=\"500\"} 5",
    mutationRule:
      "rules:\n  - name: error-spike\n    match:\n      metric_name: http_requests_total\n      labels:\n        status: \"500\"\n    mutator:\n      type: spike\n      params:\n        multiplier: 50\n        duration: 20s\n        interval: 90s\n        interval_jitter: 15s",
  },
  {
    id: "spike-latency",
    group: "Spikes (Sudden Surges)",
    name: "Database Latency Spike",
    metricDump:
      "# HELP db_query_duration_seconds Database query duration\n# TYPE db_query_duration_seconds histogram\ndb_query_duration_seconds_bucket{le=\"0.01\"} 800\ndb_query_duration_seconds_bucket{le=\"0.1\"} 950\ndb_query_duration_seconds_bucket{le=\"1\"} 995\ndb_query_duration_seconds_bucket{le=\"+Inf\"} 1000\ndb_query_duration_seconds_sum 12.5\ndb_query_duration_seconds_count 1000",
    mutationRule:
      "rules:\n  - name: db-latency-spike\n    match:\n      metric_name: db_query_duration_seconds_sum\n    mutator:\n      type: spike\n      params:\n        multiplier: 30\n        duration: 15s\n        interval: 2m\n        interval_jitter: 20s",
  },

  // ── Trends ───────────────────────────────────────────────────────────────
  {
    id: "trend-memory-leak",
    group: "Trends (Gradual Degradation)",
    name: "Memory Leak",
    metricDump:
      "# HELP process_resident_memory_bytes Resident memory size in bytes\n# TYPE process_resident_memory_bytes gauge\nprocess_resident_memory_bytes 104857600",
    mutationRule:
      "rules:\n  - name: memory-leak\n    match:\n      metric_name: process_resident_memory_bytes\n    mutator:\n      type: trend\n      params:\n        rate_per_second: 51200",
  },
  {
    id: "trend-disk-fill",
    group: "Trends (Gradual Degradation)",
    name: "Disk Filling Up",
    metricDump:
      "# HELP node_filesystem_avail_bytes Filesystem available bytes\n# TYPE node_filesystem_avail_bytes gauge\nnode_filesystem_avail_bytes{device=\"/dev/sda1\",mountpoint=\"/\"} 10737418240",
    mutationRule:
      "rules:\n  - name: disk-fill\n    match:\n      metric_name: node_filesystem_avail_bytes\n    mutator:\n      type: trend\n      params:\n        rate_per_second: -1048576",
  },
  {
    id: "trend-connection-pool",
    group: "Trends (Gradual Degradation)",
    name: "Connection Pool Exhaustion",
    metricDump:
      "# HELP db_connections_active Active database connections\n# TYPE db_connections_active gauge\ndb_connections_active 5\n# HELP db_connections_max Maximum database connections\n# TYPE db_connections_max gauge\ndb_connections_max 100",
    mutationRule:
      "rules:\n  - name: conn-pool-trend\n    match:\n      metric_name: db_connections_active\n    mutator:\n      type: trend\n      params:\n        rate_per_second: 0.5",
  },
  {
    id: "trend-error-rate",
    group: "Trends (Gradual Degradation)",
    name: "Slowly Rising Error Rate",
    metricDump:
      "# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{status=\"200\"} 50000\nhttp_requests_total{status=\"500\"} 100",
    mutationRule:
      "rules:\n  - name: rising-errors\n    match:\n      metric_name: http_requests_total\n      labels:\n        status: \"500\"\n    mutator:\n      type: trend\n      params:\n        rate_per_second: 2",
  },

  // ── Jitter ───────────────────────────────────────────────────────────────
  {
    id: "jitter-network",
    group: "Jitter (Instability & Noise)",
    name: "Network Packet Loss Noise",
    metricDump:
      "# HELP node_network_receive_drop_total Network receive drops\n# TYPE node_network_receive_drop_total counter\nnode_network_receive_drop_total{device=\"eth0\"} 42",
    mutationRule:
      "rules:\n  - name: network-jitter\n    match:\n      metric_name: node_network_receive_drop_total\n    mutator:\n      type: jitter\n      params:\n        variance: 0.4",
  },
  {
    id: "jitter-cpu",
    group: "Jitter (Instability & Noise)",
    name: "Unstable CPU Usage",
    metricDump:
      "# HELP node_cpu_seconds_total CPU seconds total\n# TYPE node_cpu_seconds_total counter\nnode_cpu_seconds_total{cpu=\"0\",mode=\"user\"} 12000\nnode_cpu_seconds_total{cpu=\"0\",mode=\"system\"} 3000\nnode_cpu_seconds_total{cpu=\"0\",mode=\"idle\"} 85000",
    mutationRule:
      "rules:\n  - name: cpu-jitter\n    match:\n      metric_name: node_cpu_seconds_total\n      labels:\n        mode: user\n    mutator:\n      type: jitter\n      params:\n        variance: 0.3",
  },
  {
    id: "jitter-response-time",
    group: "Jitter (Instability & Noise)",
    name: "Erratic Response Times",
    metricDump:
      "# HELP http_response_time_seconds HTTP response time\n# TYPE http_response_time_seconds gauge\nhttp_response_time_seconds{handler=\"/api/users\"} 0.05\nhttp_response_time_seconds{handler=\"/api/orders\"} 0.12",
    mutationRule:
      "rules:\n  - name: response-jitter\n    match:\n      metric_name: http_response_time_seconds\n    mutator:\n      type: jitter\n      params:\n        variance: 0.6",
  },
  {
    id: "jitter-queue",
    group: "Jitter (Instability & Noise)",
    name: "Flapping Queue Depth",
    metricDump:
      "# HELP job_queue_depth Current job queue depth\n# TYPE job_queue_depth gauge\njob_queue_depth{queue=\"default\"} 15\njob_queue_depth{queue=\"priority\"} 3",
    mutationRule:
      "rules:\n  - name: queue-jitter\n    match:\n      metric_name: job_queue_depth\n    mutator:\n      type: jitter\n      params:\n        variance: 0.8",
  },

  // ── Outages ──────────────────────────────────────────────────────────────
  {
    id: "outage-service-down",
    group: "Outages (Complete Failures)",
    name: "Service Complete Outage",
    metricDump:
      "# HELP up Service health (1=up, 0=down)\n# TYPE up gauge\nup{job=\"api-server\",instance=\"10.0.0.1:8080\"} 1\nup{job=\"api-server\",instance=\"10.0.0.2:8080\"} 1\nup{job=\"api-server\",instance=\"10.0.0.3:8080\"} 1",
    mutationRule:
      "rules:\n  - name: service-outage\n    match:\n      metric_name: up\n    mutator:\n      type: spike\n      params:\n        multiplier: 0\n        duration: 5m\n        interval: 10m",
  },
  {
    id: "outage-zero-traffic",
    group: "Outages (Complete Failures)",
    name: "Zero Incoming Traffic",
    metricDump:
      "# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{method=\"GET\",status=\"200\"} 8000\nhttp_requests_total{method=\"POST\",status=\"200\"} 1500",
    mutationRule:
      "rules:\n  - name: traffic-blackout\n    match:\n      metric_name: http_requests_total\n    mutator:\n      type: spike\n      params:\n        multiplier: 0\n        duration: 3m\n        interval: 8m\n        interval_jitter: 1m",
  },
  {
    id: "outage-db-connection",
    group: "Outages (Complete Failures)",
    name: "Database Connection Loss",
    metricDump:
      "# HELP db_connections_active Active database connections\n# TYPE db_connections_active gauge\ndb_connections_active{pool=\"primary\"} 20\ndb_connections_active{pool=\"replica\"} 10\n# HELP db_query_errors_total Total DB query errors\n# TYPE db_query_errors_total counter\ndb_query_errors_total 3",
    mutationRule:
      "rules:\n  - name: db-outage-connections\n    match:\n      metric_name: db_connections_active\n    mutator:\n      type: spike\n      params:\n        multiplier: 0\n        duration: 4m\n        interval: 12m\n  - name: db-outage-errors\n    match:\n      metric_name: db_query_errors_total\n    mutator:\n      type: spike\n      params:\n        multiplier: 100\n        duration: 4m\n        interval: 12m",
  },
  {
    id: "outage-partial-degradation",
    group: "Outages (Complete Failures)",
    name: "Partial Region Failure",
    metricDump:
      "# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{region=\"us-east\",status=\"200\"} 5000\nhttp_requests_total{region=\"us-west\",status=\"200\"} 4800\nhttp_requests_total{region=\"eu-west\",status=\"200\"} 3200\nhttp_requests_total{region=\"us-east\",status=\"503\"} 5\nhttp_requests_total{region=\"us-west\",status=\"503\"} 3",
    mutationRule:
      "rules:\n  - name: region-failure\n    match:\n      metric_name: http_requests_total\n      labels:\n        region: eu-west\n        status: \"200\"\n    mutator:\n      type: spike\n      params:\n        multiplier: 0\n        duration: 6m\n        interval: 15m\n  - name: region-errors\n    match:\n      metric_name: http_requests_total\n      labels:\n        region: eu-west\n        status: \"503\"\n    mutator:\n      type: spike\n      params:\n        multiplier: 200\n        duration: 6m\n        interval: 15m",
  },

  // ── Waves ────────────────────────────────────────────────────────────────
  {
    id: "wave-business-hours",
    group: "Waves (Cyclical Patterns)",
    name: "Business Hours Traffic",
    metricDump:
      "# HELP http_requests_total Total HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total{method=\"GET\",status=\"200\"} 3000",
    mutationRule:
      "rules:\n  - name: business-hours-wave\n    match:\n      metric_name: http_requests_total\n    mutator:\n      type: wave\n      params:\n        amplitude: 0.7\n        frequency: 0.0000231",
  },
  {
    id: "wave-heartbeat",
    group: "Waves (Cyclical Patterns)",
    name: "Heartbeat / Health Check",
    metricDump:
      "# HELP probe_success Probe success (1=ok)\n# TYPE probe_success gauge\nprobe_success{job=\"blackbox\",target=\"https://example.com\"} 1",
    mutationRule:
      "rules:\n  - name: heartbeat-wave\n    match:\n      metric_name: probe_success\n    mutator:\n      type: wave\n      params:\n        amplitude: 0.1\n        frequency: 0.0167",
  },
  {
    id: "wave-gc-pressure",
    group: "Waves (Cyclical Patterns)",
    name: "Periodic GC Pressure",
    metricDump:
      "# HELP go_gc_duration_seconds GC pause duration\n# TYPE go_gc_duration_seconds summary\ngo_gc_duration_seconds{quantile=\"0.5\"} 0.0001\ngo_gc_duration_seconds{quantile=\"0.9\"} 0.0003\ngo_gc_duration_seconds{quantile=\"0.99\"} 0.001\ngo_gc_duration_seconds_sum 0.25\ngo_gc_duration_seconds_count 500",
    mutationRule:
      "rules:\n  - name: gc-wave\n    match:\n      metric_name: go_gc_duration_seconds_sum\n    mutator:\n      type: wave\n      params:\n        amplitude: 0.5\n        frequency: 0.05",
  },
  {
    id: "wave-batch-job",
    group: "Waves (Cyclical Patterns)",
    name: "Scheduled Batch Job Load",
    metricDump:
      "# HELP worker_jobs_processed_total Jobs processed by workers\n# TYPE worker_jobs_processed_total counter\nworker_jobs_processed_total{worker=\"batch-1\"} 1200\nworker_jobs_processed_total{worker=\"batch-2\"} 1150\n# HELP worker_cpu_utilization Worker CPU utilization ratio\n# TYPE worker_cpu_utilization gauge\nworker_cpu_utilization{worker=\"batch-1\"} 0.15\nworker_cpu_utilization{worker=\"batch-2\"} 0.12",
    mutationRule:
      "rules:\n  - name: batch-load-wave\n    match:\n      metric_name: worker_cpu_utilization\n    mutator:\n      type: wave\n      params:\n        amplitude: 0.8\n        frequency: 0.00278",
  },
];
