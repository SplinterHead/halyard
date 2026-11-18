<template>
  <DataTablePage
    title="Git Syncs"
    subtitle="Automated Swarm stack reconciliation from Git repository compose definitions."
    :items="syncs"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-sync-off"
    empty-title="No Git Syncs Found"
    empty-description="Create a Git sync to automate the deployment and periodic reconciliation of your Swarm stacks."
    search-placeholder="Search syncs..."
    :sort-by="[{ key: 'stack_name', order: 'asc' }]"
    @refresh="fetchSyncs"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" @click="openAddDialog">
        + Add Git Sync
      </button>
    </template>

    <template #top>
      <!-- Add/Edit Sync Modal -->
      <AppModal
        v-model="dialog"
        :title="`${isEdit ? 'Edit' : 'Add'} Git Sync`"
        max-width="580px"
        :persistent="saving"
      >
        <div class="form-field">
          <label class="form-label" for="sync-stack">Stack Name</label>
          <input
            id="sync-stack"
            v-model="editedSync.stack_name"
            type="text"
            class="form-input"
            placeholder="my-app"
          />
          <span v-if="editedSync.stack_name && !stackNameValid" class="form-error">
            Use lowercase letters, numbers, and dashes only.
          </span>
          <span v-else class="form-hint">Lowercase letters, numbers, and dashes only.</span>
        </div>

        <div class="form-field">
          <label class="form-label" for="sync-repo">Git Repository</label>
          <select
            id="sync-repo"
            v-model="editedSync.repository_id"
            class="form-select"
            @change="fetchBranches(editedSync.repository_id)"
          >
            <option value="" disabled>Select repository...</option>
            <option v-for="repo in repos" :key="repo.id" :value="repo.id">{{ repo.name }}</option>
          </select>
        </div>

        <div class="form-field">
          <label class="form-label" for="sync-branch">Branch</label>
          <select
            id="sync-branch"
            v-model="editedSync.branch"
            class="form-select"
            :disabled="!editedSync.repository_id || loadingBranches"
          >
            <option value="" disabled>{{ loadingBranches ? 'Loading branches...' : 'Select branch...' }}</option>
            <option v-for="br in branches" :key="br" :value="br">{{ br }}</option>
          </select>
        </div>

        <div class="form-field">
          <label class="form-label" for="sync-path">Compose File Path</label>
          <div class="flex-row">
            <input
              id="sync-path"
              v-model="editedSync.path"
              type="text"
              class="form-input mono"
              placeholder="compose.yml"
              :disabled="!editedSync.repository_id || !editedSync.branch"
            />
            <button
              type="button"
              class="btn"
              :disabled="!editedSync.repository_id || !editedSync.branch"
              @click="openFileBrowser"
            >
              Browse
            </button>
          </div>
        </div>

        <div class="config-grid">
          <label class="nested-box flex-between" style="cursor: pointer;">
            <span>
              <strong class="fw-500 text-1" style="display: block;">Additional Files</strong>
              <span class="text-2" style="font-size: 12px;">Pull sibling files like .env</span>
            </span>
            <input v-model="editedSync.pull_additional_files" type="checkbox" role="switch" class="toggle-switch" />
          </label>
          <label class="nested-box flex-between" style="cursor: pointer;">
            <span>
              <strong class="fw-500 text-1" style="display: block;">Auto Sync</strong>
              <span class="text-2" style="font-size: 12px;">Reconcile periodically</span>
            </span>
            <input v-model="editedSync.auto_sync" type="checkbox" role="switch" class="toggle-switch" />
          </label>
        </div>

        <div v-if="error" class="alert-banner alert-err">
          <span>{{ error }}</span>
        </div>

        <template #footer>
          <button type="button" class="btn" :disabled="saving" @click="closeDialog">Cancel</button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="saving || !isFormValid"
            @click="saveSync"
          >
            {{ saving ? 'Saving...' : (isEdit ? 'Save' : 'Add') }}
          </button>
        </template>
      </AppModal>

      <!-- Delete Confirmation Modal -->
      <AppModal v-model="deleteDialog" title="Delete Git Sync?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          This will stop automatic updates for <strong class="text-1">{{ syncToDelete?.stack_name }}</strong>.
          The existing stack in the Swarm will not be removed.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteSync">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </template>
      </AppModal>

      <!-- File Browser Modal -->
      <AppModal v-model="fileBrowserDialog" title="Browse Repository" max-width="520px">
        <div class="nested-box flex-between">
          <span class="mono text-2">/{{ currentPath }}</span>
          <button type="button" class="btn btn-sm" :disabled="currentPath === ''" @click="goUp">
            Up
          </button>
        </div>

        <Loader :loading="loadingFiles" />

        <div v-if="!loadingFiles" class="surface-card" style="max-block-size: 360px; overflow-y: auto;">
          <div v-if="currentPath !== ''" class="workload-row" @click="goUp">
            <div class="flex-row">
              <v-icon size="16" class="text-2">mdi-folder-upload-outline</v-icon>
              <span class="mono fw-500">..</span>
            </div>
          </div>

          <div
            v-for="file in repoFiles"
            :key="file.path"
            class="workload-row"
            :style="!file.is_dir && !isYamlFile(file.name) ? 'opacity: 0.45; cursor: not-allowed;' : ''"
            @click="(file.is_dir || isYamlFile(file.name)) && selectFile(file)"
          >
            <div class="flex-row">
              <v-icon size="16" :class="file.is_dir ? 'text-accent' : 'text-2'">
                {{ file.is_dir ? 'mdi-folder' : 'mdi-file-code-outline' }}
              </v-icon>
              <span class="mono" :class="file.is_dir ? 'fw-500 text-1' : 'text-2'">{{ file.name }}</span>
            </div>
            <span v-if="!file.is_dir && isYamlFile(file.name)" class="status-badge">YAML</span>
          </div>

          <div v-if="repoFiles.length === 0" class="card-body text-center text-3">
            No files found in this directory.
          </div>
        </div>

        <template #footer>
          <button type="button" class="btn" @click="fileBrowserDialog = false">Cancel</button>
        </template>
      </AppModal>
    </template>

    <template #item.stack_name="{ item }">
      <div class="flex-row">
        <span class="status-dot" :class="getSyncDot(item.last_status)"></span>
        <span class="fw-500" :class="item.last_status === 'Out of Date' ? 'text-warn' : 'text-1'">
          {{ item.stack_name }}
        </span>
      </div>
    </template>

    <template #item.repository_id="{ item }">
      <div class="d-flex flex-column" style="gap: 2px;">
        <span class="fw-500 text-1">{{ getRepoName(item.repository_id) }}</span>
        <span class="mono text-3" style="font-size: 11.5px;">{{ item.branch }}</span>
      </div>
    </template>

    <template #item.last_applied_sha="{ value }">
      <span v-if="value" class="status-badge mono">{{ value.substring(0, 7) }}</span>
      <span v-else class="text-3">Never</span>
    </template>

    <template #item.path="{ value }">
      <span class="mono text-2">{{ value }}</span>
    </template>

    <template #item.pull_additional_files="{ value }">
      <span class="status-badge">{{ value ? 'Enabled' : 'Disabled' }}</span>
    </template>

    <template #item.auto_sync="{ value }">
      <span class="status-badge">{{ value ? 'Enabled' : 'Disabled' }}</span>
    </template>

    <template #item.last_sync_at="{ value }">
      <div class="text-end">
        <RelativeTime v-if="value && value !== '0001-01-01T00:00:00Z'" :value="value" />
        <span v-else class="text-3">Never</span>
      </div>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center" style="gap: 4px;">
        <button
          type="button"
          class="btn btn-icon btn-sm"
          title="Force Sync Now"
          aria-label="Force Sync Now"
          :disabled="syncingId === item.id"
          @click="forceSync(item)"
        >
          <v-icon size="15">mdi-sync</v-icon>
        </button>
        <button
          type="button"
          class="btn btn-icon btn-sm"
          title="Edit Sync"
          aria-label="Edit Sync"
          @click="openEditDialog(item)"
        >
          <v-icon size="15">mdi-pencil-outline</v-icon>
        </button>
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Sync"
          aria-label="Delete Sync"
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
            <span class="status-dot" :class="getSyncDot(item.last_status)"></span>
            <span class="fw-500 text-1 text-truncate">{{ item.stack_name }}</span>
          </div>
          <div class="flex-row" style="gap: 4px;">
            <span v-if="item.last_applied_sha" class="status-badge mono">
              {{ item.last_applied_sha.substring(0, 7) }}
            </span>
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Force Sync Now"
              aria-label="Force Sync Now"
              :disabled="syncingId === item.id"
              @click="forceSync(item)"
            >
              <v-icon size="14">mdi-sync</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Edit Sync"
              aria-label="Edit Sync"
              @click="openEditDialog(item)"
            >
              <v-icon size="14">mdi-pencil-outline</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm btn-danger-subtle"
              title="Delete Sync"
              aria-label="Delete Sync"
              @click="confirmDelete(item)"
            >
              <v-icon size="14">mdi-delete-outline</v-icon>
            </button>
          </div>
        </div>
        <div class="mobile-card-bottom">
          <span class="mono text-truncate">{{ item.path }}</span>
          <span class="flex-shrink-0">
            {{ item.last_sync_at && item.last_sync_at !== '0001-01-01T00:00:00Z' ? formatRelativeTime(item.last_sync_at) : 'Never' }}
          </span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import DataTablePage from '../../components/DataTablePage.vue';
