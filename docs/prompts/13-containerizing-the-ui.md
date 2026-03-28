Let's containerize the MetricForge Web UI so it can be deployed directly into our Kubernetes cluster alongside the Controller.

Requirements:

1. **Create `Dockerfile.ui` in the root directory:**
   - Use a multi-stage build.
   - **Builder Stage:** Use `node:20-alpine`. Set working dir to `/app`. Copy `ui/package.json` and install dependencies. Copy the rest of the `/ui` source code and run the Vite build command (`npm run build`).
   - **Production Stage:** Use `nginx:alpine`. Copy the compiled static files from the builder stage (`/app/dist`) into Nginx's default serving directory (`/usr/share/nginx/html`). 
   - Expose port 80.

2. **Update the Kubernetes Manifests:**
   - In `/k8s/controller/`, create a new file `ui-deployment.yaml`.
   - Define a Deployment named `metricforge-ui` running the `metricforge-ui:latest` image.
   - Define a Service named `metricforge-ui-svc` exposing port 80.
   
3. **Important Nginx/Vite Config Note:**
   - When the React UI runs in the browser, it needs to know where the Controller API is. Ensure the API client in `client.ts` uses relative paths (e.g., `/api/simulators` instead of `http://localhost:8080/api/simulators`).
   - To make this work in K8s without CORS issues, please also generate a custom `nginx.conf` for the UI container that proxies requests from `/api/` directly to our backend K8s service `http://faultline-controller.faultline.svc.cluster.local:80`.
