Now that we have the parsed metrics, let's build the Mutation Engine for PromSim.

Create a new Go package at `/pkg/mutator`. The goal is to define rules that modify metric values programmatically over time to simulate real-world behaviors.

Requirements:
1. Define a `Mutator` interface with a method like: `Apply(currentValue float64, elapsed time.Duration) float64`. 
2. Implement the following concrete mutator types that satisfy this interface:
   - `Jitter`: Takes a `variance` percentage and adds/subtracts random noise to the value.
   - `Trend`: Takes a `ratePerSecond` and applies a steady linear increase or decrease based on elapsed time.
   - `Spike`: Takes a `multiplier` and a `duration`. It triggers a spike for that duration, then returns to the original value.
   - `Wave`: Takes an `amplitude` and `frequency` to simulate cyclic behaviors (like daily traffic) using a sine wave.
3. Create a `Rule` struct that ties a specific `Mutator` to a metric label selector (e.g., "apply this Jitter to any metric where name=process_cpu_seconds_total").
4. Write table-driven tests in `mutator_test.go` to verify the math for each mutator over simulated time steps.
5. Keep the code isolated to the `/pkg/mutator` package. Do not hook it up to the parser or build the HTTP server yet. Focus purely on robust mathematical transformations and testing.
