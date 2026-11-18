<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">Docker Registries</h1>
        <p class="page-subtitle">Container registry credentials used for automated image pulls and updates.</p>
      </div>
      <div class="header-actions">
        <button type="button" class="btn btn-primary" @click="openAddDialog">
          + Add Registry
        </button>
      </div>
    </div>

    <Loader :loading="loading" />

    <!-- Anonymous Docker Hub Rate Limit Banner -->
    <div v-if="dockerHubLimit && !hasDockerHubCredential" class="alert-banner alert-info">
      <div class="flex-row" style="flex-wrap: wrap;">
        <span class="fw-500">Docker Hub Pull Rate Limit:</span>
        <span class="mono text-1">
          {{ dockerHubLimit.remaining }} / {{ dockerHubLimit.limit }} pulls remaining
        </span>
        <span v-if="dockerHubLimit.reset" class="text-2" style="font-size: 12px;">
          (Resets in {{ formatResetTime(dockerHubLimit.reset) }})
        </span>
      </div>
      <button type="button" class="btn btn-primary btn-sm" @click="addDockerHubPreset">
        Authenticate Docker Hub
      </button>
    </div>

    <!-- Empty State -->
    <div v-if="registries.length === 0 && !loading" class="surface-card card-body text-center" style="padding: 48px 24px;">
      <v-icon size="42" class="text-3 mb-3">mdi-database-lock</v-icon>
      <div class="card-title" style="font-size: 16px;">No Registries Configured</div>
      <p class="page-subtitle" style="margin-block-start: 6px;">
        Add registry credentials to pull private images during automated deployments and reconciliations.
      </p>
    </div>

    <!-- Registry Grid -->
    <div v-else class="registries-grid">
      <section v-for="reg in registries" :key="reg.id" class="surface-card">
        <div class="card-header">
          <div class="flex-row" style="min-width: 0;">
            <v-icon size="18" class="text-accent">{{ getRegistryIcon(reg) }}</v-icon>
            <span class="card-title text-truncate">{{ reg.name }}</span>
          </div>
          <div class="flex-row" style="gap: 4px;">
            <button
              type="button"
              class="btn btn-icon btn-sm"
              title="Edit Registry"
              aria-label="Edit Registry"
              @click="openEditDialog(reg)"
            >
              <v-icon size="14">mdi-pencil-outline</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm btn-danger-subtle"
              title="Delete Registry"
              aria-label="Delete Registry"
              @click="confirmDelete(reg)"
            >
              <v-icon size="14">mdi-delete-outline</v-icon>
            </button>
          </div>
        </div>

        <div class="card-body" style="display: flex; flex-direction: column; gap: 12px;">
          <div class="config-field">
            <span class="config-label">Registry URL</span>
            <span class="config-val mono text-2">{{ reg.url || 'Docker Hub' }}</span>
          </div>
          <div class="config-field">
            <span class="config-label">Username</span>
            <span class="config-val mono">{{ reg.username }}</span>
          </div>

          <div v-if="isDockerHub(reg) && dockerHubLimit" class="nested-box" style="display: flex; flex-direction: column; gap: 8px;">
            <div class="flex-between" style="font-size: 12px;">
              <span class="text-2">Docker Hub Limit</span>
              <span class="mono fw-500" :class="getLimitColorClass">
                {{ dockerHubLimit.remaining }} / {{ dockerHubLimit.limit }} left
              </span>
            </div>
            <div class="telemetry-track">
              <div
                class="telemetry-fill"
                :class="getLimitTelemetryClass"
                :style="{ width: `${Math.min(100, (dockerHubLimit.remaining / dockerHubLimit.limit) * 100)}%` }"
              ></div>
            </div>
            <div class="flex-between" style="font-size: 11.5px;">
              <span v-if="dockerHubLimit.reset" class="text-3">
                Reset in {{ formatResetTime(dockerHubLimit.reset) }}
              </span>
              <span v-else></span>
              <button
                type="button"
                class="btn btn-icon btn-sm"
                :disabled="fetchingLimit"
                aria-label="Refresh rate limit"
                @click="fetchDockerHubLimit"
              >
                <v-icon size="14">mdi-refresh</v-icon>
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Add/Edit Registry Modal -->
    <AppModal
      v-model="dialog"
      :title="editedIndex === -1 ? 'Add Registry' : 'Edit Registry'"
      max-width="560px"
      :persistent="saving"
    >
      <div class="form-field">
        <label class="form-label" for="reg-preset">Preset (Optional)</label>
        <select id="reg-preset" class="form-select" @change="onPresetChange">
          <option value="">Choose a popular registry preset...</option>
          <option v-for="preset in presets" :key="preset.name" :value="preset.name">
            {{ preset.name }} ({{ preset.url }})
          </option>
        </select>
      </div>

      <div class="form-field">
        <label class="form-label" for="reg-name">Friendly Name</label>
        <input
          id="reg-name"
          v-model="editedItem.name"
          type="text"
          class="form-input"
          placeholder="e.g. My Private Registry"
        />
      </div>

      <div class="form-field">
        <label class="form-label" for="reg-url">Registry URL</label>
        <input
          id="reg-url"
          v-model="editedItem.url"
          type="text"
          class="form-input mono"
          placeholder="e.g. https://index.docker.io/v1/ or ghcr.io"
        />
        <span class="form-hint">Leave blank for Docker Hub</span>
      </div>

      <div class="config-grid">
        <div class="form-field">
          <label class="form-label" for="reg-user">Username</label>
          <input
            id="reg-user"
            v-model="editedItem.username"
            type="text"
            class="form-input"
            placeholder="username"
          />
        </div>
        <div class="form-field">
          <label class="form-label" for="reg-pass">Password / Token</label>
          <input
            id="reg-pass"
            v-model="editedItem.password"
            type="password"
            class="form-input"
            placeholder="••••••••••••"
          />
        </div>
      </div>

      <template #footer>
        <button type="button" class="btn" :disabled="saving" @click="dialog = false">Cancel</button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="saving || !isFormValid"
          @click="save"
        >
          {{ saving ? 'Saving...' : 'Save' }}
        </button>
      </template>
    </AppModal>

    <!-- Delete Confirmation Modal -->
    <AppModal v-model="deleteDialog" title="Delete Registry?" :persistent="deleting">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to delete the credentials for
        <strong class="text-1">{{ itemToDelete?.name }}</strong>?
        This may cause future deployments to fail if they require these credentials.
      </p>
      <template #footer>
        <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
        <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteItem">
          {{ deleting ? 'Deleting...' : 'Delete' }}
        </button>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import Loader from '../../components/Loader.vue';
