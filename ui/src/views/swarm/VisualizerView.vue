<template>
  <div class="fill-height d-flex flex-column visualizer-container">
    <!-- Header -->
    <div class="px-6 py-5 d-flex align-center flex-wrap visualizer-header justify-space-between">
      <div>
        <h1 class="text-h4 font-weight-bold d-flex align-center">
          <v-icon color="primary" class="me-3" size="32">mdi-sitemap-outline</v-icon>
          Swarm Visualizer
        </h1>
        <p class="text-subtitle-2 text-grey-darken-1 mt-1">
          Real-time cluster topology. Visualizing nodes and active container distribution.
        </p>

        <!-- Left-aligned Sort By Option -->
        <div class="sort-container mt-3">
          <span class="sort-label">Sort By:</span>
          <v-btn-toggle
            v-model="sortBy"
            mandatory
            divided
            variant="outlined"
            density="comfortable"
            color="primary"
            rounded="lg"
            class="glass-toggle"
          >
            <v-btn value="hostname" size="small">
              <v-icon size="14" class="me-1">mdi-sort-alphabetical-ascending</v-icon>
              Name
            </v-btn>
            <v-btn value="ip" size="small">
              <v-icon size="14" class="me-1">mdi-ip</v-icon>
              IP
            </v-btn>
          </v-btn-toggle>
        </div>
      </div>
      
      <!-- Filters and Search -->
      <div class="d-flex align-center filter-row flex-wrap py-1">
        <v-text-field
          v-model="searchQuery"
          prepend-inner-icon="mdi-magnify"
          placeholder="Filter containers, services, stacks..."
          variant="solo-filled"
          density="compact"
          flat
          hide-details
          rounded="lg"
          class="search-input glass-input"
          style="width: 280px"
        ></v-text-field>

        <v-btn-toggle
          v-model="roleFilter"
          mandatory
          divided
          variant="outlined"
          density="comfortable"
          color="primary"
          rounded="lg"
          class="glass-toggle"
        >
          <v-btn value="all" size="small">All Nodes</v-btn>
          <v-btn value="manager" size="small">Managers</v-btn>
          <v-btn value="worker" size="small">Workers</v-btn>
        </v-btn-toggle>

        <v-btn-toggle
          v-model="containerFilter"
          mandatory
          divided
          variant="outlined"
          density="comfortable"
          color="primary"
          rounded="lg"
          class="glass-toggle"
        >
          <v-btn value="all" size="small">
            <v-icon size="14" class="me-1">mdi-package-variant-closed</v-icon>
            All
          </v-btn>
          <v-btn value="running" size="small">
            <v-icon size="14" class="me-1">mdi-play-circle-outline</v-icon>
            Running
          </v-btn>
        </v-btn-toggle>

        <v-btn
          icon="mdi-refresh"
          @click="loadData"
          :loading="loading"
          size="small"
          class="refresh-btn glass-btn"
          flat
        ></v-btn>
      </div>
    </div>

    <!-- Main Content Layout -->
    <v-divider class="mx-4 my-2 border-opacity-25"></v-divider>

    <Loader :loading="loading" class="mx-4" />

    <!-- Empty State -->
    <div v-if="filteredNodes.length === 0 && !loading" class="d-flex flex-column align-center justify-center flex-grow-1">
      <v-icon size="64" color="grey-lighten-1" class="mb-4">mdi-lan-disconnect</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No nodes match filters</h3>
      <p class="text-body-2 text-grey mt-1">Try adjusting your filters or search query.</p>
    </div>

    <!-- Visualizer Grid -->
    <div v-else class="columns-wrapper flex-grow-1 pa-4 overflow-x-auto overflow-y-hidden d-flex">
      <div 
        v-for="node in filteredNodes" 
        :key="node.node_id"
        class="node-column d-flex flex-column me-6"
        :class="{ 'manager-column': node.role === 'manager' }"
      >
        <!-- Node Card Header -->
        <div class="node-header pa-4 glass-card mb-4 rounded-xl border border-light" @click="goToNodeDetail(node)">
          <div class="d-flex align-center justify-space-between mb-0">
            <h3 
              class="text-h6 font-weight-bold text-truncate font-mono text-white d-flex align-center" 
              style="max-width: 20ch;"
              :title="node.hostname"
            >
              <v-icon size="18" class="me-2" :color="node.status === 'ready' ? 'success' : 'error'">
                {{ node.status === 'ready' ? 'mdi-checkbox-blank-circle' : 'mdi-alert-circle' }}
              </v-icon>
              {{ node.hostname.length > 20 ? node.hostname.substring(0, 20) + '...' : node.hostname }}
            </h3>
            <v-chip
              size="x-small"
              :color="node.role === 'manager' ? 'primary' : 'grey-lighten-1'"
              variant="flat"
              class="font-weight-bold text-uppercase"
            >
              {{ node.role === 'manager' ? 'Manager' : 'Worker' }}
            </v-chip>
          </div>
          <span class="text-caption text-grey-darken-1 font-mono d-block mb-2" style="font-size: 10px !important;">{{ node.ip }}</span>

          <!-- Real-Time Metrics inside Node Header -->
          <div class="d-flex flex-column gap-2 mt-2">
            <div>
              <div class="d-flex justify-space-between text-caption font-mono mb-1 text-grey-lighten-2">
                <span>CPU Usage</span>
                <span>{{ Math.round(node.cpu_usage || 0) }}%</span>
              </div>
              <v-progress-linear
                :model-value="node.cpu_usage || 0"
                color="primary"
                height="6"
                rounded
                class="glass-progress"
              ></v-progress-linear>
            </div>

            <div>
              <div class="d-flex justify-space-between text-caption font-mono mb-1 text-grey-lighten-2">
                <span>Memory</span>
                <span>{{ formatBytes(node.memory_usage || 0) }} / {{ formatBytes(node.memory_total || 0) }}</span>
              </div>
              <v-progress-linear
                :model-value="node.memory_total ? ((node.memory_usage / node.memory_total) * 100) : 0"
                color="secondary"
                height="6"
                rounded
                class="glass-progress"
              ></v-progress-linear>
            </div>
          </div>
        </div>

        <!-- Containers inside Node -->
        <div class="containers-list flex-grow-1 overflow-y-auto pe-1 d-flex flex-column gap-3">
          <div class="text-caption font-weight-bold text-grey-darken-1 px-2 d-flex justify-space-between align-center">
            <span>CONTAINERS ({{ getContainersForNode(node.hostname).length }})</span>
          </div>

          <div v-if="getContainersForNode(node.hostname).length === 0" class="d-flex flex-column align-center justify-center py-8 glass-card border border-dashed rounded-xl flex-grow-1 opacity-60">
            <v-icon size="32" color="grey-darken-2" class="mb-2">mdi-package-variant-closed-remove</v-icon>
            <span class="text-caption text-grey-darken-2 font-mono">No active containers</span>
          </div>

          <div
            v-for="container in getContainersForNode(node.hostname)"
            :key="container.id"
            class="container-card glass-card rounded-xl border border-light pa-4 cursor-pointer position-relative overflow-hidden"
            :class="`state-${container.state.toLowerCase()}`"
            @click="goToContainerDetail(container)"
          >
            <!-- Background Glow Effect for Running State -->
            <div class="glow-layer"></div>

            <div class="d-flex align-center justify-space-between mb-2">
              <span class="text-caption font-weight-bold font-mono tracking-wide text-primary text-truncate" style="max-width: 70%" :title="container.stack">
                {{ container.stack || 'No Stack' }}
              </span>
              <v-chip
                size="x-small"
                :color="getStateColor(container.state)"
                variant="tonal"
                class="font-weight-bold text-uppercase"
              >
                {{ container.state }}
              </v-chip>
            </div>

            <h4 class="text-body-1 font-weight-bold text-white text-truncate mb-1" :title="formatNames(container.names)">
              {{ formatNames(container.names) }}
            </h4>
            
            <p class="text-caption text-grey-darken-1 text-truncate mb-3" :title="container.image">
              {{ formatImage(container.image) }}
            </p>

            <div class="d-flex align-center justify-space-between text-caption font-mono mt-1 text-grey-lighten-3 pt-2 border-t border-light border-opacity-10">
              <span class="text-truncate" style="max-width: 60%">{{ container.service }}</span>
              <v-icon v-if="!container.up_to_date" color="warning" size="16" title="Container image is out of date">
                mdi-alert-circle-outline
              </v-icon>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import Loader from '../../components/Loader.vue'

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

