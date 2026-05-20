<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Git Settings</h1>
      <v-spacer></v-spacer>
      <v-btn color="primary" @click="saveSettings" :loading="saving" flat>
        Save Changes
      </v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <div class="pa-4">
      <v-card border flat class="bg-surface max-width-800">
        <v-card-text class="pa-6">
          <div class="d-flex align-center mb-6">
            <v-icon color="primary" class="me-4" size="32">mdi-sync</v-icon>
            <div>
              <div class="text-h6 font-weight-bold">Sync Concurrency</div>
              <div class="text-caption text-grey">
                How many Git Syncs to process at the same time. Set to 0 for
                full parallelism, this could impact the system with lots of
                active syncs
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
                v-model="settings.git_sync_concurrency"
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
            <v-icon color="success" class="me-4" size="32"
              >mdi-clock-outline</v-icon
            >
            <div>
              <div class="text-h6 font-weight-bold">Sync Interval</div>
              <div class="text-caption text-grey">
                How often (in minutes) to check source repositories for new
                commits when Auto Sync is enabled.
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
                v-model="settings.git_sync_interval"
                type="number"
                variant="outlined"
                density="compact"
                style="width: 80px"
                hide-details
              ></v-text-field>
            </template>
          </v-slider>

          <v-alert
            v-if="success"
            type="success"
            variant="tonal"
            density="compact"
            class="mt-6"
            closable
            @click:close="success = false"
          >
            Settings saved successfully
          </v-alert>
        </v-card-text>
      </v-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";

interface Settings {
  git_sync_concurrency: number;
  git_sync_interval: number;
}

const settings = ref<Settings>({
  git_sync_concurrency: 5,
  git_sync_interval: 5,
});
const loading = ref(false);
const saving = ref(false);
const success = ref(false);

const fetchSettings = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/settings");
    settings.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch settings:", error);
  } finally {
    loading.value = false;
  }
};

const saveSettings = async () => {
  saving.value = true;
  success.value = false;
  try {
    const response = await fetch("/api/settings", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(settings.value),
    });
    if (response.ok) {
      success.value = true;
      setTimeout(() => {
        success.value = false;
      }, 3000);
    }
  } catch (error) {
    console.error("Failed to save settings:", error);
  } finally {
    saving.value = false;
  }
};

onMounted(fetchSettings);
</script>

<style scoped>
.max-width-800 {
  max-width: 800px;
}
</style>
