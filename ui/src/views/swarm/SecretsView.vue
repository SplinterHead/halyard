<template>
  <DataTablePage
    title="Swarm Secrets"
    subtitle="Encrypted Swarm secrets injected securely into service containers."
    :items="secrets"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-key-remove"
    empty-title="No secrets found"
    empty-description="Create secrets to securely pass sensitive data like passwords and tokens to your services."
    search-placeholder="Search secrets..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    @refresh="fetchSecrets"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" @click="dialog = true">
        + Create Secret
      </button>
    </template>

    <template #top>
      <AppModal v-model="dialog" title="Create Swarm Secret" max-width="560px" :persistent="saving">
        <div class="form-field">
          <label class="form-label" for="secret-name">Secret Name</label>
          <input
            id="secret-name"
            v-model="newSecret.name"
            type="text"
            class="form-input"
            placeholder="db_password"
          />
        </div>

        <div class="form-field">
          <label class="form-label" for="secret-data">Secret Data</label>
          <textarea
            id="secret-data"
            v-model="newSecret.data"
            class="form-textarea mono"
            rows="6"
            placeholder="Paste your secret content here..."
          ></textarea>
        </div>

        <template #footer>
          <button type="button" class="btn" :disabled="saving" @click="closeDialog">Cancel</button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="saving || !newSecret.name.trim() || !newSecret.data"
            @click="createSecret"
          >
            {{ saving ? 'Creating...' : 'Create' }}
          </button>
        </template>
      </AppModal>

      <AppModal v-model="deleteDialog" title="Remove Secret?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove the secret
          <strong class="text-1">{{ secretToDelete?.name }}</strong>?
          This will fail if it is currently in use by any services.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteSecret">
            {{ deleting ? 'Removing...' : 'Remove' }}
          </button>
        </template>
      </AppModal>
    </template>

    <template #item.name="{ value }">
      <span class="fw-500 text-1">{{ value }}</span>
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
          title="Remove Secret"
          aria-label="Remove Secret"
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
          <span class="fw-500 text-1 text-truncate">{{ item.name }}</span>
          <button
            type="button"
            class="btn btn-icon btn-sm btn-danger-subtle"
            title="Remove Secret"
            aria-label="Remove Secret"
            @click="confirmDelete(item)"
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
import { formatRelativeTime } from '../../utils/time';

interface Secret {
  id: string;
  name: string;
  created_at: string;
}

const secrets = ref<Secret[]>([]);
const loading = ref(false);
const dialog = ref(false);
const saving = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const secretToDelete = ref<Secret | null>(null);

const newSecret = ref({
  name: '',
  data: '',
});

const headers = [
  { title: 'Name', key: 'name', sortable: true, align: 'start' as const },
  { title: 'ID', key: 'id', width: '160px', align: 'start' as const },
  { title: 'Created', key: 'created_at', width: '140px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '90px', align: 'center' as const, sortable: false },
];

const fetchSecrets = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/secrets');
    if (response.ok) {
      secrets.value = (await response.json()) || [];
    }
  } catch (error) {
    console.error('Failed to fetch secrets:', error);
  } finally {
    loading.value = false;
  }
};

const createSecret = async () => {
  if (!newSecret.value.name.trim() || !newSecret.value.data) return;
  saving.value = true;
  try {
    const response = await fetch('/api/secrets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newSecret.value),
    });
    if (response.ok) {
      await fetchSecrets();
      closeDialog();
    } else {
      alert('Failed to create secret: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to create secret:', err);
  } finally {
    saving.value = false;
  }
};

const confirmDelete = (secret: Secret) => {
  secretToDelete.value = secret;
  deleteDialog.value = true;
};

const deleteSecret = async () => {
  if (!secretToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/secrets/${secretToDelete.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchSecrets();
      deleteDialog.value = false;
    } else {
      alert('Failed to remove secret: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to delete secret:', err);
  } finally {
    deleting.value = false;
  }
};

const closeDialog = () => {
  dialog.value = false;
  newSecret.value = { name: '', data: '' };
};

onMounted(fetchSecrets);
</script>
