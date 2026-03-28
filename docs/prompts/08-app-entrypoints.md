Let's wire up the application packages by creating the `main.go` entrypoints for PromSim.

We need two separate binaries. Create them at `/cmd/simulator/main.go` and `/cmd/controller/main.go`.

Requirements for `/cmd/simulator/main.go`:
1. Read the file paths for the Prometheus dump and the rules YAML. Default to `/etc/faultline/dump.txt` and `/etc/faultline/rules.yaml`, but allow overriding via environment variables (`PROMSIM_DUMP_PATH`, `PROMSIM_RULES_PATH`).
2. Call `parser.ParseDump` to load the initial metrics.
3. Call `config.LoadConfig` and `config.ApplyRules` to attach the mutators to the parsed metrics.
4. Initialize the `server.SimulatorServer` with the mutated metrics and start an HTTP server on port 8080.
5. Implement graceful shutdown by capturing `os.Interrupt` and `syscall.SIGTERM`.

Requirements for `/cmd/controller/main.go`:
1. Initialize the Kubernetes client from `/pkg/k8s`. It should attempt to use `rest.InClusterConfig()` first. If that fails (e.g., when running locally), it should fall back to using the local kubeconfig file path from the `KUBECONFIG` environment variable or the default `~/.kube/config`.
2. Initialize the `api.Handler` using the configured K8s client.
3. Set up the HTTP routes using `api.NewRouter`.
4. Start the HTTP server on port 8080.
5. Implement graceful shutdown by capturing `os.Interrupt` and `syscall.SIGTERM`.

General Requirements:
- Use the standard `log/slog` package for structured JSON logging in both files.
- Ensure any errors during startup (like a missing dump file or bad K8s config) cause the program to log the error and exit with a non-zero status code (`os.Exit(1)`).
- Keep the code clean and strictly focused on wiring the dependencies. Do not write the Dockerfiles yet.
