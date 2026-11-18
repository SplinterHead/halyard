<template>
  <DataTablePage
    title="Cluster Containers"
    subtitle="All running and exited containers discovered across Swarm nodes."
    :items="containers"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-package-variant-closed-remove"
    empty-title="No containers found"
    empty-description="Your cluster is currently empty. Deploy services or stacks to see containers here."
    search-placeholder="Search containers..."
    :sort-by="[{ key: 'names', order: 'asc' }]"
    clickable-rows
    @refresh="fetchContainers"
    @click:row="goToDetail"
  >
    <template #item.names="{ item }">
      <div class="d-flex flex-column" style="gap: 2px; min-width: 0;">
        <div class="flex-row" style="min-width: 0;">
          <span class="status-dot" :class="getStateDot(item.state)"></span>
          <span class="fw-500 text-1 text-truncate" style="max-width: 520px;" :title="formatNames(item.names)">
            {{ formatNames(item.names) }}
          </span>
        </div>
        <span class="mono text-3 text-truncate" style="max-width: 520px; padding-inline-start: 15px;" :title="item.image">
          {{ formatImage(item.image) }}
        </span>
      </div>
    </template>

    <template #item.up_to_date="{ item }">
      <span class="status-badge">
        <span class="status-dot" :class="item.up_to_date ? 'dot-ok' : 'dot-warn'"></span>
        {{ item.up_to_date ? 'Current' : 'Out of Date' }}
      </span>
    </template>

    <template #item.service="{ item }">
      <div class="d-flex flex-column" style="gap: 2px;">
        <span class="fw-500 text-1 text-truncate" style="max-width: 180px;" :title="item.service">{{ item.service || '—' }}</span>
        <span class="text-3 text-truncate" style="font-size: 12px; max-width: 180px;" :title="item.stack">{{ item.stack || '—' }}</span>
      </div>
    </template>

    <template #item.status="{ value }">
      <span class="text-2 text-truncate d-inline-block" style="max-width: 170px;" :title="value">{{ value }}</span>
    </template>

    <template #item.node="{ value }">
      <span class="text-2 text-truncate d-inline-block" style="max-width: 180px;" :title="value">{{ value }}</span>
    </template>

    <template #item.created_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
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
            <span class="status-dot" :class="getStateDot(item.state)"></span>
            <span class="fw-500 text-1 text-truncate">{{ formatNames(item.names) }}</span>
          </div>
          <span class="text-2" style="font-size: 12px;">{{ formatStateLabel(item.state) }}</span>
        </div>
        <div class="mobile-card-bottom">
          <span><span class="text-3">Node:</span> {{ item.node }}</span>
          <span><span class="text-3">Stack:</span> {{ item.stack || '—' }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import DataTablePage from '../../components/DataTablePage.vue';
import RelativeTime from '../../components/RelativeTime.vue';

const router = useRouter();

interface Container {
  id: string;
  names: string[];
  image: string;
  state: string;
  status: string;
  node: string;
  service: string;
  stack: string;
  up_to_date: boolean;
  created_at: string;
}

const containers = ref<Container[]>([]);
const loading = ref(false);

const headers = [
  { title: 'Container / Image', key: 'names', sortable: true, align: 'start' as const },
  { title: 'Update', key: 'up_to_date', width: '130px', align: 'start' as const },
  { title: 'Service / Stack', key: 'service', width: '200px', sortable: true, align: 'start' as const },
  { title: 'Status', key: 'status', width: '190px', align: 'start' as const },
  { title: 'Node', key: 'node', width: '200px', sortable: true, align: 'start' as const },
  { title: 'Created', key: 'created_at', width: '130px', align: 'end' as const },
];

const getStateDot = (state: string) => {
  switch ((state || '').toLowerCase()) {
    case 'running':
      return 'dot-ok';
    case 'exited':
      return 'dot-err';
    case 'paused':
    case 'restarting':
      return 'dot-warn';
    default:
      return '';
  }
};

const formatStateLabel = (state: string) => {
  if (!state) return 'Unknown';
  return state.charAt(0).toUpperCase() + state.slice(1).toLowerCase();
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

const formatNames = (names: string[]) => {
  if (!names || names.length === 0) return '-';
  return names[0].replace(/^\//, '');
};

const fetchContainers = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/containers');
    if (response.ok) {
      containers.value = (await response.json()) || [];
    }
  } catch (error) {
    console.error('Failed to fetch containers:', error);
  } finally {
    loading.value = false;
  }
};

const goToDetail = (_event: any, { item }: any) => {
  router.push({
    name: 'container-detail',
    params: { id: item.id },
    query: { node: item.node },
  });
};

onMounted(fetchContainers);
</script>
