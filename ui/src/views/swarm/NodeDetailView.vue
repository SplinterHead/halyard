<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <div class="status-line">
          <button type="button" class="section-link" @click="$router.push('/swarm/nodes')">
            ← Nodes
          </button>
        </div>
        <h1 class="page-title">
          <span>{{ node?.hostname || 'Node Detail' }}</span>
          <span v-if="node" class="status-badge">
            <span class="status-dot" :class="node.status === 'ready' ? 'dot-ok' : 'dot-err'"></span>
            {{ formatLabel(node.status) }}
          </span>
          <span v-if="node" class="status-badge">{{ formatLabel(node.role) }}</span>
          <span v-if="node" class="status-badge">
            <span class="status-dot" :class="node.availability === 'active' ? 'dot-ok' : (node.availability === 'drain' ? 'dot-err' : 'dot-warn')"></span>
            {{ formatLabel(node.availability) }}
          </span>
        </h1>
        <p v-if="node" class="page-subtitle mono">{{ node.node_id }}</p>
      </div>

      <div v-if="node" class="header-actions">
        <button
          v-if="node.availability === 'active'"
          type="button"
          class="btn"
          @click="triggerSwarmAction('drain')"
        >
          Drain
        </button>
        <button
          v-else-if="node.availability === 'drain'"
          type="button"
          class="btn btn-primary"
          @click="triggerSwarmAction('activate')"
        >
          Activate
        </button>

        <button
          v-if="node.role === 'worker'"
          type="button"
          class="btn"
          @click="triggerSwarmAction('promote')"
        >
          Promote
        </button>
        <button
          v-else-if="node.role === 'manager'"
          type="button"
          class="btn"
          @click="triggerSwarmAction('demote')"
        >
          Demote
        </button>

        <button type="button" class="btn btn-danger-subtle" @click="rebootDialog.show = true">
          Reboot
        </button>
        <button type="button" class="btn btn-icon" :disabled="loading" aria-label="Refresh" @click="fetchNodeDetail">
          <v-icon size="16">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <Loader :loading="loading && !node" />

    <template v-if="node">
      <!-- Alert Banners -->
      <div v-if="node.restart_required" class="alert-banner alert-warn">
        <span>This node has pending system updates and requires a restart.</span>
        <button type="button" class="btn btn-sm" @click="rebootDialog.show = true">Reboot Now</button>
      </div>

      <div v-if="node.pending_updates > 0" class="alert-banner alert-info">
        <span>
          There are <strong class="fw-600">{{ node.pending_updates }}</strong> package updates available for this node.
        </span>
        <button type="button" class="btn btn-primary btn-sm" @click="openUpdateDialog">Run Updates</button>
      </div>

      <!-- 3 Key Metric Cards -->
      <section class="metrics-row" style="grid-template-columns: repeat(3, minmax(0, 1fr));">
        <div class="metric-box" style="cursor: default;">
          <span class="metric-label">CPU Usage</span>
          <span class="metric-number">{{ Math.round(node.cpu_usage || 0) }}%</span>
          <div class="telemetry-track" style="margin-block-start: 6px;">
            <div
              class="telemetry-fill"
              :class="getTelemetryClass(node.cpu_usage || 0)"
              :style="{ width: `${Math.min(100, Math.round(node.cpu_usage || 0))}%` }"
            ></div>
          </div>
        </div>

        <div class="metric-box" style="cursor: default;">
          <span class="metric-label">Memory Usage</span>
          <span class="metric-number">{{ formatBytes(node.memory_usage) }}</span>
          <span class="metric-meta">of {{ formatBytes(node.memory_total) }} ({{ memPercent }}%)</span>
        </div>

        <div class="metric-box" style="cursor: default;">
          <span class="metric-label">Uptime</span>
          <span class="metric-number">{{ formatUptime(node.uptime) }}</span>
          <span class="metric-meta">Since last reboot</span>
        </div>
      </section>

      <!-- Detail Grid: System Info & Labels -->
      <div class="detail-grid">
        <section class="surface-card col-7" aria-labelledby="node-sys-title">
          <div class="card-header">
            <span id="node-sys-title" class="card-title">System Information</span>
          </div>
          <div class="card-body">
            <div class="config-grid">
              <div class="config-field">
                <span class="config-label">Operating System</span>
                <span class="config-val">{{ node.os || '-' }}</span>
              </div>
              <div class="config-field">
                <span class="config-label">Architecture</span>
                <span class="config-val">{{ node.architecture || '-' }}</span>
              </div>
              <div class="config-field">
                <span class="config-label">CPUs</span>
                <span class="config-val">{{ node.cpus }} Cores</span>
              </div>
              <div class="config-field">
                <span class="config-label">Physical Memory</span>
                <span class="config-val">{{ formatBytes(node.memory) }}</span>
              </div>
              <div class="config-field">
                <span class="config-label">Docker Engine</span>
                <span class="config-val">{{ node.engine_version }}</span>
              </div>
              <div class="config-field">
                <span class="config-label">IP Address</span>
                <span class="config-val mono">{{ node.ip }}</span>
              </div>
            </div>
          </div>
        </section>

        <section class="surface-card col-5" aria-labelledby="node-labels-title">
          <div class="card-header">
            <span id="node-labels-title" class="card-title">Node Labels</span>
            <button type="button" class="btn btn-sm" @click="showAddLabel = true">+ Add Label</button>
          </div>
          <div class="card-body">
            <div v-if="!Object.keys(node.labels || {}).length" class="text-center text-3">
              No custom labels assigned.
            </div>
            <div v-else class="env-list">
              <div v-for="(val, key) in node.labels" :key="key" class="env-row">
                <span class="env-key" :title="String(key)">{{ key }}</span>
                <span class="env-val flex-between">
                  <span class="mono">{{ val || '—' }}</span>
                  <button
                    type="button"
                    class="btn btn-icon btn-sm btn-danger-subtle"
                    title="Remove label"
                    aria-label="Remove label"
                    @click="confirmDeleteLabel(String(key))"
                  >
                    <v-icon size="14">mdi-close</v-icon>
                  </button>
                </span>
              </div>
            </div>
          </div>
        </section>
      </div>

      <!-- Historical Graphs -->
      <node-history-charts
        ref="historyCharts"
        :node-id="node.node_id"
        :total-memory="node.memory_total || node.memory"
      />
    </template>

    <!-- Delete Label Modal -->
    <AppModal v-model="showConfirmDelete" title="Delete Label?" :persistent="deletingLabel">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to delete the label <strong class="text-1 mono">{{ labelToDelete }}</strong>?
      </p>
      <template #footer>
        <button type="button" class="btn" :disabled="deletingLabel" @click="showConfirmDelete = false">Cancel</button>
        <button type="button" class="btn btn-danger" :disabled="deletingLabel" @click="deleteLabel">
          {{ deletingLabel ? 'Deleting...' : 'Delete Label' }}
        </button>
      </template>
    </AppModal>

    <!-- Add Label Modal -->
    <AppModal v-model="showAddLabel" title="Add Node Label" :persistent="savingLabel">
      <div class="form-field">
        <label class="form-label" for="label-key">Key</label>
        <input id="label-key" v-model="newLabel.key" type="text" class="form-input mono" placeholder="e.g. env" />
      </div>
      <div class="form-field">
        <label class="form-label" for="label-val">Value (Optional)</label>
        <input id="label-val" v-model="newLabel.value" type="text" class="form-input mono" placeholder="e.g. production" />
      </div>
      <template #footer>
        <button type="button" class="btn" :disabled="savingLabel" @click="showAddLabel = false">Cancel</button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="savingLabel || !newLabel.key.trim()"
          @click="saveLabel"
        >
          {{ savingLabel ? 'Saving...' : 'Add Label' }}
        </button>
      </template>
    </AppModal>

    <!-- Confirm Swarm Action Modal -->
    <AppModal v-model="confirmDialog.show" :title="confirmDialog.title" :persistent="updatingNode">
      <p class="text-2" style="margin: 0;">{{ confirmDialog.text }}</p>
      <template #footer>
        <button type="button" class="btn" :disabled="updatingNode" @click="confirmDialog.show = false">Cancel</button>
        <button
          type="button"
          class="btn"
          :class="confirmDialog.color === 'error' ? 'btn-danger' : 'btn-primary'"
          :disabled="updatingNode"
          @click="executeSwarmAction"
        >
          {{ updatingNode ? 'Updating...' : confirmDialog.actionText }}
        </button>
      </template>
    </AppModal>

    <!-- Reboot Confirmation Modal -->
    <AppModal v-model="rebootDialog.show" title="Reboot Node" :persistent="isRebootingHost">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to reboot node <strong class="text-1">{{ node?.hostname }}</strong>?
        We recommend draining the node first to safely reschedule its active containers.
      </p>
      <template #footer>
        <button type="button" class="btn" :disabled="isRebootingHost" @click="rebootDialog.show = false">Cancel</button>
        <button
          type="button"
          class="btn btn-danger-subtle"
          :disabled="isRebootingHost"
          @click="executeHostReboot(false)"
        >
          Reboot Immediately
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="isRebootingHost"
          @click="executeHostReboot(true)"
        >
          Drain & Reboot
        </button>
      </template>
    </AppModal>

    <!-- System Updates Modal -->
    <AppModal
      v-model="updateDialog.show"
      title="Run System Updates"
      max-width="520px"
      :persistent="updateDialog.updating"
    >
      <div v-if="updateDialog.error" class="alert-banner alert-err">
        {{ updateDialog.error }}
      </div>

      <div v-if="updateDialog.loading" class="py-6 text-center">
        <Loader :loading="true" />
        <span class="text-2">Fetching pending packages...</span>
      </div>

      <div v-else-if="updateDialog.packages.length === 0" class="text-center text-3 py-4">
        No packages found.
      </div>

      <div v-else class="surface-card" style="max-block-size: 80vh; overflow-y: auto;">
        <div v-for="(pkg, idx) in updateDialog.packages" :key="idx" class="workload-row" style="padding: 10px 14px; cursor: default;">
          <span class="mono text-1">{{ pkg.name }}</span>
          <span class="status-badge">
            <span
              class="status-dot"
              :class="pkg.status === 'success' ? 'dot-ok' : (pkg.status === 'error' ? 'dot-err' : 'dot-warn')"
            ></span>
            {{ pkg.status }}
          </span>
        </div>
      </div>

      <template #footer>
        <button type="button" class="btn" :disabled="updateDialog.updating" @click="closeUpdateDialog">Cancel</button>
        <button
          v-if="!updateDialog.completed"
          type="button"
          class="btn btn-primary"
          :disabled="updateDialog.updating || updateDialog.packages.length === 0"
          @click="confirmUpdate"
        >
          {{ updateDialog.updating ? 'Updating...' : 'Confirm Update' }}
        </button>
        <button v-else type="button" class="btn btn-primary" @click="closeUpdateDialog">Done</button>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRoute } from 'vue-router';
