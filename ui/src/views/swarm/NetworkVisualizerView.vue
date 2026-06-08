<template>
  <div class="fill-height d-flex flex-column visualizer-container">
    <!-- Header -->
    <div class="px-6 py-5 d-flex align-center flex-wrap visualizer-header justify-space-between">
      <div>
        <h1 class="text-h4 font-weight-bold d-flex align-center">
          <v-icon color="primary" class="me-3" size="32">mdi-lan-connect</v-icon>
          Network Topology
        </h1>
        <p class="text-subtitle-2 text-grey-darken-1 mt-1">
          Visualizing Swarm networks, CIDR ranges, and connected containers.
        </p>
      </div>
      
      <!-- Filters and Search -->
      <div class="d-flex align-center filter-row flex-wrap py-1">
        <v-text-field
          v-model="searchQuery"
          prepend-inner-icon="mdi-magnify"
          placeholder="Filter networks, subnets, containers..."
          variant="solo-filled"
          density="compact"
          flat
          hide-details
          rounded="lg"
          class="search-input glass-input"
          style="width: 280px"
        ></v-text-field>

        <v-btn-toggle
          v-model="scopeFilter"
          mandatory
          divided
          variant="outlined"
          density="comfortable"
          color="primary"
          rounded="lg"
          class="glass-toggle"
        >
          <v-btn value="all" size="small">All Scopes</v-btn>
          <v-btn value="swarm" size="small">Swarm (Overlay)</v-btn>
          <v-btn value="local" size="small">Local</v-btn>
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
    <div v-if="filteredNetworks.length === 0 && !loading" class="d-flex flex-column align-center justify-center flex-grow-1">
      <v-icon size="64" color="grey-lighten-1" class="mb-4">mdi-lan-disconnect</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No networks match filters</h3>
      <p class="text-body-2 text-grey mt-1">Try adjusting your filters or search query.</p>
    </div>

    <!-- Visualizer Grid -->
    <div v-else class="columns-wrapper flex-grow-1 pa-4 overflow-x-auto overflow-y-hidden d-flex">
      <div 
        v-for="network in filteredNetworks" 
        :key="network.id"
        class="network-column d-flex flex-column me-6"
        :class="{ 'overlay-column': network.scope === 'swarm' }"
      >
        <!-- Network Card Header -->
        <div class="network-header pa-4 glass-card mb-4 rounded-xl border border-light" @click="goToNetworkDetail(network)">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption font-weight-bold text-uppercase tracking-wider font-mono text-grey-lighten-1 d-flex align-center">
              <v-icon size="14" class="me-1" :color="network.scope === 'swarm' ? 'info' : 'grey'">
                {{ network.scope === 'swarm' ? 'mdi-earth' : 'mdi-laptop' }}
              </v-icon>
              {{ network.scope }}
            </span>
            <v-chip
              size="x-small"
              :color="network.driver === 'overlay' ? 'primary' : 'secondary'"
              variant="flat"
              class="font-weight-bold text-uppercase"
            >
              {{ network.driver }}
            </v-chip>
          </div>

          <h3 class="text-h6 font-weight-bold text-truncate font-mono mb-1 text-white" :title="network.name">
            {{ network.name }}
          </h3>
          <span class="text-caption text-grey-darken-1 font-mono d-block mb-3" :title="network.id">
            ID: {{ network.id.substring(0, 12) }}
          </span>

          <!-- Network Metrics/Details -->
          <div class="d-flex flex-column gap-2 mt-2 pt-2 border-t border-light border-opacity-10">
            <div>
              <div class="d-flex justify-space-between text-caption font-mono text-grey-lighten-2">
                <span>Subnet (CIDR)</span>
                <span class="text-success">{{ network.subnet || 'Auto / None' }}</span>
              </div>
            </div>
            <div>
              <div class="d-flex justify-space-between text-caption font-mono text-grey-lighten-2">
                <span>Gateway</span>
                <span>{{ network.gateway || '-' }}</span>
              </div>
            </div>
            <div v-if="network.node !== 'Swarm' && network.node !== 'Multi-Node'">
              <div class="d-flex justify-space-between text-caption font-mono text-grey-lighten-2">
                <span>Node</span>
                <span>{{ network.node }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Containers inside Network -->
        <div class="containers-list flex-grow-1 overflow-y-auto pe-1 d-flex flex-column gap-3">
          <div class="text-caption font-weight-bold text-grey-darken-1 px-2 d-flex justify-space-between align-center">
            <span>ATTACHED ({{ getContainersForNetwork(network.name).length }})</span>
          </div>

          <div v-if="getContainersForNetwork(network.name).length === 0" class="d-flex flex-column align-center justify-center py-8 glass-card border border-dashed rounded-xl flex-grow-1 opacity-60">
            <v-icon size="32" color="grey-darken-2" class="mb-2">mdi-link-variant-off</v-icon>
            <span class="text-caption text-grey-darken-2 font-mono">No connected containers</span>
          </div>

          <div
            v-for="container in getContainersForNetwork(network.name)"
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
              <span class="text-truncate" style="max-width: 60%">Node: {{ container.node }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Loader from '../../components/Loader.vue'

const router = useRouter()

interface Network {
  id: string
  name: string
  node: string
  driver: string
  scope: string
  subnet: string
  gateway: string
  stack: string
  created_at: string
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
  networks: string[]
  up_to_date: boolean
  created_at: string
}

const networks = ref<Network[]>([])
const containers = ref<Container[]>([])
const loading = ref(false)

const searchQuery = ref('')
const scopeFilter = ref<'all' | 'swarm' | 'local'>('all')

// Fetch API data
const loadData = async () => {
  loading.value = true
  try {
    const [networksRes, containersRes] = await Promise.all([
      fetch('/api/networks'),
      fetch('/api/containers')
    ])
    
    if (networksRes.ok) networks.value = await networksRes.json()
    if (containersRes.ok) containers.value = await containersRes.json()
  } catch (error) {
    console.error('Failed to load visualizer data:', error)
  } finally {
    loading.value = false
  }
}

// Filtering Networks
const filteredNetworks = computed(() => {
  return networks.value.filter(network => {
    // Scope filter
    if (scopeFilter.value !== 'all' && network.scope !== scopeFilter.value) {
      return false
    }
    
    // Search query filter (if query matches network name, subnet, or container, show it)
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      const matchesNetwork = network.name.toLowerCase().includes(query) || 
                             (network.subnet && network.subnet.includes(query)) ||
                             (network.driver && network.driver.includes(query))
      const hasMatchingContainer = getContainersForNetwork(network.name).some(c => 
        formatNames(c.names).toLowerCase().includes(query) ||
        (c.stack && c.stack.toLowerCase().includes(query)) ||
        (c.service && c.service.toLowerCase().includes(query))
      )
      
      return matchesNetwork || hasMatchingContainer
    }
    
    return true
  })
})

// Helper: Get containers mapping to a specific network by name
const getContainersForNetwork = (networkName: string) => {
  return containers.value.filter(container => {
    return container.networks && container.networks.includes(networkName)
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

// Navigation Handlers
const goToNetworkDetail = (network: Network) => {
  router.push({ name: 'network-detail', params: { id: network.id } })
}

const goToContainerDetail = (container: Container) => {
  router.push({
    name: 'container-detail',
    params: { id: container.id },
    query: { node: container.node }
  })
}

onMounted(loadData)
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
  justify-content: left;
}

.network-column {
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

.network-header {
  cursor: pointer;
}

.network-header:hover {
  background: rgba(255, 255, 255, 0.055) !important;
  border-color: rgba(var(--v-theme-primary), 0.35) !important;
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.3) !important;
}

.overlay-column .network-header {
  border-left: 3px solid rgba(var(--v-theme-info), 0.7) !important;
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

.visualizer-header {
  gap: 24px;
}

.filter-row {
  gap: 20px;
}
</style>
