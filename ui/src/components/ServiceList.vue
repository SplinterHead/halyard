<template>
  <DataTablePage
    title="Swarm Services"
    subtitle="Replicated and global Docker Swarm services across the cluster."
    :items="services"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-server-off"
    empty-title="No services found"
    empty-description="Deploy services to your cluster to manage and monitor them here."
    search-placeholder="Search services..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    clickable-rows
    @refresh="fetchServices"
    @click:row="goToDetail"
  >
    <template #top>
      <AppModal v-model="deleteDialog" title="Delete Service?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove the service
          <strong class="text-1">{{ serviceToDelete?.name }}</strong>?
          This will permanently stop and remove all tasks for this service.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">
            Cancel
          </button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteService">
            {{ deleting ? 'Removing...' : 'Remove Service' }}
          </button>
        </template>
      </AppModal>

      <AppModal v-model="errorDialog" title="Service Deletion Failed">
        <p class="text-2" style="margin: 0;">The service could not be deleted.</p>
        <pre class="log-viewport text-err" style="margin: 0; white-space: pre-wrap;">{{ errorMessage }}</pre>
        <template #footer>
          <button type="button" class="btn btn-primary" @click="errorDialog = false">Dismiss</button>
        </template>
      </AppModal>
    </template>

    <template #item.name="{ item }">
      <div class="d-flex flex-column" style="gap: 2px;">
        <div class="flex-row">
          <span
            class="status-dot"
            :class="item.running === item.replicas ? 'dot-ok' : (item.running === 0 && item.replicas > 0 ? 'dot-err' : 'dot-warn')"
          ></span>
          <span class="fw-500 text-1">{{ item.name }}</span>
        </div>
        <span class="text-3" style="font-size: 12px; padding-inline-start: 15px;">
          {{ item.stack || 'No stack' }}
        </span>
      </div>
    </template>

    <template #item.replicas="{ item }">
      <div class="flex-row">
        <v-icon
          size="15"
          class="text-2"
          :title="item.mode"
          :aria-label="item.mode"
        >
          {{ item.mode === 'global' ? 'mdi-earth' : 'mdi-view-grid-outline' }}
        </v-icon>
        <span class="status-badge">
          <span class="status-dot" :class="item.running === item.replicas ? 'dot-ok' : 'dot-warn'"></span>
          {{ item.running }} / {{ item.replicas }}
        </span>
      </div>
    </template>

    <template #item.image="{ value }">
      <span class="mono text-2 text-truncate d-inline-block" style="max-width: 240px;" :title="value">
        {{ formatImage(value) }}
      </span>
    </template>

    <template #item.ports="{ value }">
      <div v-if="value && value.length" class="flex-row" style="flex-wrap: wrap; gap: 4px;">
        <span v-for="port in value" :key="port" class="status-badge mono">{{ port }}</span>
      </div>
      <span v-else class="text-3">—</span>
    </template>

    <template #item.updated_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
      </div>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center">
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Service"
          aria-label="Delete Service"
          @click.stop="confirmDelete(item)"
        >
          <v-icon size="15">mdi-delete-outline</v-icon>
        </button>
      </div>
    </template>

    <!-- Mobile Curated 2-Row Card -->
    <template #mobile-item="{ item }">
      <div
        class="mobile-item-card"
        role="button"
        tabindex="0"
        @click="goToDetail($event, { item })"
        @keydown.enter="goToDetail($event, { item })"
      >
        <div class="mobile-card-top">
          <div class="flex-row" style="min-width: 0;">
            <span
              class="status-dot"
              :class="item.running === item.replicas ? 'dot-ok' : (item.running === 0 && item.replicas > 0 ? 'dot-err' : 'dot-warn')"
            ></span>
            <span class="fw-500 text-1 text-truncate">{{ item.name }}</span>
          </div>
          <span class="status-badge">{{ item.running }}/{{ item.replicas }}</span>
        </div>
        <div class="mobile-card-bottom">
          <span><span class="text-3">Stack:</span> {{ item.stack || '—' }}</span>
          <span><span class="text-3">Updated:</span> {{ formatRelativeTime(item.updated_at) }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import DataTablePage from './DataTablePage.vue';
import AppModal from './AppModal.vue';
import RelativeTime from './RelativeTime.vue';
import { formatRelativeTime } from '../utils/time';

const router = useRouter();

interface Service {
  id: string;
  name: string;
  stack: string;
  image: string;
  mode: string;
  replicas: number;
  running: number;
  ports: string[];
  updated_at: string;
}

const services = ref<Service[]>([]);
const loading = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const serviceToDelete = ref<Service | null>(null);
const errorDialog = ref(false);
const errorMessage = ref('');

const headers = [
  { title: 'Service', key: 'name', sortable: true, align: 'start' as const },
  { title: 'Replicas', key: 'replicas', width: '140px', align: 'start' as const },
  { title: 'Image', key: 'image', width: '260px', sortable: true, align: 'start' as const },
  { title: 'Ports', key: 'ports', width: '180px', align: 'start' as const },
  { title: 'Updated', key: 'updated_at', width: '130px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '80px', align: 'center' as const, sortable: false },
];

const goToDetail = (_: any, { item }: any) => {
  router.push(`/swarm/services/${item.id}`);
};

const formatImage = (image: string) => {
  if (!image) return '-';
  let clean = image.split('@')[0];
  const slashParts = clean.split('/');
  const lastPart = slashParts[slashParts.length - 1];
  if (lastPart.includes(':')) {
    const colonParts = lastPart.split(':');
    slashParts[slashParts.length - 1] = colonParts.slice(0, -1).join(':');
    clean = slashParts.join('/');
  }
  return clean;
};

const fetchServices = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/services');
    if (response.ok) {
      services.value = (await response.json()) || [];
    }
  } catch (error) {
    console.error('Failed to fetch services:', error);
  } finally {
    loading.value = false;
  }
};

const confirmDelete = (service: Service) => {
  serviceToDelete.value = service;
  deleteDialog.value = true;
};

const deleteService = async () => {
  if (!serviceToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/services?id=${encodeURIComponent(serviceToDelete.value.id)}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchServices();
      deleteDialog.value = false;
    } else {
      errorMessage.value = await response.text();
      errorDialog.value = true;
    }
  } catch (err) {
    errorMessage.value = String(err);
    errorDialog.value = true;
  } finally {
    deleting.value = false;
    serviceToDelete.value = null;
  }
};

onMounted(fetchServices);
</script>
