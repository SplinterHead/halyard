<template>
  <DataTablePage
    title="Cluster Nodes"
    subtitle="Connected Docker Swarm cluster nodes and live resource telemetry."
    :items="nodes"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-lan-disconnect"
    empty-title="No nodes detected"
    empty-description="Wait for the Halyard agent to report back from your cluster nodes."
    search-placeholder="Search nodes..."
    :sort-by="[{ key: 'hostname', order: 'asc' }]"
    clickable-rows
    @refresh="fetchNodes"
    @click:row="goToDetail"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" @click="openJoinDialog">
        + Add Node
      </button>
    </template>

    <template #top>
      <AppModal v-model="joinDialog" title="Add Node to Swarm Cluster" max-width="640px">
        <div v-if="loadingTokens" class="py-6 text-center">
          <Loader :loading="true" />
          <span class="text-2" style="font-size: 13px;">Retrieving Swarm Join Tokens...</span>
        </div>

        <div v-else-if="tokenError" class="alert-banner alert-err">
          <span><strong>Failed to fetch tokens:</strong> {{ tokenError }}</span>
        </div>

        <template v-else>
          <p class="text-2" style="font-size: 13px; margin: 0;">
            Execute the appropriate command below directly inside the terminal of the node you wish to join.
          </p>

          <div>
            <div class="flex-between" style="margin-block-end: 6px;">
              <span class="fw-500 text-1" style="font-size: 13px;">Worker Node (Recommended)</span>
              <button type="button" class="btn btn-sm" @click="copyWorker">
                {{ copiedWorker ? 'Copied!' : 'Copy Command' }}
              </button>
            </div>
            <pre class="log-viewport" style="font-size: 11.5px; white-space: pre-wrap; word-break: break-all; margin: 0;">{{ tokens.worker_command }}</pre>
          </div>

          <div>
            <div class="flex-between" style="margin-block-end: 6px;">
              <span class="fw-500 text-1" style="font-size: 13px;">Manager Node (Quorum Member)</span>
              <button type="button" class="btn btn-sm" @click="copyManager">
                {{ copiedManager ? 'Copied!' : 'Copy Command' }}
              </button>
            </div>
            <pre class="log-viewport" style="font-size: 11.5px; white-space: pre-wrap; word-break: break-all; margin: 0;">{{ tokens.manager_command }}</pre>
          </div>

          <div class="nested-box text-2" style="font-size: 12px;">
            Ensure ports <code class="text-1">2377/tcp</code>, <code class="text-1">7946/tcp/udp</code>, and <code class="text-1">4789/udp</code> are open between cluster nodes.
          </div>
        </template>

        <template #footer>
          <button type="button" class="btn" @click="joinDialog = false">Done</button>
        </template>
      </AppModal>
    </template>

    <template #item.hostname="{ item }">
      <div class="flex-row">
        <span class="status-dot" :class="item.status === 'ready' ? 'dot-ok' : 'dot-err'"></span>
        <span class="fw-500 text-1">{{ item.hostname }}</span>
      </div>
    </template>

    <template #item.alerts="{ item }">
      <div class="d-flex align-center justify-end" style="gap: 6px;">
        <span
          v-if="item.pending_updates > 0"
          class="status-badge"
          :title="`${item.pending_updates} package updates available`"
        >
          ↓ {{ item.pending_updates }}
        </span>
        <span
          v-if="item.restart_required"
          class="status-badge text-warn"
          title="Restart required"
        >
          ↻
        </span>
      </div>
    </template>

    <template #item.role="{ value }">
      <span :class="value === 'manager' ? 'text-1 fw-500' : 'text-2'">
        {{ formatRole(value) }}
      </span>
    </template>

    <template #item.cpu_usage="{ value }">
      <div class="telemetry-cell">
        <span>{{ Math.round(value || 0) }}%</span>
        <div class="telemetry-track">
          <div
            class="telemetry-fill"
            :class="getTelemetryClass(value || 0)"
            :style="{ width: `${Math.min(100, Math.round(value || 0))}%` }"
          ></div>
        </div>
      </div>
    </template>

    <template #item.memory_usage="{ item }">
      <div
        class="telemetry-cell"
        :title="`${formatBytes(item.memory_usage)} / ${formatBytes(item.memory_total)}`"
      >
        <span>{{ getMemPercent(item) }}%</span>
        <div class="telemetry-track">
          <div
            class="telemetry-fill"
            :class="getTelemetryClass(getMemPercent(item))"
            :style="{ width: `${Math.min(100, getMemPercent(item))}%` }"
          ></div>
        </div>
      </div>
    </template>

    <template #item.ip="{ value }">
      <span class="mono text-2">{{ value }}</span>
    </template>

    <template #item.version="{ value }">
      <span class="text-2">{{ value }}</span>
    </template>

    <template #item.uptime="{ value }">
      <span class="text-2 text-end d-block">{{ formatUptime(value) }}</span>
    </template>

    <!-- Mobile Curated 2-Row Card -->
    <template #mobile-item="{ item }">
      <div
        class="mobile-item-card"
        role="button"
        tabindex="0"
        @click="goToDetail($event, { item })"
        @keydown.enter="goToDetail($event, { item })"
      >
        <div class="mobile-card-top">
          <div class="flex-row" style="min-width: 0;">
            <span class="status-dot" :class="item.status === 'ready' ? 'dot-ok' : 'dot-err'"></span>
            <span class="fw-500 text-1 text-truncate">{{ item.hostname }}</span>
          </div>
          <div class="flex-row" style="gap: 6px;">
            <span v-if="item.pending_updates > 0" class="status-badge" title="Updates available">
              ↓ {{ item.pending_updates }}
            </span>
            <span v-if="item.restart_required" class="status-badge text-warn" title="Restart required">
              ↻
            </span>
            <span :class="item.role === 'manager' ? 'text-1 fw-500' : 'text-2'" style="font-size: 12.5px;">
              {{ formatRole(item.role) }}
            </span>
          </div>
        </div>
        <div class="mobile-card-bottom">
          <span><span class="text-3">Engine:</span> {{ item.version }}</span>
          <span><span class="text-3">Uptime:</span> {{ formatUptime(item.uptime) }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import DataTablePage from '../../components/DataTablePage.vue';
import AppModal from '../../components/AppModal.vue';
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
  pending_updates: number;
  restart_required: boolean;
}

