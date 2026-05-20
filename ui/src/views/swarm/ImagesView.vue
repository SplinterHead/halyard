<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Cluster Images</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-layers-search-outline"
        @click="checkAllImages"
        :loading="checkingAll"
        variant="tonal"
        color="primary"
        class="me-2"
        size="small"
      >
        Check All Updates
      </v-btn>
      <v-btn
        icon="mdi-refresh"
        @click="fetchImages"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <!-- Data Table -->
    <div v-if="images.length === 0 && !loading" class="flex-grow-1 d-flex flex-column align-center justify-center">
      <v-icon size="80" color="grey-lighten-1" class="mb-4">mdi-image-off-outline</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No Images Found</h3>
      <p class="text-body-1 text-grey-darken-1 mt-2 mb-6 text-center" style="max-width: 500px">
        No Docker images were found stored on any active Swarm node.
      </p>
    </div>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="images"
        :loading="loading"
        :row-props="getRowProps"
        item-value="ui_key"
        class="bg-transparent"
        hover
        density="comfortable"
        items-per-page="25"
      >
        <!-- Repository Column (with registry icon) -->
        <template v-slot:item.repository="{ item }">
          <div class="d-flex align-center">
            <v-icon
              :icon="getRegistryIcon(item.repository)"
              color="primary"
              size="20"
              class="me-3"
            ></v-icon>
            <div class="d-flex flex-column">
              <span class="text-body-2 font-weight-bold text-truncate-400" :title="item.repository">
                {{ item.repository.split('@')[0] }}
              </span>
              <span class="text-caption text-grey text-truncate-200 font-mono" :title="item.tag">
                {{ item.tag.split('@')[0] }}
              </span>
            </div>
          </div>
        </template>

        <!-- SHA Column -->
        <template v-slot:item.id="{ value }">
          <code class="font-mono text-caption text-grey" :title="value">
            {{ value.replace('sha256:', '').substring(0, 12) }}
          </code>
        </template>

        <!-- Architecture Column -->
        <template v-slot:item.architecture="{ value }">
          <v-chip size="x-small" variant="outlined" color="primary" label class="font-mono text-uppercase">
            {{ value }}
          </v-chip>
        </template>

        <!-- Node Column -->
        <template v-slot:item.node="{ value }">
          <span class="font-mono text-body-2">{{ value }}</span>
        </template>

        <!-- Size Column -->
        <template v-slot:item.size="{ value }">
          <span class="text-body-2">{{ formatSize(value) }}</span>
        </template>

        <!-- In Use Column -->
        <template v-slot:item.in_use="{ value }">
          <v-chip
            size="x-small"
            :color="value ? 'success' : 'grey'"
            variant="tonal"
            label
          >
            {{ value ? 'In Use' : 'Unused' }}
          </v-chip>
        </template>

        <!-- Actions Column -->
        <template v-slot:item.actions="{ item }">
          <div class="d-flex align-center justify-center gap-1">
            <!-- Delete Action Button -->
            <v-btn
              icon="mdi-delete-outline"
              size="x-small"
              variant="text"
              color="error"
              @click="confirmDelete(item)"
              title="Delete Image"
            ></v-btn>
          </div>
        </template>
      </v-data-table>
    </div>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="450px" persistent>
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2 d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-circle-outline</v-icon>
          <span class="text-h6 font-weight-bold">Delete Docker Image?</span>
        </v-card-title>
        
        <v-card-text class="pa-6 pt-2">
          <p class="text-body-2 mb-4">
            Are you sure you want to remove this image tag from <strong>{{ imageToDelete?.node }}</strong>?
          </p>
          <div class="bg-black bg-opacity-20 rounded-lg pa-3 mb-4 font-mono text-caption">
            <div><strong>Image:</strong> {{ imageToDelete?.repository }}:{{ imageToDelete?.tag }}</div>
            <div class="text-truncate-200"><strong>ID:</strong> {{ imageToDelete?.id }}</div>
            <div><strong>Node:</strong> {{ imageToDelete?.node }}</div>
          </div>
          
          <v-alert
            v-if="imageToDelete?.in_use"
            type="warning"
            variant="tonal"
            density="compact"
            class="mb-4 text-caption"
          >
            This image is currently in use by one or more containers. Deleting it standardly will fail unless you force it.
          </v-alert>

          <v-checkbox
            v-model="forceDelete"
            label="Force delete image (equivalent to rmi -f)"
            color="error"
            density="compact"
            hide-details
          ></v-checkbox>
        </v-card-text>

        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="closeDeleteDialog" :disabled="deleting">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteImage"
            :loading="deleting"
          >
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

interface ImageInfo {
  id: string
  repository: string
  tag: string
  node: string
  node_id: string
  size: number
  architecture: string
  in_use: boolean
  created_at: string
  ui_key?: string
  // UI states:
  up_to_date_status?: 'unchecked' | 'checking' | 'yes' | 'no' | 'error'
  check_error?: string
}

