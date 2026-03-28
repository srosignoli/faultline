Let's build the configuration loader and rule matching engine for PromSim.

Create a new Go package at `/pkg/config`. This package will bridge the user's YAML definitions with our `mutator` and `parser` packages.

Requirements:
1. Define a `Config` struct that can unmarshal a YAML file containing a list of `rules`. 
2. A rule should contain:
   - `Name` (string)
   - `Match`: A struct containing `MetricName` (string) and `Labels` (map[string]string).
   - `Mutator`: A struct containing `Type` (string: jitter, trend, spike, wave) and `Params` (map[string]interface{}).
3. Write a function `LoadConfig(path string) (*Config, error)` that reads and parses the YAML file.
4. Write a function `ApplyRules(metrics []parser.Metric, cfg *Config)` or similar logic. This function should iterate through the metrics, check if they match a rule's `MetricName` and `Labels`, and if so, instantiate the correct `mutator.Mutator` using a factory pattern based on the `Type` and `Params`.
5. Include robust table-driven tests in `config_test.go`. Mock a YAML file and an array of `parser.Metric` structs, and verify that the correct mutators are assigned to the correct metrics based on label matching.
6. Use the standard `gopkg.in/yaml.v3` library for YAML parsing.
