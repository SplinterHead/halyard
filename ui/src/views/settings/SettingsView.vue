<template>
  <div class="fill-height d-flex flex-column">
    <!-- Header -->
    <div class="pa-2 pb-0 d-flex align-center flex-wrap gap-2">
      <div><h1 class="text-h4 font-weight-bold">Settings</h1></div>
      <v-spacer></v-spacer>
      <v-btn color="primary" @click="saveSettings" :loading="saving" flat prepend-icon="mdi-content-save-outline">
        Save Changes
      </v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <div class="pa-4 flex-grow-1 overflow-y-auto">
      <div class="d-flex flex-column gap-6 max-width-900">
        <!-- GitOps Settings Card -->
        <v-card border flat class="bg-surface rounded-lg">
          <v-card-text class="pa-6">
            <div class="d-flex align-center mb-6">
              <v-icon color="primary" class="me-4" size="32">mdi-sync</v-icon>
              <div>
                <div class="text-h6 font-weight-bold">Sync Concurrency</div>
                <div class="text-caption text-grey">
                  How many Git Syncs to process at the same time. Set to 0 for full parallelism.
                </div>
              </div>
            </div>

            <v-slider
              v-model="settings.git_sync_concurrency"
              :min="0"
              :max="20"
              :step="1"
              thumb-label="always"
              color="primary"
              class="mb-8"
              show-ticks="always"
            >
              <template v-slot:append>
                <v-text-field
                  v-model.number="settings.git_sync_concurrency"
                  type="number"
                  variant="outlined"
                  density="compact"
                  style="width: 80px"
                  hide-details
                ></v-text-field>
              </template>
            </v-slider>

            <v-divider class="my-6"></v-divider>

            <div class="d-flex align-center mb-6">
              <v-icon color="success" class="me-4" size="32">mdi-clock-outline</v-icon>
              <div>
                <div class="text-h6 font-weight-bold">Sync Interval</div>
                <div class="text-caption text-grey">
                  How often (in minutes) to check source repositories for new commits when Auto Sync is enabled.
                </div>
              </div>
            </div>

            <v-slider
              v-model="settings.git_sync_interval"
              :min="1"
              :max="60"
              :step="1"
              thumb-label="always"
              color="success"
              show-ticks="always"
            >
              <template v-slot:append>
                <v-text-field
                  v-model.number="settings.git_sync_interval"
                  type="number"
                  variant="outlined"
                  density="compact"
                  style="width: 80px"
                  hide-details
                ></v-text-field>
              </template>
            </v-slider>
          </v-card-text>
        </v-card>

        <!-- Image Poll Options Card -->
        <v-card border flat class="bg-surface rounded-lg">
          <v-card-text class="pa-6">
            <div class="d-flex align-center justify-space-between mb-6 flex-wrap gap-2">
              <div class="d-flex align-center">
                <v-icon color="primary" class="me-4" size="32">mdi-layers-search-outline</v-icon>
                <div>
                  <div class="text-h6 font-weight-bold">Image Update Polling</div>
                  <div class="text-caption text-grey">
                    Periodically inspect images in the Swarm and pull updates from remote registries.
                  </div>
                </div>
              </div>
              <v-btn
                color="primary"
                variant="tonal"
                size="small"
                prepend-icon="mdi-refresh"
                @click="triggerManualPoll"
                :loading="polling"
              >
                Check Now
              </v-btn>
            </div>

            <!-- Interval Slider -->
            <div class="d-flex align-center justify-space-between mb-2">
              <div class="text-subtitle-2 font-weight-bold">Check Interval</div>
              <v-chip
                :color="settings.image_update_interval > 0 ? 'success' : 'grey'"
                variant="tonal"
                size="small"
                label
                class="font-mono font-weight-bold"
              >
                {{ settings.image_update_interval > 0 ? `Every ${settings.image_update_interval} hr${settings.image_update_interval > 1 ? 's' : ''}` : 'Disabled' }}
              </v-chip>
            </div>

            <v-slider
              v-model="settings.image_update_interval"
              :min="0"
              :max="72"
              :step="1"
              thumb-label="always"
              color="primary"
              class="mb-2"
              show-ticks="always"
            >
              <template v-slot:append>
                <v-text-field
                  v-model.number="settings.image_update_interval"
                  type="number"
                  variant="outlined"
                  density="compact"
                  style="width: 90px"
                  suffix="hrs"
                  hide-details
                ></v-text-field>
              </template>
            </v-slider>
            <div class="text-caption text-grey ps-1 mb-6">
              Set to <strong>0</strong> to disable automatic scheduled checks (manual check will still be available).
            </div>

            <v-divider class="my-6"></v-divider>

            <!-- Update Stacks Checkbox -->
            <div class="d-flex align-start">
              <v-checkbox
                v-model="settings.update_stacks"
                color="warning"
                density="comfortable"
                hide-details
                class="pt-0"
              >
                <template v-slot:label>
                  <div>
                    <div class="d-flex align-center gap-2">
                      <span class="font-weight-bold text-body-1">Update Stacks</span>
                      <v-chip
                        :color="settings.update_stacks ? 'warning' : 'grey'"
                        variant="tonal"
                        size="x-small"
                        label
                        class="font-mono"
                      >
                        {{ settings.update_stacks ? 'Auto-Restart Enabled' : 'Disabled' }}
                      </v-chip>
                    </div>
                    <div class="text-caption text-grey mt-1">
                      Automatically restart services when updated images are pulled to nodes.
                      Replicated services are scaled -1 then +1; global services are killed node by node.
                    </div>
                  </div>
                </template>
              </v-checkbox>
            </div>

            <!-- Manual poll feedback alert -->
            <v-alert
              v-if="pollAlert"
              :type="pollAlertType"
              variant="tonal"
              density="comfortable"
              class="mt-6 rounded-lg border-s-4"
              closable
              @click:close="pollAlert = false"
            >
              {{ pollAlertMessage }}
            </v-alert>
          </v-card-text>
        </v-card>
      </div>

      <!-- Save notification -->
      <v-snackbar
        v-model="saveSuccess"
        color="success"
        variant="flat"
        location="top right"
        :timeout="3000"
      >
        <div class="d-flex align-center">
          <v-icon class="me-2">mdi-check-circle-outline</v-icon>
          <span>Settings saved successfully</span>
        </div>
      </v-snackbar>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";

