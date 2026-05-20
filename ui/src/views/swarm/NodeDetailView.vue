<template>
  <div class="fill-height d-flex flex-column">
    <!-- Header Section -->
    <div class="pa-4 d-flex align-center">
      <v-btn icon="mdi-arrow-left" @click="$router.back()" variant="text" class="me-2"></v-btn>
      <div>
        <h1 class="text-h4 font-weight-bold">{{ node?.hostname || 'Node Detail' }}</h1>
        <div class="d-flex align-center mt-1">
          <v-chip size="x-small" :color="statusColor" class="text-uppercase">{{ node?.status }}</v-chip>
          <span class="text-caption text-grey ms-2">{{ node?.node_id }}</span>
        </div>
        
        <div class="d-flex align-center mt-2 flex-wrap gap-2">
          <!-- Subtitle Labels -->
          <div v-for="(val, key) in node?.labels" :key="key" class="label-chip-mini label-chip-hover">
            <span class="label-key-mini">{{ key }}</span>
            <span class="label-val-mini">{{ val }}</span>
            <v-btn
              icon="mdi-close-circle"
              size="x-small"
              variant="text"
              color="error"
              class="delete-label-btn"
              @click="confirmDeleteLabel(key)"
            ></v-btn>
          </div>

          <!-- Add Label Button -->
          <v-btn
            size="x-small"
            variant="tonal"
            color="primary"
            prepend-icon="mdi-plus"
            class="rounded-sm px-2"
            height="18"
            style="font-size: 10px; text-transform: none"
            @click="showAddLabel = true"
          >
            Add Label
          </v-btn>
        </div>
      </div>
      <v-spacer></v-spacer>
      <v-chip size="small" variant="tonal" color="primary" prepend-icon="mdi-shield-check" class="text-uppercase font-weight-bold">
        {{ node?.role }}
      </v-chip>
    </div>

    <!-- Confirm Delete Label Dialog -->
    <v-dialog v-model="showConfirmDelete" max-width="400">
      <v-card class="solid-card pa-6">
        <v-card-title class="text-h6 font-weight-bold px-0 pb-4">Delete Label</v-card-title>
        <v-card-text class="pa-0 text-body-1">
          Are you sure you want to delete the label <strong>{{ labelToDelete }}</strong>?
        </v-card-text>
        <v-card-actions class="px-0 pt-6">
          <v-spacer></v-spacer>
          <v-btn variant="text" color="grey" @click="showConfirmDelete = false">Cancel</v-btn>
          <v-btn
            variant="flat"
            color="error"
            :loading="deletingLabel"
            @click="deleteLabel"
          >Delete Label</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Add Label Dialog -->
    <v-dialog v-model="showAddLabel" max-width="400">
      <v-card class="solid-card pa-6">
        <v-card-title class="text-h6 font-weight-bold px-0 pb-4">Add Node Label</v-card-title>
        <v-card-text class="pa-0">
          <v-text-field
            v-model="newLabel.key"
            label="Key"
            variant="outlined"
            density="comfortable"
            color="primary"
            class="mb-2"
            hide-details
            placeholder="e.g. env"
          ></v-text-field>
          <v-text-field
            v-model="newLabel.value"
            label="Value (optional)"
            variant="outlined"
            density="comfortable"
            color="primary"
            hide-details
            placeholder="e.g. production"
          ></v-text-field>
        </v-card-text>
        <v-card-actions class="px-0 pt-6">
          <v-spacer></v-spacer>
          <v-btn variant="text" color="grey" @click="showAddLabel = false">Cancel</v-btn>
          <v-btn
            variant="flat"
            color="primary"
            :loading="savingLabel"
            :disabled="!newLabel.key"
            @click="saveLabel"
          >Add Label</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-divider class="mb-6"></v-divider>

    <v-row v-if="loading && !node" justify="center" class="mt-8">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </v-row>

    <v-container v-else-if="node" fluid class="pa-0 flex-grow-1 overflow-y-auto">
      <v-row class="px-4">
        <!-- Technical Details -->
        <v-col cols="12">
          <v-card class="glass-card pa-4 mb-4" elevation="0">
            <h3 class="text-subtitle-1 font-weight-bold mb-4 d-flex align-center">
              <v-icon size="small" class="me-2">mdi-information-outline</v-icon>
              System Information
            </h3>
            <v-list bg-color="transparent" density="compact" class="d-flex flex-wrap">
              <v-list-item class="px-0 flex-grow-1" style="min-width: 250px">
                <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 100px">Operating System</span></template>
                <v-list-item-title class="text-body-2">{{ node.os }}</v-list-item-title>
              </v-list-item>
              <v-list-item class="px-0 flex-grow-1" style="min-width: 250px">
                <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 100px">Architecture</span></template>
                <v-list-item-title class="text-body-2">{{ node.architecture }}</v-list-item-title>
              </v-list-item>
              <v-list-item class="px-0 flex-grow-1" style="min-width: 250px">
                <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 100px">CPUs</span></template>
                <v-list-item-title class="text-body-2">{{ node.cpus }} Cores</v-list-item-title>
              </v-list-item>
              <v-list-item class="px-0 flex-grow-1" style="min-width: 250px">
                <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 100px">Physical Memory</span></template>
                <v-list-item-title class="text-body-2">{{ formatBytes(node.memory) }}</v-list-item-title>
              </v-list-item>
              <v-list-item class="px-0 flex-grow-1" style="min-width: 250px">
                <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 100px">Docker Version</span></template>
                <v-list-item-title class="text-body-2"><code>{{ node.engine_version }}</code></v-list-item-title>
              </v-list-item>
              <v-list-item class="px-0 flex-grow-1" style="min-width: 250px">
                <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 100px">IP Address</span></template>
                <v-list-item-title class="text-body-2"><code>{{ node.ip }}</code></v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card>
        </v-col>

        <!-- Live Metrics Cards -->
        <v-col cols="12" md="4">
          <v-card class="glass-card h-100 pa-4" elevation="0">
            <div class="d-flex align-center mb-4">
              <v-icon color="primary" class="me-2">mdi-cpu-64-bit</v-icon>
              <span class="text-subtitle-2 font-weight-bold text-grey">CPU Usage</span>
            </div>
            <div class="d-flex align-end justify-space-between">
              <span class="text-h3 font-weight-bold">{{ Math.ceil(node.cpu_usage) }}%</span>
              <v-progress-circular
                :model-value="node.cpu_usage"
                color="primary"
                size="64"
                width="8"
              >
                <v-icon size="small">mdi-percent</v-icon>
              </v-progress-circular>
            </div>
          </v-card>
        </v-col>

        <v-col cols="12" md="4">
          <v-card class="glass-card h-100 pa-4" elevation="0">
            <div class="d-flex align-center mb-4">
              <v-icon color="secondary" class="me-2">mdi-memory</v-icon>
              <span class="text-subtitle-2 font-weight-bold text-grey">Memory Usage</span>
            </div>
            <div class="d-flex align-end justify-space-between">
              <div>
                <div class="text-h3 font-weight-bold">{{ formatBytes(node.memory_usage) }}</div>
                <div class="text-caption text-grey">of {{ formatBytes(node.memory_total) }}</div>
              </div>
              <v-progress-circular
                :model-value="(node.memory_usage / node.memory_total) * 100"
                color="secondary"
                size="64"
                width="8"
              >
                <v-icon size="small">mdi-memory</v-icon>
              </v-progress-circular>
            </div>
          </v-card>
        </v-col>

        <v-col cols="12" md="4">
          <v-card class="glass-card h-100 pa-4" elevation="0">
            <div class="d-flex align-center mb-4">
              <v-icon color="success" class="me-2">mdi-clock-outline</v-icon>
              <span class="text-subtitle-2 font-weight-bold text-grey">Uptime</span>
            </div>
            <div class="text-h3 font-weight-bold mt-2">{{ formatUptime(node.uptime) }}</div>
            <div class="text-caption text-grey mt-1">Since last agent report</div>
          </v-card>
        </v-col>

        <!-- Historical Graphs -->
        <v-col cols="12">
          <node-history-charts ref="historyCharts" :node-id="node.node_id" />
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import NodeHistoryCharts from '../../components/NodeHistoryCharts.vue'

