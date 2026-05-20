<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Cluster Nodes</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-plus-box-outline"
        color="primary"
        variant="tonal"
        class="me-2 animate-pulse-glow"
        @click="openJoinDialog"
      >
        Add Node
      </v-btn>
      <v-btn
        icon="mdi-refresh"
        @click="fetchNodes"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>
    
    <v-row v-if="loading" justify="center" class="mt-8">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </v-row>

    <v-row v-else-if="nodes.length === 0" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="64" color="grey-lighten-1" class="mb-4">mdi-lan-disconnect</v-icon>
        <h3 class="text-h5 text-grey-darken-1">No nodes detected</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          Wait for the Halyard agent to report back from your cluster nodes.
        </p>
      </v-col>
    </v-row>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="nodes"
        :loading="loading"
        :sort-by="[{ key: 'hostname', order: 'asc' }]"
        :row-props="getRowProps"
        class="bg-transparent"
        density="comfortable"
        @click:row="goToDetail"
        items-per-page="25"
      >
        <template v-slot:item.hostname="{ value }">
          <code>{{ value }}</code>
        </template>

        <template v-slot:item.role="{ value }">
          <div class="text-center text-caption text-uppercase font-weight-bold" :class="value === 'manager' ? 'text-primary' : 'text-grey'">
            {{ value }}
          </div>
        </template>

        <template v-slot:item.ip="{ value }">
          <code>{{ value }}</code>
        </template>

        <template v-slot:item.cpu_usage="{ value }">
          <div style="width: 120px">
            <v-progress-linear
              :model-value="value"
              color="primary"
              height="18"
              rounded
            >
              <template v-slot:default="{ value }">
                <span class="text-caption font-weight-bold">{{ Math.ceil(value) }}%</span>
              </template>
            </v-progress-linear>
          </div>
        </template>

        <template v-slot:item.memory_usage="{ item }">
          <div class="d-flex flex-column" style="width: 150px">
            <span class="text-caption text-grey-lighten-1">{{ formatBytes(item.memory_usage) }} / {{ formatBytes(item.memory_total) }}</span>
            <v-progress-linear
              :model-value="(item.memory_usage / item.memory_total) * 100"
              color="secondary"
              height="4"
              rounded
            ></v-progress-linear>
          </div>
        </template>

        <template v-slot:item.version="{ value }">
          <code>{{ value }}</code>
        </template>

        <template v-slot:item.uptime="{ value }">
          <span class="text-caption">{{ formatUptime(value) }}</span>
        </template>

      </v-data-table>
    </div>
  </div>

    <!-- Swarm Join Dialog -->
    <v-dialog v-model="joinDialog" max-width="700px">
      <v-card border flat class="bg-surface glass-panel">
        <v-card-title class="pa-6 pb-2 d-flex align-center">
          <v-icon color="primary" class="me-2" size="24">mdi-docker</v-icon>
          <span class="text-h5 font-weight-bold">Add Node to Swarm Cluster</span>
          <v-spacer></v-spacer>
          <v-btn icon="mdi-close" variant="text" size="small" @click="joinDialog = false"></v-btn>
        </v-card-title>
        
        <v-card-text class="pa-6 pt-2">
          <div v-if="loadingTokens" class="d-flex flex-column align-center py-8">
            <v-progress-circular indeterminate color="primary" size="48" class="mb-4"></v-progress-circular>
            <span class="text-body-2 text-grey">Retrieving Swarm Join Tokens...</span>
          </div>

          <div v-else-if="tokenError" class="text-center py-8">
            <v-icon color="error" size="48" class="mb-4">mdi-alert-circle-outline</v-icon>
            <h4 class="text-h6 text-error">Failed to fetch tokens</h4>
            <p class="text-body-2 text-grey mt-1">{{ tokenError }}</p>
          </div>

          <div v-else>
            <p class="text-body-2 text-grey mb-6">
              To expand your Docker Swarm cluster, execute the appropriate command below directly inside the terminal of the node you wish to join.
            </p>

            <!-- Join as Worker -->
            <div class="mb-6">
              <div class="d-flex align-center justify-space-between mb-2">
                <span class="text-subtitle-2 font-weight-bold text-primary">Join as Worker (Recommended)</span>
                <v-btn
                  size="x-small"
                  variant="tonal"
                  :prepend-icon="copiedWorker ? 'mdi-check' : 'mdi-content-copy'"
                  :color="copiedWorker ? 'success' : 'default'"
                  @click="copyWorker"
                >
                  {{ copiedWorker ? 'Copied!' : 'Copy Command' }}
                </v-btn>
              </div>
              <div class="pa-4 bg-black bg-opacity-30 rounded-lg border border-opacity-10 position-relative font-mono text-caption overflow-x-auto text-grey-lighten-2" style="white-space: pre-wrap; font-family: var(--font-mono) !important;">
                {{ tokens.worker_command }}
              </div>
            </div>

            <!-- Join as Manager -->
            <div>
              <div class="d-flex align-center justify-space-between mb-2">
                <span class="text-subtitle-2 font-weight-bold text-secondary">Join as Manager</span>
                <v-btn
                  size="x-small"
                  variant="tonal"
                  :prepend-icon="copiedManager ? 'mdi-check' : 'mdi-content-copy'"
                  :color="copiedManager ? 'success' : 'default'"
                  @click="copyManager"
                >
                  {{ copiedManager ? 'Copied!' : 'Copy Command' }}
                </v-btn>
              </div>
              <div class="pa-4 bg-black bg-opacity-30 rounded-lg border border-opacity-10 position-relative font-mono text-caption overflow-x-auto text-grey-lighten-2" style="white-space: pre-wrap; font-family: var(--font-mono) !important;">
                {{ tokens.manager_command }}
              </div>
            </div>

            <!-- Alert banner -->
            <v-alert
              type="info"
              variant="tonal"
              density="compact"
              class="mt-6 text-caption"
              icon="mdi-information-outline"
            >
              Note: Make sure port <code class="text-primary font-weight-bold">2377/tcp</code> (Swarm clustering), <code class="text-primary font-weight-bold">7946/tcp/udp</code> (Node communication), and <code class="text-primary font-weight-bold">4789/udp</code> (Overlay network) are open on your firewalls.
            </v-alert>
          </div>
        </v-card-text>
      </v-card>
    </v-dialog>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

