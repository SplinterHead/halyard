<template>
  <div class="h-100 d-flex flex-column gap-4">
    <!-- Header -->
    <div class="d-flex align-center justify-space-between glass-panel px-6 py-4 rounded-lg">
      <div class="d-flex align-center gap-4">
        <v-btn icon="mdi-arrow-left" variant="text" size="small" @click="$router.back()"></v-btn>
        <div>
          <h2 class="text-h5 font-weight-bold text-gradient mb-1">User Profile</h2>
          <div class="text-caption text-grey">Manage your account and security settings</div>
        </div>
      </div>
    </div>

    <!-- Content -->
    <v-row class="flex-grow-1">
      <!-- Profile Info -->
      <v-col cols="12" md="6">
        <v-card class="glass-panel h-100 rounded-lg pa-6 d-flex flex-column align-center justify-center">
          <v-avatar size="256" color="primary" variant="tonal" class="mb-6" style="border: 4px solid rgba(139, 92, 246, 0.5)">
            <span class="text-h1 font-weight-bold">{{ userInitials }}</span>
          </v-avatar>
          <h3 class="text-h4 font-weight-bold mb-2">{{ user.real_name || 'Operator' }}</h3>
          <div class="text-subtitle-1 text-grey">{{ user.username || 'unknown' }}</div>
        </v-card>
      </v-col>

      <!-- Password Change -->
      <v-col cols="12" md="6">
        <v-card class="glass-panel h-100 rounded-lg pa-6">
          <h3 class="text-h6 font-weight-bold mb-4">Change Password</h3>
          <v-divider class="mb-6 border-opacity-25" color="white"></v-divider>

          <v-form ref="passwordForm" @submit.prevent="handlePasswordUpdate" v-model="formValid">
            <v-alert
              v-if="statusMessage"
              :type="statusType"
              variant="tonal"
              density="compact"
              class="mb-6 text-caption rounded-lg border-s-4"
              closable
              @click:close="statusMessage = ''"
            >
              {{ statusMessage }}
            </v-alert>

            <div class="mb-4">
              <div class="text-caption font-weight-bold mb-2">
                Current Password
              </div>
              <v-text-field
                v-model="currentPassword"
                type="password"
                placeholder="••••••••"
                prepend-inner-icon="mdi-lock-outline"
                variant="outlined"
                density="comfortable"
                color="primary"
                :rules="[v => !!v || 'Required']"
                hide-details="auto"
                bg-color="rgba(0, 0, 0, 0.2)"
                required
              ></v-text-field>
            </div>

            <div class="mb-4">
              <div class="text-caption font-weight-bold mb-2">
                New Password
              </div>
              <v-text-field
                v-model="newPassword"
                type="password"
                placeholder="••••••••"
                prepend-inner-icon="mdi-lock-reset"
                variant="outlined"
                density="comfortable"
                color="primary"
                :rules="[
                  v => !!v || 'Required',
                  v => v.length >= 8 || 'Must be at least 8 characters'
                ]"
                hide-details="auto"
                bg-color="rgba(0, 0, 0, 0.2)"
                required
              ></v-text-field>
            </div>

            <div class="mb-6">
              <div class="text-caption font-weight-bold mb-2">
                Confirm New Password
              </div>
              <v-text-field
                v-model="confirmPassword"
                type="password"
                placeholder="••••••••"
                prepend-inner-icon="mdi-lock-check"
                variant="outlined"
                density="comfortable"
                color="primary"
                :rules="[
                  v => !!v || 'Required',
                  v => v === newPassword || 'Passwords must match'
                ]"
                hide-details="auto"
                bg-color="rgba(0, 0, 0, 0.2)"
                required
              ></v-text-field>
            </div>

            <v-btn
              type="submit"
              color="primary"
              size="large"
              class="font-weight-bold tracking-wide w-100"
              :loading="loading"
              :disabled="!formValid"
              elevation="4"
              prepend-icon="mdi-content-save"
            >
              Update Password
            </v-btn>
          </v-form>
        </v-card>
      </v-col>
    </v-row>

    <!-- Appearance Settings -->
    <v-row>
      <v-col cols="12">
        <v-card class="glass-panel rounded-lg pa-6">
          <div class="d-flex align-center justify-space-between mb-4">
            <h3 class="text-h6 font-weight-bold">Appearance Settings</h3>
            <v-btn color="primary" @click="savePreferences" :loading="savingPrefs" flat>
              Save Preferences
            </v-btn>
          </div>
          <v-divider class="mb-6 border-opacity-25" color="white"></v-divider>
          
          <div class="d-flex align-center mb-6">
            <v-icon color="primary" class="me-4" size="32">mdi-palette</v-icon>
            <div>
              <div class="text-h6 font-weight-bold">Container Log Colors</div>
              <div class="text-caption text-grey">
                Customize the standard ANSI colors used to format container logs.
              </div>
            </div>
          </div>

          <v-row class="mt-4">
            <v-col cols="12" sm="6" md="3" v-for="color in colors" :key="color.key">
              <div class="d-flex align-center mb-2">
                <span class="text-subtitle-2 text-capitalize me-auto">{{ color.label }}</span>
              </div>
              <v-text-field
                v-model="preferences['ansi-' + color.key]"
                variant="outlined"
                density="compact"
                hide-details
                placeholder="#000000"
              >
                <template v-slot:append-inner>
                  <input
                    type="color"
                    v-model="preferences['ansi-' + color.key]"
                    class="color-picker-input cursor-pointer"
                  />
                </template>
              </v-text-field>
            </v-col>
          </v-row>

          <v-alert
            v-if="prefsSuccess"
            type="success"
            variant="tonal"
            density="compact"
            class="mt-6"
            closable
            @click:close="prefsSuccess = false"
          >
            Preferences saved successfully
          </v-alert>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'

