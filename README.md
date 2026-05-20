# <img src="ui/public/logo.png" width="64" height="64" align="center"> Halyard

**Halyard** is a modern, lightweight, and visual GitOps management platform designed specifically for Docker Swarm clusters. Much like a halyard is used to hoist sails, this platform allows you to effortlessly deploy and manage stacks across your Swarm cluster with precision. It bridges the gap between your Git repositories and your Swarm cluster, providing automated reconciliation, deep observability, and a premium dashboard experience.

![Halyard Dashboard](https://raw.githubusercontent.com/lewis-england/halyard/main/ui/public/banner.png) _(Placeholder for your actual screenshot)_

## ✨ Key Features

### 🔄 GitOps Reconciliation

- **Auto-Sync**: Periodically checks your remote Git repositories (GitHub, GitLab, Self-hosted) for new commits and automatically redeploys your stacks.
- **Selective Sync**: Choose between syncing a single `compose.yml` file or pulling the entire repository for complex stacks requiring `.env` files or additional config assets.
- **Manual Overrides**: Force an immediate sync at any time with a single click, fetching the absolute latest SHA from your branch.

### 🧹 Advanced Cluster-Wide Pruning

- **Multi-Node Distribution**: Trigger cleanups across your entire cluster at once. The Manager leverages internal Swarm overlay routing to trigger concurrent resource prunes across all active worker Agent nodes simultaneously.
- **Granular Operator Controls**: A premium, modal operator dialog lets you customize precisely what resources to purge, covering:
  - **Unused Containers**: Removes stopped containers.
  - **Unused Networks**: Removes networks with no associated tasks.
  - **Unused Volumes**: Safely purges unmounted local volumes (highlighted with built-in UI caution safety warnings).
  - **Unused Images**: Clears out untagged dangling images.
- **Deep Image Cleanups (-a)**: A nested transition option to run full unused image prunes, purging all images that do not have at least one active container associated with them.

### 🔒 Docker Hub Rate Limit Quota Tracking

- **Automated Pull Quotas**: Monitors Docker Hub pull rate limits directly from the cluster nodes.
- **Credential-Aware Fetching**: Automatically uses configured registry credentials to pull authenticated quota headers, with an immediate, transparent fallback to anonymous public IP limits if no credentials are configured.
- **Card-Integrated Progress Meters**: Configured Docker Hub registry boxes showcase a custom progress linear meter tracking exact remaining pulls and a manual trigger to force a live rate refresh.
- **Anonymous Alert Banner**: Displays a prominent, high-fidelity alert banner if Docker Hub is unauthenticated, detailing remaining IP pulls and offering a one-click button to immediately pre-fill and authenticate the Docker Hub preset.

### 👁️ Deep Observability & UI Polish

- **Real-time Metrics**: Live CPU, Memory, and Uptime streaming for every node in the cluster via persistent WebSockets.
- **Historical Analytics**: Automated persistence of node resource usage into a local SQLite database for 24-hour performance trend analysis.
- **Interactive Graphs**: Dynamic time-series performance charts (CPU/Memory) integrated directly into node views using ApexCharts.
- **Node Detail Dashboard**: Granular technical inspection of Swarm nodes, including OS details, architecture, Docker engine metadata, and custom label management.
- **Standardized Technical Sizing**: Unified action controls, row tools, and refresh elements globally to a crisp `36px` footprint for clean visual balance.
- **Production Pagination**: All resource tables are standardized to default to **25 items per page**, keeping large-scale Swarm list cleanups accessible.
- **Clean Label Formatting**: Strips digest SHAs and tags from active container and service listings, showing beautiful, human-readable image names.
- **Relocated Card Actions**: Registry cards feature relocated edit and delete controls inside header title append slots, making card lists highly compact and modern.
- **Automated Diagnostics**: Instantly triggers image update inspections immediately upon loading the Images view.
- **Container Log Streaming**: Instant, low-latency log streaming for any container across the entire cluster.
- **Visual Status Indicators**: Stable, non-intrusive status bars across all resource tables providing instant health feedback.
- **Events & Audit Trail**: A dedicated **Events** view provides a chronological history of every deployment, including commit SHAs, timestamps, and full execution logs.

---

## 🏗️ Architecture

Halyard consists of two primary components:

1. **Halyard Manager**: The central control plane, REST API host, SQLite state store, and Vue 3 frontend host.
2. **Halyard Agent**: A lightweight daemon deployed globally (`mode: global`) across every node to stream host-local metrics, volumes, logs, and command prunes.

---

## 🚀 Getting Started

### 1. Deployment

Deploy Halyard to your cluster using the provided stack file:

```bash
docker stack deploy -c deploy/docker-stack.yml halyard
```

### 2. Connect a Repository

1.  Navigate to **Git -> Repositories**.
2.  Add your repository URL and credentials.
3.  Go to **Git -> Syncs** and create a new sync entry for your Stack.

---

## 🗺️ Roadmap

- [x] **GitOps Engine**: Automatic reconciliation of stacks from Git.
- [x] **Observability**: Real-time cluster-wide metrics and volume inspection.
- [x] **Historical Trends**: 24h resource usage analytics.
- [x] **Container Log Streaming**: View live logs for any container directly in the dashboard.
- [x] **Advanced Prune options**: Cluster-wide custom pruning controls.
- [x] **Registry Quotas**: Real-time Docker Hub rate limit tracking.
- [ ] **Webhook Support**: Trigger syncs via GitHub/GitLab webhooks.
- [ ] **Auth & RBAC**: Secure the dashboard with OIDC/LDAP.
- [ ] **Self-Update**: Enable Halyard to manage its own versioning via its GitOps engine.
- [x] **Secret Injection**: Direct management of Docker Secrets and Configs via the UI.

---

## 🛠️ Development

### Prerequisites

- Go 1.22+
- Node.js 20+ (with NPM)
- A running Docker Swarm cluster

### Local Setup

```bash
# Clone the repo
git clone https://github.com/SplinterHead/halyard.git

# Run Manager (API + Static UI)
make build-all
./manager

# Hot-reload UI for development
cd ui && npm run dev
```

---

## ⚖️ License

Distributed under the MIT License. See `LICENSE` for more information.

---

**Built with ❤️ for the Docker Swarm Community.**
