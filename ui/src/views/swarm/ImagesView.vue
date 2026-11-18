<template>
  <DataTablePage
    title="Node Images"
    subtitle="Container images stored across cluster nodes and remote digest status."
    :items="images"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-image-broken-variant"
    empty-title="No images found"
    empty-description="No images are currently downloaded on any of the cluster nodes."
    search-placeholder="Search images..."
    :sort-by="[{ key: 'repository', order: 'asc' }]"
    @refresh="fetchImages"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" :disabled="checkingAll" @click="checkAllImages">
        {{ checkingAll ? 'Checking...' : 'Check All Updates' }}
      </button>
    </template>

    <template #top>
      <AppModal v-model="deleteDialog" title="Delete Docker Image?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove this image tag from
          <strong class="text-1">{{ imageToDelete?.node }}</strong>?
        </p>

        <div class="nested-box mono" style="font-size: 12px; display: flex; flex-direction: column; gap: 4px;">
          <div><span class="text-3">Image:</span> {{ imageToDelete?.repository }}:{{ imageToDelete?.tag }}</div>
          <div class="text-truncate"><span class="text-3">ID:</span> {{ imageToDelete?.id }}</div>
          <div><span class="text-3">Node:</span> {{ imageToDelete?.node }}</div>
        </div>

        <div v-if="imageToDelete?.in_use" class="alert-banner alert-warn">
          This image is currently in use by one or more containers. Deleting it requires force mode.
        </div>

        <label class="nested-box flex-between" style="cursor: pointer;">
          <span>
            <strong class="fw-500 text-1" style="display: block;">Force delete image</strong>
            <span class="text-2" style="font-size: 12px;">Equivalent to docker rmi -f</span>
          </span>
          <input v-model="forceDelete" type="checkbox" role="switch" class="toggle-switch" />
        </label>

        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="closeDeleteDialog">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteImage">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </template>
      </AppModal>
    </template>

    <template #item.repository="{ item }">
      <div class="d-flex flex-column" style="gap: 2px; min-width: 0;">
        <div class="flex-row" style="min-width: 0;">
          <span class="status-dot" :class="getUpdateDot(item.up_to_date_status)"></span>
          <span class="mono fw-500 text-1 text-truncate" style="max-width: 520px;" :title="item.repository">
            {{ item.repository.split('@')[0] }}
          </span>
        </div>
        <span class="mono text-3 text-truncate" style="max-width: 520px; padding-inline-start: 15px;" :title="item.tag">
          {{ item.tag.split('@')[0] }}
        </span>
      </div>
    </template>

    <template #item.id="{ value }">
      <span class="mono text-2" :title="value">
        {{ value.replace('sha256:', '').substring(0, 12) }}
      </span>
    </template>

    <template #item.architecture="{ value }">
      <span class="status-badge mono">{{ value }}</span>
    </template>

    <template #item.node="{ value }">
      <span class="text-2 text-truncate d-inline-block" style="max-width: 170px;" :title="value">{{ value }}</span>
    </template>

    <template #item.size="{ value }">
      <span class="text-end d-block text-2" style="white-space: nowrap;">{{ formatSize(value) }}</span>
    </template>

    <template #item.in_use="{ value }">
      <span class="status-badge">
        <span class="status-dot" :class="value ? 'dot-ok' : ''"></span>
        {{ value ? 'In Use' : 'Unused' }}
      </span>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center">
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Image"
          aria-label="Delete Image"
          @click="confirmDelete(item)"
        >
          <v-icon size="15">mdi-delete-outline</v-icon>
        </button>
      </div>
    </template>

    <!-- Mobile Curated 2-Row Card -->
    <template #mobile-item="{ item }">
      <div class="mobile-item-card" style="cursor: default;">
        <div class="mobile-card-top">
          <div class="flex-row" style="min-width: 0;">
            <span class="status-dot" :class="getUpdateDot(item.up_to_date_status)"></span>
            <span class="mono fw-500 text-1 text-truncate">{{ item.repository.split('@')[0] }}:{{ item.tag.split('@')[0] }}</span>
          </div>
          <span class="status-badge">{{ item.in_use ? 'In Use' : 'Unused' }}</span>
        </div>
        <div class="mobile-card-bottom">
          <span><span class="text-3">Node:</span> {{ item.node }}</span>
          <span><span class="text-3">Size:</span> {{ formatSize(item.size) }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import DataTablePage from '../../components/DataTablePage.vue';
import AppModal from '../../components/AppModal.vue';

interface ImageInfo {
  id: string;
  repository: string;
  tag: string;
  node: string;
  node_id: string;
  size: number;
  architecture: string;
  in_use: boolean;
  created_at: string;
  ui_key?: string;
  up_to_date_status?: 'unchecked' | 'checking' | 'yes' | 'no' | 'error';
  check_error?: string;
}

const images = ref<ImageInfo[]>([]);
const loading = ref(false);
const checkingAll = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const imageToDelete = ref<ImageInfo | null>(null);
const forceDelete = ref(false);

const headers = [
  { title: 'Image Name', key: 'repository', sortable: true, align: 'start' as const },
  { title: 'SHA', key: 'id', width: '130px', sortable: true, align: 'start' as const },
  { title: 'Arch', key: 'architecture', width: '100px', align: 'start' as const },
  { title: 'Node', key: 'node', width: '185px', sortable: true, align: 'start' as const },
  { title: 'Size', key: 'size', width: '140px', sortable: true, align: 'end' as const },
  { title: 'Status', key: 'in_use', width: '110px', sortable: true, align: 'start' as const },
  { title: 'Actions', key: 'actions', width: '80px', align: 'center' as const, sortable: false },
];

const getUpdateDot = (status?: string) => {
  if (status === 'yes') return 'dot-ok';
  if (status === 'no') return 'dot-err';
  if (status === 'checking') return 'dot-warn';
  return '';
};

const fetchImages = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/images');
    if (response.ok) {
      const data = await response.json();
      images.value = (data || []).map((img: ImageInfo) => ({
        ...img,
        ui_key: `${img.node_id}-${img.id}-${img.repository}-${img.tag}`,
        up_to_date_status: 'unchecked',
      }));
      checkAllImages();
    }
  } catch (error) {
    console.error('Failed to fetch images:', error);
  } finally {
    loading.value = false;
  }
};