const route = useRoute()
const node = ref<any>(null)
const loading = ref(true)
const historyCharts = ref<any>(null)
let ws: WebSocket | null = null

const showAddLabel = ref(false)
const savingLabel = ref(false)
const newLabel = ref({ key: '', value: '' })

const showConfirmDelete = ref(false)
const deletingLabel = ref(false)
const labelToDelete = ref('')

const confirmDeleteLabel = (key: string) => {
  labelToDelete.value = key
  showConfirmDelete.value = true
}

const deleteLabel = async () => {
  if (!labelToDelete.value) return
  deletingLabel.value = true
  try {
    const response = await fetch('/api/nodes/labels/remove', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: node.value.node_id,
        key: labelToDelete.value
      })
    })

    if (response.ok) {
      if (node.value.labels) {
        delete node.value.labels[labelToDelete.value]
      }
      showConfirmDelete.value = false
      labelToDelete.value = ''
    }
  } catch (err) {
    console.error('Failed to delete label:', err)
  } finally {
    deletingLabel.value = false
  }
}

const saveLabel = async () => {
  if (!newLabel.value.key) return
  savingLabel.value = true
  try {
    const response = await fetch('/api/nodes/labels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: node.value.node_id,
        key: newLabel.value.key,
        value: newLabel.value.value
      })
    })

    if (response.ok) {
      // Update local state and close
      if (!node.value.labels) node.value.labels = {}
      node.value.labels[newLabel.value.key] = newLabel.value.value
      showAddLabel.value = false
      newLabel.value = { key: '', value: '' }
    }
  } catch (err) {
    console.error('Failed to save label:', err)
  } finally {
    savingLabel.value = false
  }
}

