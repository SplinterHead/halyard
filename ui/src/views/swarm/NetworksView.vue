<template>
  <DataTablePage
    title="Swarm Networks"
    :items="networks"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-network-off-outline"
    empty-title="No networks found"
    empty-description="Networks created for your services will appear here once detected."
    search-placeholder="Search networks..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    :row-props="getRowProps"
    @refresh="fetchNetworks"
    @click:row="goToDetail"
  >
    <template v-slot:actions>
      <v-btn
          prepend-icon="mdi-plus"
          color="primary"
          flat
          @click="dialog = true"
        >
          Create Network
        </v-btn>
    </template>
    <template v-slot:top>
      <v-dialog v-model="dialog" max-width="600px" persistent>
            <v-card border flat class="bg-surface">
              <v-card-title class="pa-6 pb-2">
                <span class="text-h5 font-weight-bold">Create Swarm Network</span>
              </v-card-title>
              
              <v-card-text class="pa-6 pt-2">
                <v-form ref="form" v-model="valid">
                  <v-text-field
                    v-model="newNetwork.name"
                    label="Network Name"
                    placeholder="my-overlay-net"
                    variant="outlined"
                    density="comfortable"
                    :rules="[v => !!v || 'Name is required']"
                    required
                    class="mb-4"
                  ></v-text-field>

                  <v-select
                    v-model="newNetwork.driver"
                    :items="['overlay', 'bridge']"
                    label="Driver"
                    variant="outlined"
                    density="comfortable"
                    class="mb-4"
                  ></v-select>

                  <v-row>
                    <v-col cols="12" md="6">
                      <v-switch
                        v-model="newNetwork.attachable"
                        label="Attachable"
                        hint="Allow standalone containers to connect"
                        persistent-hint
                        color="primary"
                        density="compact"
                      ></v-switch>
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-switch
                        v-model="newNetwork.internal"
                        label="Internal"
                        hint="Restrict external access"
                        persistent-hint
                        color="primary"
                        density="compact"
                      ></v-switch>
                    </v-col>
                  </v-row>

                  <v-divider class="my-6"></v-divider>
                  <div class="text-subtitle-2 mb-2">IPAM Configuration (Optional)</div>
                  
                  <v-row>
                    <v-col cols="12" md="8">
                      <v-text-field
                        v-model="newNetwork.ipam.subnet"
                        label="Subnet (CIDR)"
                        placeholder="10.0.10.0/24"
                        variant="outlined"
                        density="comfortable"
                      ></v-text-field>
                    </v-col>
                    <v-col cols="12" md="4">
                      <v-text-field
                        v-model="newNetwork.ipam.gateway"
                        label="Gateway"
                        placeholder="10.0.10.1"
                        variant="outlined"
                        density="comfortable"
                      ></v-text-field>
                    </v-col>
                  </v-row>

                  <div v-if="newNetwork.driver === 'overlay'" class="mt-4">
                    <v-checkbox
                      v-model="newNetwork.encrypted"
                      label="Enable Encryption (IPSEC)"
                      color="primary"
                      hide-details
                      density="compact"
                    ></v-checkbox>
                  </div>
                </v-form>
              </v-card-text>

              <v-card-actions class="pa-6 pt-0">
                <v-spacer></v-spacer>
                <v-btn variant="text" @click="closeDialog" :disabled="saving">Cancel</v-btn>
                <v-btn
                  color="primary"
                  variant="flat"
                  @click="createNetwork"
                  :loading="saving"
                  :disabled="!valid"
                >
                  Create
                </v-btn>
              </v-card-actions>
            </v-card>
          </v-dialog>
      <v-dialog v-model="deleteDialog" max-width="400px">
            <v-card border flat class="bg-surface">
              <v-card-title class="pa-6 pb-2">Delete Network?</v-card-title>
              <v-card-text class="pa-6 pt-0">
                Are you sure you want to remove the network <strong>{{ networkToDelete?.name }}</strong>? This action cannot be undone.
              </v-card-text>
              <v-card-actions class="pa-6 pt-0">
                <v-spacer></v-spacer>
                <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
                <v-btn
                  color="error"
                  variant="flat"
                  @click="deleteNetwork"
                  :loading="deleting"
                >Remove Network</v-btn>
              </v-card-actions>
            </v-card>
          </v-dialog>
      <v-dialog v-model="errorDialog" max-width="500px">
            <v-card border flat class="bg-surface">
              <v-card-title class="pa-6 pb-2 text-error d-flex align-center">
                <v-icon color="error" class="me-2">mdi-alert-circle</v-icon>
                Network Deletion Failed
              </v-card-title>
              <v-card-text class="pa-6 pt-0">
                <p class="mb-4">The network could not be deleted. This usually happens if it is still being used by one or more services or containers.</p>
                <div class="bg-black bg-opacity-20 pa-4 rounded-lg font-mono text-caption text-error border border-error border-opacity-20">
                  {{ errorMessage }}
                </div>
              </v-card-text>
              <v-card-actions class="pa-6 pt-0">
                <v-spacer></v-spacer>
                <v-btn variant="flat" color="primary" @click="errorDialog = false">Dismiss</v-btn>
              </v-card-actions>
            </v-card>
          </v-dialog>
    </template>
    <template v-slot:item.name="{ item }">
      <div class="d-flex flex-column">
        <span class="text-body-2 font-weight-bold">{{ item.name }}</span>
        <span class="text-caption text-grey">{{ item.stack === '-' ? 'System' : item.stack }}</span>
      </div>
    </template>

    <template v-slot:item.internal="{ value }">
      <v-chip
        size="x-small"
        :color="value ? 'success' : 'grey'"
        variant="tonal"
        label
      >
        {{ value ? 'Yes' : 'No' }}
      </v-chip>
    </template>

    <template v-slot:item.attachable="{ value }">
      <v-chip
        size="x-small"
        :color="value ? 'success' : 'grey'"
        variant="tonal"
        label
      >
        {{ value ? 'Yes' : 'No' }}
      </v-chip>
    </template>

    <template v-slot:item.subnet="{ value }">
      <code class="font-mono text-caption" v-if="value">{{ value }}</code>
      <span v-else class="text-caption text-grey-lighten-1">-</span>
    </template>

    <template v-slot:item.ip_usage="{ item }">
      <div v-if="item.total_ips > 0" class="d-flex flex-column" style="width: 150px">
        <div class="d-flex justify-space-between align-center mb-1">
          <span class="text-caption font-mono text-grey-lighten-1">
            {{ item.used_ips }} / {{ item.total_ips }}
          </span>
          <span class="text-caption font-mono text-grey" style="font-size: 0.75rem">
            {{ Math.round((item.used_ips / item.total_ips) * 100) }}%
          </span>
        </div>
        <v-progress-linear
          :model-value="(item.used_ips / item.total_ips) * 100"
          :color="getIpUsageColor(item.used_ips, item.total_ips)"
          height="4"
          rounded
        ></v-progress-linear>
      </div>
      <span v-else class="text-caption text-grey-lighten-1">-</span>
    </template>

    <template v-slot:item.created_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
      </div>
    </template>

    <template v-slot:item.actions="{ item }">
      <div class="d-flex justify-center">
        <v-btn
          icon="mdi-delete-outline"
          size="x-small"
          variant="text"
          color="error"
          @click.stop="confirmDelete(item)"
        ></v-btn>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import DataTablePage from "../../components/DataTablePage.vue";
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import RelativeTime from '../../components/RelativeTime.vue'

