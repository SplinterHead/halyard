# Halyard: Docker Swarm GitOps & Observability Platform

Halyard is a modern, lightweight observability and management suite for Docker Swarm clusters. It combines real-time cluster monitoring with a robust GitOps workflow for deploying and syncing stacks.

## 🏗 Architecture

Halyard follows a **Manager-Agent** architecture designed to overcome the distributed nature of Docker Swarm.

### 1. Halyard Manager (Central Brain)
- **Role**: Serves as the primary API gateway and frontend host.
- **Data Aggregation**: Polls or receives data from multiple agents across the cluster.
- **GitOps Engine**: Manages the lifecycle of Swarm stacks by syncing with Git repositories.
- **State Management**: Uses a local SQLite database (`/data/halyard.db`) to track Git repository metadata, sync history, and deployment events.
- **Networking**: Typically deployed as a Swarm service with 1 replica, accessible via port 8080.

### 2. Halyard Agent (Node Observer)
- **Role**: Runs on **every node** in the Swarm cluster (`mode: global`).
- **Local Metrics**: Collects CPU, memory, and disk I/O metrics directly from the host Docker daemon.
- **Volume Insight**: Inspects local volume mounts and usage, identifying NFS/CIFS types and "In Use" status.
- **Log Streaming**: Provides a WebSocket-based log streaming service for containers running on its specific node.
- **API**: Exposes a REST API on port 9090 (internal to the Swarm overlay network).

## 🛠 Technology Stack

### Backend (Go)
- **Docker SDK**: Deep integration with the Docker Engine API for orchestration, volume management, and real-time events.
- **Gorilla WebSocket**: Used for low-latency log streaming from containers to the browser.
- **SQLite**: Embedded database for GitOps state persistence.
- **Multi-Arch Support**: Native builds for `linux/amd64` (Xeon/Intel) and `linux/arm64` (Raspberry Pi).

### Frontend (Vue.js 3)
- **Framework**: Vite + Vue 3 (Composition API) + TypeScript.
- **Styling**: Vuetify 3 with a customized dark theme.
- **State**: Reactive refs for real-time dashboard updates.
- **Design Language**: High-contrast dark mode, monospaced technical identifiers, and glassmorphic micro-animations.

## 📂 Project Structure

```text
.
├── api/                # Shared Go types (the "Contract" between Agent, Manager, and UI)
├── cmd/
│   ├── agent/         # Agent entry point
│   └── manager/       # Manager entry point
├── deploy/            # Dockerfiles and Swarm stack definitions
├── internal/
│   ├── agent/         # Node-local logic (metrics, volume inspection)
│   ├── manager/       # Cluster-wide logic (aggregation, GitOps, DB)
│   └── pkg/           # Shared utilities (Docker client wrappers, log demuxing)
├── ui/                 # Vue.js frontend
└── Makefile           # Multi-arch build and deployment automation
```

## 🧩 Design Decisions & Patterns

### Agent-Manager Data Flow
Docker Swarm's standard API doesn't provide node-local stats or volume mount details for the entire cluster from a single endpoint. Halyard solves this by having the Manager query each Agent's internal IP within the overlay network.

### Volume Detection Logic
The system implements a multi-stage detection for volume types:
1. **Driver Check**: Identifies 3rd-party drivers (Gluster, Portworx).
2. **Options Inspection**: Looks for `type=nfs` or `o=bind` in volume options.
3. **Mountpoint Heuristics**: Secondary check for paths containing "nfs".

### UI/UX Standards
- **Monospacing**: All IDs, Node names, and metrics use monospaced fonts (`Roboto Mono` or `Courier New`) to ensure alignment and technical clarity.
- **Friendly Names**: The UI prioritizes `com.docker.stack.namespace` and `com.docker.compose.volume` labels to present human-readable names over raw hex IDs.
- **Status Consistency**: Unified color palette for status (Success = Green, Warning = Amber, Error = Red, Inactive = Grey).

## 🚀 Deployment

The project is built using a `Makefile` that handles multi-architecture compilation:
- `make build-all`: Builds local architecture images.
- `make build-multi`: Uses `docker buildx` to build for `amd64` and `arm64`.

Stacks are deployed via `deploy/docker-stack.yml`, which mounts the Docker socket to both the Manager and the Agents.

## 📝 For Future Gemini Instances

- **API Extensions**: When adding new metrics or features, always update `api/types.go` first.
- **Agent Handlers**: New node-local features must be implemented in `internal/agent` and exposed via `cmd/agent/main.go`.
- **UI Consistency**: Follow the pattern in `VolumesView.vue` for new list views (pagination, friendly name resolution, and standardized headers).
- **GitOps Logic**: The `GitWorker` in `internal/manager/git_worker.go` is the heartbeat of deployment; ensure all stack operations are idempotent.
