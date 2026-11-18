<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <div class="status-line">
          <button type="button" class="section-link" @click="$router.back()">← Back</button>
        </div>
        <h1 class="page-title">User Profile</h1>
        <p class="page-subtitle">Manage your operator account, password, and appearance preferences.</p>
      </div>
    </div>

    <div class="detail-grid">
      <!-- Profile Info (5 cols) -->
      <section class="surface-card col-5" aria-labelledby="profile-account-title">
        <div class="card-header">
          <span id="profile-account-title" class="card-title">Operator Account</span>
        </div>
        <div class="card-body d-flex flex-column align-center justify-center text-center" style="padding: 36px 20px; gap: 12px;">
          <div class="profile-avatar">{{ userInitials }}</div>
          <div class="fw-600 text-1" style="font-size: 18px;">{{ user.real_name || 'Operator' }}</div>
          <div class="mono text-2">{{ user.username || 'unknown' }}</div>
        </div>
      </section>

      <!-- Change Password (7 cols) -->
      <section class="surface-card col-7" aria-labelledby="profile-password-title">
        <div class="card-header">
          <span id="profile-password-title" class="card-title">Change Password</span>
        </div>
        <form class="card-body" style="display: flex; flex-direction: column; gap: 14px;" @submit.prevent="handlePasswordUpdate">
          <div
            v-if="statusMessage"
            class="alert-banner"
            :class="statusType === 'error' ? 'alert-err' : 'alert-ok'"
          >
            <span>{{ statusMessage }}</span>
            <button type="button" class="btn btn-sm" @click="statusMessage = ''">Dismiss</button>
          </div>

          <div class="form-field">
            <label class="form-label" for="pw-current">Current Password</label>
            <input
              id="pw-current"
              v-model="currentPassword"
              type="password"
              class="form-input"
              placeholder="••••••••"
              required
            />
          </div>

          <div class="form-field">
            <label class="form-label" for="pw-new">New Password</label>
            <input
              id="pw-new"
              v-model="newPassword"
              type="password"
              class="form-input"
              placeholder="••••••••"
              required
            />
            <span v-if="newPassword && newPassword.length < 8" class="form-error">
              Must be at least 8 characters
            </span>
          </div>

          <div class="form-field">
            <label class="form-label" for="pw-confirm">Confirm New Password</label>
            <input
              id="pw-confirm"
              v-model="confirmPassword"
              type="password"
              class="form-input"
              placeholder="••••••••"
              required
            />
            <span v-if="confirmPassword && confirmPassword !== newPassword" class="form-error">
              Passwords must match
            </span>
          </div>

          <div class="d-flex justify-end" style="margin-block-start: 4px;">
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="loading || !passwordFormValid"
            >
              {{ loading ? 'Updating...' : 'Update Password' }}
            </button>
          </div>
        </form>
      </section>

      <!-- Appearance & Log Colors (12 cols) -->
      <section class="surface-card col-12" aria-labelledby="profile-appearance-title">
        <div class="card-header">
          <span id="profile-appearance-title" class="card-title">Appearance & Log Colors</span>
          <button type="button" class="btn btn-primary btn-sm" :disabled="savingPrefs" @click="savePreferences">
            {{ savingPrefs ? 'Saving...' : 'Save Preferences' }}
          </button>
        </div>

        <div class="card-body" style="display: flex; flex-direction: column; gap: 20px;">
          <div class="flex-between" style="flex-wrap: wrap; gap: 12px;">
            <div>
              <div class="fw-500 text-1">Color Scheme</div>
              <div class="text-2" style="font-size: 12.5px;">Switch between Deep Slate (dark), Alabaster (light), or system default.</div>
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

          <div style="border-block-start: 1px solid var(--border-2); padding-block-start: 16px;">
            <div class="fw-500 text-1" style="margin-block-end: 4px;">Container Log ANSI Colors</div>
            <div class="text-2" style="font-size: 12.5px; margin-block-end: 14px;">
              Customize the standard ANSI colors used to format container logs.
            </div>

            <div class="ansi-grid">
              <div v-for="color in colors" :key="color.key" class="nested-box flex-between" style="gap: 10px;">
                <div class="config-field" style="flex: 1; min-width: 0;">
                  <label class="config-label" :for="`ansi-${color.key}`">{{ color.label }}</label>
                  <input
                    :id="`ansi-${color.key}`"
                    v-model="preferences['ansi-' + color.key]"
                    type="text"
                    class="form-input mono"
                    style="height: 28px; font-size: 12px;"
                    placeholder="#000000"
                  />
                </div>
                <input
                  v-model="preferences['ansi-' + color.key]"
                  type="color"
                  class="color-picker-input"
                  :aria-label="`${color.label} color picker`"
                />
              </div>
            </div>
          </div>

          <div v-if="prefsSuccess" class="alert-banner alert-ok">
            <span>Preferences saved successfully.</span>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useTheme } from '../../theme';

