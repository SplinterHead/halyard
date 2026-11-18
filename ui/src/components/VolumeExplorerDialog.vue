<template>
  <AppModal
    v-model="internalModel"
    :title="`Volume Explorer: ${volumeName}`"
    max-width="860px"
  >
    <div class="nested-box flex-between" style="flex-wrap: wrap; gap: 10px;">
      <div class="flex-row mono" style="flex-wrap: wrap; gap: 4px;">
        <button
          v-for="(crumb, idx) in breadcrumbItems"
          :key="crumb.path"
          type="button"
          class="section-link"
          :class="{ 'text-1 fw-500': idx === breadcrumbItems.length - 1 }"
          :disabled="crumb.disabled"
          @click="navigate(crumb.path)"
        >
          <span>{{ crumb.title }}</span>
          <span v-if="idx < breadcrumbItems.length - 1 && crumb.path !== '/'" class="text-3">/</span>
        </button>
      </div>

      <div class="flex-row">
        <input
          v-model="searchQuery"
          type="search"
          class="search-input"
          placeholder="Filter files..."
          aria-label="Filter files"
        />
        <button type="button" class="btn btn-icon" :disabled="loading" aria-label="Refresh" @click="fetchFiles">
          <v-icon size="15">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <div class="surface-card">
      <div class="table-wrapper" style="max-block-size: 55dvh;">
        <table class="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th class="text-end">Size</th>
              <th>Modified</th>
              <th>Permissions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="text-center text-3">Loading directory...</td>
            </tr>
            <tr v-else-if="filteredFiles.length === 0">
              <td colspan="4" class="text-center text-3">This directory is empty</td>
            </tr>
            <tr
              v-for="file in filteredFiles"
              :key="file.path"
              :class="{ 'clickable-row': file.is_dir }"
              @click="handleRowClick(file)"
            >
              <th>
                <div class="flex-row">
                  <v-icon size="16" :class="file.is_dir ? 'text-accent' : 'text-3'">
                    {{ file.is_dir ? 'mdi-folder' : 'mdi-file-outline' }}
                  </v-icon>
                  <span class="mono" :class="file.is_dir ? 'text-1 fw-500' : 'text-2'">{{ file.name }}</span>
                </div>
              </th>
              <td class="text-end mono text-2">
                {{ file.is_dir ? '—' : formatBytes(file.size) }}
              </td>
              <td><RelativeTime :value="file.modified_at" /></td>
              <td class="mono text-2">{{ file.permissions }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn" @click="internalModel = false">Close</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import AppModal from './AppModal.vue';
import RelativeTime from './RelativeTime.vue';

const props = defineProps<{
  modelValue: boolean;
  volumeName: string;
  nodeName: string;
}>();

const emit = defineEmits(['update:modelValue']);

const internalModel = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
});

const searchQuery = ref('');
const loading = ref(false);
const files = ref<any[]>([]);
const currentPath = ref('/');

const sortedFiles = computed(() => {
  return [...files.value].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
    return (a.name || '').localeCompare(b.name || '');
  });
});

const filteredFiles = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return sortedFiles.value;
  return sortedFiles.value.filter((f) => (f.name || '').toLowerCase().includes(q));
});

const breadcrumbItems = computed(() => {
  const parts = currentPath.value.split('/').filter((p) => p);
  const items = [{ title: '/', disabled: parts.length === 0, path: '/' }];

  let accumulatedPath = '';
  for (let i = 0; i < parts.length; i++) {
    accumulatedPath += '/' + parts[i];
    items.push({
      title: parts[i],
      disabled: i === parts.length - 1,
      path: accumulatedPath,
    });
  }
  return items;
});

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

const fetchFiles = async () => {
  if (!props.volumeName || !props.nodeName) return;

  loading.value = true;
  try {
    const params = new URLSearchParams({
      node: props.nodeName,
      name: props.volumeName,
      path: currentPath.value,
    });

    const response = await fetch(`/api/volumes/browse?${params.toString()}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    files.value = (await response.json()) || [];
  } catch (error) {
    console.error('Failed to fetch files:', error);
    files.value = [];
  } finally {
    loading.value = false;
  }
};

const handleRowClick = (item: any) => {
  if (item.is_dir) {
    navigate(item.path);
  }
};

const navigate = (path: string) => {
  currentPath.value = path;
  searchQuery.value = '';
  fetchFiles();
};

watch(internalModel, (newVal) => {
  if (newVal) {
    currentPath.value = '/';
    searchQuery.value = '';
    fetchFiles();
  }
});
</script>