import AppModal from '../../components/AppModal.vue';
import Loader from '../../components/Loader.vue';
import RelativeTime from '../../components/RelativeTime.vue';
import { formatRelativeTime } from '../../utils/time';

interface GitRepo {
  id: string;
  name: string;
}

interface GitSync {
  id?: string;
  name: string;
  repository_id: string;
  stack_name: string;
  branch: string;
  path: string;
  pull_additional_files: boolean;
  auto_sync: boolean;
  last_applied_sha: string;
  last_sync_at?: string | null;
  last_status: string;
  last_error: string;
  created_at?: string | null;
}

const syncs = ref<GitSync[]>([]);
const repos = ref<GitRepo[]>([]);
const branches = ref<string[]>([]);
const loading = ref(false);
const loadingBranches = ref(false);
const dialog = ref(false);
const deleteDialog = ref(false);
const saving = ref(false);
const deleting = ref(false);
const syncingId = ref<string | null>(null);
const isEdit = ref(false);
const error = ref('');
const syncToDelete = ref<GitSync | null>(null);
const fileBrowserDialog = ref(false);
const loadingFiles = ref(false);
const repoFiles = ref<any[]>([]);
const currentPath = ref('');

const editedSync = ref<GitSync>({
  name: '',
  repository_id: '',
  stack_name: '',
  branch: '',
  path: 'compose.yml',
  pull_additional_files: true,
  auto_sync: true,
  last_applied_sha: '',
  last_sync_at: null,
  last_status: 'Ready',
  last_error: '',
  created_at: null,
});

