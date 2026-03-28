Let's finalize the Kubernetes infrastructure for MetricForge by adding Prometheus monitoring and tying everything together with Kustomize.

Please create and update the following files:

1. **Create `/k8s/controller/servicemonitor.yaml`:**
   - `apiVersion: monitoring.coreos.com/v1`, `kind: ServiceMonitor`.
   - `metadata.name`: `metricforge-simulators`.
   - `spec.selector.matchLabels`: Set this to `app: metricforge-worker`. (This is the critical link that tells Prometheus to watch our dynamically generated pods).
   - `spec.endpoints`: Add an endpoint with `port: web`, `path: /metrics`, and `interval: 15s`.
   - `spec.namespaceSelector.matchNames`: Array containing `faultline`.

2. **Create `/k8s/controller/kustomization.yaml`:**
   - Define the `namespace: faultline`.
   - Include all the manifests in this directory under the `resources` block:
     - `serviceaccount.yaml`
     - `rbac.yaml`
     - `deployment.yaml` (The Go Controller)
     - `service.yaml`
     - `ui-deployment.yaml` (The containerized React UI)
     - `servicemonitor.yaml`

3. **Verify/Update the Go Controller (`/pkg/k8s/client.go`):**
   - Review the `CreateSimulator` method we built earlier.
   - Ensure the K8s `Service` it generates explicitly sets the label `app: metricforge-worker`.
   - Ensure the K8s `Service` it generates explicitly names the exposed port `"web"` (e.g., `Name: "web"` in the ServicePort struct) so the ServiceMonitor can find it.
