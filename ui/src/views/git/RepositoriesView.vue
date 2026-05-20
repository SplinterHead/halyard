<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Git Repositories</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-plus"
        color="primary"
        @click="openAddDialog"
        flat
        class="me-2"
      >
        Add Repository
      </v-btn>
      <v-btn
        icon="mdi-refresh"
        @click="fetchRepos"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <v-row v-if="loading" justify="center" class="mt-8">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </v-row>

    <div v-else-if="repos.length === 0" class="flex-grow-1 d-flex flex-column align-center justify-center">
      <v-icon size="80" color="grey-lighten-1" class="mb-4">mdi-git</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No Repositories Found</h3>
      <p class="text-body-1 text-grey-darken-1 mt-2 mb-6 text-center" style="max-width: 500px">
        Add a Git repository to start syncing your stacks with source code and automate your deployments. Use the button above to add your first repository.
      </p>
    </div>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="repos"
        :loading="loading"
        :sort-by="[{ key: 'name', order: 'asc' }]"
        :row-props="getRowProps"
        class="bg-transparent"
        density="comfortable"
        items-per-page="25"
      >
        <template v-slot:item.name="{ item }">
          <div class="d-flex flex-column">
            <span class="text-body-2 font-weight-bold">{{ item.name }}</span>
            <span class="text-caption text-grey text-truncate-400" :title="item.description">{{ item.description || 'No description' }}</span>
          </div>
        </template>

        <template v-slot:item.url="{ value }">
          <div class="d-flex align-center">
            <v-icon size="16" color="grey" class="me-2">mdi-git</v-icon>
            <code class="text-caption text-truncate-400" :title="value">{{ value }}</code>
          </div>
        </template>

        <template v-slot:item.created_at="{ value }">
          <RelativeTime :value="value" />
        </template>

        <template v-slot:item.actions="{ item }">
          <div class="d-flex justify-center">
            <v-btn
              icon="mdi-connection"
              size="x-small"
              variant="text"
              color="success"
              @click="testConnection(item)"
              :loading="testingId === item.id"
              class="me-1"
              title="Test Connection"
            ></v-btn>
            <v-btn
              icon="mdi-pencil-outline"
              size="x-small"
              variant="text"
              color="primary"
              @click="openEditDialog(item)"
              class="me-1"
            ></v-btn>
            <v-btn
              icon="mdi-delete-outline"
              size="x-small"
              variant="text"
              color="error"
              @click="confirmDelete(item)"
            ></v-btn>
          </div>
        </template>

      </v-data-table>
    </div>

    <!-- Add/Edit Repo Dialog -->
    <v-dialog v-model="dialog" max-width="600px" persistent>
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">
          <span class="text-h5 font-weight-bold">{{ isEdit ? 'Edit' : 'Add' }} Git Repository</span>
        </v-card-title>
        
        <v-card-text class="pa-6 pt-2">
          <v-form ref="form" v-model="valid">
            <v-row>
              <v-col cols="12">
                <v-text-field
                  v-model="editedRepo.name"
                  label="Friendly Name"
                  placeholder="e.g. Production Stack"
                  required
                  :rules="[v => !!v || 'Name is required']"
                  variant="outlined"
                  density="comfortable"
                ></v-text-field>
              </v-col>
              
              <v-col cols="12">
                <v-text-field
                  v-model="editedRepo.url"
                  label="Git URL"
                  placeholder="https://github.com/org/repo.git"
                  required
                  :rules="[v => !!v || 'URL is required']"
                  variant="outlined"
                  density="comfortable"
                ></v-text-field>
              </v-col>

              <v-col cols="6">
                <v-text-field
                  v-model="editedRepo.username"
                  label="Username"
                  placeholder="Optional"
                  variant="outlined"
                  density="comfortable"
                ></v-text-field>
              </v-col>

              <v-col cols="6">
                <v-text-field
                  v-model="editedRepo.token"
                  label="Token / Password"
                  type="password"
                  placeholder="Optional"
                  variant="outlined"
                  density="comfortable"
                ></v-text-field>
              </v-col>

              <v-col cols="12">
                <v-textarea
                  v-model="editedRepo.description"
                  label="Description"
                  rows="2"
                  variant="outlined"
                  density="comfortable"
                  hide-details
                ></v-textarea>
              </v-col>
            </v-row>
          </v-form>

          <v-alert
            v-if="error"
            type="error"
            variant="tonal"
            density="compact"
            class="mt-4"
            closable
            @click:close="error = ''"
          >
            {{ error }}
          </v-alert>
        </v-card-text>

        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="closeDialog" :disabled="saving">Cancel</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            @click="saveRepo"
            :loading="saving"
            :disabled="!valid"
          >
            {{ isEdit ? 'Save' : 'Add' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Delete Repository?</v-card-title>
        <v-card-text class="pa-6 pt-0">
          This will also delete all Git Syncs associated with this repository. This action cannot be undone.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" @click="deleteRepo" :loading="deleting">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import RelativeTime from '../../components/RelativeTime.vue'

interface GitRepo {
  id?: string
  name: string
  url: string
  username: string
  token: string
  description: string
  last_status: string
  last_error: string
  created_at?: string
}

const repos = ref<GitRepo[]>([])
const loading = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const valid = ref(false)
const saving = ref(false)
const deleting = ref(false)
const isEdit = ref(false)
const error = ref('')
const repoToDelete = ref<GitRepo | null>(null)
const testingId = ref<string | null>(null)

const editedRepo = ref<GitRepo>({
  name: '',
  url: '',
  username: '',
  token: '',
  description: '',
  last_status: '',
  last_error: '',
  created_at: ''
})

const getRowProps = ({ item }: any) => {
  return {
    class: `status-bar-row ${item.last_status === "Failed" ? "status-error" : "status-success"}`,
  };
};

const headers = [
  { title: "Repository", key: "name", sortable: true, align: "start" as const },
  { title: "URL", key: "url", sortable: true, align: "start" as const },
  { title: "Created", key: "created_at", width: "200px", align: "start" as const },
  { title: "Actions", key: "actions", width: "120px", align: "center" as const, sortable: false },
];

const fetchRepos = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/repos')
    repos.value = await response.json()
  } catch (error) {
    console.error('Failed to fetch repos:', error)
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  isEdit.value = false
  resetEditedRepo()
  dialog.value = true
}

const openEditDialog = (repo: GitRepo) => {
  isEdit.value = true
  editedRepo.value = { ...repo }
  dialog.value = true
}

const confirmDelete = (repo: GitRepo) => {
  repoToDelete.value = repo
  deleteDialog.value = true
}

const resetEditedRepo = () => {
  editedRepo.value = {
    name: '',
    url: '',
    username: '',
    token: '',
    description: '',
    last_status: '',
    last_error: '',
    created_at: ''
  }
}

const saveRepo = async () => {
  saving.value = true
  error.value = ''
  try {
    const url = isEdit.value ? `/api/repos/${editedRepo.value.id}` : '/api/repos'
    const method = isEdit.value ? 'PUT' : 'POST'
    
    const payload = { ...editedRepo.value }
    if (!isEdit.value) {
      delete payload.id
      delete (payload as any).created_at
    }

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    
    const result = await response.json()
    
    if (result.last_status === 'Failed') {
      error.value = result.last_error || 'Connection failed'
      await fetchRepos()
      return
    }
    
    await fetchRepos()
    closeDialog()
  } catch (err: any) {
    error.value = err.message
  } finally {
    saving.value = false
  }
}

const deleteRepo = async () => {
  if (!repoToDelete.value) return
  deleting.value = true
  try {
    const response = await fetch(`/api/repos/${repoToDelete.value.id}`, {
      method: 'DELETE'
    })
    if (response.ok) {
      await fetchRepos()
      deleteDialog.value = false
    }
  } catch (err) {
    console.error('Failed to delete repo:', err)
  } finally {
    deleting.value = false
  }
}

const closeDialog = () => {
  dialog.value = false
  error.value = ''
  resetEditedRepo()
}

const testConnection = async (repo: GitRepo) => {
  if (!repo.id) return
  testingId.value = repo.id
  try {
    const response = await fetch(`/api/repos/${repo.id}/test`, {
      method: "POST",
    });
    if (!response.ok) {
      const err = await response.text();
      // We don't alert here because the status row will turn red and show the error on next fetch
    }
    await fetchRepos();
  } catch (err) {
    console.error("Failed to test connection:", err);
  } finally {
    testingId.value = null;
  }
};

onMounted(fetchRepos)
</script>