const stackNameValid = computed(() => /^[a-z0-9-]+$/.test(editedSync.value.stack_name));
const isFormValid = computed(() => {
  return Boolean(
    stackNameValid.value &&
      editedSync.value.repository_id &&
      editedSync.value.branch &&
      editedSync.value.path.trim()
  );
});

const headers = [
  { title: 'Stack', key: 'stack_name', sortable: true, align: 'start' as const },
  { title: 'Repository / Branch', key: 'repository_id', width: '190px', align: 'start' as const },
  { title: 'Commit', key: 'last_applied_sha', width: '100px', align: 'start' as const },
  { title: 'File Path', key: 'path', width: '220px', align: 'start' as const },
  { title: 'Pull Files', key: 'pull_additional_files', width: '100px', align: 'start' as const },
  { title: 'Auto Sync', key: 'auto_sync', width: '100px', align: 'start' as const },
  { title: 'Updated', key: 'last_sync_at', width: '130px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '120px', align: 'center' as const, sortable: false },
];

const getSyncDot = (status: string) => {
  if (status === 'Failed') return 'dot-err';
  if (status === 'Out of Date' || status === 'Syncing') return 'dot-warn';
  return 'dot-ok';
};

const fetchRepos = async () => {
  try {
    const response = await fetch('/api/repos');
    if (response.ok) {
      repos.value = (await response.json()) || [];
    }
  } catch (err) {
    console.error('Failed to fetch repos:', err);
  }
};

const fetchBranches = async (repoId: string) => {
  if (!repoId) return;
  loadingBranches.value = true;
  branches.value = [];
  try {
    const response = await fetch(`/api/repos/${repoId}/branches`);
    if (response.ok) {
      branches.value = (await response.json()) || [];
    }
  } catch (err) {
    console.error('Failed to fetch branches:', err);
  } finally {
    loadingBranches.value = false;
  }
};

const fetchSyncs = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/syncs');
    if (response.ok) {
      syncs.value = (await response.json()) || [];
    }
    if (repos.value.length === 0) await fetchRepos();
  } catch (err) {
    console.error('Failed to fetch syncs:', err);
  } finally {
    loading.value = false;
  }
};

