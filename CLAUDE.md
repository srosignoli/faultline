# Project: FaultLine (Prometheus Exporter Simulator)

## Project Overview
This project dynamically simulates Prometheus exporters based on static metric dumps. It allows users to manipulate metric values (spikes, waves, drops) programmatically to test monitoring and alerting systems. The system is orchestrated within Kubernetes.

## Architecture
- **/cmd/simulator**: A Go-based worker that parses a Prometheus text dump, applies mutation rules, and serves a `/metrics` endpoint.
- **/cmd/controller**: A Go-based API that manages the lifecycle of simulators. It interacts with the Kubernetes API to spawn simulators as Deployments/Pods.
- **/ui**: A React/Vite/Tailwind frontend for uploading dumps and managing running simulators.
- **/k8s**: Base Helm charts or Kubernetes manifests for the controller.

## Tech Stack
- Backend: Go 1.22+, client_golang (Prometheus), client-go (Kubernetes).
- Frontend: React, Vite, Tailwind CSS, TypeScript.
- Containerization: Docker.

## Coding Guidelines
- **Go:** Write idiomatic Go. Use standard error handling. Favor standard library where possible, except for Prometheus and K8s integrations. Always write table-driven tests for parsing logic.
- **React:** Use functional components and hooks. Use Tailwind for all styling.
- **Kubernetes:** Simulators should be deployed with minimal privileges. Configuration (metric dumps, mutation rules) should be passed to simulators via ConfigMaps.
- **Agent Rules:** Before committing K8s code, ensure RBAC rules (ServiceAccounts, Roles) are updated if the controller needs new permissions.

## Build & Test Commands
- **Run Go tests:** `go test ./... -v`
- **Build Simulator:** `go build -o bin/simulator ./cmd/simulator`
- **Build Controller:** `go build -o bin/controller ./cmd/controller`
- **Run UI locally:** `cd ui && npm run dev`
- **Lint Go:** `golangci-lint run`