import Loader from '../../components/Loader.vue';
import AppModal from '../../components/AppModal.vue';
import NodeHistoryCharts from '../../components/NodeHistoryCharts.vue';

const route = useRoute();
const node = ref<any>(null);
const loading = ref(true);
const historyCharts = ref<any>(null);
let ws: WebSocket | null = null;
let retryTimer: ReturnType<typeof setTimeout> | null = null;
let unmounted = false;

const showAddLabel = ref(false);
const savingLabel = ref(false);
const newLabel = ref({ key: '', value: '' });

const showConfirmDelete = ref(false);
const deletingLabel = ref(false);
const labelToDelete = ref('');

const updatingNode = ref(false);
const confirmDialog = ref({
  show: false,
  title: '',
  text: '',
  actionText: '',
  color: 'primary',
  payload: {} as any,
});

const isRebootingHost = ref(false);
const rebootDialog = ref({
  show: false,
  draining: false,
});

interface PackageUpdate {
  name: string;
  status: 'pending' | 'success' | 'error';
}

const updateDialog = ref({
  show: false,
  loading: false,
  updating: false,
  completed: false,
  packages: [] as PackageUpdate[],
  error: '',
  eventSource: null as EventSource | null,
});

const memPercent = computed(() => {
  if (!node.value?.memory_total) return 0;
  return Math.round((node.value.memory_usage / node.value.memory_total) * 100);
});

