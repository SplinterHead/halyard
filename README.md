# <img src="ui/public/logo.png" width="64" height="64" align="center"> Halyard

![Screenshot of Cluster Overview (Home) page](/docs/wiki/screens/03-home.png)

**Halyard** is a modern, lightweight, and intuitive GitOps & observability platform designed specifically for Docker Swarm clusters. Built from the ground up to give you full visibility and automated deployment capabilities, Halyard hoists your Swarm services, stacks, and nodes into a unified control center.

With Halyard, you can stop SSH-ing into your manager nodes to deploy stacks, read logs, browse volume files, or prune resources. Instead, manage everything from a centralized, real-time visual dashboard that automatically synchronizes your services with Git.

---

## 🚀 What Halyard Can Do For You

### 🔄 Automated GitOps & Stacks Deployment
Keep your infrastructure in perfect sync with your codebase without manual intervention.
* **Continuous Git Reconciliation**: Point Halyard to a Git repository (GitHub, GitLab, or self-hosted), browse branches and compose files directly in the UI, and let Halyard automatically pull and sync your compose definitions to the cluster.
* **Smart Configurations & `.env` Support**: Pull additional sibling files alongside your compose file so environment files and local configs are ready when deploying complex stacks, with configurable sync concurrency and intervals.
* **One-Click Manual Syncs & Volume Cleanup**: Trigger immediate stack reconciliations on demand, and optionally remove stack-associated volumes when deleting stacks.

### 📊 Single Pane-of-Glass Observability
Gain complete visual clarity over what is running inside your cluster.
* **Swarm Visualizer & Real-Time Metrics**: Track live CPU, memory, uptime, and container placement across every cluster node at once.
* **Historical Performance Trends**: Analyze node CPU and memory trends over the last 24 hours with interactive ApexCharts.
* **Container Log Streaming & Interactive Terminal**: Stream ANSI-colored WebSocket logs (with customizable ANSI palette, line numbers, and timestamps) or attach an interactive `xterm.js` PTY shell directly from your browser.
* **Complete Asset Inspector**: Deeply inspect Swarm nodes, services, containers, deduplicated node images, networks (with live IP usage meters), volumes (including a node-local **Volume Explorer** file browser), secrets, and viewable configurations.

### 🧹 One-Click Resource Deletion & Swarm Pruning
Reclaim wasted server disk space safely and efficiently.
* **Node-Local Volume Deletion & Inspection**: Browse files inside local volumes and safely remove unused volumes through node agents.
* **Service & Container Lifecycle Controls**: Scale, start, stop, restart, roll back, or delete Swarm services and containers directly from the UI with confirmation prompts.
* **Cluster-Wide Pruning**: Trigger concurrent cleanups of unused containers, networks, dangling/unused images, and unused volumes across cluster nodes.
* **Node OS Updates**: View pending `apt` updates and reboot requirements per node, and stream live package upgrades via SSE directly from Node Detail.

### 🔒 Registry Protection & Rate Limit Meters
Never get rate-limited by registry pull limits again.
* **Registry Rate-Limit Progress Meters**: Monitor your Docker Hub pull rate limit directly on your registry cards.
* **Credential-Aware Fetching**: Displays authenticated quota limits when credentials are set, with anonymous limit fallback tracking for public IPs.
* **Quick Registry Presets**: One-step setup presets for Docker Hub, GHCR, GitLab, AWS ECR, Google Artifact Registry / GCR, Quay, and Azure ACR.

### 📦 Container Image Update Polling & Automated Restarts
Keep your running containers and stacks continuously up to date with upstream image releases.
* **Configurable Update Polling Timer**: Set how often (in hours) Halyard scans each unique image and architecture combination across all Swarm nodes.
* **Deduplicated Multi-Arch Checks**: Inspects remote registries once per unique image+platform pair to preserve rate limits.
* **Automated Node Image Pulling**: Automatically pulls new image revisions to all relevant Swarm nodes running that architecture.
* **Automated Stack Restarts**: When enabled via **Update Stacks**, triggers native Docker Swarm rolling updates (`ServiceUpdate` with stale digest unpinning and registry re-resolution), updating both replicated and global services with zero downtime.

---

## ⚡ Quick Start

Deploy Halyard to your Swarm cluster with a single command:

```bash
docker stack deploy -c deploy/docker-stack.yml halyard
```

Open your browser and navigate to `http://<your-manager-ip>:8080` to create your operator account and start managing your cluster.

### Connecting a Repository
1. Navigate to **GitOps → Repositories** and add your repository details.
2. Go to **GitOps → Git Syncs** and create a new sync entry to deploy your stack from Git.

---

## ✨ Built for the Modern Swarm Operator

* **Hierarchical Surface Interface**: Purpose-built dark (**Deep Slate**) and light (**Alabaster**) themes with system preference matching, monospaced technical identifiers, and responsive desktop/mobile navigation.
* **Highly Responsive**: Real-time tables, paginated task views, and viewport-bounded log/terminal panels designed to fit wide monitors and mobile screens alike.
* **Ultra-Lightweight**: Built on high-performance Go and Vue.js 3 + TypeScript, consuming minimal CPU and memory.
* **Zero Complex Setup**: Embedded SQLite state storage with no external databases required.

---

## 🗺️ Feature Roadmap

- [x] **Automated GitOps Engine**: Continuous sync of stacks from Git with branch & compose file browsing.
- [x] **Multi-Node Observability**: Real-time metrics, Swarm Visualizer, and 24h node history charts.
- [x] **Live Container Logs & Interactive Terminal**: WebSocket log streaming with ANSI color customization and browser PTY shell (`xterm.js`).
- [x] **Cluster-Wide Cleanups & Volume Explorer**: Granular pruning controls and node-local volume file browsing.
- [x] **Registry Limit Gauges**: Live Docker Hub quota metrics and quick presets.
- [x] **Container Image Update Polling**: Automated checks, node pulls, and stack rolling restarts.
- [x] **Network IP Usage & Detail Inspection**: Real-time subnet utilization bars and connected endpoint tables.
- [x] **Secret & Config Management**: View configs and manage secrets visually.
- [x] **Node OS Updates**: Trigger and monitor OS package updates across the cluster.
- [ ] **Webhook Integrations**: Sync instantly when code is pushed.
- [ ] **OIDC / LDAP Authentication**: Secure access control.
- [ ] **Self-Managing Stacks**: Allow Halyard to self-update via its own GitOps pipeline.

---

## 📚 Documentation

* [Platform User Guide](docs/USER_GUIDE.md)
* [Page-by-Page Wiki & Screenshots](docs/wiki)

---

## ⚖️ License

Distributed under the MIT License. See `LICENSE` for details.

**Built with ❤️ for the Docker Swarm Community.**