import AppModal from '../../components/AppModal.vue';

interface Registry {
  id?: string;
  name: string;
  url: string;
  username: string;
  password?: string;
  created_at?: string;
}

interface DockerHubLimit {
  limit: number;
  remaining: number;
  reset: number;
}

interface RegistryPreset {
  name: string;
  url: string;
  icon: string;
  placeholderName: string;
}

const presets: RegistryPreset[] = [
  { name: 'Docker Hub', url: 'https://index.docker.io/v1/', icon: 'mdi-docker', placeholderName: 'Docker Hub' },
  { name: 'GitHub Container Registry', url: 'ghcr.io', icon: 'mdi-github', placeholderName: 'GitHub GHCR' },
  { name: 'GitLab Container Registry', url: 'registry.gitlab.com', icon: 'mdi-gitlab', placeholderName: 'GitLab Registry' },
  { name: 'AWS ECR (Private)', url: '123456789012.dkr.ecr.us-east-1.amazonaws.com', icon: 'mdi-aws', placeholderName: 'AWS ECR Private' },
  { name: 'AWS ECR (Public)', url: 'public.ecr.aws', icon: 'mdi-aws', placeholderName: 'AWS ECR Public' },
  { name: 'Google Artifact Registry', url: 'us-central1-docker.pkg.dev', icon: 'mdi-google-cloud', placeholderName: 'Google Artifact Registry' },
  { name: 'Google Container Registry', url: 'gcr.io', icon: 'mdi-google-cloud', placeholderName: 'Google GCR' },
  { name: 'Red Hat Quay', url: 'quay.io', icon: 'mdi-redhat', placeholderName: 'Red Hat Quay' },
  { name: 'Azure Container Registry', url: 'myregistry.azurecr.io', icon: 'mdi-microsoft-azure', placeholderName: 'Azure ACR' },
];

const registries = ref<Registry[]>([]);
const dockerHubLimit = ref<DockerHubLimit | null>(null);
const fetchingLimit = ref(false);
const loading = ref(true);
const dialog = ref(false);
const deleteDialog = ref(false);
const saving = ref(false);
const deleting = ref(false);
const editedIndex = ref(-1);
const itemToDelete = ref<Registry | null>(null);

const defaultItem: Registry = {
  name: '',
  url: '',
  username: '',
  password: '',
};

const editedItem = ref<Registry>({ ...defaultItem });

const isFormValid = computed(() => {
  return Boolean(editedItem.value.name.trim() && editedItem.value.username.trim() && editedItem.value.password);
});

const isDockerHub = (reg: Registry) => {
  const url = reg.url?.toLowerCase() || '';
  return !url || url.includes('docker.io') || url.includes('docker.com');
};