const nodes = ref<NodeStats[]>([]);
const loading = ref(false);

const joinDialog = ref(false);
const loadingTokens = ref(false);
const tokenError = ref('');
const copiedWorker = ref(false);
const copiedManager = ref(false);
const tokens = ref({
  worker_command: '',
  manager_command: '',
});

const headers = [
  { title: 'Hostname', key: 'hostname', sortable: true, align: 'start' as const },
  { title: '', key: 'alerts', sortable: false, align: 'end' as const, width: '72px' },
  { title: 'Role', key: 'role', sortable: true, width: '100px', align: 'start' as const },
  { title: 'CPU', key: 'cpu_usage', sortable: true, align: 'end' as const, width: '110px' },
  { title: 'Memory', key: 'memory_usage', sortable: true, align: 'end' as const, width: '110px' },
  { title: 'IP', key: 'ip', sortable: true, width: '145px', align: 'start' as const },
  { title: 'Engine', key: 'version', sortable: true, width: '110px', align: 'start' as const },
  { title: 'Uptime', key: 'uptime', sortable: true, width: '110px', align: 'end' as const },
];

const formatRole = (role: string) => {
  if (!role) return 'Worker';
  return role.charAt(0).toUpperCase() + role.slice(1).toLowerCase();
};

const getMemPercent = (item: NodeStats) => {
  if (!item.memory_total) return 0;
  return Math.round((item.memory_usage / item.memory_total) * 100);
};

const getTelemetryClass = (pct: number) => {
  if (pct >= 90) return 'is-err';
  if (pct >= 75) return 'is-warn';
  return '';
};

const openJoinDialog = async () => {
  joinDialog.value = true;
  loadingTokens.value = true;
  tokenError.value = '';
  try {
    const response = await fetch('/api/swarm/tokens');
    if (response.ok) {
      tokens.value = await response.json();
    } else {
      tokenError.value = await response.text();
    }
  } catch (err) {
    tokenError.value = String(err);
  } finally {
    loadingTokens.value = false;
  }
};

const copyWorker = () => {
  navigator.clipboard.writeText(tokens.value.worker_command);
  copiedWorker.value = true;
  setTimeout(() => { copiedWorker.value = false; }, 2000);
};

const copyManager = () => {
  navigator.clipboard.writeText(tokens.value.manager_command);
  copiedManager.value = true;
  setTimeout(() => { copiedManager.value = false; }, 2000);
};

const goToDetail = (_event: any, { item }: any) => {
  router.push({ name: 'node-detail', params: { id: item.node_id } });
};

let ws: WebSocket | null = null;
let retryTimer: ReturnType<typeof setTimeout> | null = null;
let unmounted = false;

const fetchNodes = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/nodes');
    if (response.ok) {
      nodes.value = (await response.json()) || [];
      setupStatsStream();
    }
  } catch (error) {
    console.error('Failed to fetch nodes:', error);
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
        node.pending_updates = stats.pending_updates;
        node.restart_required = stats.restart_required;
      }
    } catch (e) {
      console.error('Failed to parse stats message:', e);
    }
  };

  ws.onclose = () => {
    if (!unmounted) {
      retryTimer = setTimeout(setupStatsStream, 5000);
    }
  };

  ws.onerror = (err) => {
    console.error('Stats stream error:', err);
    ws?.close();
  };
};

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
};

const formatUptime = (seconds: number) => {
  if (!seconds) return '-';
  const days = Math.floor(seconds / (24 * 3600));
  const hours = Math.floor((seconds % (24 * 3600)) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  return `${hours}h ${minutes}m`;
};

onMounted(fetchNodes);

onUnmounted(() => {
  unmounted = true;
  if (retryTimer) clearTimeout(retryTimer);
  if (ws) ws.close();
});
</script>
