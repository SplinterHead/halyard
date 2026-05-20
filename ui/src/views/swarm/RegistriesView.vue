<template>
  <v-container fluid>
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Docker Registries</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-plus"
        color="primary"
        variant="flat"
        @click="openAddDialog"
      >
        Add Registry
      </v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <!-- Dynamic Docker Hub rate limit indicator for anonymous usage -->
    <v-alert
      v-if="dockerHubLimit && !hasDockerHubCredential"
      type="info"
      variant="tonal"
      class="mb-6 rounded-xl border-0"
      density="comfortable"
      prepend-icon="mdi-shield-alert-outline"
    >
      <div class="d-flex flex-wrap align-center justify-space-between w-100">
        <div class="d-flex align-center">
          <v-icon color="info" size="large" class="me-3">mdi-docker</v-icon>
          <div>
            <span class="font-weight-bold">Docker Hub Pull Rate Limit:</span>
            <span class="font-mono ms-2 text-primary font-weight-bold">{{ dockerHubLimit.remaining }} / {{ dockerHubLimit.limit }} pulls remaining</span> for this cluster's public IP.
          </div>
        </div>
        <div class="d-flex align-center mt-2 mt-sm-0">
          <span v-if="dockerHubLimit.reset" class="text-caption text-grey-lighten-1 me-4">Resets in {{ formatResetTime(dockerHubLimit.reset) }}</span>
          <v-btn
            prepend-icon="mdi-key-variant"
            size="small"
            color="primary"
            variant="flat"
            @click="addDockerHubPreset"
            class="rounded-lg"
          >
            Authenticate Docker Hub
          </v-btn>
        </div>
      </div>
    </v-alert>

    <v-row v-if="loading" justify="center" class="mt-8">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </v-row>

    <div v-else-if="registries.length === 0" class="flex-grow-1 d-flex flex-column align-center justify-center py-12">
      <v-icon size="80" color="grey-lighten-1" class="mb-4">mdi-database-lock</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No Registries Configured</h3>
      <p class="text-body-1 text-grey-darken-1 mt-2 mb-6 text-center" style="max-width: 500px">
        Add registry credentials to pull private images during automated deployments and reconciliations. Use the button above to add your first registry.
      </p>
    </div>

    <v-row v-else>
      <v-col v-for="reg in registries" :key="reg.id" cols="12" md="6" lg="4">
        <v-card variant="outlined" class="registry-card">
          <v-card-item>
            <template v-slot:prepend>
              <v-avatar color="primary" variant="tonal" size="48">
                <v-icon>{{ getRegistryIcon(reg) }}</v-icon>
              </v-avatar>
            </template>
            <v-card-title class="font-weight-bold">{{ reg.name }}</v-card-title>
            <v-card-subtitle class="font-mono">{{ reg.url || 'Docker Hub' }}</v-card-subtitle>

            <template v-slot:append>
              <div class="d-flex align-center">
                <v-btn icon="mdi-pencil" variant="text" size="small" class="me-1" @click.stop="openEditDialog(reg)"></v-btn>
                <v-btn icon="mdi-delete" variant="text" color="error" size="small" @click.stop="confirmDelete(reg)"></v-btn>
              </div>
            </template>
          </v-card-item>

          <v-divider></v-divider>

          <v-card-text>
            <div class="d-flex align-center mb-2">
              <v-icon size="small" class="mr-2" color="grey">mdi-account</v-icon>
              <span class="text-body-2 font-mono">{{ reg.username }}</span>
            </div>
            <div class="d-flex align-center" :class="{ 'mb-4': isDockerHub(reg) }">
              <v-icon size="small" class="mr-2" color="grey">mdi-key-variant</v-icon>
              <span class="text-body-2 font-mono">••••••••••••</span>
            </div>

            <!-- Rate Limit Info for Docker Hub -->
            <v-expand-transition>
              <div v-if="isDockerHub(reg) && dockerHubLimit" class="mt-4 pt-3 border-t border-grey-darken-3">
                <div class="d-flex align-center justify-space-between mb-1">
                  <span class="text-caption text-grey-lighten-1">Docker Hub Limit</span>
                  <span class="text-caption font-weight-bold font-mono" :class="getLimitColorClass">
                    {{ dockerHubLimit.remaining }} / {{ dockerHubLimit.limit }} left
                  </span>
                </div>
                <v-progress-linear
                  :model-value="(dockerHubLimit.remaining / dockerHubLimit.limit) * 100"
                  height="6"
                  rounded
                  :color="getLimitProgressColor"
                  class="mb-2"
                ></v-progress-linear>
                <div class="d-flex justify-space-between align-center">
                  <span v-if="dockerHubLimit.reset" class="text-caption text-grey-darken-1">Reset in {{ formatResetTime(dockerHubLimit.reset) }}</span>
                  <v-spacer v-else></v-spacer>
                  <v-btn
                    icon="mdi-refresh"
                    variant="text"
                    size="x-small"
                    density="comfortable"
                    @click.stop="fetchDockerHubLimit"
                    :loading="fetchingLimit"
                  ></v-btn>
                </div>
              </div>
            </v-expand-transition>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Add/Edit Dialog -->
    <v-dialog v-model="dialog" max-width="600px">
      <v-card>
        <v-card-title class="pa-6 pb-0">
          <span class="text-h5 font-weight-bold">{{ editedIndex === -1 ? 'Add Registry' : 'Edit Registry' }}</span>
        </v-card-title>
        <v-card-text class="pt-4">
          <v-form ref="form" v-model="valid">
            <v-text-field
              v-model="editedItem.name"
              label="Friendly Name"
              placeholder="e.g. My Private Registry"
              required
              :rules="[v => !!v || 'Name is required']"
              variant="outlined"
              density="comfortable"
            ></v-text-field>

            <v-text-field
              v-model="editedItem.url"
              label="Registry URL"
              placeholder="e.g. https://index.docker.io/v1/ or my.registry.com"
              hint="Leave blank for Docker Hub"
              persistent-hint
              variant="outlined"
              density="comfortable"
              class="mt-4"
            >
              <template v-slot:append-inner>
                <v-menu location="bottom end">
                  <template v-slot:activator="{ props }">
                    <v-btn
                      v-bind="props"
                      variant="text"
                      density="compact"
                      icon="mdi-chevron-down"
                      title="Popular Registries"
                      class="mr-n2"
                    ></v-btn>
                  </template>
                  <v-list density="comfortable" nav min-width="260" class="pa-2 preset-dropdown-list">
                    <v-list-subheader class="text-overline font-weight-bold pl-2">Popular Registries</v-list-subheader>
                    <v-list-item
                      v-for="preset in presets"
                      :key="preset.name"
                      @click="applyPreset(preset)"
                      class="rounded-lg mb-1"
                    >
                      <template v-slot:prepend>
                        <v-avatar size="28" color="primary" variant="tonal" class="mr-2">
                          <v-icon size="16">{{ preset.icon }}</v-icon>
                        </v-avatar>
                      </template>
                      <v-list-item-title class="font-weight-medium text-body-2">{{ preset.name }}</v-list-item-title>
                      <v-list-item-subtitle class="font-mono text-caption text-grey-darken-1">{{ preset.url }}</v-list-item-subtitle>
                    </v-list-item>
                  </v-list>
                </v-menu>
              </template>
            </v-text-field>


            <v-row class="mt-2">
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="editedItem.username"
                  label="Username"
                  required
                  :rules="[v => !!v || 'Username is required']"
                  variant="outlined"
                  density="comfortable"
                ></v-text-field>
              </v-col>
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="editedItem.password"
                  label="Password / Token"
                  type="password"
                  required
                  :rules="[v => !!v || 'Password is required']"
                  variant="outlined"
                  density="comfortable"
                ></v-text-field>
              </v-col>
            </v-row>
          </v-form>
        </v-card-text>
        <v-card-actions class="pa-4">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="dialog = false">Cancel</v-btn>
          <v-btn color="primary" variant="flat" :disabled="!valid" @click="save" :loading="saving">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation -->
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card>
        <v-card-title class="text-h5">Delete Registry?</v-card-title>
        <v-card-text>
          Are you sure you want to delete the credentials for <b>{{ itemToDelete?.name }}</b>?
          This may cause future deployments to fail if they require these credentials.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" @click="deleteItem" :loading="deleting">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