const searchQuery = ref('')

interface Network {
  id: string
  name: string
  scope: string
  subnet: string
  stack: string
  created_at: string
  internal: boolean
  attachable: boolean
  used_ips: number
  total_ips: number
}

const networks = ref<Network[]>([])
const loading = ref(false)
const dialog = ref(false)
const saving = ref(false)
const valid = ref(false)

const newNetwork = ref({
  name: '',
  driver: 'overlay',
  attachable: true,
  internal: false,
  encrypted: false,
  ipam: {
    subnet: '',
    gateway: ''
  }
})

const getRowProps = ({ item }: any) => {
  return {
    class: `status-bar-row ${item.scope === "swarm" ? "status-info" : "status-grey"}`,
  };
};

const getIpUsageColor = (used: number, total: number) => {
  if (!total) return 'primary'
  const pct = (used / total) * 100
  if (pct >= 90) return 'error'
  if (pct >= 75) return 'warning'
  return 'primary'
}

const headers = [
  {
    title: "Network / Stack",
    key: "name",
    sortable: true,
    align: "start" as const,
  },
  {
    title: "Internal",
    key: "internal",
    width: "110px",
    align: "start" as const,
  },
  {
    title: "Attachable",
    key: "attachable",
    width: "110px",
    align: "start" as const,
  },
  {
    title: "Subnet",
    key: "subnet",
    align: "start" as const,
  },
  {
    title: "Ip Usage",
    key: "ip_usage",
    width: "180px",
    align: "start" as const,
    value: (item: Network) => item.total_ips > 0 ? (item.used_ips / item.total_ips) : 0,
  },
  {
    title: "Created",
    key: "created_at",
    width: "140px",
    align: "end" as const,
  },
  {
    title: "Actions",
    key: "actions",
    width: "80px",
    align: "center" as const,
    sortable: false,
  },
];