interface NodeStats {
  node_id: string
  hostname: string
  status: string
  availability: string
  role: string
  version: string
  ip: string
  cpu_usage: number
  memory_usage: number
  memory_total: number
  uptime: number
}

const nodes = ref<NodeStats[]>([])
const loading = ref(false)

const joinDialog = ref(false)
const loadingTokens = ref(false)
const tokenError = ref('')
const copiedWorker = ref(false)
const copiedManager = ref(false)
const tokens = ref({
  worker_command: '',
  manager_command: '',
})

const openJoinDialog = async () => {
  joinDialog.value = true
  loadingTokens.value = true
  tokenError.value = ''
  try {
    const response = await fetch('/api/swarm/tokens')
    if (response.ok) {
      tokens.value = await response.json()
    } else {
      tokenError.value = await response.text()
    }
  } catch (err) {
    tokenError.value = String(err)
  } finally {
    loadingTokens.value = false
  }
}

const copyWorker = () => {
  navigator.clipboard.writeText(tokens.value.worker_command)
  copiedWorker.value = true
  setTimeout(() => { copiedWorker.value = false }, 2000)
}

const copyManager = () => {
  navigator.clipboard.writeText(tokens.value.manager_command)
  copiedManager.value = true
  setTimeout(() => { copiedManager.value = false }, 2000)
}

const goToDetail = (event: any, { item }: any) => {
  router.push({ name: 'node-detail', params: { id: item.node_id } })
}

const getRowProps = ({ item }: any) => {
  return {
    class: `status-bar-row cursor-pointer ${item.status === "ready" ? "status-success" : "status-error"}`,
  };
};

const headers = [
  { title: "Hostname", key: "hostname", sortable: true, align: "start" as const },
  { title: "Role", key: "role", width: "100px", align: "center" as const },
  { title: 'IP', key: 'ip', align: 'start' as const },
  { title: 'CPU', key: 'cpu_usage', align: 'start' as const },
  { title: 'Memory', key: 'memory_usage', align: 'start' as const },
  { title: 'Engine', key: 'version', align: 'start' as const },
  { title: 'Uptime', key: 'uptime', align: 'start' as const },
]

const fetchNodes = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/nodes')
    nodes.value = await response.json()
    // Start streaming once we have the initial list
    setupStatsStream()
  } catch (error) {
    console.error('Failed to fetch nodes:', error)
  } finally {
    loading.value = false
  }
}

let ws: WebSocket | null = null

const setupStatsStream = () => {
  if (ws) ws.close()

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('halyard_token') || '';
  const wsUrl = `${protocol}//${window.location.host}/api/nodes/stream?token=${encodeURIComponent(token)}`;
  ws = new WebSocket(wsUrl);

  ws.onmessage = (event) => {
    try {
      const stats = JSON.parse(event.data);
      const index = nodes.value.findIndex(n => n.node_id === stats.node_id);
      if (index !== -1) {
        // Only update the dynamic stats
        const node = nodes.value[index];
        node.cpu_usage = stats.cpu_usage;
        node.memory_usage = stats.memory_usage;
        node.memory_total = stats.memory_total;
        node.uptime = stats.uptime;
      }
    } catch (e) {
      console.error('Failed to parse stats message:', e);
    }
  };

  ws.onclose = () => {
    console.log('Stats stream closed. Retrying in 5s...');
    setTimeout(setupStatsStream, 5000);
  };

  ws.onerror = (err) => {
    console.error('Stats stream error:', err);
    ws?.close();
  };
};

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const formatUptime = (seconds: number) => {
  if (!seconds) return '-'
  const days = Math.floor(seconds / (24 * 3600))
  const hours = Math.floor((seconds % (24 * 3600)) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${days}d ${hours}h ${minutes}m`
}

onMounted(() => {
  fetchNodes()
})

import { onUnmounted } from 'vue'
onUnmounted(() => {
  if (ws) ws.close()
})
</script>