interface Container {
  id: string
  names: string[]
  image: string
  state: string
  status: string
  node: string
  service: string
  stack: string
  up_to_date: boolean
  created_at: string
}

const nodes = ref<NodeStats[]>([])
const containers = ref<Container[]>([])
const loading = ref(false)

const searchQuery = ref('')
const roleFilter = ref<'all' | 'manager' | 'worker'>('all')
const containerFilter = ref<'all' | 'running'>('running')
const sortBy = ref<'hostname' | 'ip'>('hostname')

// Fetch API data
const loadData = async () => {
  loading.value = true
  try {
    const [nodesRes, containersRes] = await Promise.all([
      fetch('/api/nodes'),
      fetch('/api/containers')
    ])
    
    if (nodesRes.ok) nodes.value = await nodesRes.json()
    if (containersRes.ok) containers.value = await containersRes.json()
    
    setupStatsStream()
  } catch (error) {
    console.error('Failed to load visualizer data:', error)
  } finally {
    loading.value = false
  }
}

// WebSocket connection for real-time node metrics
let ws: WebSocket | null = null

const setupStatsStream = () => {
  if (ws) ws.close()

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('halyard_token') || ''
  const wsUrl = `${protocol}//${window.location.host}/api/nodes/stream?token=${encodeURIComponent(token)}`
  ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    try {
      const stats = JSON.parse(event.data)
      const index = nodes.value.findIndex(n => n.node_id === stats.node_id)
      if (index !== -1) {
        const node = nodes.value[index]
        node.cpu_usage = stats.cpu_usage
        node.memory_usage = stats.memory_usage
        node.memory_total = stats.memory_total
        node.uptime = stats.uptime
      }
    } catch (e) {
      console.error('Failed to parse real-time stats:', e)
    }
  }

  ws.onclose = () => {
    setTimeout(setupStatsStream, 5000)
  }

  ws.onerror = (err) => {
    console.error('Real-time stats stream error:', err)
    ws?.close()
  }
}