const passwordForm = ref<any>(null)
const user = ref<any>({})
const userInitials = ref('??')

const colors = [
  { key: 'black', label: 'Black' },
  { key: 'red', label: 'Red' },
  { key: 'green', label: 'Green' },
  { key: 'yellow', label: 'Yellow' },
  { key: 'blue', label: 'Blue' },
  { key: 'magenta', label: 'Magenta' },
  { key: 'cyan', label: 'Cyan' },
  { key: 'white', label: 'White' },
]

const defaultColors: Record<string, string> = {
  'ansi-black': '#000000',
  'ansi-red': '#bb0000',
  'ansi-green': '#00bb00',
  'ansi-yellow': '#bbbb00',
  'ansi-blue': '#0000bb',
  'ansi-magenta': '#bb00bb',
  'ansi-cyan': '#00bbbb',
  'ansi-white': '#ffffff',
}

const preferences = ref<Record<string, string>>({})
const savingPrefs = ref(false)
const prefsSuccess = ref(false)

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const formValid = ref(false)
const loading = ref(false)

const statusMessage = ref('')
const statusType = ref<'success' | 'error'>('success')

onMounted(() => {
  const userStr = localStorage.getItem('halyard_user')
  if (userStr) {
    try {
      const parsed = JSON.parse(userStr)
      user.value = parsed
      if (parsed.real_name) {
        userInitials.value = parsed.real_name.split(' ').map((n: string) => n[0]).join('').substring(0, 2).toUpperCase()
      } else if (parsed.username) {
        userInitials.value = parsed.username.substring(0, 2).toUpperCase()
      }
      
      if (parsed.preferences) {
        preferences.value = { ...defaultColors, ...parsed.preferences }
      } else {
        preferences.value = { ...defaultColors }
      }
    } catch (e) {
      console.error('Failed to parse user', e)
    }
  }
})

const handlePasswordUpdate = async () => {
  if (!formValid.value) return

  loading.value = true
  statusMessage.value = ''

  try {
    const token = localStorage.getItem('halyard_token')
    const res = await fetch('/api/auth/password', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        current_password: currentPassword.value,
        new_password: newPassword.value
      })
    })

    if (!res.ok) {
      const errText = await res.text()
      throw new Error(errText || 'Failed to update password')
    }

    statusType.value = 'success'
    statusMessage.value = 'Password updated successfully'
    
    // Reset form
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    
    nextTick(() => {
      passwordForm.value?.resetValidation()
    })
  } catch (err: any) {
    console.error('Password update failed:', err)
    statusType.value = 'error'
    statusMessage.value = err.message || 'An error occurred'
  } finally {
    loading.value = false
  }
}

const savePreferences = async () => {
  savingPrefs.value = true
  prefsSuccess.value = false

  try {
    const token = localStorage.getItem('halyard_token')
    const res = await fetch('/api/user/preferences', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(preferences.value)
    })

    if (!res.ok) {
      throw new Error('Failed to save preferences')
    }

    const updatedPrefs = await res.json()
    
    // Update local user
    if (user.value) {
      user.value.preferences = updatedPrefs
      localStorage.setItem('halyard_user', JSON.stringify(user.value))
    }

    // Apply CSS vars
    const root = document.documentElement;
    for (const [key, value] of Object.entries(updatedPrefs)) {
      if (value) {
        root.style.setProperty(`--${key}`, value as string);
      }
    }

    prefsSuccess.value = true
    setTimeout(() => prefsSuccess.value = false, 3000)
  } catch (err) {
    console.error('Failed to save preferences:', err)
  } finally {
    savingPrefs.value = false
  }
}
</script>

<style scoped>
.tracking-wider {
  letter-spacing: 0.06em !important;
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
