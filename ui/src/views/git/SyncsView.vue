<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Git Syncs</h1>
      <v-spacer></v-spacer>
      <v-btn
        prepend-icon="mdi-sync"
        color="primary"
        @click="openAddDialog"
        flat
        class="me-2"
      >
        Add Git Sync
      </v-btn>
      <v-btn
        icon="mdi-refresh"
        @click="fetchSyncs"
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

    <div v-else-if="syncs.length === 0" class="flex-grow-1 d-flex flex-column align-center justify-center">
      <v-icon size="80" color="grey-lighten-1" class="mb-4">mdi-sync-off</v-icon>
      <h3 class="text-h5 text-grey-darken-1">No Git Syncs Found</h3>
      <p class="text-body-1 text-grey-darken-1 mt-2 mb-6 text-center" style="max-width: 500px">
        Create a Git sync to automate the deployment and periodic reconciliation of your Swarm stacks. Use the button above to create your first sync.
      </p>
    </div>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="syncs"
        :loading="loading"
        :sort-by="[{ key: 'name', order: 'asc' }]"
        :row-props="getRowProps"
        class="bg-transparent"
        density="comfortable"
        items-per-page="25"
      >
        <template v-slot:item.stack_name="{ item }">
          <span 
            class="text-body-2 font-weight-bold"
            :class="item.last_status === 'Out of Date' ? 'text-error' : ''"
          >
            {{ item.stack_name }}
          </span>
        </template>

        <template v-slot:item.repository_id="{ item }">
          <div class="d-flex flex-column">
            <span class="text-caption font-weight-bold">{{
              getRepoName(item.repository_id)
            }}</span>
            <div class="d-flex align-center mt-n1">
              <v-icon size="12" class="me-1 text-grey"
                >mdi-source-branch</v-icon
              >
              <code
                class="text-caption text-grey"
                style="background: transparent"
                >{{ item.branch }}</code
              >
            </div>
          </div>
        </template>

        <template v-slot:item.last_applied_sha="{ value }">
          <div v-if="value" class="d-flex align-center">
            <v-icon size="14" class="me-1 text-grey">mdi-source-commit</v-icon>
            <code class="text-caption text-primary font-weight-bold">{{
              value.substring(0, 7)
            }}</code>
          </div>
          <span v-else class="text-caption text-grey">Never</span>
        </template>

        <template v-slot:item.path="{ value }">
          <code class="text-caption">{{ value }}</code>
        </template>

        <template v-slot:item.pull_additional_files="{ value }">
          <v-chip
            :color="value ? 'primary' : 'grey-lighten-1'"
            size="x-small"
            label
            class="text-uppercase font-weight-bold"
          >
            {{ value ? 'Enabled' : 'Disabled' }}
          </v-chip>
        </template>

        <template v-slot:item.auto_sync="{ value }">
          <v-chip
            :color="value ? 'primary' : 'grey-lighten-1'"
            size="x-small"
            label
            class="text-uppercase font-weight-bold"
          >
            {{ value ? 'Enabled' : 'Disabled' }}
          </v-chip>
        </template>

        <template v-slot:item.last_sync_at="{ value }">
          <RelativeTime :value="value" v-if="value && value !== '0001-01-01T00:00:00Z'" />
          <span v-else class="text-caption text-grey">Never</span>
        </template>

        <template v-slot:item.created_at="{ value }">
          <RelativeTime :value="value" />
        </template>

        <template v-slot:item.actions="{ item }">
          <div class="d-flex justify-center">
            <v-btn
              icon="mdi-sync"
              size="x-small"
              variant="text"
              color="success"
              @click="forceSync(item)"
              :loading="syncingId === item.id"
              title="Force Sync Now"
            ></v-btn>
            <v-btn
              icon="mdi-pencil-outline"
              size="x-small"
              variant="text"
              color="primary"
              @click="openEditDialog(item)"
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

    <!-- Add/Edit Sync Dialog -->
    <v-dialog v-model="dialog" max-width="600px" persistent>
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">
          <span class="text-h5 font-weight-bold"
            >{{ isEdit ? "Edit" : "Add" }} Git Sync</span
          >
        </v-card-title>

        <v-card-text class="pa-6 pt-2">
          <v-form ref="form" v-model="valid">
            <v-row>
              <v-col cols="12">
                <v-text-field
                  v-model="editedSync.stack_name"
                  label="Stack Name"
                  placeholder="my-app"
                  variant="outlined"
                  density="comfortable"
                  hint="Use lowercase letters and dashes only. No spaces or capitals."
                  persistent-hint
                  :rules="stackNameRules"
                  required
                ></v-text-field>
              </v-col>

              <v-col cols="12">
                <v-select
                  v-model="editedSync.repository_id"
                  :items="repos"
                  item-title="name"
                  item-value="id"
                  label="Git Repository"
                  variant="outlined"
                  density="comfortable"
                  :rules="[(v) => !!v || 'Repository is required']"
                  required
                  @update:model-value="fetchBranches"
                ></v-select>
              </v-col>

              <v-col cols="12">
                <v-select
                  v-model="editedSync.branch"
                  :items="branches"
                  label="Branch"
                  required
                  :rules="[(v) => !!v || 'Branch is required']"
                  :loading="loadingBranches"
                  :disabled="!editedSync.repository_id"
                  variant="outlined"
                  density="comfortable"
                ></v-select>
              </v-col>

              <v-col cols="12">
                <v-text-field
                  v-model="editedSync.path"
                  label="Compose File Path"
                  placeholder="compose.yml"
                  required
                  :rules="[(v) => !!v || 'Path is required']"
                  variant="outlined"
                  density="comfortable"
                  append-inner-icon="mdi-folder-search-outline"
                  @click:append-inner="openFileBrowser"
                  :disabled="!editedSync.repository_id || !editedSync.branch"
                >
                  <template v-slot:append-inner>
                    <v-tooltip location="top" text="Browse Repository">
                      <template v-slot:activator="{ props }">
                        <v-icon
                          v-bind="props"
                          @click="openFileBrowser"
                          :disabled="!editedSync.repository_id || !editedSync.branch"
                        >
                          mdi-folder-search-outline
                        </v-icon>
                      </template>
                    </v-tooltip>
                  </template>
                </v-text-field>
              </v-col>

              <v-col cols="12" md="6" class="pe-4">
                <div class="d-flex align-start px-2 mt-2">
                  <v-switch
                    v-model="editedSync.pull_additional_files"
                    color="primary"
                    hide-details
                    density="compact"
                    class="mt-n1"
                  ></v-switch>
                  <div class="ms-6">
                    <div class="text-subtitle-2 font-weight-bold">
                      Additional Files
                    </div>
                    <div
                      class="text-caption text-grey"
                      style="font-size: 0.7rem !important; line-height: 1.2"
                    >
                      Pull additional files like .env in the same directory.
                    </div>
                  </div>
                </div>
              </v-col>

              <v-col cols="12" md="6" class="ps-4">
                <div class="d-flex align-start px-2 mt-2">
                  <v-switch
                    v-model="editedSync.auto_sync"
                    color="primary"
                    hide-details
                    density="compact"
                    class="mt-n1"
                  ></v-switch>
                  <div class="ms-6">
                    <div class="text-subtitle-2 font-weight-bold">
                      Auto Sync
                    </div>
                    <div
                      class="text-caption text-grey"
                      style="font-size: 0.7rem !important; line-height: 1.2"
                    >
                      Periodically reconcile the stack with the source
                      repository.
                    </div>
                  </div>
                </div>
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
          <v-btn variant="text" @click="closeDialog" :disabled="saving"
            >Cancel</v-btn
          >
          <v-btn
            color="primary"
            variant="flat"
            @click="saveSync"
            :loading="saving"
            :disabled="!valid"
          >
            {{ isEdit ? "Save" : "Add" }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Delete Git Sync?</v-card-title>
        <v-card-text class="pa-6 pt-0">
          This will stop automatic updates for this stack. The existing stack in
          the Swarm will not be removed.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteSync"
            :loading="deleting"
            >Delete</v-btn
          >
        </v-card-actions>
      </v-card>
    </v-dialog>
    
    <!-- File Browser Dialog -->
    <v-dialog v-model="fileBrowserDialog" max-width="500px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-4 pb-2 d-flex align-center">
          <v-btn 
            icon="mdi-chevron-left" 
            variant="text" 
            size="small" 
            @click="goUp" 
            :disabled="currentPath === ''"
          ></v-btn>
          <div class="ms-2 d-flex flex-column">
            <span class="text-subtitle-1 font-weight-bold">Browse Repository</span>
            <span class="text-caption text-grey mt-n1">/{{ currentPath }}</span>
          </div>
          <v-spacer></v-spacer>
          <v-btn icon="mdi-close" variant="text" size="small" @click="fileBrowserDialog = false"></v-btn>
        </v-card-title>
        <v-divider></v-divider>
        <v-card-text class="pa-0" style="max-height: 450px; overflow-y: auto;">
          <div v-if="loadingFiles" class="pa-8 text-center">
            <v-progress-circular indeterminate color="primary"></v-progress-circular>
            <div class="text-caption mt-2 text-grey">Reading repository...</div>
          </div>
          <v-list v-else density="compact" nav class="bg-transparent">
            <v-list-item v-if="currentPath !== ''" @click="goUp" prepend-icon="mdi-folder-upload-outline">
              <v-list-item-title class="font-weight-bold">..</v-list-item-title>
            </v-list-item>
            
            <template v-if="repoFiles.length">
              <v-list-item 
                v-for="file in repoFiles" 
                :key="file.path" 
                @click="selectFile(file)" 
                :prepend-icon="file.is_dir ? 'mdi-folder' : 'mdi-file-code-outline'"
                :active="editedSync.path === file.path"
                :disabled="!file.is_dir && !isYamlFile(file.name)"
              >
                <v-list-item-title :class="file.is_dir ? 'font-weight-bold' : ''">{{ file.name }}</v-list-item-title>
                <template v-slot:append v-if="!file.is_dir && isYamlFile(file.name)">
                   <v-icon size="small" color="primary">mdi-check-circle-outline</v-icon>
                </template>
              </v-list-item>
            </template>
            <div v-else class="pa-8 text-center text-grey text-caption">
              No files found in this directory.
            </div>
          </v-list>
        </v-card-text>
        <v-divider></v-divider>
        <v-card-actions class="pa-2 px-4">
          <span class="text-caption text-grey">Select a .yml or .yaml file</span>
          <v-spacer></v-spacer>
          <v-btn variant="text" density="comfortable" size="small" @click="fileBrowserDialog = false">Cancel</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import RelativeTime from "../../components/RelativeTime.vue";

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
const valid = ref(false);
const saving = ref(false);
const deleting = ref(false);
const syncingId = ref<string | null>(null);
const isEdit = ref(false);
const error = ref("");
const syncToDelete = ref<GitSync | null>(null);
const fileBrowserDialog = ref(false);
const loadingFiles = ref(false);
const repoFiles = ref<any[]>([]);
const currentPath = ref("");

const editedSync = ref<GitSync>({
  name: "",
  repository_id: "",
  stack_name: "",
  branch: "",
  path: "compose.yml",
  pull_additional_files: false,
  auto_sync: false,
  last_applied_sha: "",
  last_sync_at: null,
  last_status: "Ready",
  last_error: "",
  created_at: null,
});

const stackNameRules = [
  (v: string) => !!v || "Stack Name is required",
  (v: string) =>
    /^[a-z0-9-]+$/.test(v) ||
    "Stack Name must be lowercase and contain no spaces or special characters (only dashes allowed)",
];

const getRowProps = ({ item }: any) => {
  const status = item.last_status;
  let colorClass = "status-success";
  if (status === "Failed" || status === "Out of Date") colorClass = "status-error";
  else if (status === "Syncing") colorClass = "status-info";
  else if (status === "Ready") colorClass = "status-grey";

  return {
    class: `status-bar-row ${colorClass}`,
  };
};

const headers = [
  {
    title: "Stack Name",
    key: "stack_name",
    sortable: true,
    align: "start" as const,
  },
  {
    title: "Repository / Branch",
    key: "repository_id",
    align: "start" as const,
  },
  {
    title: "Commit",
    key: "last_applied_sha",
    width: "120px",
    align: "start" as const,
  },
  { title: "File Path", key: "path", align: "start" as const },
  {
    title: "Pull Files",
    key: "pull_additional_files",
    width: "120px",
    align: "center" as const,
  },
  {
    title: "Auto Sync",
    key: "auto_sync",
    width: "120px",
    align: "center" as const,
  },
  {
    title: "Created",
    key: "created_at",
    width: "200px",
    align: "start" as const,
  },
  {
    title: "Updated",
    key: "last_sync_at",
    width: "150px",
    align: "start" as const,
  },
  {
    title: "Actions",
    key: "actions",
    width: "120px",
    align: "center" as const,
    sortable: false,
  },
];

const fetchRepos = async () => {
  try {
    const response = await fetch("/api/repos");
    repos.value = await response.json();
  } catch (err) {
    console.error("Failed to fetch repos:", err);
  }
};

const fetchBranches = async (repoId: string) => {
  if (!repoId) return;
  loadingBranches.value = true;
  branches.value = [];
  try {
    const response = await fetch(`/api/repos/${repoId}/branches`);
    branches.value = await response.json();
  } catch (err) {
    console.error("Failed to fetch branches:", err);
  } finally {
    loadingBranches.value = false;
  }
};

const fetchSyncs = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/syncs");
    syncs.value = await response.json();
    if (repos.value.length === 0) await fetchRepos();
  } catch (error) {
    console.error("Failed to fetch syncs:", error);
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
    name: "",
    repository_id: "",
    stack_name: "",
    branch: "",
    path: "compose.yml",
    pull_additional_files: false,
    auto_sync: false,
    last_applied_sha: "",
    last_sync_at: null,
    last_status: "Ready",
    last_error: "",
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
    const response = await fetch(`/api/syncs/${sync.id}/sync`, {
      method: "POST",
    });
    if (!response.ok) {
      const text = await response.text();
      alert("Failed to trigger sync: " + text);
    }
    await fetchSyncs();
  } catch (err) {
    console.error("Failed to force sync:", err);
  } finally {
    syncingId.value = null;
  }
};

