Let's containerize the PromSim application. We need to create Dockerfiles for both the Simulator and the Controller using multi-stage builds to keep the final images small and secure.

Create two files in the root directory: `Dockerfile.simulator` and `Dockerfile.controller`.

Requirements for BOTH Dockerfiles:
1. Use a multi-stage build.
2. **Builder Stage:**
   - Use `golang:1.22-alpine` as the base image.
   - Set the working directory to `/app`.
   - Copy `go.mod` and `go.sum` first, then run `go mod download` (to cache dependencies).
   - Copy the rest of the source code.
   - Build the binary statically. Set `CGO_ENABLED=0` and `GOOS=linux`.
3. **Final Stage:**
   - Use `alpine:latest` as the base image (so we have basic debugging tools like `cat` and `sh` to inspect mounted ConfigMaps).
   - Install `ca-certificates` (crucial for the Controller to talk to the K8s API securely).
   - Create a non-root user and group named `faultline`.
   - Copy the compiled binary from the builder stage into `/usr/local/bin/`.
   - Set the user to `faultline` (`USER faultline`).
   - Expose port `8080`.

Specifics for `Dockerfile.simulator`:
- The build command should be: `go build -a -installsuffix cgo -o simulator ./cmd/simulator`
- Set the `ENTRYPOINT` to `["simulator"]`

Specifics for `Dockerfile.controller`:
- The build command should be: `go build -a -installsuffix cgo -o controller ./cmd/controller`
- Set the `ENTRYPOINT` to `["controller"]`
