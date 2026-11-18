<template>
  <DataTablePage
    title="Swarm Configs"
    subtitle="Non-sensitive Swarm configuration files mounted into cluster services."
    :items="configs"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-code-braces"
    empty-title="No Configs Found"
    empty-description="Swarm configurations allow you to store non-sensitive data, such as configuration files, in the swarm."
    search-placeholder="Search configs..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    clickable-rows
    @refresh="fetchConfigs"
    @click:row="openViewDialog"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" @click="dialog = true">
        + Create Config
      </button>
    </template>

    <template #top>
      <!-- Create Config Modal -->
      <AppModal v-model="dialog" title="Create Swarm Config" max-width="560px" :persistent="saving">
        <div class="form-field">
          <label class="form-label" for="config-name">Config Name</label>
          <input
            id="config-name"
            v-model="newConfig.name"
            type="text"
            class="form-input"
            placeholder="app_config"
          />
        </div>

        <div class="form-field">
          <label class="form-label" for="config-data">Config Data</label>
          <textarea
            id="config-data"
            v-model="newConfig.data"
            class="form-textarea mono"
            rows="6"
            placeholder="Paste your configuration content here..."
          ></textarea>
        </div>

        <template #footer>
          <button type="button" class="btn" :disabled="saving" @click="closeDialog">Cancel</button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="saving || !newConfig.name.trim() || !newConfig.data"
            @click="createConfig"
          >
            {{ saving ? 'Creating...' : 'Create' }}
          </button>
        </template>
      </AppModal>

      <!-- Delete Confirmation Modal -->
      <AppModal v-model="deleteDialog" title="Remove Config?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove the configuration
          <strong class="text-1">{{ configToDelete?.name }}</strong>?
          This will fail if it is currently in use by any services.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteConfig">
            {{ deleting ? 'Removing...' : 'Remove' }}
          </button>
        </template>
      </AppModal>

      <!-- View Config Modal -->
      <AppModal v-model="viewDialog" :title="`View Config: ${viewedConfig?.name || ''}`" max-width="760px">
        <Loader :loading="loadingView" />
        <pre
          v-if="!loadingView && viewedConfigData"
          class="log-viewport"
          style="margin: 0; max-block-size: 60dvh; white-space: pre-wrap; word-break: break-all;"
        >{{ viewedConfigData }}</pre>
        <div v-else-if="!loadingView && !viewedConfigData" class="text-center text-3 py-6">
          No data or unable to load.
        </div>
        <template #footer>
          <button type="button" class="btn" @click="viewDialog = false">Close</button>
        </template>
      </AppModal>
    </template>

    <template #item.name="{ value }">
      <span class="fw-500 text-1">{{ formatConfigName(value) }}</span>
    </template>

    <template #item.id="{ value }">
      <span class="mono text-2">{{ value.substring(0, 12) }}</span>
    </template>

    <template #item.created_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
      </div>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center">
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Remove Config"
          aria-label="Remove Config"
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
        @click="openViewDialog($event, { item })"
        @keydown.enter="openViewDialog($event, { item })"
      >
        <div class="mobile-card-top">
          <span class="fw-500 text-1 text-truncate">{{ formatConfigName(item.name) }}</span>
          <button
            type="button"
            class="btn btn-icon btn-sm btn-danger-subtle"
            title="Remove Config"
            aria-label="Remove Config"
            @click.stop="confirmDelete(item)"
          >
            <v-icon size="14">mdi-delete-outline</v-icon>
          </button>
        </div>
        <div class="mobile-card-bottom">
          <span class="mono">{{ item.id.substring(0, 12) }}</span>
          <span><span class="text-3">Created:</span> {{ formatRelativeTime(item.created_at) }}</span>
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
import Loader from '../../components/Loader.vue';
import { formatRelativeTime } from '../../utils/time';

interface Config {
  id: string;
  name: string;
  created_at: string;
}

const formatConfigName = (name: string) => name.replace(/-v\d+$/, '');

const configs = ref<Config[]>([]);
const loading = ref(false);
const dialog = ref(false);
const saving = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const configToDelete = ref<Config | null>(null);
const viewDialog = ref(false);
const loadingView = ref(false);
const viewedConfig = ref<Config | null>(null);
const viewedConfigData = ref('');

const newConfig = ref({
  name: '',
  data: '',
});

const headers = [
  { title: 'Name', key: 'name', sortable: true, align: 'start' as const },
  { title: 'ID', key: 'id', width: '160px', align: 'start' as const },
  { title: 'Created', key: 'created_at', width: '140px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '90px', align: 'center' as const, sortable: false },
];

const fetchConfigs = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/configs');
    if (response.ok) {
      const rawConfigs: Config[] = (await response.json()) || [];
      const grouped = new Map<string, { config: Config; version: number }>();
      for (const c of rawConfigs) {
        const match = c.name.match(/^(.*?)(?:-v(\d+))?$/);
        if (match) {
          const baseName = match[1];
          const version = match[2] ? parseInt(match[2], 10) : 1;
          const existing = grouped.get(baseName);
          if (!existing || version > existing.version) {
            grouped.set(baseName, { config: c, version });
          }
        }
      }
      configs.value = Array.from(grouped.values()).map((v) => v.config);
    }
  } catch (error) {
    console.error('Failed to fetch configs:', error);
  } finally {
    loading.value = false;
  }
};

const createConfig = async () => {
  if (!newConfig.value.name.trim() || !newConfig.value.data) return;
  saving.value = true;
  try {
    const response = await fetch('/api/configs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newConfig.value),
    });
    if (response.ok) {
      await fetchConfigs();
      closeDialog();
    } else {
      alert('Failed to create config: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to create config:', err);
  } finally {
    saving.value = false;
  }
};

const confirmDelete = (config: Config) => {
  configToDelete.value = config;
  deleteDialog.value = true;
};

const deleteConfig = async () => {
  if (!configToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/configs/${configToDelete.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchConfigs();
      deleteDialog.value = false;
    } else {
      alert('Failed to remove config: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to delete config:', err);
  } finally {
    deleting.value = false;
  }
};

const closeDialog = () => {
  dialog.value = false;
  newConfig.value = { name: '', data: '' };
};

const openViewDialog = async (_event: any, { item }: { item: Config }) => {
  viewedConfig.value = item;
  viewedConfigData.value = '';
  viewDialog.value = true;
  loadingView.value = true;
  try {
    const response = await fetch(`/api/configs/${item.id}`);
    if (response.ok) {
      const detail = await response.json();
      let data = detail.data || '';
      try {
        if (data && /^[A-Za-z0-9+/=]+$/.test(data)) {
          const decoded = atob(data);
          if (/^[\x20-\x7E\t\r\n]*$/.test(decoded)) {
            data = decoded;
          }
        }
      } catch {
        // Not base64
      }
      viewedConfigData.value = data || 'No content';
    } else {
      viewedConfigData.value = 'Failed to load configuration data.';
    }
  } catch {
    viewedConfigData.value = 'Error loading configuration data.';
  } finally {
    loadingView.value = false;
  }
};

onMounted(fetchConfigs);
</script>
