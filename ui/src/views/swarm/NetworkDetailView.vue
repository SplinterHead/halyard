<template>
  <div class="fill-height d-flex flex-column pa-4">
    <!-- Header -->
    <div class="d-flex align-center mb-6" v-if="network">
      <v-btn
        icon="mdi-arrow-left"
        variant="text"
        @click="$router.back()"
        class="me-2"
      ></v-btn>
      <div>
        <h1 class="text-h4 font-weight-bold d-flex align-center">
          Network: {{ network.name }}
          <v-chip
            color="primary"
            size="small"
            label
            class="ms-4 text-uppercase font-weight-bold"
          >
            {{ network.driver }}
          </v-chip>
        </h1>
        <p class="text-subtitle-1 text-grey-lighten-1">
          {{ network.scope }} scope // {{ network.id.substring(0, 12) }}
        </p>
      </div>
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-refresh"
        @click="fetchDetail"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-row v-if="network">
      <!-- Info Cards -->
      <v-col cols="12" md="4">
        <v-card class="glass-card h-100 pa-4">
          <div class="d-flex align-center mb-4">
            <v-icon color="primary" class="me-2">mdi-ip-network</v-icon>
            <span class="text-subtitle-2 font-weight-bold text-grey">IPAM Configuration</span>
          </div>
          <div v-if="network.ipam_configs && network.ipam_configs.length">
            <div v-for="(config, idx) in network.ipam_configs" :key="idx" class="mb-4">
              <div class="text-caption text-grey">Subnet</div>
              <div class="text-h6 font-mono">{{ config.subnet || '-' }}</div>
              <div class="text-caption text-grey mt-2">Gateway</div>
              <div class="text-h6 font-mono">{{ config.gateway || '-' }}</div>
            </div>
          </div>
          <div v-else class="text-body-2 text-grey">No IPAM configuration found.</div>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="glass-card h-100 pa-4">
          <div class="d-flex align-center mb-4">
            <v-icon color="secondary" class="me-2">mdi-cog-outline</v-icon>
            <span class="text-subtitle-2 font-weight-bold text-grey">Properties</span>
          </div>
          <v-list density="compact" bg-color="transparent">
            <v-list-item class="px-0">
              <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 80px">Internal</span></template>
              <v-chip size="x-small" :color="network.internal ? 'warning' : 'grey'">{{ network.internal }}</v-chip>
            </v-list-item>
            <v-list-item class="px-0">
              <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 80px">Attachable</span></template>
              <v-chip size="x-small" :color="network.attachable ? 'success' : 'grey'">{{ network.attachable }}</v-chip>
            </v-list-item>
            <v-list-item class="px-0">
              <template v-slot:prepend><span class="text-caption text-grey me-4" style="width: 80px">Ingress</span></template>
              <v-chip size="x-small" :color="network.ingress ? 'info' : 'grey'">{{ network.ingress }}</v-chip>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="glass-card h-100 pa-4">
          <div class="d-flex align-center mb-4">
            <v-icon color="success" class="me-2">mdi-clock-outline</v-icon>
            <span class="text-subtitle-2 font-weight-bold text-grey">Metadata</span>
          </div>
          <div class="text-caption text-grey">Created At</div>
          <div class="text-body-2 mb-4">{{ new Date(network.created_at).toLocaleString() }}</div>
          
          <div v-if="network.labels && Object.keys(network.labels).length">
            <div class="text-caption text-grey mb-1">Labels</div>
            <div class="d-flex flex-wrap gap-1">
              <v-chip v-for="(v, k) in network.labels" :key="k" size="x-small" variant="tonal" class="font-mono">
                {{ k }}={{ v }}
              </v-chip>
            </div>
          </div>
        </v-card>
      </v-col>

      <!-- Connected Containers -->
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="pa-6 pb-2 d-flex align-center">
            <v-icon size="20" class="me-2">mdi-cube-outline</v-icon>
            <span class="font-weight-bold">Connected Containers</span>
            <v-spacer></v-spacer>
            <v-chip size="small" variant="tonal">{{ network.containers?.length || 0 }} Containers</v-chip>
          </v-card-title>
          
          <v-card-text class="pa-0">
            <v-data-table
              :headers="containerHeaders"
              :items="network.containers || []"
              class="bg-transparent"
              density="comfortable"
              hover
              items-per-page="25"
              @click:row="goToContainer"
            >
              <template v-slot:item.name="{ value }">
                <span class="text-body-2 font-weight-bold">{{ value }}</span>
              </template>
              <template v-slot:item.ipv4="{ value }">
                <code class="text-caption">{{ value || '-' }}</code>
              </template>
              <template v-slot:item.ipv6="{ value }">
                <code class="text-caption">{{ value || '-' }}</code>
              </template>
              <template v-slot:item.id="{ value }">
                <code class="text-caption text-grey">{{ value.substring(0, 12) }}</code>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="loading && !network" class="fill-height d-flex align-center justify-center">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const network = ref<any>(null)
const loading = ref(true)

const containerHeaders = [
  { title: 'Name', key: 'name', sortable: true, align: 'start' as const },
  { title: 'IPv4 Address', key: 'ipv4', sortable: true, align: 'start' as const },
  { title: 'IPv6 Address', key: 'ipv6', sortable: true, align: 'start' as const },
  { title: 'Container ID', key: 'id', sortable: true, align: 'start' as const },
]

const fetchDetail = async () => {
  const id = route.params.id as string
  if (!id) return

  loading.value = true
  try {
    const response = await fetch(`/api/networks/detail?id=${id}`)
    if (response.ok) {
      network.value = await response.json()
    } else {
      console.error('Failed to fetch network detail')
    }
  } catch (err) {
    console.error('Error fetching network detail:', err)
  } finally {
    loading.value = false
  }
}

const goToContainer = (_: any, { item }: any) => {
  // Note: We don't have the node ID here easily, but many containers 
  // in the same network might be on different nodes.
  // We'll try to navigate if we can find a way to resolve the node.
  // For now, we'll just link to the detail if we have enough info.
  router.push(`/swarm/containers/${item.id}`)
}

onMounted(fetchDetail)
</script>

<style scoped>
.gap-1 { gap: 4px; }
</style>
