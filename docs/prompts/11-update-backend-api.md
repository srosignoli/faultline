Let's update the MetricForge Controller API to return the active mutation rules for each simulator.

Modify the `GET /api/simulators` endpoint in `/pkg/api` and the corresponding list function in `/pkg/k8s`.

Requirements:
1. Update the `Simulator` struct in the API package to include an `ActiveRules` field. This should be an array of the `Rule` structs we defined in `/pkg/config` (which include Name, Match, and Mutator with its Params).
2. When the K8s client lists the running simulators, it should also fetch the `{name}-config` ConfigMap for each instance.
3. Parse the `rules.yaml` string from that ConfigMap and attach the parsed rules to the `Simulator` response object.
4. Update `api_test.go` to mock this ConfigMap retrieval and verify the JSON response includes the rule data and parameters (like variance, multiplier, etc.).