// Filtering and Sorting Nodes
const filteredNodes = computed(() => {
  const result = nodes.value.filter(node => {
    // Role filter
    if (roleFilter.value !== 'all' && node.role !== roleFilter.value) {
      return false
    }
    
    // Search query filter (if query matches node hostname or IP, show it)
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      const matchesNode = node.hostname.toLowerCase().includes(query) || node.ip.includes(query)
      const hasMatchingContainer = getContainersForNode(node.hostname).length > 0
      
      return matchesNode || hasMatchingContainer
    }
    
    return true
  })

  // Sort nodes
  return result.sort((a, b) => {
    if (sortBy.value === 'hostname') {
      return a.hostname.localeCompare(b.hostname)
    } else if (sortBy.value === 'ip') {
      // Clean up IP sorting by converting to octets for natural numerical sorting if possible
      const partsA = a.ip.split('.').map(Number)
      const partsB = b.ip.split('.').map(Number)
      if (partsA.length === 4 && partsB.length === 4 && !partsA.some(isNaN) && !partsB.some(isNaN)) {
        for (let i = 0; i < 4; i++) {
          if (partsA[i] !== partsB[i]) {
            return partsA[i] - partsB[i]
          }
        }
      }
      return a.ip.localeCompare(b.ip)
    }
    return 0
  })
})

// Helper: Get containers mapping to a specific node by hostname
const getContainersForNode = (hostname: string) => {
  return containers.value.filter(container => {
    // Match container host node name
    const matchNode = container.node.toLowerCase() === hostname.toLowerCase()
    if (!matchNode) return false

    // Filter by running state if selected
    if (containerFilter.value === 'running' && container.state.toLowerCase() !== 'running') {
      return false
    }

    // Search query matching
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      const containerName = formatNames(container.names).toLowerCase()
      const containerImage = container.image.toLowerCase()
      const containerService = container.service.toLowerCase()
      const containerStack = (container.stack || '').toLowerCase()
      
      return containerName.includes(query) || 
             containerImage.includes(query) || 
             containerService.includes(query) || 
             containerStack.includes(query)
    }

    return true
  })
}

// Formatting helpers
const formatImage = (image: string) => {
  if (!image) return '-'
  let clean = image.split('@')[0]
  const slashParts = clean.split('/')
  const lastPart = slashParts[slashParts.length - 1]
  if (lastPart.includes(':')) {
    const colonParts = lastPart.split(':')
    slashParts[slashParts.length - 1] = colonParts.slice(0, -1).join(':')
    clean = slashParts.join('/')
  }
  return clean
}