const images = ref<ImageInfo[]>([])
const loading = ref(false)
const checkingAll = ref(false)

const deleteDialog = ref(false)
const deleting = ref(false)
const imageToDelete = ref<ImageInfo | null>(null)
const forceDelete = ref(false)

const headers = [
  { title: 'Image Name', key: 'repository', sortable: true, align: 'start' as const },
  { title: 'SHA', key: 'id', width: '150px', sortable: true, align: 'start' as const },
  { title: 'Architecture', key: 'architecture', width: '130px', align: 'center' as const },
  { title: 'Node', key: 'node', sortable: true, align: 'start' as const },
  { title: 'Size', key: 'size', width: '120px', sortable: true, align: 'end' as const },
  { title: 'Status', key: 'in_use', width: '120px', sortable: true, align: 'center' as const },
  { title: 'Actions', key: 'actions', width: '120px', align: 'center' as const, sortable: false },
]

const getRowProps = ({ item }: any) => {
  let statusClass = 'status-grey'
  if (item.up_to_date_status === 'yes') {
    statusClass = 'status-success' // Green for up-to-date
  } else if (item.up_to_date_status === 'no') {
    statusClass = 'status-error' // Red for out-of-date
  } else if (item.up_to_date_status === 'error') {
    statusClass = 'status-grey' // Grey for error
  } else {
    // Unchecked default
    statusClass = 'status-grey'
  }
  return {
    class: `status-bar-row ${statusClass}`
  }
}

const fetchImages = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/images')
    if (response.ok) {
      const data = await response.json()
      images.value = data.map((img: ImageInfo) => ({
        ...img,
        ui_key: `${img.node_id}-${img.id}-${img.repository}-${img.tag}`,
        up_to_date_status: 'unchecked'
      }))
      // Automatically trigger update check for all images on load
      checkAllImages()
    } else {
      console.error('Failed to fetch images:', await response.text())
    }
  } catch (error) {
    console.error('Failed to fetch images:', error)
  } finally {
    loading.value = false
  }
}

const checkUpdate = async (item: ImageInfo) => {
  item.up_to_date_status = 'checking'
  try {
    const response = await fetch(
      `/api/images/check?node_id=${item.node_id}&repository=${encodeURIComponent(item.repository)}&tag=${encodeURIComponent(item.tag)}&id=${item.id}`,
      { method: 'POST' }
    )
    if (response.ok) {
      const data = await response.json()
      item.up_to_date_status = data.up_to_date ? 'yes' : 'no'
    } else {
      const errText = await response.text()
      item.up_to_date_status = 'error'
      item.check_error = errText
    }
  } catch (error: any) {
    item.up_to_date_status = 'error'
    item.check_error = error.message || 'Network error'
  }
}

const checkAllImages = async () => {
  checkingAll.value = true
  const targets = images.value.filter(img => img.up_to_date_status !== 'checking')
  await Promise.all(targets.map(img => checkUpdate(img)))
  checkingAll.value = false
}

const getRegistryIcon = (repo: string): string => {
  const lower = repo.toLowerCase()
  if (lower.includes('ghcr.io')) return 'mdi-github'
  if (lower.includes('gcr.io') || lower.includes('pkg.dev')) return 'mdi-google'
  if (lower.includes('quay.io')) return 'mdi-redhat'
  if (lower.includes('public.ecr.aws') || lower.includes('.dkr.ecr.')) return 'mdi-aws'
  if (lower.includes('azurecr.io')) return 'mdi-microsoft-azure'
  if (lower.includes('gitlab.com')) return 'mdi-gitlab'
  if (lower.includes('lscr.io')) return 'mdi-linux'
  return 'mdi-docker' // Default docker hub whale
}

const formatSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const confirmDelete = (img: ImageInfo) => {
  imageToDelete.value = img
  forceDelete.value = false
  deleteDialog.value = true
}

const closeDeleteDialog = () => {
  deleteDialog.value = false
  imageToDelete.value = null
  forceDelete.value = false
}

const deleteImage = async () => {
  if (!imageToDelete.value) return
  deleting.value = true
  try {
    const response = await fetch(
      `/api/images?node_id=${imageToDelete.value.node_id}&id=${imageToDelete.value.id}&force=${forceDelete.value}`,
      { method: 'DELETE' }
    )
    if (response.ok) {
      await fetchImages()
      closeDeleteDialog()
    } else {
      const err = await response.text()
      alert(`Failed to delete image: ${err}`)
    }
  } catch (error) {
    console.error('Failed to delete image:', error)
    alert(`Failed to delete image: ${error}`)
  } finally {
    deleting.value = false
  }
}

onMounted(fetchImages)
</script>