const openAddDialog = () => {
  isEdit.value = false;
  resetEditedSync();
  fetchRepos();
  dialog.value = true;
};

const openEditDialog = async (sync: GitSync) => {
  isEdit.value = true;
  editedSync.value = { ...sync };
  await fetchRepos();
  await fetchBranches(sync.repository_id);
  dialog.value = true;
};

const confirmDelete = (sync: GitSync) => {
  syncToDelete.value = sync;
  deleteDialog.value = true;
};

const resetEditedSync = () => {
  editedSync.value = {
    name: '',
    repository_id: '',
    stack_name: '',
    branch: '',
    path: 'compose.yml',
    pull_additional_files: true,
    auto_sync: true,
    last_applied_sha: '',
    last_sync_at: null,
    last_status: 'Ready',
    last_error: '',
    created_at: null,
  };
  branches.value = [];
};

const getRepoName = (id: string) => {
  const repo = repos.value.find((r) => r.id === id);
  return repo ? repo.name : id;
};

const forceSync = async (sync: GitSync) => {
  if (!sync.id) return;
  syncingId.value = sync.id;
  try {
    const response = await fetch(`/api/syncs/${sync.id}/sync`, { method: 'POST' });
    if (!response.ok) {
      alert('Failed to trigger sync: ' + (await response.text()));
    }
    await fetchSyncs();
  } catch (err) {
    console.error('Failed to force sync:', err);
  } finally {
    syncingId.value = null;
  }
};

const saveSync = async () => {
  if (!isFormValid.value) return;
  saving.value = true;
  error.value = '';
  try {
    const url = isEdit.value ? `/api/syncs/${editedSync.value.id}` : '/api/syncs';
    const method = isEdit.value ? 'PUT' : 'POST';

    const payload = {
      ...editedSync.value,
      name: editedSync.value.stack_name,
    };

    if (!isEdit.value) {
      delete payload.id;
      delete payload.created_at;
    }

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const text = await response.text();
      try {
        const data = JSON.parse(text);
        throw new Error(data.message || data.error || 'Failed to save sync');
      } catch {
        throw new Error(text || 'Failed to save sync');
      }
    }

    const result = await response.json();
    if (result.last_status === 'Failed') {
      error.value = result.last_error || 'Validation failed';
      await fetchSyncs();
      return;
    }

    await fetchSyncs();
    closeDialog();
  } catch (err: any) {
    error.value = err.message;
  } finally {
    saving.value = false;
  }
};

const deleteSync = async () => {
  if (!syncToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/syncs/${syncToDelete.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchSyncs();
      deleteDialog.value = false;
    }
  } catch (err) {
    console.error('Failed to delete sync:', err);
  } finally {
    deleting.value = false;
  }
};

const closeDialog = () => {
  dialog.value = false;
  error.value = '';
  resetEditedSync();
};

const openFileBrowser = async () => {
  if (!editedSync.value.repository_id || !editedSync.value.branch) return;
  fileBrowserDialog.value = true;
  currentPath.value = '';
  await fetchFiles('');
};

const fetchFiles = async (path: string) => {
  loadingFiles.value = true;
  try {
    const repoId = editedSync.value.repository_id;
    const branch = editedSync.value.branch;
    const response = await fetch(
      `/api/repos/${repoId}/files?branch=${encodeURIComponent(branch)}&path=${encodeURIComponent(path)}`
    );
    if (response.ok) {
      repoFiles.value = (await response.json()) || [];
      currentPath.value = path;
    }
  } catch (err) {
    console.error('Error fetching files:', err);
  } finally {
    loadingFiles.value = false;
  }
};

const selectFile = async (file: any) => {
  if (file.is_dir) {
    await fetchFiles(file.path);
  } else {
    editedSync.value.path = file.path;
    fileBrowserDialog.value = false;
  }
};

const goUp = async () => {
  if (currentPath.value === '') return;
  const parts = currentPath.value.split('/');
  parts.pop();
  await fetchFiles(parts.join('/'));
};

const isYamlFile = (name: string) => name.endsWith('.yml') || name.endsWith('.yaml');

onMounted(fetchSyncs);
</script>
