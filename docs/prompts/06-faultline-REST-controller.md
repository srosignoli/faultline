Let's build the REST API for the PromSim controller. This API will wrap our `k8s.Client` so we can manage simulators via HTTP.

Create a new Go package at `/pkg/api`.

Requirements:
1. Define a `Handler` struct that holds a reference to the `k8s.Client` interface (you may need to extract an interface from the K8s package if you haven't already, to make mocking easier).
2. Create a function `NewRouter(h *Handler) *http.ServeMux` that sets up the following routes using Go 1.22+ standard library routing:
   - `GET /api/simulators`: Calls the K8s List method and returns a JSON array of running instances.
   - `POST /api/simulators`: Expects a JSON body with `Name` (string), `DumpPayload` (string), and `RulesPayload` (string). It should validate these fields and call the K8s Create method. Return a 201 Created status on success.
   - `DELETE /api/simulators/{name}`: Extracts the `{name}` wildcard from the URL and calls the K8s Delete method. Return a 204 No Content on success.
3. Define standard JSON request and response structs for these endpoints. Ensure appropriate HTTP status codes (400 for bad input, 500 for K8s errors) and write JSON error messages.
4. Implement table-driven tests in `api_test.go` using the `net/http/httptest` package. You will need to create a mock implementation of the K8s client to inject into the `Handler` so you can test the HTTP layer in isolation without a real cluster.
5. Keep the code strictly within standard library packages (`net/http`, `encoding/json`). Do not implement the `main.go` entrypoint yet, just focus on the isolated API handlers and routing.