const formatNames = (names: string[]) => {
  if (!names || names.length === 0) return '-'
  return names[0].replace(/^\//, '')
}

const getStateColor = (state: string) => {
  switch (state.toLowerCase()) {
    case 'running': return 'success'
    case 'exited': return 'error'
    case 'created': return 'info'
    case 'paused':
    case 'restarting': return 'warning'
    default: return 'grey'
  }
}

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

// Navigation Handlers
const goToNodeDetail = (node: NodeStats) => {
  router.push({ name: 'node-detail', params: { id: node.node_id } })
}

const goToContainerDetail = (container: Container) => {
  router.push({
    name: 'container-detail',
    params: { id: container.id },
    query: { node: container.node }
  })
}

onMounted(loadData)

onUnmounted(() => {
  if (ws) ws.close()
})
</script>

<style scoped>
.visualizer-container {
  overflow: hidden;
}

.columns-wrapper {
  display: flex;
  overflow-x: auto;
  align-items: stretch;
  min-height: 500px;
  justify-content: center;
  justify-content: safe center;
}

.node-column {
  min-width: 320px;
  max-width: 360px;
  width: 320px;
  height: 100%;
}

.containers-list {
  background: rgba(255, 255, 255, 0.015);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.03);
  padding: 8px;
}

/* Glassmorphic Cards styling */
.glass-card {
  background: rgba(255, 255, 255, 0.035) !important;
  backdrop-filter: blur(16px) saturate(140%) !important;
  -webkit-backdrop-filter: blur(16px) saturate(140%) !important;
  border: 1px solid rgba(255, 255, 255, 0.075) !important;
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.2) !important;
}

.node-header {
  cursor: pointer;
}

.node-header:hover {
  background: rgba(255, 255, 255, 0.055) !important;
  border-color: rgba(var(--v-theme-primary), 0.35) !important;
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.3) !important;
}

.manager-column .node-header {
  border-left: 3px solid rgba(var(--v-theme-primary), 0.7) !important;
}

.container-card {
  z-index: 1;
}

.container-card:hover {
  background: rgba(255, 255, 255, 0.065) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
  transform: translateY(-3px) scale(1.02);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.4) !important;
}

/* Glow Effect Layer */
.glow-layer {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  opacity: 0;
  transition: opacity 0.3s ease;
  z-index: -1;
  background: radial-gradient(circle at 80% 20%, rgba(var(--v-theme-primary), 0.08) 0%, transparent 60%);
}

.container-card:hover .glow-layer {
  opacity: 1;
}

/* Dynamic State Styling */
.container-card.state-running:hover {
  border-color: rgba(76, 175, 80, 0.4) !important;
}

.container-card.state-running:hover .glow-layer {
  background: radial-gradient(circle at 80% 20%, rgba(76, 175, 80, 0.12) 0%, transparent 60%);
}

.container-card.state-exited:hover {
  border-color: rgba(244, 67, 54, 0.4) !important;
}

.container-card.state-exited:hover .glow-layer {
  background: radial-gradient(circle at 80% 20%, rgba(244, 67, 54, 0.12) 0%, transparent 60%);
}

/* Glass Inputs & Controls */
.glass-input :deep(.v-field) {
  background: rgba(255, 255, 255, 0.035) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  backdrop-filter: blur(8px) !important;
  border-radius: 12px !important;
  transition: border-color 0.2s ease;
}

.glass-input :deep(.v-field--focused) {
  border-color: rgba(var(--v-theme-primary), 0.5) !important;
}

.glass-toggle {
  background: rgba(255, 255, 255, 0.02) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  backdrop-filter: blur(8px) !important;
}

.glass-toggle :deep(.v-btn) {
  border-color: transparent !important;
}

.glass-btn {
  background: rgba(255, 255, 255, 0.035) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  backdrop-filter: blur(8px) !important;
  transition: all 0.2s ease;
}

.glass-btn:hover {
  background: rgba(255, 255, 255, 0.065) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

.glass-progress :deep(.v-progress-linear__background) {
  background: rgba(255, 255, 255, 0.1) !important;
  opacity: 1 !important;
}

/* Custom Scrollbar for columns */
.columns-wrapper::-webkit-scrollbar {
  height: 8px;
}
.columns-wrapper::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.1);
}
.columns-wrapper::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}
.columns-wrapper::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.15);
}

.containers-list::-webkit-scrollbar {
  width: 4px;
}
.containers-list::-webkit-scrollbar-track {
  background: transparent;
}
.containers-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 2px;
}
.containers-list::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.15);
}

/* Custom Spacing for Visualizer Header & Filters */
.visualizer-header {
  gap: 24px;
}

.filter-row {
  gap: 20px; /* Custom elegant spacing between visualizer filters */
  margin-left: auto;
  justify-content: flex-end;
}

/* Sort Container on Left */
.sort-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sort-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.4);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-family: var(--font-family);
}
</style>
