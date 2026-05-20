# <img src="ui/public/logo.png" width="64" height="64" align="center"> Halyard

**Halyard** is a modern, lightweight, and intuitive GitOps & observability platform designed specifically for Docker Swarm clusters. Built from the ground up to give you full visibility and automated deployment capabilities, Halyard hoists your Swarm services, stacks, and nodes into a unified, stunning control center.

With Halyard, you can stop SSH-ing into your manager nodes to deploy stacks, read logs, or prune resources. Instead, manage everything from a centralized, real-time visual dashboard that automatically synchronizes your services with Git.

---

## 🚀 What Halyard Can Do For You

### 🔄 Automated GitOps & Stacks Deployment
Keep your infrastructure in perfect sync with your codebase without manual intervention.
* **Continuous Git Reconciliation**: Point Halyard to a Git repository (GitHub, GitLab, or self-hosted), specify a branch, and let Halyard automatically pull and sync your `compose.yml` definitions to the cluster.
* **Smart Configurations & `.env` Support**: Full capability to pull down complete repositories, ensuring your local assets, environment files, and configurations are ready when deploying complex stacks.
* **One-Click Manual Syncs**: Instantly force an immediate stack reconciliation to deploy hotfixes or latest commits.
* **Deployment Audit Trail**: Access a chronological history of all deployment events, commit SHAs, and sync details in a clean unified feed.

### 📊 Single Pane-of-Glass Observability
Gain complete visual clarity over what is running inside your cluster.
* **Real-Time Node Metrics**: Track live CPU usage, memory utilization, and node status across your entire cluster at once.
* **Historical Performance Trends**: Analyze resource usage trends over the last 24 hours to scale resources properly.
* **Container Log Streaming**: Access low-latency, real-time log streams for any container in the cluster with a single click—no shell command required.
* **Complete Asset Inspector**: Deeply inspect Swarm nodes, active containers, images, networks, volumes, secrets, and configurations.

### 🧹 One-Click Resource Deletion & Swarm Pruning
Reclaim wasted server disk space safely and efficiently.
* **Node-Local Volume Deletion**: Easily remove unused volumes directly from the interface. Halyard handles the background routing to locate the correct node and delete it safely.
* **Service Lifecycle Controls**: Scale, configure, or terminate Swarm services directly from the UI with clear confirmation safety prompts.
* **Cluster-Wide Pruning**: Trigger concurrent cleanups of unused resources (stopped containers, networks, volumes, and dangling images) across all worker nodes simultaneously.

### 🔒 Registry Protection & Rate Limit Meters
Never get rate-limited by registry pull limits again.
* **Registry Rate-Limit Progress Meters**: Proactively monitor your Docker Hub pull rate limit right on your registry cards.
* **Credential-Aware Fetching**: Displays authenticated quota limits if credentials are set, with seamless anonymous limit fallback tracking for public IPs.
* **Authentication Banner Alerts**: Highlights public registry setups with one-click actions to authenticate and secure your pull quotas.

---

## ⚡ Quick Start

Deploy Halyard to your Swarm cluster with a single command:

```bash
docker stack deploy -c deploy/docker-stack.yml halyard
```

Open your browser and navigate to `http://<your-manager-ip>:8080` to start managing your cluster.

### Connecting a Repository
1. Navigate to **Git -> Repositories** and add your repository details.
2. Go to **Git -> Syncs** and create a new sync entry to deploy your stack from Git.

---

## ✨ Built for the Modern Swarm Operator

* **Premium Glassmorphic Interface**: A high-contrast, beautiful dark-mode theme designed specifically for technical operations.
* **Highly Responsive**: Optimized real-time reactive tables that let you monitor hundreds of services without lag.
* **Ultra-Lightweight**: Built on high-performance Go and Vue.js 3, consuming negligible CPU and memory on your host.
* **Zero Complex Setup**: No heavy databases or external state stores required. Simply deploy the stack and go.

---

## 🗺️ Feature Roadmap

- [x] **Automated GitOps Engine**: Continuous sync of stacks from Git.
- [x] **Multi-Node Observability**: Real-time metrics and volume detection.
- [x] **Live Container Logs**: High-performance streaming of logs to the UI.
- [x] **Cluster-Wide Cleanups**: Advanced, granular pruning controls.
- [x] **Registry Limit Gauges**: Live Docker Hub quota metrics.
- [x] **Secret & Config Management**: View and modify secrets and configurations visually.
- [ ] **Webhook Integrations**: Sync instantly when code is pushed.
- [ ] **OIDC / LDAP Authentication**: Secure access control.
- [ ] **Self-Managing Stacks**: Allow Halyard to self-update via its own GitOps pipeline.

---

## ⚖️ License

Distributed under the MIT License. See `LICENSE` for details.

**Built with ❤️ for the Docker Swarm Community.**