const statusColor = computed(() => {
  if (!node.value) return 'grey'
  return node.value.status === 'ready' ? 'success' : 'error'
})

const fetchNodeDetail = async () => {
  loading.value = true
  try {
    const response = await fetch(`/api/nodes/detail?id=${route.params.id}`)
    if (response.ok) {
      node.value = await response.json()
      setupStatsStream()
    }
  } catch (err) {
    console.error('Failed to fetch node detail:', err)
  } finally {
    loading.value = false
  }
}

const setupStatsStream = () => {
  if (ws) ws.close()
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/api/nodes/stream`;
  ws = new WebSocket(wsUrl);

  ws.onmessage = (event) => {
    try {
      const stats = JSON.parse(event.data);
      if (node.value && stats.node_id === node.value.node_id) {
        node.value.cpu_usage = stats.cpu_usage;
        node.value.memory_usage = stats.memory_usage;
        node.value.memory_total = stats.memory_total;
        node.value.uptime = stats.uptime;

        // Update charts in real-time
        if (historyCharts.value) {
          historyCharts.value.addStats(stats);
        }
      }
    } catch (e) {
      console.error('Failed to parse stats:', e);
    }
  };

  ws.onclose = () => {
    if (route.name === 'node-detail') {
      setTimeout(setupStatsStream, 5000);
    }
  };
}

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

onMounted(fetchNodeDetail)
onUnmounted(() => {
  if (ws) ws.close()
})
</script>

<style scoped>
.label-chip-mini {
  display: flex;
  font-size: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 3px;
  overflow: hidden;
  height: 18px;
  transition: all 0.2s ease;
}

.label-chip-hover:hover {
  border-color: rgba(244, 67, 54, 0.4);
}

.delete-label-btn {
  width: 0 !important;
  min-width: 0 !important;
  opacity: 0;
  padding: 0 !important;
  height: 18px !important;
  transition: all 0.2s ease;
  margin-left: -4px;
}

.label-chip-hover:hover .delete-label-btn {
  width: 24px !important;
  opacity: 1;
  margin-left: 0;
}

.label-key-mini {
  background: rgba(139, 92, 246, 0.15);
  color: #A78BFA;
  padding: 0 6px;
  display: flex;
  align-items: center;
  font-weight: bold;
}

.label-val-mini {
  background: rgba(255, 255, 255, 0.03);
  color: #ccc;
  padding: 0 6px;
  display: flex;
  align-items: center;
}

.gap-1 { gap: 4px; }
.gap-2 { gap: 8px; }
</style>