const hasDockerHubCredential = computed(() => registries.value.some((r) => isDockerHub(r)));

const formatResetTime = (seconds: number) => {
  if (!seconds) return '0s';
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return remainingMinutes === 0 ? `${hours}h` : `${hours}h ${remainingMinutes}m`;
};

const getLimitColorClass = computed(() => {
  if (!dockerHubLimit.value) return 'text-2';
  const percent = (dockerHubLimit.value.remaining / dockerHubLimit.value.limit) * 100;
  if (percent > 50) return 'text-ok';
  if (percent > 20) return 'text-warn';
  return 'text-err';
});

const getLimitTelemetryClass = computed(() => {
  if (!dockerHubLimit.value) return '';
  const percent = (dockerHubLimit.value.remaining / dockerHubLimit.value.limit) * 100;
  if (percent <= 20) return 'is-err';
  if (percent <= 50) return 'is-warn';
  return '';
});

const applyPreset = (preset: RegistryPreset) => {
  editedItem.value.url = preset.url;
  if (
    !editedItem.value.name ||
    presets.some((p) => p.placeholderName === editedItem.value.name || p.name === editedItem.value.name)
  ) {
    editedItem.value.name = preset.placeholderName;
  }
};

const onPresetChange = (e: Event) => {
  const name = (e.target as HTMLSelectElement).value;
  const found = presets.find((p) => p.name === name);
  if (found) applyPreset(found);
};

const getRegistryIcon = (reg: Registry) => {
  const url = reg.url?.toLowerCase() || '';
  if (!url || url.includes('docker.io') || url.includes('docker.com')) return 'mdi-docker';
  if (url.includes('ghcr.io') || url.includes('github.com')) return 'mdi-github';
  if (url.includes('gitlab.com')) return 'mdi-gitlab';
  if (url.includes('amazonaws.com') || url.includes('ecr.aws')) return 'mdi-aws';
  if (url.includes('pkg.dev') || url.includes('gcr.io')) return 'mdi-google-cloud';
  if (url.includes('quay.io')) return 'mdi-redhat';
  if (url.includes('azurecr.io')) return 'mdi-microsoft-azure';
  return 'mdi-server';
};

const fetchDockerHubLimit = async () => {
  fetchingLimit.value = true;
  try {
    const response = await fetch('/api/registries/dockerhub-limit');
    if (response.ok) {
      dockerHubLimit.value = await response.json();
    }
  } catch (error) {
    console.error('Failed to fetch Docker Hub rate limit:', error);
  } finally {
    fetchingLimit.value = false;
  }
};

const addDockerHubPreset = () => {
  openAddDialog();
  const dockerHubPreset = presets.find((p) => p.name === 'Docker Hub');
  if (dockerHubPreset) applyPreset(dockerHubPreset);
};

const fetchRegistries = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/registries');
    if (response.ok) {
      registries.value = (await response.json()) || [];
    }
  } catch (error) {
    console.error('Failed to fetch registries:', error);
  } finally {
    loading.value = false;
  }
};

const openAddDialog = () => {
  editedIndex.value = -1;
  editedItem.value = { ...defaultItem };
  dialog.value = true;
};

const openEditDialog = (item: Registry) => {
  editedIndex.value = registries.value.indexOf(item);
  editedItem.value = { ...item };
  dialog.value = true;
};

const confirmDelete = (item: Registry) => {
  itemToDelete.value = item;
  deleteDialog.value = true;
};

const deleteItem = async () => {
  if (!itemToDelete.value?.id) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/registries/${itemToDelete.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchRegistries();
      fetchDockerHubLimit();
      deleteDialog.value = false;
    }
  } catch (error) {
    console.error('Failed to delete registry:', error);
  } finally {
    deleting.value = false;
  }
};

const save = async () => {
  if (!isFormValid.value) return;
  saving.value = true;
  try {
    const isEdit = editedIndex.value > -1;
    const url = isEdit ? `/api/registries/${editedItem.value.id}` : '/api/registries';
    const method = isEdit ? 'PUT' : 'POST';

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editedItem.value),
    });

    if (response.ok) {
      await fetchRegistries();
      fetchDockerHubLimit();
      dialog.value = false;
    }
  } catch (error) {
    console.error('Failed to save registry:', error);
  } finally {
    saving.value = false;
  }
};

onMounted(async () => {
  await fetchRegistries();
  fetchDockerHubLimit();
});
</script>

<style scoped>
.registries-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

@container (width < 820px) {
  .registries-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@container (width < 560px) {
  .registries-grid {
    grid-template-columns: 1fr;
  }
}
</style>