const checkUpdate = async (item: ImageInfo) => {
  item.up_to_date_status = 'checking';
  try {
    const response = await fetch(
      `/api/images/check?node_id=${item.node_id}&repository=${encodeURIComponent(item.repository)}&tag=${encodeURIComponent(item.tag)}&id=${item.id}`,
      { method: 'POST' }
    );
    if (response.ok) {
      const data = await response.json();
      item.up_to_date_status = data.up_to_date ? 'yes' : 'no';
    } else {
      item.up_to_date_status = 'error';
      item.check_error = await response.text();
    }
  } catch (error: any) {
    item.up_to_date_status = 'error';
    item.check_error = error.message || 'Network error';
  }
};

const checkAllImages = async () => {
  checkingAll.value = true;
  const targets = images.value.filter((img) => img.up_to_date_status !== 'checking');
  await Promise.all(targets.map((img) => checkUpdate(img)));
  checkingAll.value = false;
};

const formatSize = (bytes: number): string => {
  if (!bytes) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

const confirmDelete = (img: ImageInfo) => {
  imageToDelete.value = img;
  forceDelete.value = false;
  deleteDialog.value = true;
};

const closeDeleteDialog = () => {
  deleteDialog.value = false;
  imageToDelete.value = null;
  forceDelete.value = false;
};

const deleteImage = async () => {
  if (!imageToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(
      `/api/images?node_id=${imageToDelete.value.node_id}&id=${imageToDelete.value.id}&force=${forceDelete.value}`,
      { method: 'DELETE' }
    );
    if (response.ok) {
      await fetchImages();
      closeDeleteDialog();
    } else {
      alert(`Failed to delete image: ${await response.text()}`);
    }
  } catch (error) {
    alert(`Failed to delete image: ${error}`);
  } finally {
    deleting.value = false;
  }
};

onMounted(fetchImages);
</script>
