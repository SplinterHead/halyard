<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">Settings</h1>
        <p class="page-subtitle">Configure appearance, GitOps reconciliation, and image polling behavior.</p>
      </div>
      <div class="header-actions">
        <button type="button" class="btn btn-primary" :disabled="saving" @click="saveSettings">
          {{ saving ? 'Saving...' : 'Save Changes' }}
        </button>
      </div>
    </div>

    <div v-if="saveSuccess" class="alert-banner alert-ok">
      <span>Settings saved successfully.</span>
    </div>

    <Loader :loading="loading" />

    <!-- Appearance Settings Card -->
    <section class="surface-card" aria-labelledby="settings-appearance-title">
      <div class="card-header">
        <span id="settings-appearance-title" class="card-title">Appearance</span>
      </div>
      <div class="card-body">
        <div class="flex-between" style="flex-wrap: wrap; gap: 16px;">
          <div>
            <div class="fw-500 text-1">Color Scheme</div>
            <div class="text-2" style="font-size: 12.5px;">
              Choose between Deep Slate (dark), Alabaster (light), or follow your operating system preference.
            </div>
          </div>
          <div class="segmented-control" role="group" aria-label="Color scheme">
            <button
              type="button"
              class="segmented-btn"
              :aria-pressed="schemePreference === 'system'"
              @click="setScheme('system')"
            >
              System ({{ systemMode === 'dark' ? 'Dark' : 'Light' }})
            </button>
            <button
              type="button"
              class="segmented-btn"
              :aria-pressed="schemePreference === 'dark'"
              @click="setScheme('dark')"
            >
              Dark (Deep Slate)
            </button>
            <button
              type="button"
              class="segmented-btn"
              :aria-pressed="schemePreference === 'light'"
              @click="setScheme('light')"
            >
              Light (Alabaster)
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- GitOps Settings Card -->
    <section class="surface-card" aria-labelledby="settings-gitops-title">
      <div class="card-header">
        <span id="settings-gitops-title" class="card-title">GitOps Reconciliation</span>
      </div>
      <div class="card-body" style="display: flex; flex-direction: column; gap: 20px;">
        <div class="config-grid">
          <div class="form-field">
            <label class="form-label" for="git-concurrency">Sync Concurrency</label>
            <input
              id="git-concurrency"
              v-model.number="settings.git_sync_concurrency"
              type="number"
              min="0"
              max="20"
              class="form-input"
            />
            <span class="form-hint">
              How many Git Syncs to process concurrently. Set to 0 for full parallelism.
            </span>
          </div>

          <div class="form-field">
            <label class="form-label" for="git-interval">Sync Interval (minutes)</label>
            <input
              id="git-interval"
              v-model.number="settings.git_sync_interval"
              type="number"
              min="1"
              max="60"
              class="form-input"
            />
            <span class="form-hint">
              How often to check source repositories for new commits when Auto Sync is enabled.
            </span>
          </div>
        </div>
      </div>
    </section>

    <!-- Image Update Polling Card -->
    <section class="surface-card" aria-labelledby="settings-image-poll-title">
      <div class="card-header">
        <span id="settings-image-poll-title" class="card-title">Image Update Polling</span>
        <button type="button" class="btn btn-sm" :disabled="polling" @click="triggerManualPoll">
          {{ polling ? 'Checking...' : 'Check Now' }}
        </button>
      </div>
      <div class="card-body" style="display: flex; flex-direction: column; gap: 18px;">
        <div class="form-field" style="max-inline-size: 360px;">
          <div class="flex-between">
            <label class="form-label" for="image-interval">Check Interval (hours)</label>
            <span class="status-badge">
              {{ settings.image_update_interval > 0 ? `Every ${settings.image_update_interval} hr${settings.image_update_interval > 1 ? 's' : ''}` : 'Disabled' }}
            </span>
          </div>
          <input
            id="image-interval"
            v-model.number="settings.image_update_interval"
            type="number"
            min="0"
            max="72"
            class="form-input"
          />
          <span class="form-hint">
            Set to 0 to disable automatic scheduled checks (manual check remains available).
          </span>
        </div>

        <label class="nested-box flex-between" style="cursor: pointer;">
          <span>
            <strong class="fw-500 text-1" style="display: block;">
              Update Stacks
              <span class="status-badge" style="margin-inline-start: 6px;">
                {{ settings.update_stacks ? 'Auto-Restart Enabled' : 'Disabled' }}
              </span>
            </strong>
            <span class="text-2" style="font-size: 12px;">
              Automatically trigger rolling service updates when newer images are pulled to nodes.
            </span>
          </span>
          <input v-model="settings.update_stacks" type="checkbox" role="switch" class="toggle-switch" />
        </label>

        <div
          v-if="pollAlert"
          class="alert-banner"
          :class="pollAlertType === 'error' ? 'alert-err' : 'alert-ok'"
        >
          <span>{{ pollAlertMessage }}</span>
          <button type="button" class="btn btn-sm" @click="pollAlert = false">Dismiss</button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import Loader from '../../components/Loader.vue';
import { useTheme } from '../../theme';

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

const { schemePreference, systemMode, setScheme } = useTheme();

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
const pollAlertType = ref<'success' | 'error' | 'info'>('info');
const pollAlertMessage = ref('');

const fetchSettings = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/settings');
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
    console.error('Failed to fetch settings:', error);
  } finally {
    loading.value = false;
  }
};

const saveSettings = async () => {
  saving.value = true;
  saveSuccess.value = false;
  try {
    const response = await fetch('/api/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings.value),
    });
    if (response.ok) {
      saveSuccess.value = true;
      setTimeout(() => { saveSuccess.value = false; }, 3000);
    }
  } catch (error) {
    console.error('Failed to save settings:', error);
  } finally {
    saving.value = false;
  }
};

const triggerManualPoll = async () => {
  polling.value = true;
  pollAlert.value = false;
  try {
    const response = await fetch('/api/images/poll', { method: 'POST' });
    if (response.ok) {
      const result: ImagePollResult = await response.json();
      pollAlertType.value = result.status === 'error' ? 'error' : 'success';
      pollAlertMessage.value = `Image check finished: scanned ${result.unique_images_count} images, updated ${result.updated_images_count} images, restarted ${result.restarted_services?.length || 0} services.`;
      pollAlert.value = true;
    } else {
      pollAlertType.value = 'error';
      pollAlertMessage.value = `Failed to trigger image update check: ${await response.text()}`;
      pollAlert.value = true;
    }
  } catch (error: any) {
    pollAlertType.value = 'error';
    pollAlertMessage.value = `Error triggering check: ${error.message || error}`;
    pollAlert.value = true;
  } finally {
    polling.value = false;
  }
};

onMounted(fetchSettings);
</script>
