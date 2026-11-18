<template>
  <DataTablePage
    title="Git Repositories"
    subtitle="Remote Git repositories connected for automated Swarm stack synchronization."
    :items="repos"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-git"
    empty-title="No Repositories Found"
    empty-description="Add a Git repository to start syncing your stacks with source code and automate your deployments."
    search-placeholder="Search repositories..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    @refresh="fetchRepos"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" @click="openAddDialog">
        + Add Repository
      </button>
    </template>

    <template #top>
      <!-- Add/Edit Repo Modal -->
      <AppModal
        v-model="dialog"
        :title="`${isEdit ? 'Edit' : 'Add'} Git Repository`"
        max-width="560px"
        :persistent="saving"
      >
        <div class="form-field">
          <label class="form-label" for="repo-name">Friendly Name</label>
          <input
            id="repo-name"
            v-model="editedRepo.name"
            type="text"
            class="form-input"
            placeholder="e.g. Production Stack"
          />
        </div>

        <div class="form-field">
          <label class="form-label" for="repo-url">Git URL</label>
          <input
            id="repo-url"
            v-model="editedRepo.url"
            type="text"
            class="form-input mono"
            placeholder="https://github.com/org/repo.git"
          />
        </div>

        <div class="config-grid">
          <div class="form-field">
            <label class="form-label" for="repo-user">Username (Optional)</label>
            <input
              id="repo-user"
              v-model="editedRepo.username"
              type="text"
              class="form-input"
              placeholder="Optional"
            />
          </div>
          <div class="form-field">
            <label class="form-label" for="repo-token">Token / Password (Optional)</label>
            <input
              id="repo-token"
              v-model="editedRepo.token"
              type="password"
              class="form-input"
              placeholder="Optional"
            />
          </div>
        </div>

        <div class="form-field">
          <label class="form-label" for="repo-desc">Description</label>
          <textarea
            id="repo-desc"
            v-model="editedRepo.description"
            class="form-textarea"
            rows="2"
            style="min-height: 64px;"
          ></textarea>
        </div>

        <div v-if="error" class="alert-banner alert-err">
          <span>{{ error }}</span>
        </div>

        <template #footer>
          <button type="button" class="btn" :disabled="saving" @click="closeDialog">Cancel</button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="saving || !editedRepo.name.trim() || !editedRepo.url.trim()"
            @click="saveRepo"
          >
            {{ saving ? 'Saving...' : (isEdit ? 'Save' : 'Add') }}
          </button>
        </template>
      </AppModal>

      <!-- Delete Confirmation Modal -->
      <AppModal v-model="deleteDialog" title="Delete Repository?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          This will also delete all Git Syncs associated with
          <strong class="text-1">{{ repoToDelete?.name }}</strong>. This action cannot be undone.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteRepo">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </template>
      </AppModal>
    </template>

    <template #item.name="{ item }">
      <div class="d-flex flex-column" style="gap: 2px;">
        <div class="flex-row">
          <span class="status-dot" :class="item.last_status === 'Failed' ? 'dot-err' : 'dot-ok'"></span>
          <span class="fw-500 text-1">{{ item.name }}</span>
        </div>
        <span class="text-3" style="font-size: 12px; padding-inline-start: 15px; overflow-wrap: anywhere;">
          {{ item.description || 'No description' }}
        </span>
      </div>
    </template>

    <template #item.url="{ value }">
      <span class="mono text-2" style="overflow-wrap: anywhere;">
        {{ value }}
      </span>
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
          title="Test Connection"
          aria-label="Test Connection"
          :disabled="testingId === item.id"
          @click="testConnection(item)"
        >
          <v-icon size="15">mdi-connection</v-icon>
        </button>
        <button
          type="button"
          class="btn btn-icon btn-sm"
          title="Edit Repository"
          aria-label="Edit Repository"
          @click="openEditDialog(item)"
        >
          <v-icon size="15">mdi-pencil-outline</v-icon>
        </button>
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Repository"
          aria-label="Delete Repository"
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
            <span class="status-dot" :class="item.last_status === 'Failed' ? 'dot-err' : 'dot-ok'"></span>
            <span class="fw-500 text-1 text-truncate">{{ item.name }}</span>
          </div>
          <div class="flex-row" style="gap: 4px;">
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Test Connection"
              aria-label="Test Connection"
              :disabled="testingId === item.id"
              @click="testConnection(item)"
            >
              <v-icon size="14">mdi-connection</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Edit Repository"
              aria-label="Edit Repository"
              @click="openEditDialog(item)"
            >
              <v-icon size="14">mdi-pencil-outline</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm btn-danger-subtle"
              title="Delete Repository"
              aria-label="Delete Repository"
              @click="confirmDelete(item)"
            >
              <v-icon size="14">mdi-delete-outline</v-icon>
            </button>
          </div>
        </div>
        <div class="mobile-card-bottom">
          <span class="mono text-truncate">{{ item.url }}</span>
          <span class="flex-shrink-0">{{ formatRelativeTime(item.created_at || '') }}</span>
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
import { formatRelativeTime } from '../../utils/time';

