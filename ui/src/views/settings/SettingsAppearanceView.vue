<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Appearance Settings</h1>
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
            <v-icon color="primary" class="me-4" size="32">mdi-palette</v-icon>
            <div>
              <div class="text-h6 font-weight-bold">Container Log Colors</div>
              <div class="text-caption text-grey">
                Customize the standard ANSI colors used to format container logs. Click the color swatch or enter a HEX code.
              </div>
            </div>
          </div>

          <v-row class="mt-4">
            <v-col cols="12" sm="6" md="4" v-for="color in colors" :key="color.key">
              <div class="d-flex align-center mb-2">
                <span class="text-subtitle-2 text-capitalize me-auto">{{ color.label }}</span>
              </div>
              <v-text-field
                v-model="settings.log_colors['ansi-' + color.key]"
                variant="outlined"
                density="compact"
                hide-details
                placeholder="#000000"
              >
                <template v-slot:append-inner>
                  <input
                    type="color"
                    v-model="settings.log_colors['ansi-' + color.key]"
                    class="color-picker-input cursor-pointer"
                  />
                </template>
              </v-text-field>
            </v-col>
          </v-row>

          <v-alert
            v-if="success"
            type="success"
            variant="tonal"
            density="compact"
            class="mt-6"
            closable
            @click:close="success = false"
          >
            Appearance settings saved successfully
          </v-alert>
        </v-card-text>
      </v-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";

const colors = [
  { key: 'black', label: 'Black' },
  { key: 'red', label: 'Red' },
  { key: 'green', label: 'Green' },
  { key: 'yellow', label: 'Yellow' },
  { key: 'blue', label: 'Blue' },
  { key: 'magenta', label: 'Magenta' },
  { key: 'cyan', label: 'Cyan' },
  { key: 'white', label: 'White' },
];

interface Settings {
  git_sync_concurrency: number;
  git_sync_interval: number;
  log_colors: Record<string, string>;
}

const settings = ref<Settings>({
  git_sync_concurrency: 5,
  git_sync_interval: 5,
  log_colors: {},
});

const defaultColors: Record<string, string> = {
  'ansi-black': '#000000',
  'ansi-red': '#bb0000',
  'ansi-green': '#00bb00',
  'ansi-yellow': '#bbbb00',
  'ansi-blue': '#0000bb',
  'ansi-magenta': '#bb00bb',
  'ansi-cyan': '#00bbbb',
  'ansi-white': '#ffffff',
};

const loading = ref(false);
const saving = ref(false);
const success = ref(false);

const fetchSettings = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/settings");
    const data = await response.json();
    if (!data.log_colors) {
      data.log_colors = { ...defaultColors };
    } else {
      // Ensure all standard keys exist
      for (const [key, defaultVal] of Object.entries(defaultColors)) {
        if (!data.log_colors[key]) {
          data.log_colors[key] = defaultVal;
        }
      }
    }
    settings.value = data;
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
      // Inject the saved CSS variables dynamically so they update instantly without reload
      const root = document.documentElement;
      for (const [key, value] of Object.entries(settings.value.log_colors)) {
        if (value) {
          root.style.setProperty(`--${key}`, value);
        }
      }
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
.color-picker-input {
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  background: transparent;
  border-radius: 4px;
  overflow: hidden;
}
.color-picker-input::-webkit-color-swatch-wrapper {
  padding: 0;
}
.color-picker-input::-webkit-color-swatch {
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 4px;
}
</style>
