<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Swarm Configs</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-plus"
        color="primary"
        @click="dialog = true"
        flat
        class="me-2"
      >
        Create Config
      </v-btn>
      <v-btn
        icon="mdi-refresh"
        @click="fetchConfigs"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <div v-if="configs.length === 0" class="flex-grow-1 d-flex flex-column align-center justify-center">
      <v-icon size="80" color="grey-lighten-1" class="mb-4">mdi-code-braces</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No Configs Found</h3>
      <p class="text-body-1 text-grey-darken-1 mt-2 mb-6 text-center" style="max-width: 500px">
        Swarm configurations allow you to store non-sensitive data, such as configuration files, in the swarm.
      </p>
    </div>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="configs"
        :loading="loading"
        :sort-by="[{ key: 'name', order: 'asc' }]"
        class="bg-transparent"
        hover
        density="comfortable"
        items-per-page="25"
      >
        <template v-slot:item.name="{ value }">
          <span class="text-body-2 font-weight-bold">{{ value }}</span>
        </template>

        <template v-slot:item.id="{ value }">
          <code class="text-caption">{{ value.substring(0, 12) }}</code>
        </template>

        <template v-slot:item.created_at="{ value }">
          <RelativeTime :value="value" />
        </template>

        <template v-slot:item.actions="{ item }">
          <div class="d-flex justify-center">
            <v-btn
              icon="mdi-delete-outline"
              size="x-small"
              variant="text"
              color="error"
              @click="confirmDelete(item)"
            ></v-btn>
          </div>
        </template>

      </v-data-table>
    </div>

    <!-- Create Config Dialog -->
    <v-dialog v-model="dialog" max-width="600px" persistent>
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">
          <span class="text-h5 font-weight-bold">Create Swarm Config</span>
        </v-card-title>
        
        <v-card-text class="pa-6 pt-2">
          <v-form ref="form" v-model="valid">
            <v-text-field
              v-model="newConfig.name"
              label="Config Name"
              placeholder="app_config"
              variant="outlined"
              density="comfortable"
              :rules="[v => !!v || 'Name is required']"
              required
              class="mb-4"
            ></v-text-field>

            <v-textarea
              v-model="newConfig.data"
              label="Config Data"
              placeholder="Paste your configuration content here..."
              variant="outlined"
              density="comfortable"
              :rules="[v => !!v || 'Data is required']"
              required
              rows="6"
              auto-grow
            ></v-textarea>
          </v-form>
        </v-card-text>

        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="closeDialog" :disabled="saving">Cancel</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            @click="createConfig"
            :loading="saving"
            :disabled="!valid"
          >
            Create
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Remove Config?</v-card-title>
        <v-card-text class="pa-6 pt-0">
          Are you sure you want to remove the configuration <strong>{{ configToDelete?.name }}</strong>? This will fail if it's currently in use by any services.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" @click="deleteConfig" :loading="deleting">Remove</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import RelativeTime from '../../components/RelativeTime.vue'

interface Config {
  id: string
  name: string
  created_at: string
}

const configs = ref<Config[]>([])
const loading = ref(false)
const dialog = ref(false)
const saving = ref(false)
const deleteDialog = ref(false)
const deleting = ref(false)
const valid = ref(false)
const configToDelete = ref<Config | null>(null)

const newConfig = ref({
  name: '',
  data: ''
})

const headers = [
  { title: 'Name', key: 'name', sortable: true, align: 'start' as const },
  { title: 'ID', key: 'id', width: '150px', align: 'start' as const },
  { title: 'Created', key: 'created_at', align: 'start' as const },
  { title: 'Actions', key: 'actions', width: '100px', align: 'center' as const, sortable: false },
]

const fetchConfigs = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/configs')
    configs.value = await response.json()
  } catch (error) {
    console.error('Failed to fetch configs:', error)
  } finally {
    loading.value = false
  }
}

const createConfig = async () => {
  saving.value = true
  try {
    const response = await fetch('/api/configs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newConfig.value)
    })
    if (response.ok) {
      await fetchConfigs()
      closeDialog()
    } else {
      const err = await response.text()
      alert('Failed to create config: ' + err)
    }
  } catch (err) {
    console.error('Failed to create config:', err)
  } finally {
    saving.value = false
  }
}

const confirmDelete = (config: Config) => {
  configToDelete.value = config
  deleteDialog.value = true
}

const deleteConfig = async () => {
  if (!configToDelete.value) return
  deleting.value = true
  try {
    const response = await fetch(`/api/configs/${configToDelete.value.id}`, {
      method: 'DELETE'
    })
    if (response.ok) {
      await fetchConfigs()
      deleteDialog.value = false
    } else {
      const err = await response.text()
      alert('Failed to remove config: ' + err)
    }
  } catch (err) {
    console.error('Failed to delete config:', err)
  } finally {
    deleting.value = false
  }
}

const closeDialog = () => {
  dialog.value = false
  newConfig.value = { name: '', data: '' }
}

onMounted(fetchConfigs)
</script>

<style scoped>
</style>