interface Settings {
  git_sync_concurrency: number;
  git_sync_interval: number;
  image_update_interval: number;
  update_stacks: boolean;
}

interface ImagePollResult {
  checked_at: string;
  status: string;
  unique_images_count: number;
  updated_images_count: number;
  restarted_services: string[];
  details: string[];
  error?: string;
}

const settings = ref<Settings>({
  git_sync_concurrency: 5,
  git_sync_interval: 5,
  image_update_interval: 0,
  update_stacks: false,
});

const loading = ref(false);
const saving = ref(false);
const saveSuccess = ref(false);

const polling = ref(false);
const pollAlert = ref(false);
const pollAlertType = ref<"success" | "error" | "info">("info");
const pollAlertMessage = ref("");

const fetchSettings = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/settings");
    if (response.ok) {
      const data = await response.json();
      settings.value = {
        git_sync_concurrency: data.git_sync_concurrency ?? 5,
        git_sync_interval: data.git_sync_interval ?? 5,
        image_update_interval: data.image_update_interval ?? 0,
        update_stacks: !!data.update_stacks,
      };
    }
  } catch (error) {
    console.error("Failed to fetch settings:", error);
  } finally {
    loading.value = false;
  }
};

const saveSettings = async () => {
  saving.value = true;
  try {
    const response = await fetch("/api/settings", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(settings.value),
    });
    if (response.ok) {
      saveSuccess.value = true;
    } else {
      console.error("Failed to save settings:", await response.text());
    }
  } catch (error) {
    console.error("Failed to save settings:", error);
  } finally {
    saving.value = false;
  }
};

const triggerManualPoll = async () => {
  polling.value = true;
  pollAlert.value = false;
  try {
    const response = await fetch("/api/images/poll", { method: "POST" });
    if (response.ok) {
      const result: ImagePollResult = await response.json();
      pollAlertType.value = result.status === "error" ? "error" : "success";
      pollAlertMessage.value = `Image check finished: scanned ${result.unique_images_count} images, updated ${result.updated_images_count} images, restarted ${result.restarted_services?.length || 0} services.`;
      pollAlert.value = true;
    } else {
      const err = await response.text();
      pollAlertType.value = "error";
      pollAlertMessage.value = `Failed to trigger image update check: ${err}`;
      pollAlert.value = true;
    }
  } catch (error: any) {
    pollAlertType.value = "error";
    pollAlertMessage.value = `Error triggering check: ${error.message || error}`;
    pollAlert.value = true;
  } finally {
    polling.value = false;
  }
};

onMounted(() => {
  fetchSettings();
});
</script>

<style scoped>
.max-width-900 {
  max-width: 900px;
}
.gap-2 {
  gap: 8px;
}
.gap-6 {
  gap: 24px;
}
</style>