const router = useRouter()
const deleteDialog = ref(false)
const deleting = ref(false)
const networkToDelete = ref<Network | null>(null)
const errorDialog = ref(false)
const errorMessage = ref('')

const goToDetail = (_: any, { item }: any) => {
  router.push({ name: 'network-detail', params: { id: item.id } })
}

const confirmDelete = (network: Network) => {
  networkToDelete.value = network
  deleteDialog.value = true
}

const deleteNetwork = async () => {
  if (!networkToDelete.value) return
  deleting.value = true
  try {
    const response = await fetch(`/api/networks?id=${networkToDelete.value.id}`, {
      method: 'DELETE'
    })

    if (response.ok) {
      await fetchNetworks()
      deleteDialog.value = false
    } else {
      const err = await response.text()
      errorMessage.value = err
      errorDialog.value = true
    }
  } catch (err) {
    console.error('Failed to delete network:', err)
    errorMessage.value = String(err)
    errorDialog.value = true
  } finally {
    deleting.value = false
    networkToDelete.value = null
  }
}

const fetchNetworks = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/networks')
    networks.value = await response.json()
  } catch (error) {
    console.error('Failed to fetch networks:', error)
  } finally {
    loading.value = false
  }
}

const createNetwork = async () => {
  saving.value = true
  try {
    const options: Record<string, string> = {}
    if (newNetwork.value.encrypted) {
      options['encrypted'] = 'true'
    }

    const payload = {
      name: newNetwork.value.name,
      driver: newNetwork.value.driver,
      attachable: newNetwork.value.attachable,
      internal: newNetwork.value.internal,
      options: options,
      ipam: (newNetwork.value.ipam.subnet) ? {
        subnet: newNetwork.value.ipam.subnet,
        gateway: newNetwork.value.ipam.gateway
      } : null
    }

    const response = await fetch('/api/networks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })

    if (response.ok) {
      await fetchNetworks()
      closeDialog()
    } else {
      const err = await response.text()
      alert('Failed to create network: ' + err)
    }
  } catch (err) {
    console.error('Failed to create network:', err)
  } finally {
    saving.value = false
  }
}

const closeDialog = () => {
  dialog.value = false
  newNetwork.value = {
    name: '',
    driver: 'overlay',
    attachable: true,
    internal: false,
    encrypted: false,
    ipam: {
      subnet: '',
      gateway: ''
    }
  }
}

onMounted(fetchNetworks)
</script>

<style scoped>
</style>
