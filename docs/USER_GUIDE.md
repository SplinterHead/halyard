# Halyard Platform User Guide

Welcome to the Halyard User Guide! This manual explains how to navigate, monitor, and manage your Docker Swarm clusters using the Halyard visual control plane.

---

## 📌 Table of Contents
1. [Dashboard (Home)](#1-dashboard-home)
2. [Swarm Visualizer](#2-swarm-visualizer)
3. [Stacks](#3-stacks)
4. [Swarm Nodes & OS Updates](#4-swarm-nodes--os-updates)
5. [Services Management](#5-services-management)
6. [Containers, Live Logs & Interactive Terminal](#6-containers-live-logs--interactive-terminal)
7. [Volumes, File Browser & Node-Local Storage](#7-volumes-file-browser--node-local-storage)
8. [Node Images & Digest Checks](#8-node-images--digest-checks)
9. [Networks & IP Usage](#9-networks--ip-usage)
10. [Docker Registries & Pull Quotas](#10-docker-registries--pull-quotas)
11. [Configs & Secrets](#11-configs--secrets)
12. [Git Repositories & Sync Mappings](#12-git-repositories--sync-mappings)
13. [System Settings, Themes & Automated Image Polling](#13-system-settings-themes--automated-image-polling)
14. [User Profile & Log Colors](#14-user-profile--log-colors)

---

## 1. Dashboard (Home)

The **Dashboard** (`/`) serves as the central control room for your Docker Swarm cluster.

### 📊 What You Can See
* **Cluster Health Header**: Overall cluster status (`All cluster systems healthy` or `Cluster issues require attention`) plus node, stack, and active service totals.
* **Summary Metric Cards**: Clickable KPI cards for **Nodes** (managers, workers, offline), **Stacks** (healthy vs. issues), **Replicas** (running vs. stopped), and **Cluster Memory / CPU** averages.
* **Active Workloads**: Prioritized list of deployed stacks showing unhealthy stacks first, Git compose file paths, service names, running/total task counts, short Git commit SHAs, and update timestamps.
* **Swarm Nodes Carousel**: Quick cards for connected nodes with live CPU %, RAM usage, and IP addresses (paginated in groups of 3).
* **GitOps Synchronization Bar**: Cluster-wide summary of how many stacks are up to date with remote Git branches or have drift detected.

### ⚙️ Operator Actions
* **Prune Swarm**: Open the cluster cleanup modal with toggle switches for **Unused Containers**, **Unused Networks**, **Dangling Images**, **All Unused Images (`-a`)**, and **Unused Volumes**.
* **Refresh**: Re-query the manager and node agents for current cluster telemetry.

---

## 2. Swarm Visualizer

The **Visualizer** (`/swarm/visualizer`) renders a live column-per-node topology of running and stopped containers across the Swarm.

### 📊 What You Can See
* **Node Columns**: Node status, hostname, Manager/Worker badge, IP address, container count, and CPU/RAM utilization bars. Click any node header to open **Node Detail**.
* **Container Tiles**: Stack label, container state dot, container name, image name, and an **Image update available** indicator when an image digest update is pending. Click any tile to open **Container Detail**.

### ⚙️ Operator Actions
* **Filter & Sort**: Filter in real time by search text, node role (**All Nodes**, **Managers**, **Workers**), container state (**All**, **Running**), and sort node columns by **Name** or **IP**.

---

## 3. Stacks

The **Stacks** (`/swarm/stacks`) view groups orchestrated Docker Swarm services by `com.docker.stack.namespace`, with unstacked services grouped under `Orphaned/Manual`.

### 📊 What You Can See
* **Stack List**: Health status dot (`Healthy` green vs. degraded amber), total service count, and latest update timestamp.
* **Stack Detail View (`/swarm/stacks/:name`)**:
  * **Services**: Constituent services with images and `running / replicas` badges.
  * **Volumes**: Referenced volumes marked **Managed** or **External**.
  * **Containers**: Associated containers with host node, runtime status, and state badges.
  * **Cluster Tasks**: Searchable, paginated task ledger (`25` items per page) tracking current **State**, **Desired** state, **Error / Status**, and **Updated** time.

### ⚙️ Operator Actions
* **Force Restart / Stop / Start / Rollback Spec**: Control stack services from the **Actions** menu on Stack Detail or restart directly from the Stacks table.
* **Remove Stack**: Delete all services in the stack namespace, with an optional **Delete associated volumes** toggle switch to permanently delete stack volumes.

---

## 4. Swarm Nodes & OS Updates

The **Nodes** (`/swarm/nodes`) dashboard monitors physical and virtual cluster hosts via Halyard's global node agents.

### 📊 What You Can See
* **Nodes Table**: Hostname, status dot, alert badges for pending OS package updates (`↓ N`) and pending restarts (`↻`), Swarm role, live CPU/Memory bars, IP, Docker Engine version, and Uptime.
* **Node Detail View (`/swarm/nodes/:id`)**:
  * **Alert Banners**: Notification banners for **Restart Required** and **Pending Package Updates**.
  * **System Information**: OS, architecture, CPU cores, physical memory, Docker Engine version, and IP address.
  * **Historical Performance (Last 24h)**: Interactive ApexCharts time-series graphs for CPU Usage (%) and Memory Usage (GB).

### ⚙️ Operator Actions
* **+ Add Node**: Fetch live Swarm join tokens and copy `docker swarm join` commands for worker or manager nodes.
* **Swarm Controls**: **Drain** or **Activate** node scheduling, **Promote** or **Demote** Swarm role, and **Reboot** (with **Reboot Immediately** or **Drain & Reboot**).
* **Run System Updates**: Inspect pending `apt` packages in an `80vh` scrollable modal and stream live upgrade progress via Server-Sent Events (SSE), followed by an immediate rescan.
* **Node Labels**: Add or remove custom Swarm placement labels directly from the UI.

---

## 5. Services Management

The **Services** (`/swarm/services`) view lists all replicated and global Swarm services across the cluster.

### 📊 What You Can See
* **Services Table**: Status dot, service name, stack subtitle, scheduling mode icon (globe for `global`, grid for `replicated`) paired with the `running / replicas` badge, formatted image, published ports, and updated timestamp.
* **Service Detail View (`/swarm/services/:id`)**:
  * **Runtime Configuration**: Split **Image** (`image:tag`) and **SHA** (`sha256:...`) fields, **Restart Policy**, and **Placement Constraints**.
  * **Networking**: Published Swarm ingress/host ports.
  * **Environment & Labels**: Aligned key/value tables of environment entries and labels.

### ⚙️ Operator Actions
* **Lifecycle Menu (`Actions`)**: **Force Restart**, **Stop (Scale to 0)**, **Start Service**, **Rollback Spec**, **Scale Service** (replicated services only), and **Delete Service**.

> [!WARNING]
> Deleting a service terminates all running container tasks representing that service across every manager and worker node.

---

## 6. Containers, Live Logs & Interactive Terminal

The **Containers** (`/swarm/containers`) view lists all active and exited containers discovered across Swarm nodes.

### 📊 What You Can See
* **Containers Table**: State dot, container name, image, update badge (**Current** vs. **Out of Date**), Service/Stack, status string, node hostname, and creation time.
* **Container Detail View (`/swarm/containers/:id`)**:
  * **Overview Tab**: Service/Stack/Node/Created metadata, split **Image** and **SHA** fields, Networking (attached networks & exposed ports), Mounts table, Environment, and Labels.
  * **Logs Tab**: Viewport-bounded WebSocket stdout/stderr log streamer with **Line Numbers** and **Timestamps** toggle switches, **Auto-scroll**, **Clear**, and customizable ANSI color rendering.
  * **Terminal Tab**: Viewport-bounded interactive shell (`xterm.js`) over WebSocket with selectable shell (`/bin/bash`, `/bin/sh`, `/bin/zsh`, `/bin/ash`) and **Connect / Disconnect** control.

### ⚙️ Operator Actions
* **Lifecycle Controls**: **Start**, **Stop**, **Restart**, and **Delete** (with a **Force delete** toggle switch to kill running containers).

---

## 7. Volumes, File Browser & Node-Local Storage

Halyard routes queries through node agents to surface node-local volume metadata across the Swarm (`/swarm/volumes`).

### 📊 What You Can See
* **Unified Volume Inspector**: Volume name, **In Use** green dot indicator, host node, owning stack badge, driver (`local`, third-party drivers), detected mount type (`local`, `nfs`, `bind`, `gluster`), and creation timestamp.

### ⚙️ Operator Actions
* **Browse Volume Files**: Open the interactive **Volume Explorer** modal for any volume to traverse directories via breadcrumbs, filter filenames, and inspect file sizes, modification dates, and permissions.
* **Delete Volume / Prune Unused**: Remove an individual unused volume on its host node or prune all unattached volumes cluster-wide.

> [!IMPORTANT]
> Volume deletion is permanent. Docker Engine prevents deletion of volumes mounted by active containers unless those containers are removed first.

---

## 8. Node Images & Digest Checks

The **Images** (`/swarm/images`) view lists container images stored on every node in the cluster (deduplicated per node, repository, and tag).

### 📊 What You Can See
* **Image Inventory**: Remote digest status dot (`Up to date`, `Out of date`, `Checking`, `Unchecked`), repository & tag, 12-char SHA, CPU architecture badge, host node, formatted size, and `In Use` / `Unused` status.

### ⚙️ Operator Actions
* **Check All Updates**: Compare local image digests against remote registries using your configured registry credentials.
* **Delete Image**: Remove a specific image tag from a node, with an optional **Force delete image** (`docker rmi -f`) toggle switch.

---

## 9. Networks & IP Usage

The **Networks** (`/swarm/networks`) page manages Swarm overlay, bridge, and host networks.

### 📊 What You Can See
* **Networks Table**: Network name, owning stack (`System` for default/manual networks), **Internal** and **Attachable** flags, **Subnet**, live **IP Usage** meter (`used/total` + percentage bar), driver, scope, and creation time.
* **Network Detail View (`/swarm/networks/:id`)**: **IPAM Configuration** (Subnet & Gateway), **Properties** (Internal, Attachable, Ingress), **Metadata** (Created At & Labels), and a searchable **Connected Containers** table showing IPv4/IPv6 addresses and clickable links to Container Detail.

### ⚙️ Operator Actions
* **+ Create Network**: Create `overlay` or `bridge` networks with **Attachable**, **Internal**, optional CIDR **Subnet** & **Gateway**, and optional **Enable Encryption (IPSEC)** toggles.
* **Delete Network**: Remove unused networks from the cluster.

---

## 10. Docker Registries & Pull Quotas

The **Docker Registries** (`/swarm/registries`) view manages registry credentials and tracks Docker Hub pull limits.

### 📊 What You Can See
* **Anonymous Rate-Limit Banner**: Displays remaining anonymous Docker Hub pulls and reset time when no Docker Hub credential is saved, with an **Authenticate Docker Hub** quick action.
* **Registry Cards**: Lists saved credentials with quick presets (`Docker Hub`, `GitHub Container Registry`, `GitLab Container Registry`, `AWS ECR`, `Google Artifact Registry`, `Google GCR`, `Red Hat Quay`, `Azure ACR`).
* **Docker Hub Limit Meter**: Real-time progress bar on Docker Hub cards showing remaining pull quota, reset countdown, and manual refresh button.

---

## 11. Configs & Secrets

Manage cluster-wide Docker Swarm configs (`/swarm/variables`) and secrets (`/swarm/secrets`).

### 📊 What You Can See & Do
* **Configs (Variables)**: List non-sensitive Swarm configs, create new configs, **click any row to inspect its contents** in a read-only viewer modal, or delete unused configs.
* **Secrets**: Create write-only encrypted Swarm secrets or remove unused secrets. Secret contents are never displayed back in the UI after creation.

---

## 12. Git Repositories & Sync Mappings

Halyard's GitOps engine connects remote Git repositories to Swarm stack deployments.

### 📊 What You Can See & Do
* **Git Repositories (`/git/repositories`)**: Add, edit, delete, and **Test Connection** on remote Git repositories (with optional username and token/password authentication).
* **Git Syncs (`/git/syncs`)**: Map a stack name to a repository, branch, and compose file path:
  * **Browse Repository**: Interactive file picker highlighting `.yml` / `.yaml` files directly on the selected remote branch.
  * **Additional Files**: Toggle pulling sibling files (such as `.env`) from the compose directory.
  * **Auto Sync**: Toggle periodic background Git reconciliation, or trigger **Force Sync Now** on demand.
  * Shows short 7-character **Commit SHA**, sync status (`Ready`, `Syncing`, `Out of Date`), and relative last sync timestamp.

---

## 13. System Settings, Themes & Automated Image Polling

All global system configuration lives in **Settings** (`/settings`).

### 🎨 Appearance
* **Color Scheme**: Switch between **System** (follows OS dark/light preference), **Dark (Deep Slate)**, and **Light (Alabaster)**. Theme toggling is also accessible from the top-right user menu.

### 🔄 GitOps Reconciliation
* **Sync Concurrency**: Limit concurrent Git sync operations (`0`–`20`, where `0` allows unrestricted parallelism).
* **Sync Interval (minutes)**: Configure how frequently Halyard polls repositories for commits when Auto Sync is enabled (`1`–`60` minutes).

### 📦 Image Update Polling
Halyard's background `ImagePoller` periodically scans unique image+platform pairs across the cluster, pulls newer tags onto matching nodes, and optionally performs zero-downtime rolling service updates:
* **Check Interval (hours)**: Set polling frequency (`1`–`72` hours, or `0` to disable scheduled checks).
* **Update Stacks**: Toggle switch (`Auto-Restart Enabled`) controlling whether services using newly pulled images are automatically updated via native Docker Swarm rolling `ServiceUpdate` (clearing stale digest pins).
* **Check Now**: Run an immediate cluster-wide poll and view scanned/updated image counts and restarted service names.

---

## 14. User Profile & Log Colors

Navigate to **Profile Settings** (`/settings/profile`) from the top-right avatar menu:
* **Operator Account**: View your account display name and username.
* **Change Password**: Update your operator login password (minimum 8 characters).
* **Appearance & Log Colors**: Choose your **Color Scheme** and customize all 8 standard ANSI foreground colors rendered inside the Container Detail **Logs** tab.

---

> **Tip**: All paginated data tables default to **25 items per page**, and responsive card layouts automatically activate on mobile viewports alongside the mobile drawer and bottom navigation bar.