const formatLabel = (val: string) => {
  if (!val) return '';
  return val.charAt(0).toUpperCase() + val.slice(1).toLowerCase();
};

const getTelemetryClass = (pct: number) => {
  if (pct >= 90) return 'is-err';
  if (pct >= 75) return 'is-warn';
  return '';
};

const openUpdateDialog = async () => {
  if (!node.value) return;
  updateDialog.value.show = true;
  updateDialog.value.loading = true;
  updateDialog.value.error = '';
  updateDialog.value.updating = false;
  updateDialog.value.completed = false;
  updateDialog.value.packages = [];

  try {
    const res = await fetch(`/api/nodes/updates/list?id=${node.value.node_id}`);
    if (res.ok) {
      const pkgs = await res.json();
      if (pkgs) {
        updateDialog.value.packages = pkgs.map((p: string) => ({ name: p, status: 'pending' }));
      }
    } else {
      updateDialog.value.error = 'Failed to load package list';
    }
  } catch {
    updateDialog.value.error = 'Failed to load package list';
  } finally {
    updateDialog.value.loading = false;
  }
};

const confirmUpdate = () => {
  if (!node.value) return;
  updateDialog.value.updating = true;
  updateDialog.value.error = '';

  const token = localStorage.getItem('halyard_token') || '';
  const eventSource = new EventSource(
    `/api/nodes/update/stream?id=${node.value.node_id}&token=${encodeURIComponent(token)}`
  );
  updateDialog.value.eventSource = eventSource;

  eventSource.onmessage = (event) => {
    const data = event.data;
    if (data === 'DONE') {
      eventSource.close();
      updateDialog.value.eventSource = null;
      updateDialog.value.updating = false;
      updateDialog.value.completed = true;
      fetchNodeDetail();
      return;
    }

    if (data.startsWith('ERROR:')) {
      updateDialog.value.error = data;
      eventSource.close();
      updateDialog.value.eventSource = null;
      updateDialog.value.updating = false;
      updateDialog.value.packages.forEach((p) => {
        if (p.status === 'pending') p.status = 'error';
      });
      return;
    }

    const match = data.match(/Setting up ([a-zA-Z0-9.\-+]+)/);
    if (match && match[1]) {
      const pkgName = match[1];
      const pkg = updateDialog.value.packages.find((p) => p.name === pkgName || pkgName.startsWith(p.name));
      if (pkg) pkg.status = 'success';
    }
  };

  eventSource.onerror = () => {
    updateDialog.value.error = 'Stream connection lost';
    eventSource.close();
    updateDialog.value.eventSource = null;
    updateDialog.value.updating = false;
    updateDialog.value.packages.forEach((p) => {
      if (p.status === 'pending') p.status = 'error';
    });
  };
};

