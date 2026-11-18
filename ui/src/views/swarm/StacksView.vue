<template>
  <DataTablePage
    title="Swarm Stacks"
    subtitle="Deployed Docker Swarm stacks and aggregate replica health."
    :items="stacks"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-layers-off-outline"
    empty-title="No stacks deployed"
    empty-description="Deploy your first docker-compose stack to see it managed here."
    search-placeholder="Search stacks..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    clickable-rows
    @refresh="fetchStacks"
    @click:row="goToDetail"
  >
    <template #top>
      <AppModal v-model="deleteDialog" title="Delete Stack?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove the stack
          <strong class="text-1">{{ stackToDelete?.name }}</strong>?
          This will stop and remove all associated services.
        </p>

        <label class="nested-box flex-between" style="cursor: pointer;">
          <span>
            <strong class="fw-500 text-1" style="display: block;">Delete associated volumes</strong>
            <span class="text-2" style="font-size: 12px;">Permanently remove volumes created for this stack</span>
          </span>
          <input v-model="deleteVolumes" type="checkbox" role="switch" class="toggle-switch" />
        </label>

        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">
            Cancel
          </button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteStack">
            {{ deleting ? 'Removing...' : 'Remove Stack' }}
          </button>
        </template>
      </AppModal>
    </template>

    <template #item.name="{ item }">
      <div class="flex-row">
        <span class="status-dot" :class="getStatusDot(item.status)"></span>
        <span class="fw-500 text-1">{{ item.name }}</span>
      </div>
    </template>

    <template #item.services="{ item }">
      <span class="text-2">{{ item.services }}</span>
    </template>

    <template #item.updated_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
      </div>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center" style="gap: 4px;">
        <button
          v-if="item.name !== 'Orphaned/Manual'"
          type="button"
          class="btn btn-icon btn-sm"
          title="Force Restart"
          aria-label="Force Restart"
          :disabled="restartingStack === item.name"
          @click.stop="restartStack(item.name)"
        >
          <v-icon size="15">mdi-restart</v-icon>
        </button>
        <button
          v-if="item.name !== 'Orphaned/Manual'"
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Stack"
          aria-label="Delete Stack"
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
            <span class="status-dot" :class="getStatusDot(item.status)"></span>
            <span class="fw-500 text-1 text-truncate">{{ item.name }}</span>
          </div>
          <div v-if="item.name !== 'Orphaned/Manual'" class="flex-row" style="gap: 4px;">
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Force Restart"
              aria-label="Force Restart"
              :disabled="restartingStack === item.name"
              @click.stop="restartStack(item.name)"
            >
              <v-icon size="14">mdi-restart</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm btn-danger-subtle"
              title="Delete Stack"
              aria-label="Delete Stack"
              @click.stop="confirmDelete(item)"
            >
              <v-icon size="14">mdi-delete-outline</v-icon>
            </button>
          </div>
        </div>
        <div class="mobile-card-bottom">
          <span><span class="text-3">Services:</span> {{ item.services }}</span>
          <span><span class="text-3">Updated:</span> {{ formatRelativeTime(item.updated_at) }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import DataTablePage from '../../components/DataTablePage.vue';
import AppModal from '../../components/AppModal.vue';
import RelativeTime from '../../components/RelativeTime.vue';
import { formatRelativeTime } from '../../utils/time';

const router = useRouter();

interface Stack {
  name: string;
  services: number;
  total_replicas: number;
  running_replicas: number;
  status: string;
  updated_at: string;
}

const stacks = ref<Stack[]>([]);
const loading = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const stackToDelete = ref<Stack | null>(null);
const restartingStack = ref<string | null>(null);
const deleteVolumes = ref(false);

const headers = [
  { title: 'Stack Name', key: 'name', sortable: true, align: 'start' as const },
  { title: 'Services', key: 'services', width: '130px', align: 'start' as const },
  { title: 'Updated', key: 'updated_at', width: '140px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '100px', align: 'center' as const, sortable: false },
];

const getStatusDot = (status: string) => {
  if (status === 'Healthy') return 'dot-ok';
  if (status === 'Degraded') return 'dot-warn';
  if (status === 'Updating') return 'dot-warn';
  return '';
};

const goToDetail = (_: any, { item }: any) => {
  router.push(`/swarm/stacks/${item.name}`);
};

const fetchStacks = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/stacks');
    if (response.ok) {
      stacks.value = (await response.json()) || [];
    }
  } catch (error) {
    console.error('Failed to fetch stacks:', error);
  } finally {
    loading.value = false;
  }
};

const confirmDelete = (stack: Stack) => {
  stackToDelete.value = stack;
  deleteVolumes.value = false;
  deleteDialog.value = true;
};

const restartStack = async (name: string) => {
  restartingStack.value = name;
  try {
    const response = await fetch(`/api/stacks/restart?name=${encodeURIComponent(name)}`, {
      method: 'POST',
    });
    if (!response.ok) {
      alert('Failed to restart stack: ' + (await response.text()));
    } else {
      await fetchStacks();
    }
  } catch (err) {
    console.error('Failed to restart stack:', err);
  } finally {
    restartingStack.value = null;
  }
};

const deleteStack = async () => {
  if (!stackToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(
      `/api/stacks/${encodeURIComponent(stackToDelete.value.name)}?deleteVolumes=${deleteVolumes.value}`,
      { method: 'DELETE' }
    );
    if (response.ok) {
      await fetchStacks();
      deleteDialog.value = false;
    } else {
      alert('Failed to remove stack: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to delete stack:', err);
  } finally {
    deleting.value = false;
  }
};

onMounted(fetchStacks);
</script>
