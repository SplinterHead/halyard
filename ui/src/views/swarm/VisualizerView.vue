<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">Swarm Visualizer</h1>
        <p class="page-subtitle">Real-time cluster topology and container distribution across nodes.</p>
      </div>
      <div class="header-actions">
        <input
          v-model="searchQuery"
          type="search"
          class="search-input"
          placeholder="Filter containers, services, stacks..."
          aria-label="Filter topology"
        />
        <button type="button" class="btn btn-icon" :disabled="loading" aria-label="Refresh" @click="loadData">
          <v-icon size="16">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <!-- Filter Toolbar -->
    <div class="flex-between" style="flex-wrap: wrap; gap: 12px;">
      <div class="flex-row" style="flex-wrap: wrap; gap: 12px;">
        <div class="segmented-control" role="group" aria-label="Filter by role">
          <button type="button" class="segmented-btn" :aria-pressed="roleFilter === 'all'" @click="roleFilter = 'all'">All Nodes</button>
          <button type="button" class="segmented-btn" :aria-pressed="roleFilter === 'manager'" @click="roleFilter = 'manager'">Managers</button>
          <button type="button" class="segmented-btn" :aria-pressed="roleFilter === 'worker'" @click="roleFilter = 'worker'">Workers</button>
        </div>

        <div class="segmented-control" role="group" aria-label="Filter by container state">
          <button type="button" class="segmented-btn" :aria-pressed="containerFilter === 'all'" @click="containerFilter = 'all'">All</button>
          <button type="button" class="segmented-btn" :aria-pressed="containerFilter === 'running'" @click="containerFilter = 'running'">Running</button>
        </div>
      </div>

      <div class="flex-row" style="gap: 8px;">
        <span class="text-3" style="font-size: 12px;">Sort:</span>
        <div class="segmented-control" role="group" aria-label="Sort nodes">
          <button type="button" class="segmented-btn" :aria-pressed="sortBy === 'hostname'" @click="sortBy = 'hostname'">Name</button>
          <button type="button" class="segmented-btn" :aria-pressed="sortBy === 'ip'" @click="sortBy = 'ip'">IP</button>
        </div>
      </div>
    </div>

    <Loader :loading="loading" />

    <!-- Empty State -->
    <div v-if="filteredNodes.length === 0 && !loading" class="surface-card card-body text-center" style="padding: 48px 24px;">
      <v-icon size="42" class="text-3 mb-3">mdi-lan-disconnect</v-icon>
      <div class="card-title" style="font-size: 16px;">No nodes match filters</div>
      <p class="page-subtitle" style="margin-block-start: 6px;">Try adjusting your filters or search query.</p>
    </div>

    <!-- Topology Columns -->
    <div v-else class="visualizer-columns">
      <section
        v-for="node in filteredNodes"
        :key="node.node_id"
        class="surface-card node-column"
      >
        <div
          class="card-header"
          style="cursor: pointer; flex-direction: column; align-items: stretch; gap: 10px;"
          role="button"
          tabindex="0"
          @click="goToNodeDetail(node)"
          @keydown.enter="goToNodeDetail(node)"
        >
          <div class="flex-between">
            <div class="flex-row" style="min-width: 0;">
              <span class="status-dot" :class="node.status === 'ready' ? 'dot-ok' : 'dot-err'"></span>
              <span class="card-title text-truncate">{{ node.hostname }}</span>
            </div>
            <span class="status-badge">{{ node.role === 'manager' ? 'Manager' : 'Worker' }}</span>
          </div>

          <div class="flex-between" style="font-size: 12px;">
            <span class="mono text-3">{{ node.ip }}</span>
            <span class="count-pill">{{ getContainersForNode(node.hostname).length }}</span>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px;">
            <div>
              <div class="flex-between" style="font-size: 11px; margin-block-end: 3px;">
                <span class="text-3">CPU</span>
                <span class="text-1 fw-500">{{ Math.round(node.cpu_usage || 0) }}%</span>
              </div>
              <div class="telemetry-track">
                <div class="telemetry-fill" :style="{ width: `${Math.min(100, Math.round(node.cpu_usage || 0))}%` }"></div>
              </div>
            </div>
            <div>
              <div class="flex-between" style="font-size: 11px; margin-block-end: 3px;">
                <span class="text-3">RAM</span>
                <span class="text-1 fw-500">{{ getMemPercent(node) }}%</span>
              </div>
              <div class="telemetry-track">
                <div class="telemetry-fill" :style="{ width: `${Math.min(100, getMemPercent(node))}%` }"></div>
              </div>
            </div>
          </div>
        </div>

        <div class="card-body" style="padding: 12px; display: flex; flex-direction: column; gap: 8px;">
          <div v-if="getContainersForNode(node.hostname).length === 0" class="text-center text-3 py-6" style="font-size: 12px;">
            No matching containers
          </div>

          <div
            v-for="container in getContainersForNode(node.hostname)"
            :key="container.id"
            class="nested-box container-tile"
            role="button"
            tabindex="0"
            @click="goToContainerDetail(container)"
            @keydown.enter="goToContainerDetail(container)"
          >
            <div class="flex-between" style="margin-block-end: 4px;">
              <span class="text-2 text-truncate" style="font-size: 11.5px; font-weight: 500;">
                {{ container.stack || 'No Stack' }}
              </span>
              <span class="status-dot" :class="getStateDot(container.state)"></span>
            </div>
            <div class="fw-500 text-1 text-truncate" style="font-size: 13px;" :title="formatNames(container.names)">
              {{ formatNames(container.names) }}
            </div>
            <div class="mono text-3 text-truncate" style="font-size: 11.5px; margin-block-start: 2px;" :title="container.image">
              {{ formatImage(container.image) }}
            </div>
            <div v-if="!container.up_to_date" class="text-warn" style="font-size: 11px; margin-block-start: 4px;">
              Image update available
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import Loader from '../../components/Loader.vue';