interface GitRepo {
  id?: string;
  name: string;
  url: string;
  username: string;
  token: string;
  description: string;
  last_status: string;
  last_error: string;
  created_at?: string;
}

const repos = ref<GitRepo[]>([]);
const loading = ref(false);
const dialog = ref(false);
const deleteDialog = ref(false);
const saving = ref(false);
const deleting = ref(false);
const isEdit = ref(false);
const error = ref('');
const repoToDelete = ref<GitRepo | null>(null);
const testingId = ref<string | null>(null);

const editedRepo = ref<GitRepo>({
  name: '',
  url: '',
  username: '',
  token: '',
  description: '',
  last_status: '',
  last_error: '',
  created_at: '',
});

const headers = [
  { title: 'Repository', key: 'name', sortable: true, align: 'start' as const },
  { title: 'URL', key: 'url', minWidth: '280px', sortable: true, align: 'start' as const },
  { title: 'Created', key: 'created_at', width: '140px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '120px', align: 'center' as const, sortable: false },
];

const fetchRepos = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/repos');
    if (response.ok) {
      repos.value = (await response.json()) || [];
    }
  } catch (err) {
    console.error('Failed to fetch repos:', err);
  } finally {
    loading.value = false;
  }
};

const openAddDialog = () => {
  isEdit.value = false;
  resetEditedRepo();
  dialog.value = true;
};

const openEditDialog = (repo: GitRepo) => {
  isEdit.value = true;
  editedRepo.value = { ...repo };
  dialog.value = true;
};

const confirmDelete = (repo: GitRepo) => {
  repoToDelete.value = repo;
  deleteDialog.value = true;
};

const resetEditedRepo = () => {
  editedRepo.value = {
    name: '',
    url: '',
    username: '',
    token: '',
    description: '',
    last_status: '',
    last_error: '',
    created_at: '',
  };
};

const saveRepo = async () => {
  if (!editedRepo.value.name.trim() || !editedRepo.value.url.trim()) return;
  saving.value = true;
  error.value = '';
  try {
    const url = isEdit.value ? `/api/repos/${editedRepo.value.id}` : '/api/repos';
    const method = isEdit.value ? 'PUT' : 'POST';
    const payload = { ...editedRepo.value };
    if (!isEdit.value) {
      delete payload.id;
      delete payload.created_at;
    }

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    const result = await response.json();
    if (result.last_status === 'Failed') {
      error.value = result.last_error || 'Connection failed';
      await fetchRepos();
      return;
    }

    await fetchRepos();
    closeDialog();
  } catch (err: any) {
    error.value = err.message;
  } finally {
    saving.value = false;
  }
};

const deleteRepo = async () => {
  if (!repoToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/repos/${repoToDelete.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchRepos();
      deleteDialog.value = false;
    }
  } catch (err) {
    console.error('Failed to delete repo:', err);
  } finally {
    deleting.value = false;
  }
};

const closeDialog = () => {
  dialog.value = false;
  error.value = '';
  resetEditedRepo();
};

const testConnection = async (repo: GitRepo) => {
  if (!repo.id) return;
  testingId.value = repo.id;
  try {
    await fetch(`/api/repos/${repo.id}/test`, { method: 'POST' });
    await fetchRepos();
  } catch (err) {
    console.error('Failed to test connection:', err);
  } finally {
    testingId.value = null;
  }
};

onMounted(fetchRepos);
</script>
