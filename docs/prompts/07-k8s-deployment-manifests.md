Let's define the Kubernetes deployment manifests and RBAC permissions for the PromSim Controller.

Create a new directory at `/k8s/controller`. We need to define the infrastructure required to run the Controller API securely in a Kubernetes cluster.

Requirements:
1. Create a `serviceaccount.yaml` defining a ServiceAccount named `faultline-controller` in the `faultline` namespace.
2. Create a `rbac.yaml` containing two resources:
   - A `Role` named `faultline-manager-role` in the `faultline` namespace. It must grant `get`, `list`, `watch`, `create`, `update`, `patch`, and `delete` permissions for:
     - API group `""` (core): resources `configmaps`, `services`.
     - API group `"apps"`: resources `deployments`.
   - A `RoleBinding` named `faultline-manager-binding` that binds the `faultline-manager-role` to the `faultline-controller` ServiceAccount.
3. Create a `deployment.yaml` for the Controller itself:
   - Name it `faultline-controller`.
   - Ensure `serviceAccountName: faultline-controller` is set so the pod assumes the RBAC identity.
   - Use a placeholder image `faultline-controller:latest`.
   - Expose the API port (e.g., 8080).
4. Create a `service.yaml` exposing the Controller deployment internally so our future UI can reach the REST API.
5. Provide a `kustomization.yaml` file (optional but recommended) to tie these manifests together easily.
6. Keep these purely as Kubernetes YAML manifests. Do not write any Go code for this step.
