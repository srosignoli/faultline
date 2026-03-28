Let's move on to Phase 2: The Control Plane. We need a Go package that interacts with the Kubernetes API to manage the lifecycle of our simulated exporters.

Create a new package at `/pkg/k8s`. 

Requirements:
1. Define a `Client` struct that wraps the `kubernetes.Clientset` from `k8s.io/client-go`. It should support both in-cluster configuration (when running in a Pod) and out-of-cluster configuration (using a local kubeconfig for testing).
2. Implement a method `CreateSimulator(ctx context.Context, name string, dumpPayload string, rulesPayload string) error`. This method must create three Kubernetes resources in a specific namespace (e.g., "faultline"):
   - A **ConfigMap** named `{name}-config` containing two data keys: `dump.txt` and `rules.yaml`.
   - A **Deployment** named `{name}-deployment` running a placeholder image `faultline-worker:latest`. It must mount the ConfigMap as a volume at `/etc/faultline/`. It should expose port 8080.
   - A **Service** named `{name}-svc` (ClusterIP) that routes traffic on port 80 to the Deployment's port 8080.
3. Add standard labels to all resources: `app=faultline-worker` and `instance={name}`.
4. Implement methods `ListSimulators(ctx context.Context)` and `DeleteSimulator(ctx context.Context, name string)` that list and clean up the Deployments, Services, and ConfigMaps associated with a specific instance name.
5. Focus strictly on the Kubernetes resource generation and API calls. Do not build the HTTP REST API for the controller yet.
6. Write tests using the `k8s.io/client-go/kubernetes/fake` package to verify that the Create, List, and Delete methods generate the correct resources in memory.
