<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Swarm Volumes</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-broom"
        color="error"
        variant="tonal"
        class="mr-2"
        @click="showPruneDialog = true"
        :loading="pruning"
      >
        Prune Unused
      </v-btn>
      <v-btn
        icon="mdi-refresh"
        @click="fetchVolumes"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <!-- Prune Confirmation Dialog -->
    <v-dialog v-model="showPruneDialog" max-width="500">
      <v-card>
        <v-card-title class="text-h5">Prune Unused Volumes?</v-card-title>
        <v-card-text>
          This will remove ALL volumes on ALL nodes that are not currently used by at least one container.
          This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="grey-darken-1" variant="text" @click="showPruneDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" @click="pruneVolumes" :loading="pruning">Prune Volumes</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="450px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2 text-h5 font-weight-bold d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-decagram</v-icon>
          Delete Volume?
        </v-card-title>
        <v-card-text class="pa-6 pt-2">
          Are you sure you want to remove the volume <strong class="font-mono text-error">{{ volumeToDelete?.name }}</strong> on node <strong>{{ volumeToDelete?.node }}</strong>? This action cannot be undone.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteVolume"
            :loading="deleting"
          >Remove Volume</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Error Dialog -->
    <v-dialog v-model="errorDialog" max-width="500px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2 text-error d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-circle</v-icon>
          Volume Deletion Failed
        </v-card-title>
        <v-card-text class="pa-6 pt-0">
          <p class="mb-4">The volume could not be deleted. This usually happens if it is still being used by one or more containers on the node.</p>
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

    <v-divider class="my-4"></v-divider>

    <v-row v-if="loading" justify="center" class="mt-8">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </v-row>

    <v-row v-else-if="volumes.length === 0" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="64" color="grey-lighten-1" class="mb-4">mdi-database-off</v-icon>
        <h3 class="text-h5 text-grey-darken-1">No volumes found</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          Volumes created by your services or manually will appear here once detected.
        </p>
      </v-col>
    </v-row>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="volumes"
        :loading="loading"
        :sort-by="[{ key: 'name', order: 'asc' }]"
        :row-props="getRowProps"
        class="bg-transparent"
        density="comfortable"
        items-per-page="25"
      >
        <template v-slot:item.name="{ item }">
          <span class="text-body-2 font-weight-bold" :title="item.name">
            {{ item.name }}
          </span>
        </template>

        <template v-slot:item.stack="{ value }">
          <div class="text-start">
            <v-chip v-if="value !== '-'" size="x-small" variant="tonal" color="primary" label>
              {{ value }}
            </v-chip>
            <span v-else class="text-caption text-grey-lighten-1">-</span>
          </div>
        </template>


        <template v-slot:item.driver="{ value }">
          <div class="text-center">
            <v-chip
              :color="getDriverColor(value)"
              size="x-small"
              label
              class="text-uppercase font-weight-bold"
            >
              {{ value }}
            </v-chip>
          </div>
        </template>

        <template v-slot:item.type="{ value }">
          <div class="text-center">
            <v-chip
              :color="getTypeColor(value)"
              size="x-small"
              label
              class="text-uppercase font-weight-bold"
            >
              {{ value }}
            </v-chip>
          </div>
        </template>

        <template v-slot:item.node="{ value }">
          <div class="text-center">
            <span class="text-caption font-mono text-grey-lighten-1">{{ value }}</span>
          </div>
        </template>

        <template v-slot:item.created_at="{ value }">
          <RelativeTime :value="value" />
        </template>

        <template v-slot:item.actions="{ item }">
          <div class="d-flex justify-center align-center">
            <v-btn
              icon="mdi-delete-outline"
              size="x-small"
              variant="text"
              color="error"
              @click.stop="confirmDelete(item)"
            ></v-btn>
          </div>
        </template>

      </v-data-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import RelativeTime from '../../components/RelativeTime.vue'

interface Volume {
  name: string
  node: string
  stack: string
  driver: string
  type: string
  in_use: boolean
  created_at: string
  labels: Record<string, string>
}

const volumes = ref<Volume[]>([])
const loading = ref(false)
const pruning = ref(false)
const showPruneDialog = ref(false)

const deleteDialog = ref(false)
const deleting = ref(false)
const volumeToDelete = ref<Volume | null>(null)
const errorDialog = ref(false)
const errorMessage = ref('')

const getRowProps = ({ item }: any) => {
  return {
    class: `status-bar-row ${item.in_use ? "status-success" : "status-grey"}`,
  };
};

const headers = [
  { title: "Volume / ID", key: "name", sortable: true, align: "start" as const },
  {
    title: "Node",
    key: "node",
    sortable: true,
    align: "center" as const,
  },
  {
    title: "Stack",
    key: "stack",
    sortable: true,
    width: "150px",
    align: "start" as const,
  },
  { title: "Driver", key: "driver", width: "150px", align: "center" as const },
  { title: "Type", key: "type", width: "150px", align: "center" as const },
  {
    title: "Created",
    key: "created_at",
    width: "150px",
    align: "start" as const,
  },
  {
    title: "Actions",
    key: "actions",
    width: "80px",
    align: "center" as const,
    sortable: false,
  },
];

const getTypeColor = (type: string) => {
  switch (type.toLowerCase()) {
    case 'local': return 'grey'
    case 'nfs': return 'info'
    case 'gluster': return 'primary'
    default: return 'secondary'
  }
}

const getDriverColor = (driver: string) => {
  switch (driver.toLowerCase()) {
    case 'local': return 'grey'
    default: return 'primary'
  }
}

const pruneVolumes = async () => {
  pruning.value = true
  showPruneDialog.value = false
  try {
    const response = await fetch('/api/volumes/prune', { method: 'POST' })
    if (response.ok) {
      await fetchVolumes()
    }
  } catch (error) {
    console.error('Failed to prune volumes:', error)
  } finally {
    pruning.value = false
  }
}

const fetchVolumes = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/volumes')
    const data = await response.json()
    // Filter out blank rows (no name or node)
    volumes.value = data.filter((v: Volume) => v.name && v.node)
  } catch (error) {
    console.error('Failed to fetch volumes:', error)
  } finally {
    loading.value = false
  }
}

const confirmDelete = (volume: Volume) => {
  volumeToDelete.value = volume
  deleteDialog.value = true
}

const deleteVolume = async () => {
  if (!volumeToDelete.value) return
  deleting.value = true
  try {
    const response = await fetch(`/api/volumes?node=${encodeURIComponent(volumeToDelete.value.node)}&name=${encodeURIComponent(volumeToDelete.value.name)}`, {
      method: 'DELETE'
    })

    if (response.ok) {
      await fetchVolumes()
      deleteDialog.value = false
    } else {
      const err = await response.text()
      errorMessage.value = err
      errorDialog.value = true
    }
  } catch (err) {
    console.error('Failed to delete volume:', err)
    errorMessage.value = String(err)
    errorDialog.value = true
  } finally {
    deleting.value = false
    volumeToDelete.value = null
  }
}

onMounted(fetchVolumes)
</script>

