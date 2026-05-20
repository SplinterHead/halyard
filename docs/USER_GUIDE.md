# Halyard Platform User Guide

Welcome to the Halyard User Guide! This manual provides detailed instructions on how to navigate, monitor, and manage your Docker Swarm clusters using the Halyard visual control plane.

Whether you are configuring GitOps auto-deployments, inspecting node resources, viewing live WebSocket log streams, or safely cleaning up unused Swarm volumes, this guide outlines the purpose and capabilities of every single page inside Halyard.

---

## 📌 Table of Contents
1. [Dashboard (Home)](#1-dashboard-home)
2. [Stacks & GitOps Deployments](#2-stacks-gitops-deployments)
3. [Swarm Nodes](#3-swarm-nodes)
4. [Services Management](#4-services-management)
5. [Volumes & Node-Local Storage](#5-volumes-node-local-storage)
6. [Containers & WebSocket Log Streaming](#6-containers-websocket-log-streaming)
7. [Networks](#7-networks)
8. [Docker Hub Registries & Pull Quotas](#8-docker-hub-registries-pull-quotas)
9. [Secrets & Configs](#9-secrets--configs)
10. [Git Repositories & Sync Mappings](#10-git-repositories--sync-mappings)
11. [Event Audit Trail](#11-event-audit-trail)

---

## 1. Dashboard (Home)

The **Dashboard** serves as the central control room for your Docker Swarm cluster. It is optimized to give operators a high-level operational pulse at a single glance.

### 📊 What You Can See
* **Active Cluster Pulse**: High-level cluster health, node counts, and error trackers.
* **Key Metrics (Real-time Gauges)**: Streamed live CPU and memory usage statistics aggregated across all online manager and worker nodes in the cluster.
* **Swarm Stacks Overview**: A list of all deployed stacks, active service counts, replica health status, and raw size footprints.
* **Nodes Roll**: Quick overview of online/offline Swarm nodes, IP addresses, system roles, and resource loads.

### ⚙️ Operator Actions
* **Force Metrics Refresh**: Instantly query all agents to update live CPU and memory loads.
* **Quick Navigation**: Click directly on any active node or stack to jump immediately into its deep inspection dashboard.

---

## 2. Stacks & GitOps Deployments

The **Stacks** view lists all active Docker compose stacks running on the cluster. Halyard uses compose stack namespaces (`com.docker.stack.namespace`) to visually group related services together.

### 📊 What You Can See
* **Stack List**: Total number of services defined within the stack, overall replica health, and creation timestamps.
* **Stack Detail View (`/swarm/stacks/:name`)**:
  * **Services Sub-List**: Scale or manage specific services belonging to this stack namespace.
  * **Task Tracker**: Monitor active container tasks, including their individual host node locations and health.
  * **Integrated Compose Viewer**: View the exact Git-reconciled YAML configuration file currently deployed to Swarm.
  * **Configs & Secrets**: Lists all Docker secrets and config files injected into the stack.

### ⚙️ Operator Actions
* **Force GitOps Reconciliation**: Click **Sync Now** inside stack details to force Halyard's GitOps engine to fetch the latest commits and redeploy the compose definition.
* **Remove Stack**: Delete the entire stack namespace from the cluster (terminates all associated services and task containers immediately).

---

## 3. Swarm Nodes

The **Nodes** dashboard monitors the physical and virtual hardware hosting your containers. It manages node assignments, labels, and roles.

### 📊 What You Can See
* **Nodes list**: Monospaced lists of node hostnames, IP addresses, system architecture, OS versions, roles (Leader, Manager, Worker), and load percentages.
* **Node Detail View (`/swarm/nodes/:id`)**:
  * **Uptime & Docker Engine Metadata**: Live system status details.
  * **Apex time-series charts**: Interactive historical graphs representing CPU and memory trends over a rolling 24-hour window.
  * **Active Task List**: Monitor all container replicas currently running on this specific node.

### ⚙️ Operator Actions
* **Label Editor**: Docker Swarm uses custom node labels to coordinate container placement filters. Halyard provides a fully interactive node metadata panel to add, edit, or delete custom label keys and values instantly without shell access.

---

## 4. Services Management

The **Services** view acts as a cluster-wide scale controller. Docker Swarm services run as replicated or global tasks across your hosts.

### 📊 What You Can See
* **Services Grid**: Lists active services, active image tags (cleaned of complex SHA hashes for readability), created/updated timestamps, and replicas counts (e.g. `3/3`).
* **Service Detail View (`/swarm/services/:id`)**:
  * **Task Containers**: View individual task states and where they reside in the overlay topology.
  * **Injected Environment Variables**: Review environmental configuration keys safely.

### ⚙️ Operator Actions
* **Scale Replica Counts**: Adjust active container counts up or down dynamically.
* **Delete Service**: Safely remove a service from the cluster via standard confirmation dialogs.

> [!WARNING]
> Deleting a service terminates all running container tasks representing that service across every manager and worker node.

---

## 5. Volumes & Node-Local Storage

Halyard offers deep observability into storage mounts, which is notoriously difficult to retrieve globally in raw Docker Swarm setups.

### 📊 What You Can See
* **Unified Volume Inspector**: View local volumes across all nodes. Displays node location, host directory mount paths, volume driver (local, NFS, GlusterFS), and "In Use" status indicators.

### ⚙️ Operator Actions
* **Delete Volume**: Safely delete individual unused volumes. Clicking delete triggers a routed API query from the Manager to the specific host node's Agent to delete the local storage directory.

> [!IMPORTANT]
> Volume deletion is destructive and permanent. To protect production workloads, Halyard checks if the volume is mounted to any active container and displays safety alerts, utilizing Docker Engine's lock safety checks.

---

## 6. Containers & WebSocket Log Streaming

The **Containers** view lists low-level docker tasks currently deployed across your cluster hosts.

### 📊 What You Can See
* **Containers list**: Active task state (`running`, `exited`, `paused`), mapped internal IPs, and parent service relationships.
* **Container Detail View (`/swarm/containers/:id`)**:
  * **Live Websocket Log Streamer**: Streams stdout/stderr from the container directly into a high-performance terminal emulator inside the browser.
  * **Resource Metrics**: Monitored live container resource consumption.

### ⚙️ Operator Actions
* **Live Logging Controls**: Search log history, toggle autoscroll, and clear local logs cache instantly.
* **Terminal Stream Access**: Pause or resume WebSocket connections dynamically to inspect traces without losing buffer history.

---

## 7. Networks

The **Networks** page handles internal cluster overlay, bridge, and host networking.

### 📊 What You Can See
* **Networks Registry**: Lists active Docker networks, driver types (`overlay`, `bridge`, `macvlan`), network scope, and subnets.
* **Network Detail View (`/swarm/networks/:id`)**:
  * **Connected Containers**: View every task currently attached to this network overlay, along with allocated IP addresses.

### ⚙️ Operator Actions
* **Orphan Cleanup**: Delete unused networks from nodes using confirmation dialogs.

---

## 8. Docker Hub Registries & Pull Quotas

Halyard tracks your Docker Hub container image download quotas to protect your cluster from unexpected pull failures.

### 📊 What You Can See
* **Linear Quota Gauges**: Real-time progress meters showing remaining anonymous or authenticated pull limits.
* **Credentials Manager**: Connect private registries or input Docker Hub accounts.
* **Anonymous Banner alert**: Proactive banner displaying public IP pull status with one-click buttons to quickly add registry credentials.

### ⚙️ Operator Actions
* **Force Quota Refresh**: Refresh limit headers from Docker Registry APIs.
* **Configure Private Registries**: Safely register customized authentication endpoints.

---

## 9. Secrets & Configs

Securely manage sensitive keys and runtime configurations.

### 📊 What You Can See
* **Credentials Ledger**: Lists all cluster-wide injected Swarm secrets and plain configs (`com.docker.swarm.config`).

### ⚙️ Operator Actions
* **Visual Secret Creator**: Input sensitive keys directly from the UI without echoing them into host shell history files.
* **Secure Deletion**: Purge obsolete configs or keys.

---

## 10. Git Repositories & Sync Mappings

This view is the engine room of Halyard's **GitOps Git-to-Cluster integration**.

### 📊 What You Can See
* **Git Repositories View (`/git/repositories`)**: Manage connected remote git repositories, SSH deployment key setups, and connection statuses.
* **Git Syncs View (`/git/syncs`)**: Main ledger mapping repositories to Swarm stacks. Shows Git URL, targeted branch, compose file path, sync interval, and last reconciliation status.

### ⚙️ Operator Actions
* **Map GitSync**: Connect a repository and branch to a new stack. Set automatic reconciliation polling intervals.
* **Auto-Sync Toggle**: Enable or disable continuous GitOps polling. If disabled, the stack will only update on manual command clicks.

---

## 11. Event Audit Trail

Monitor the history of all environment changes in a unified view.

### 📊 What You Can See
* **Chronological Events**: Logs every GitOps action, manual sync trigger, resource prune, and container deployment event with full commit descriptions and timestamps.

### ⚙️ Operator Actions
* **Filter Logs**: Search event listings to isolate deployment errors.

---

> **Tip**: All data-tables inside the Halyard UI default to a clean **25-items-per-page** pagination structure with unified `36px` control sizes, providing a comfortable visual flow for small to large enterprise Swarm clusters.
