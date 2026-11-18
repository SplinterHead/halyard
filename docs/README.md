# Halyard Documentation

Welcome to the Halyard documentation. Instead of mirroring menus button-by-button, these guides are organized around operational workflows across your Docker Swarm cluster.

## Guides

1. **[Getting Started & Deployment](getting-started.md)**
   Manager–Agent architecture, Swarm stack deployment (`deploy/docker-stack.yml`), environment configuration, multi-arch builds (`amd64` / `arm64`), first-run onboarding, and login.

2. **[Cluster Observability & Nodes](cluster-observability.md)**
   Cluster Overview dashboard, cluster-wide resource pruning, Swarm Visualizer topology, adding nodes via Swarm join tokens, 24-hour historical performance charts, node labels, and live OS updates.

3. **[Workloads: Stacks, Services & Containers](workloads.md)**
   Managing Swarm stacks (`Managed` / `External` volumes, task ledgers, volume-aware stack removal), replicated vs. global services, container lifecycle actions, viewport-bounded WebSocket logs, and interactive `xterm.js` shells.

4. **[Storage, Images, Networks & Secrets](storage-and-networking.md)**
   Agent-driven volume type detection, **Volume Explorer** file browsing, deduplicated node image inventories and digest checks, network subnet IP usage tracking, viewable Swarm configs, and write-only Swarm secrets.

5. **[GitOps: Repositories & Stack Syncs](gitops.md)**
   Connecting Git repositories, connection testing, mapping stacks to branches and `.yml` compose paths via the interactive repository browser, pulling sibling files (`.env`), and controlling auto-sync vs. manual reconciliation.

6. **[Settings, Image Polling, Registries & Profile](settings-and-security.md)**
   Unified System Settings (**Appearance**, **GitOps Reconciliation**, **Image Update Polling**), Docker Hub rate-limit meters and registry presets, operator password changes, and custom ANSI log colors.
