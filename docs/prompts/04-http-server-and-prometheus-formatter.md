Let's build the HTTP server and Prometheus formatter for Faultline 

Create a new Go package at `/pkg/server`. This package will serve the `/metrics` endpoint, calculating mutated values on the fly.

Requirements:
1. Define a `SimulatorServer` struct. It should hold:
   - The list of parsed metrics (and their attached mutators from the config layer).
   - A `StartTime` (time.Time) recorded when the server initializes.
2. Implement an `http.HandlerFunc` for the `/metrics` route.
3. Inside the handler, iterate over the metrics. For each metric:
   - Calculate `elapsed := time.Since(s.StartTime)`.
   - If the metric has a mutator attached, calculate `currentValue = mutator.Apply(baseValue, elapsed)`. If not, use the `baseValue`.
4. Write the output to the `http.ResponseWriter` strictly following the Prometheus text exposition format:
   - Include the `# HELP` and `# TYPE` lines before the metric data.
   - Format the labels correctly: `metric_name{label1="value1",label2="value2"} currentValue`.
5. Write table-driven tests in `server_test.go` using the `net/http/httptest` package. Mock a request to `/metrics` and verify that the output string matches the expected Prometheus format, and that the mutator math is correctly applied based on a mocked StartTime.
