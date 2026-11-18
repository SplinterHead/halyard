<template>
  <DataTablePage
    title="Swarm Volumes"
    subtitle="Persistent storage volumes across cluster nodes and active mount status."
    :items="volumes"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-database-off"
    empty-title="No volumes found"
    empty-description="Volumes created by your services or manually will appear here once detected."
    search-placeholder="Search volumes..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    @refresh="fetchVolumes"
  >
    <template #actions>
      <button type="button" class="btn btn-danger-subtle" :disabled="pruning" @click="showPruneDialog = true">
        {{ pruning ? 'Pruning...' : 'Prune Unused' }}
      </button>
    </template>

    <template #top>
      <AppModal v-model="showPruneDialog" title="Prune Unused Volumes?" :persistent="pruning">
        <p class="text-2" style="margin: 0;">
          This will remove ALL volumes on ALL nodes that are not currently used by at least one container.
          This action cannot be undone.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="pruning" @click="showPruneDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="pruning" @click="pruneVolumes">
            {{ pruning ? 'Pruning...' : 'Prune Volumes' }}
          </button>
        </template>
      </AppModal>

      <AppModal v-model="deleteDialog" title="Delete Volume?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove the volume
          <strong class="text-1 mono">{{ volumeToDelete?.name }}</strong>
          on node <strong class="text-1">{{ volumeToDelete?.node }}</strong>? This action cannot be undone.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteVolume">
            {{ deleting ? 'Removing...' : 'Remove Volume' }}
          </button>
        </template>
      </AppModal>

      <AppModal v-model="errorDialog" title="Volume Deletion Failed">
        <p class="text-2" style="margin: 0;">
          The volume could not be deleted. This usually happens if it is still being used by one or more containers on the node.
        </p>
        <pre class="log-viewport text-err" style="margin: 0; white-space: pre-wrap;">{{ errorMessage }}</pre>
        <template #footer>
          <button type="button" class="btn btn-primary" @click="errorDialog = false">Dismiss</button>
        </template>
      </AppModal>

      <VolumeExplorerDialog
        v-model="explorerDialog"
        :volume-name="selectedVolumeName"
        :node-name="selectedNodeName"
      />
    </template>

    <template #item.name="{ item }">
      <div class="flex-row" style="min-width: 0;">
        <span class="status-dot" :class="item.in_use ? 'dot-ok' : ''"></span>
        <span class="fw-500 text-1 text-truncate" style="max-width: 520px;" :title="item.name">{{ item.name }}</span>
      </div>
    </template>

    <template #item.node="{ value }">
      <span class="text-2 text-truncate d-inline-block" style="max-width: 170px;" :title="value">{{ value }}</span>
    </template>

    <template #item.stack="{ value }">
      <span v-if="value && value !== '-'" class="status-badge">{{ value }}</span>
      <span v-else class="text-3">—</span>
    </template>

    <template #item.driver="{ value }">
      <span class="status-badge">{{ value }}</span>
    </template>

    <template #item.type="{ value }">
      <span class="status-badge">{{ value }}</span>
    </template>

    <template #item.created_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
      </div>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center" style="gap: 4px;">
        <button
          type="button"
          class="btn btn-icon btn-sm"
          title="Browse Volume Files"
          aria-label="Browse Volume Files"
          @click.stop="openExplorer(item)"
        >
          <v-icon size="15">mdi-folder-search-outline</v-icon>
        </button>
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Volume"
          aria-label="Delete Volume"
          @click.stop="confirmDelete(item)"
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
            <span class="status-dot" :class="item.in_use ? 'dot-ok' : ''"></span>
            <span class="fw-500 text-1 text-truncate">{{ item.name }}</span>
          </div>
          <div class="flex-row" style="gap: 4px;">
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Browse Volume Files"
              aria-label="Browse Volume Files"
              @click.stop="openExplorer(item)"
            >
              <v-icon size="14">mdi-folder-search-outline</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm btn-danger-subtle"
              title="Delete Volume"
              aria-label="Delete Volume"
              @click.stop="confirmDelete(item)"
            >
              <v-icon size="14">mdi-delete-outline</v-icon>
            </button>
          </div>
        </div>
        <div class="mobile-card-bottom">
          <span><span class="text-3">Node:</span> {{ item.node }}</span>
          <span><span class="text-3">Type:</span> {{ item.type }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import DataTablePage from '../../components/DataTablePage.vue';
import AppModal from '../../components/AppModal.vue';
import RelativeTime from '../../components/RelativeTime.vue';
import VolumeExplorerDialog from '../../components/VolumeExplorerDialog.vue';

interface Volume {
  name: string;
  node: string;
  stack: string;
  driver: string;
  type: string;
  in_use: boolean;
  created_at: string;
  labels: Record<string, string>;
}

const volumes = ref<Volume[]>([]);
const loading = ref(false);
const pruning = ref(false);
const showPruneDialog = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const volumeToDelete = ref<Volume | null>(null);
const errorDialog = ref(false);
const errorMessage = ref('');
const explorerDialog = ref(false);
const selectedVolumeName = ref('');
const selectedNodeName = ref('');

const headers = [
  { title: 'Volume / ID', key: 'name', sortable: true, align: 'start' as const },
  { title: 'Node', key: 'node', width: '185px', sortable: true, align: 'start' as const },
  { title: 'Stack', key: 'stack', sortable: true, width: '150px', align: 'start' as const },
  { title: 'Driver', key: 'driver', width: '110px', align: 'start' as const },
  { title: 'Type', key: 'type', width: '110px', align: 'start' as const },
  { title: 'Created', key: 'created_at', width: '130px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '90px', align: 'center' as const, sortable: false },
];

const pruneVolumes = async () => {
  pruning.value = true;
  showPruneDialog.value = false;
  try {
    const response = await fetch('/api/volumes/prune', { method: 'POST' });
    if (response.ok) {
      await fetchVolumes();
    }
  } catch (error) {
    console.error('Failed to prune volumes:', error);
  } finally {
    pruning.value = false;
  }
};

const fetchVolumes = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/volumes');
    if (response.ok) {
      const data = await response.json();
      volumes.value = (data || []).filter((v: Volume) => v.name && v.node);
    }
  } catch (error) {
    console.error('Failed to fetch volumes:', error);
  } finally {
    loading.value = false;
  }
};

const confirmDelete = (volume: Volume) => {
  volumeToDelete.value = volume;
  deleteDialog.value = true;
};

const openExplorer = (volume: Volume) => {
  selectedVolumeName.value = volume.name;
  selectedNodeName.value = volume.node;
  explorerDialog.value = true;
};

const deleteVolume = async () => {
  if (!volumeToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(
      `/api/volumes?node=${encodeURIComponent(volumeToDelete.value.node)}&name=${encodeURIComponent(volumeToDelete.value.name)}`,
      { method: 'DELETE' }
    );
    if (response.ok) {
      await fetchVolumes();
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
    volumeToDelete.value = null;
  }
};

onMounted(fetchVolumes);
</script>