const closeUpdateDialog = () => {
  if (updateDialog.value.eventSource) {
    updateDialog.value.eventSource.close();
  }
  updateDialog.value.show = false;
};

const executeHostReboot = async (drain: boolean) => {
  if (!node.value) return;
  isRebootingHost.value = true;
  rebootDialog.value.draining = drain;
  try {
    const url = `/api/nodes/host/reboot?id=${node.value.node_id}${drain ? '&drain=true' : ''}`;
    const response = await fetch(url, { method: 'POST' });
    if (response.ok) {
      rebootDialog.value.show = false;
      if (drain) {
        node.value.availability = 'drain';
      }
    }
  } catch (err) {
    console.error('Failed to trigger host reboot:', err);
  } finally {
    isRebootingHost.value = false;
  }
};

const triggerSwarmAction = (action: 'drain' | 'activate' | 'promote' | 'demote') => {
  if (!node.value) return;

  if (action === 'drain') {
    confirmDialog.value = {
      show: true,
      title: 'Drain Node',
      text: `Are you sure you want to drain node "${node.value.hostname}"? Active service replicas will be rescheduled to other nodes.`,
      actionText: 'Drain Node',
      color: 'warning',
      payload: { availability: 'drain' },
    };
  } else if (action === 'activate') {
    confirmDialog.value = {
      show: true,
      title: 'Activate Node',
      text: `Are you sure you want to activate node "${node.value.hostname}"? It will resume accepting Swarm tasks.`,
      actionText: 'Activate Node',
      color: 'success',
      payload: { availability: 'active' },
    };
  } else if (action === 'promote') {
    confirmDialog.value = {
      show: true,
      title: 'Promote to Manager',
      text: `Are you sure you want to promote node "${node.value.hostname}" to a Swarm Manager?`,
      actionText: 'Promote',
      color: 'info',
      payload: { role: 'manager' },
    };
  } else if (action === 'demote') {
    confirmDialog.value = {
      show: true,
      title: 'Demote to Worker',
      text: `Are you sure you want to demote node "${node.value.hostname}" to a Swarm Worker? It will no longer participate in cluster coordination.`,
      actionText: 'Demote',
      color: 'error',
      payload: { role: 'worker' },
    };
  }
};