const router = useRouter();

interface NodeStats {
  node_id: string;
  hostname: string;
  status: string;
  availability: string;
  role: string;
  version: string;
  ip: string;
  cpu_usage: number;
  memory_usage: number;
  memory_total: number;
  uptime: number;
}

interface Container {
  id: string;
  names: string[];
  image: string;
  state: string;
  status: string;
  node: string;
  service: string;
  stack: string;
  up_to_date: boolean;
  created_at: string;
}

const nodes = ref<NodeStats[]>([]);
const containers = ref<Container[]>([]);
const loading = ref(false);
const searchQuery = ref('');
const roleFilter = ref<'all' | 'manager' | 'worker'>('all');
const containerFilter = ref<'all' | 'running'>('running');
const sortBy = ref<'hostname' | 'ip'>('hostname');

let ws: WebSocket | null = null;
let retryTimer: ReturnType<typeof setTimeout> | null = null;
let unmounted = false;

const loadData = async () => {
  loading.value = true;
  try {
    const [nodesRes, containersRes] = await Promise.all([
      fetch('/api/nodes'),
      fetch('/api/containers'),
    ]);
    if (nodesRes.ok) nodes.value = (await nodesRes.json()) || [];
    if (containersRes.ok) containers.value = (await containersRes.json()) || [];
    setupStatsStream();
  } catch (error) {
    console.error('Failed to load visualizer data:', error);
  } finally {
    loading.value = false;
  }
};