const { schemePreference, systemMode, setScheme } = useTheme();

const user = ref<any>({});
const userInitials = ref('??');

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

const preferences = ref<Record<string, string>>({ ...defaultColors });
const savingPrefs = ref(false);
const prefsSuccess = ref(false);

const currentPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const loading = ref(false);
const statusMessage = ref('');
const statusType = ref<'success' | 'error'>('success');

const passwordFormValid = computed(() => {
  return Boolean(
    currentPassword.value &&
      newPassword.value.length >= 8 &&
      newPassword.value === confirmPassword.value
  );
});

onMounted(() => {
  const userStr = localStorage.getItem('halyard_user');
  if (userStr) {
    try {
      const parsed = JSON.parse(userStr);
      user.value = parsed;
      if (parsed.real_name) {
        userInitials.value = parsed.real_name
          .split(' ')
          .map((n: string) => n[0])
          .join('')
          .substring(0, 2)
          .toUpperCase();
      } else if (parsed.username) {
        userInitials.value = parsed.username.substring(0, 2).toUpperCase();
      }

      if (parsed.preferences) {
        preferences.value = { ...defaultColors, ...parsed.preferences };
      }
    } catch (e) {
      console.error('Failed to parse user', e);
    }
  }
});

const handlePasswordUpdate = async () => {
  if (!passwordFormValid.value) return;

  loading.value = true;
  statusMessage.value = '';

  try {
    const token = localStorage.getItem('halyard_token');
    const res = await fetch('/api/auth/password', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        current_password: currentPassword.value,
        new_password: newPassword.value,
      }),
    });

    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText || 'Failed to update password');
    }

    statusType.value = 'success';
    statusMessage.value = 'Password updated successfully';
    currentPassword.value = '';
    newPassword.value = '';
    confirmPassword.value = '';
  } catch (err: any) {
    statusType.value = 'error';
    statusMessage.value = err.message || 'An error occurred';
  } finally {
    loading.value = false;
  }
};

const savePreferences = async () => {
  savingPrefs.value = true;
  prefsSuccess.value = false;

  try {
    const token = localStorage.getItem('halyard_token');
    const res = await fetch('/api/user/preferences', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(preferences.value),
    });

    if (!res.ok) {
      throw new Error('Failed to save preferences');
    }

    const updatedPrefs = await res.json();
    if (user.value) {
      user.value.preferences = updatedPrefs;
      localStorage.setItem('halyard_user', JSON.stringify(user.value));
    }

    const root = document.documentElement;
    for (const [key, value] of Object.entries(updatedPrefs)) {
      if (value && key.startsWith('ansi-')) {
        root.style.setProperty(`--${key}`, value as string);
      }
    }

    prefsSuccess.value = true;
    setTimeout(() => { prefsSuccess.value = false; }, 3000);
  } catch (err) {
    console.error('Failed to save preferences:', err);
  } finally {
    savingPrefs.value = false;
  }
};
</script>

<style scoped>
.profile-avatar {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: var(--surface-2);
  border: 1px solid var(--border-1);
  color: var(--text-1);
  display: grid;
  place-items: center;
  font-size: 28px;
  font-weight: 600;
}

.ansi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

@container (width < 760px) {
  .ansi-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@container (width < 460px) {
  .ansi-grid {
    grid-template-columns: 1fr;
  }
}

.color-picker-input {
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--border-1);
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  flex-shrink: 0;
}
</style>