interface Registry {
  id?: string
  name: string
  url: string
  username: string
  password?: string
  created_at?: string
}

interface DockerHubLimit {
  limit: number
  remaining: number
  reset: number
}

const registries = ref<Registry[]>([])
const dockerHubLimit = ref<DockerHubLimit | null>(null)
const fetchingLimit = ref(false)

const isDockerHub = (reg: Registry) => {
  const url = reg.url?.toLowerCase() || ''
  return !url || url.includes('docker.io') || url.includes('docker.com')
}

const hasDockerHubCredential = computed(() => {
  return registries.value.some(r => isDockerHub(r))
})

const fetchDockerHubLimit = async () => {
  fetchingLimit.value = true
  try {
    const response = await fetch('/api/registries/dockerhub-limit')
    if (response.ok) {
      dockerHubLimit.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to fetch Docker Hub rate limit:', error)
  } finally {
    fetchingLimit.value = false
  }
}

const addDockerHubPreset = () => {
  openAddDialog()
  const dockerHubPreset = presets.find(p => p.name === 'Docker Hub')
  if (dockerHubPreset) {
    applyPreset(dockerHubPreset)
  }
}

const formatResetTime = (seconds: number) => {
  if (!seconds) return '0s'
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  if (remainingMinutes === 0) return `${hours}h`
  return `${hours}h ${remainingMinutes}m`
}

const getLimitColorClass = computed(() => {
  if (!dockerHubLimit.value) return 'text-grey'
  const percent = (dockerHubLimit.value.remaining / dockerHubLimit.value.limit) * 100
  if (percent > 50) return 'text-success'
  if (percent > 20) return 'text-warning'
  return 'text-error'
})

const getLimitProgressColor = computed(() => {
  if (!dockerHubLimit.value) return 'grey'
  const percent = (dockerHubLimit.value.remaining / dockerHubLimit.value.limit) * 100
  if (percent > 50) return 'success'
  if (percent > 20) return 'warning'
  return 'error'
})
const loading = ref(true)
const dialog = ref(false)
const deleteDialog = ref(false)
const valid = ref(false)
const saving = ref(false)
const deleting = ref(false)
const editedIndex = ref(-1)
const itemToDelete = ref<Registry | null>(null)

const defaultItem: Registry = {
  name: '',
  url: '',
  username: '',
  password: '',
}

const editedItem = ref<Registry>({ ...defaultItem })

interface RegistryPreset {
  name: string
  url: string
  icon: string
  placeholderName: string
}

const presets: RegistryPreset[] = [
  {
    name: 'Docker Hub',
    url: 'https://index.docker.io/v1/',
    icon: 'mdi-docker',
    placeholderName: 'Docker Hub'
  },
  {
    name: 'GitHub Container Registry',
    url: 'ghcr.io',
    icon: 'mdi-github',
    placeholderName: 'GitHub GHCR'
  },
  {
    name: 'GitLab Container Registry',
    url: 'registry.gitlab.com',
    icon: 'mdi-gitlab',
    placeholderName: 'GitLab Registry'
  },
  {
    name: 'AWS ECR (Private)',
    url: '123456789012.dkr.ecr.us-east-1.amazonaws.com',
    icon: 'mdi-aws',
    placeholderName: 'AWS ECR Private'
  },
  {
    name: 'AWS ECR (Public)',
    url: 'public.ecr.aws',
    icon: 'mdi-aws',
    placeholderName: 'AWS ECR Public'
  },
  {
    name: 'Google Artifact Registry',
    url: 'us-central1-docker.pkg.dev',
    icon: 'mdi-google-cloud',
    placeholderName: 'Google Artifact Registry'
  },
  {
    name: 'Google Container Registry',
    url: 'gcr.io',
    icon: 'mdi-google-cloud',
    placeholderName: 'Google GCR'
  },
  {
    name: 'Red Hat Quay',
    url: 'quay.io',
    icon: 'mdi-redhat',
    placeholderName: 'Red Hat Quay'
  },
  {
    name: 'Azure Container Registry',
    url: 'myregistry.azurecr.io',
    icon: 'mdi-microsoft-azure',
    placeholderName: 'Azure ACR'
  },
]

const applyPreset = (preset: RegistryPreset) => {
  editedItem.value.url = preset.url
  if (!editedItem.value.name || presets.some(p => p.placeholderName === editedItem.value.name || p.name === editedItem.value.name)) {
    editedItem.value.name = preset.placeholderName
  }
}

const getRegistryIcon = (reg: Registry) => {
  const url = reg.url?.toLowerCase() || ''
  if (!url || url.includes('docker.io') || url.includes('docker.com')) return 'mdi-docker'
  if (url.includes('ghcr.io') || url.includes('github.com')) return 'mdi-github'
  if (url.includes('gitlab.com')) return 'mdi-gitlab'
  if (url.includes('amazonaws.com') || url.includes('ecr.aws')) return 'mdi-aws'
  if (url.includes('pkg.dev') || url.includes('gcr.io')) return 'mdi-google-cloud'
  if (url.includes('quay.io')) return 'mdi-redhat'
  if (url.includes('azurecr.io')) return 'mdi-microsoft-azure'
  return 'mdi-server'
}

const fetchRegistries = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/registries')
    if (response.ok) {
      registries.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to fetch registries:', error)
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  editedIndex.value = -1
  editedItem.value = { ...defaultItem }
  dialog.value = true
}

const openEditDialog = (item: Registry) => {
  editedIndex.value = registries.value.indexOf(item)
  editedItem.value = { ...item }
  dialog.value = true
}

const confirmDelete = (item: Registry) => {
  itemToDelete.value = item
  deleteDialog.value = true
}

const deleteItem = async () => {
  if (!itemToDelete.value?.id) return
  deleting.value = true
  try {
    const response = await fetch(`/api/registries/${itemToDelete.value.id}`, {
      method: 'DELETE',
    })
    if (response.ok) {
      await fetchRegistries()
      fetchDockerHubLimit()
      deleteDialog.value = false
    }
  } catch (error) {
    console.error('Failed to delete registry:', error)
  } finally {
    deleting.value = false
  }
}

const save = async () => {
  saving.value = true
  try {
    const isEdit = editedIndex.value > -1
    const url = isEdit ? `/api/registries/${editedItem.value.id}` : '/api/registries'
    const method = isEdit ? 'PUT' : 'POST'

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editedItem.value),
    })

    if (response.ok) {
      await fetchRegistries()
      fetchDockerHubLimit()
      dialog.value = false
    }
  } catch (error) {
    console.error('Failed to save registry:', error)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await fetchRegistries()
  fetchDockerHubLimit()
})
</script>

<style scoped>
.registry-card {
  transition: transform 0.2s, box-shadow 0.2s;
  border-color: rgba(255, 255, 255, 0.1);
}
.registry-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3) !important;
  border-color: var(--v-primary-base);
}
.preset-dropdown-list {
  background: #18181b !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5) !important;
}
</style>