const executeSwarmAction = async () => {
  const payload = confirmDialog.value.payload;
  if (!payload || !node.value) return;

  updatingNode.value = true;
  try {
    const response = await fetch('/api/nodes/update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: node.value.node_id,
        ...payload,
      }),
    });
    if (response.ok) {
      await fetchNodeDetail();
      confirmDialog.value.show = false;
    }
  } catch (err) {
    console.error('Failed to update node status:', err);
  } finally {
    updatingNode.value = false;
  }
};

const confirmDeleteLabel = (key: string) => {
  labelToDelete.value = key;
  showConfirmDelete.value = true;
};

const deleteLabel = async () => {
  if (!labelToDelete.value) return;
  deletingLabel.value = true;
  try {
    const response = await fetch('/api/nodes/labels/remove', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: node.value.node_id,
        key: labelToDelete.value,
      }),
    });
    if (response.ok) {
      if (node.value.labels) delete node.value.labels[labelToDelete.value];
      showConfirmDelete.value = false;
      labelToDelete.value = '';
    }
  } catch (err) {
    console.error('Failed to delete label:', err);
  } finally {
    deletingLabel.value = false;
  }
};

const saveLabel = async () => {
  if (!newLabel.value.key.trim()) return;
  savingLabel.value = true;
  try {
    const response = await fetch('/api/nodes/labels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: node.value.node_id,
        key: newLabel.value.key.trim(),
        value: newLabel.value.value,
      }),
    });
    if (response.ok) {
      if (!node.value.labels) node.value.labels = {};
      node.value.labels[newLabel.value.key.trim()] = newLabel.value.value;
      showAddLabel.value = false;
      newLabel.value = { key: '', value: '' };
    }
  } catch (err) {
    console.error('Failed to save label:', err);
  } finally {
    savingLabel.value = false;
  }
};

const fetchNodeDetail = async () => {
  loading.value = true;
  try {
    const response = await fetch(`/api/nodes/detail?id=${route.params.id}`);
    if (response.ok) {
      node.value = await response.json();
      setupStatsStream();
    }
  } catch (err) {
    console.error('Failed to fetch node detail:', err);
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
      if (node.value && stats.node_id === node.value.node_id) {
        node.value.cpu_usage = stats.cpu_usage;
        node.value.memory_usage = stats.memory_usage;
        node.value.memory_total = stats.memory_total;
        node.value.uptime = stats.uptime;
        node.value.pending_updates = stats.pending_updates;
        node.value.restart_required = stats.restart_required;
        if (historyCharts.value) {
          historyCharts.value.addStats(stats);
        }
      }
    } catch (e) {
      console.error('Failed to parse stats:', e);
    }
  };

  ws.onclose = () => {
    if (!unmounted && route.name === 'node-detail') {
      retryTimer = setTimeout(setupStatsStream, 5000);
    }
  };
};

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B';
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

onMounted(fetchNodeDetail);

onUnmounted(() => {
  unmounted = true;
  if (retryTimer) clearTimeout(retryTimer);
  if (ws) ws.close();
});
</script>