const saveSync = async () => {
  saving.value = true;
  error.value = "";
  try {
    const url = isEdit.value
      ? `/api/syncs/${editedSync.value.id}`
      : "/api/syncs";
    const method = isEdit.value ? "PUT" : "POST";

    const payload = {
      ...editedSync.value,
      name: editedSync.value.stack_name, // Ensure name is sync'd with stack_name
    };

    if (!isEdit.value) {
      delete payload.id;
      delete (payload as any).created_at;
    }

    const response = await fetch(url, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const text = await response.text();
      try {
        const data = JSON.parse(text);
        throw new Error(data.message || data.error || "Failed to save sync");
      } catch {
        throw new Error(text || "Failed to save sync");
      }
    }

    const result = await response.json();

    if (result.last_status === "Failed") {
      error.value = result.last_error || "Validation failed";
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
      method: "DELETE",
    });
    if (response.ok) {
      await fetchSyncs();
      deleteDialog.value = false;
    }
  } catch (err) {
    console.error("Failed to delete sync:", err);
  } finally {
    deleting.value = false;
  }
};

const closeDialog = () => {
  dialog.value = false;
  error.value = "";
  resetEditedSync();
};

const openFileBrowser = async () => {
  if (!editedSync.value.repository_id || !editedSync.value.branch) return;
  fileBrowserDialog.value = true;
  currentPath.value = "";
  await fetchFiles("");
};

const fetchFiles = async (path: string) => {
  loadingFiles.value = true;
  try {
    const repoId = editedSync.value.repository_id;
    const branch = editedSync.value.branch;
    const response = await fetch(
      `/api/repos/${repoId}/files?branch=${branch}&path=${path}`
    );
    if (response.ok) {
      repoFiles.value = await response.json();
      currentPath.value = path;
    } else {
      const err = await response.text();
      console.error("Failed to fetch files:", err);
    }
  } catch (err) {
    console.error("Error fetching files:", err);
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
  if (currentPath.value === "") return;
  const parts = currentPath.value.split("/");
  parts.pop();
  await fetchFiles(parts.join("/"));
};

const isYamlFile = (name: string) => {
  return name.endsWith(".yml") || name.endsWith(".yaml");
};

onMounted(fetchSyncs);
</script>

<style scoped>
</style>