const setupStatsStream = () => {
  if (ws) ws.close();
  if (retryTimer) clearTimeout(retryTimer);
  if (unmounted) return;

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('halyard_token') || '';
  const wsUrl = `${protocol}//${window.location.host}/api/nodes/stream?token=${encodeURIComponent(token)}`;
  ws = new WebSocket(wsUrl);

  ws.onmessage = (event) => {
    try {
      const stats = JSON.parse(event.data);
      const index = nodes.value.findIndex((n) => n.node_id === stats.node_id);
      if (index !== -1) {
        const node = nodes.value[index];
        node.cpu_usage = stats.cpu_usage;
        node.memory_usage = stats.memory_usage;
        node.memory_total = stats.memory_total;
        node.uptime = stats.uptime;
      }
    } catch (e) {
      console.error('Failed to parse real-time stats:', e);
    }
  };

  ws.onclose = () => {
    if (!unmounted) {
      retryTimer = setTimeout(setupStatsStream, 5000);
    }
  };

  ws.onerror = () => {
    ws?.close();
  };
};

const filteredNodes = computed(() => {
  const result = nodes.value.filter((node) => {
    if (roleFilter.value !== 'all' && node.role !== roleFilter.value) {
      return false;
    }
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase();
      const matchesNode = node.hostname.toLowerCase().includes(query) || node.ip.includes(query);
      const hasMatchingContainer = getContainersForNode(node.hostname).length > 0;
      return matchesNode || hasMatchingContainer;
    }
    return true;
  });

  return result.sort((a, b) => {
    if (sortBy.value === 'hostname') {
      return a.hostname.localeCompare(b.hostname);
    }
    const partsA = a.ip.split('.').map(Number);
    const partsB = b.ip.split('.').map(Number);
    if (partsA.length === 4 && partsB.length === 4 && !partsA.some(isNaN) && !partsB.some(isNaN)) {
      for (let i = 0; i < 4; i++) {
        if (partsA[i] !== partsB[i]) return partsA[i] - partsB[i];
      }
    }
    return a.ip.localeCompare(b.ip);
  });
});

const getContainersForNode = (hostname: string) => {
  return containers.value.filter((container) => {
    if (container.node.toLowerCase() !== hostname.toLowerCase()) return false;
    if (containerFilter.value === 'running' && container.state.toLowerCase() !== 'running') {
      return false;
    }
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase();
      return (
        formatNames(container.names).toLowerCase().includes(query) ||
        container.image.toLowerCase().includes(query) ||
        container.service.toLowerCase().includes(query) ||
        (container.stack || '').toLowerCase().includes(query)
      );
    }
    return true;
  });
};

const getMemPercent = (node: NodeStats) => {
  if (!node.memory_total) return 0;
  return Math.round((node.memory_usage / node.memory_total) * 100);
};

const getStateDot = (state: string) => {
  switch ((state || '').toLowerCase()) {
    case 'running':
      return 'dot-ok';
    case 'exited':
      return 'dot-err';
    default:
      return 'dot-warn';
  }
};

const formatImage = (image: string) => {
  if (!image) return '-';
  let clean = image.split('@')[0];
  const slashParts = clean.split('/');
  const lastPart = slashParts[slashParts.length - 1];
  if (lastPart.includes(':')) {
    const colonParts = lastPart.split(':');
    slashParts[slashParts.length - 1] = colonParts.slice(0, -1).join(':');
    clean = slashParts.join('/');
  }
  return clean;
};

const formatNames = (names: string[]) => {
  if (!names || names.length === 0) return '-';
  return names[0].replace(/^\//, '');
};

const goToNodeDetail = (node: NodeStats) => {
  router.push({ name: 'node-detail', params: { id: node.node_id } });
};

const goToContainerDetail = (container: Container) => {
  router.push({
    name: 'container-detail',
    params: { id: container.id },
    query: { node: container.node },
  });
};

onMounted(loadData);

onUnmounted(() => {
  unmounted = true;
  if (retryTimer) clearTimeout(retryTimer);
  if (ws) ws.close();
});
</script>

<style scoped>
.visualizer-columns {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
  gap: 16px;
  align-items: start;
}

.container-tile {
  cursor: pointer;
  transition: border-color 0.15s ease;
}

.container-tile:hover {
  border-color: var(--accent);
}
</style>
