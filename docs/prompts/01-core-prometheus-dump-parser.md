Please scaffold the core metric parsing engine for our FaultLine project. 

Create a new Go package at `/pkg/parser`. The goal of this package is to read a standard Prometheus exposition format text file (a metrics dump) and parse it into an internal Go struct that we can easily mutate later.

Requirements:
1. Define a robust internal data structure for the parsed metrics. We need to store:
   - Metric Name
   - Labels (as a map of key-value pairs)
   - Metric Type (Gauge, Counter, Histogram, Summary)
   - Help text
   - Current Value (float64)
2. Write a function `ParseDump(io.Reader) ([]Metric, error)` that reads standard Prometheus text output. It must correctly handle `# HELP` and `# TYPE` lines and associate them with the subsequent metric values.
3. Keep in mind that later, a "mutation engine" will need to iterate over these structs to modify their 'Current Value' based on rules, so design the structs to be easily mutable.
4. Create a comprehensive suite of table-driven tests in `parser_test.go` using a mock Prometheus text payload that includes at least one Counter, one Gauge, and metrics with multiple labels.
5. Ensure the code is strictly formatted and follows standard Go idioms. Do not implement the HTTP server or mutation logic yet, just the parser and its tests.
