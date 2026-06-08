<template>
  <v-dialog v-model="internalModel" max-width="75vw" scrollable>
    <v-card class="bg-surface h-100" style="max-height: 85vh;">
      <v-card-title class="pa-4 d-flex align-center border-b">
        <v-icon class="mr-2 text-primary">mdi-folder-network-outline</v-icon>
        <span class="text-h6 font-weight-bold">Volume Explorer: <span class="font-mono text-body-1 ml-1 text-primary">{{ volumeName }}</span></span>
        <v-spacer></v-spacer>
        <v-btn icon="mdi-close" variant="text" @click="internalModel = false" size="small"></v-btn>
      </v-card-title>
      
      <div class="px-4 py-2 bg-grey-darken-4 border-b d-flex align-center flex-wrap gap-4">
        <v-breadcrumbs :items="breadcrumbItems" class="pa-0">
          <template v-slot:item="{ item }">
            <v-breadcrumbs-item
              :href="item.href"
              :disabled="item.disabled"
              @click.prevent="navigate(item.path)"
              class="cursor-pointer font-mono d-flex align-center"
            >
              <v-icon v-if="item.path === '/'" size="small">mdi-home</v-icon>
              <span v-else>{{ item.title }}</span>
            </v-breadcrumbs-item>
          </template>
        </v-breadcrumbs>
        <v-spacer></v-spacer>
        <v-text-field
          v-model="searchQuery"
          prepend-inner-icon="mdi-magnify"
          placeholder="Filter files..."
          variant="solo-filled"
          density="compact"
          flat
          hide-details
          rounded="lg"
          class="search-input glass-input"
          style="width: 240px"
        ></v-text-field>
        <v-btn icon="mdi-refresh" variant="text" size="small" @click="fetchFiles" :loading="loading"></v-btn>
      </div>

      <v-card-text class="pa-0">
        <v-data-table
          :headers="headers"
          :items="files"
          :loading="loading"
          :search="searchQuery"
          hover
          density="compact"
          class="bg-transparent"
          :sort-by="[{ key: 'is_dir', order: 'desc' }, { key: 'name', order: 'asc' }]"
          hide-default-footer
          :items-per-page="-1"
        >
          <template v-slot:item.name="{ item }">
            <div class="d-flex align-center cursor-pointer py-2" @click="handleRowClick(item)">
              <v-icon :color="item.is_dir ? 'primary' : 'grey'" class="mr-3" size="small">
                {{ item.is_dir ? 'mdi-folder' : 'mdi-file-outline' }}
              </v-icon>
              <span :class="{'text-primary font-weight-bold': item.is_dir, 'font-mono': true}">{{ item.name }}</span>
            </div>
          </template>
          
          <template v-slot:item.size="{ item }">
            <span v-if="!item.is_dir" class="text-caption text-grey-lighten-1 font-mono">{{ formatBytes(item.size) }}</span>
            <span v-else class="text-caption text-grey-darken-1">-</span>
          </template>
          
          <template v-slot:item.modified_at="{ item }">
            <span class="text-caption font-mono text-grey-lighten-1"><RelativeTime :value="item.modified_at" /></span>
          </template>

          <template v-slot:item.permissions="{ item }">
            <code class="text-caption text-grey-lighten-1 bg-transparent px-0">{{ item.permissions }}</code>
          </template>

          <template v-slot:no-data>
            <div class="pa-8 text-center text-grey">
              <v-icon size="large" class="mb-2">mdi-folder-open-outline</v-icon>
              <div>This directory is empty</div>
            </div>
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import RelativeTime from './RelativeTime.vue'

const searchQuery = ref('')

const props = defineProps<{
  modelValue: boolean
  volumeName: string
  nodeName: string
}>()

const emit = defineEmits(['update:modelValue'])

const internalModel = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const loading = ref(false)
const files = ref<any[]>([])
const currentPath = ref('/')

const headers = [
  { title: "Name", key: "name", sortable: true },
  { title: "Size", key: "size", sortable: true, width: "100px", align: "end" as const },
  { title: "Modified", key: "modified_at", sortable: true, width: "150px" },
  { title: "Permissions", key: "permissions", sortable: false, width: "120px" },
]

const breadcrumbItems = computed(() => {
  const parts = currentPath.value.split('/').filter(p => p)
  const items = [
    { title: '/', disabled: false, href: '#', path: '/' }
  ]
  
  let accumulatedPath = ''
  for (let i = 0; i < parts.length; i++) {
    accumulatedPath += '/' + parts[i]
    items.push({
      title: parts[i],
      disabled: i === parts.length - 1,
      href: '#',
      path: accumulatedPath
    })
  }
  return items
})

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fetchFiles = async () => {
  if (!props.volumeName || !props.nodeName) return
  
  loading.value = true
  try {
    const params = new URLSearchParams({
      node: props.nodeName,
      name: props.volumeName,
      path: currentPath.value
    })
    
    const response = await fetch(`/api/volumes/browse?${params.toString()}`)
    if (!response.ok) {
      throw new Error(await response.text())
    }
    const data = await response.json()
    files.value = data || []
  } catch (error) {
    console.error('Failed to fetch files:', error)
    files.value = []
  } finally {
    loading.value = false
  }
}

const handleRowClick = (item: any) => {
  if (item.is_dir) {
    navigate(item.path)
  }
}

const navigate = (path: string) => {
  currentPath.value = path
  searchQuery.value = ''
  fetchFiles()
}

watch(internalModel, (newVal) => {
  if (newVal) {
    currentPath.value = '/'
    searchQuery.value = ''
    fetchFiles()
  }
})
</script>

<style scoped>
.v-data-table :deep(th) {
  font-weight: 600 !important;
  text-transform: uppercase;
  font-size: 0.75rem !important;
  letter-spacing: 0.05em;
}
.font-mono {
  font-family: 'Roboto Mono', 'Courier New', monospace !important;
}
</style>
