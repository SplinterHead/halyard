<template>
  <div class="auth-wrapper d-flex align-center justify-center fill-height w-100">
    <!-- Glow backgrounds -->
    <div class="glow-bg glow-purple"></div>
    <div class="glow-bg glow-blue"></div>

    <v-card 
      class="glass-panel auth-card pa-8 d-flex flex-column"
      width="480"
      max-width="95%"
      elevation="24"
    >
      <div class="text-center mb-6">
        <v-avatar size="64" class="mb-4 animate-float logo-avatar">
          <v-img src="/logo.png" contain />
        </v-avatar>
        <h1 class="text-h4 font-weight-bold mb-1 text-gradient">Cluster Setup</h1>
        <p class="text-caption text-mono text-grey-darken-1 text-uppercase tracking-widest">
          Initialize Single-Operator Account
        </p>
      </div>

      <v-divider class="mb-6 border-opacity-25" color="white"></v-divider>

      <!-- Informative zero-trust header -->
      <v-alert
        type="info"
        variant="tonal"
        class="mb-6 text-caption text-mono rounded-lg border-s-4 info-alert"
        density="comfortable"
      >
        Welcome! Halyard is running in secure standalone mode. Please establish the primary operator account credentials below.
      </v-alert>

      <v-form @submit.prevent="handleOnboarding" v-model="formValid">
        <v-alert
          v-if="errorMessage"
          type="error"
          variant="tonal"
          density="compact"
          class="mb-6 text-caption text-mono rounded-lg border-s-4"
          closable
          @click:close="errorMessage = ''"
        >
          {{ errorMessage }}
        </v-alert>

        <!-- Real Name Input -->
        <div class="mb-4">
          <div class="text-caption text-mono font-weight-bold text-primary mb-2 text-uppercase tracking-wider">
            Operator Real Name
          </div>
          <v-text-field
            v-model="realName"
            placeholder="Lewis England"
            prepend-inner-icon="mdi-badge-account-horizontal-outline"
            variant="outlined"
            density="comfortable"
            color="primary"
            class="text-mono"
            :rules="[v => !!v || 'Real name is required']"
            hide-details="auto"
            bg-color="rgba(0, 0, 0, 0.2)"
            required
            flat
          ></v-text-field>
        </div>

        <!-- Username Input -->
        <div class="mb-4">
          <div class="text-caption text-mono font-weight-bold text-primary mb-2 text-uppercase tracking-wider">
            System Username
          </div>
          <v-text-field
            v-model="username"
            placeholder="lewis-england"
            prepend-inner-icon="mdi-account-outline"
            variant="outlined"
            density="comfortable"
            color="primary"
            class="text-mono"
            :rules="[v => !!v || 'Username is required', v => /^[a-zA-Z0-9_-]+$/.test(v) || 'Alphanumeric, underscores, hyphens only']"
            hide-details="auto"
            bg-color="rgba(0, 0, 0, 0.2)"
            required
            flat
          ></v-text-field>
        </div>

        <!-- Password Input -->
        <div class="mb-4">
          <div class="text-caption text-mono font-weight-bold text-primary mb-2 text-uppercase tracking-wider">
            Master Secure Password
          </div>
          <v-text-field
            v-model="password"
            placeholder="••••••••"
            prepend-inner-icon="mdi-lock-outline"
            :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
            :type="showPassword ? 'text' : 'password'"
            variant="outlined"
            density="comfortable"
            color="primary"
            class="text-mono"
            :rules="[v => !!v || 'Password is required']"
            hide-details="auto"
            bg-color="rgba(0, 0, 0, 0.2)"
            @click:append-inner="showPassword = !showPassword"
            required
            flat
          ></v-text-field>
        </div>

        <!-- Confirm Password Input -->
        <div class="mb-4">
          <div class="text-caption text-mono font-weight-bold text-primary mb-2 text-uppercase tracking-wider">
            Confirm Password
          </div>
          <v-text-field
            v-model="confirmPassword"
            placeholder="••••••••"
            prepend-inner-icon="mdi-lock-check-outline"
            :type="showPassword ? 'text' : 'password'"
            variant="outlined"
            density="comfortable"
            color="primary"
            class="text-mono"
            :rules="[v => !!v || 'Please confirm your password']"
            hide-details="auto"
            bg-color="rgba(0, 0, 0, 0.2)"
            required
            flat
          ></v-text-field>
        </div>

        <!-- Real-Time Password Checklist -->
        <div class="checklist pa-4 rounded-lg bg-black-opacity mb-6 border-thin">
          <div class="text-caption text-mono text-grey mb-2 uppercase tracking-wide">Security Policies:</div>
          
          <div class="d-flex align-center gap-2 mb-2 text-caption text-mono">
            <v-icon size="16" :color="passLengthValid ? 'success' : 'grey'">
              {{ passLengthValid ? 'mdi-check-circle' : 'mdi-circle-outline' }}
            </v-icon>
            <span :class="passLengthValid ? 'text-success' : 'text-grey-darken-1'">
              Length >= 8 characters ({{ password.length }}/8)
            </span>
          </div>

          <div class="d-flex align-center gap-2 text-caption text-mono">
            <v-icon size="16" :color="passwordsMatch ? 'success' : 'grey'">
              {{ passwordsMatch ? 'mdi-check-circle' : 'mdi-circle-outline' }}
            </v-icon>
            <span :class="passwordsMatch ? 'text-success' : 'text-grey-darken-1'">
              Passwords match identically
            </span>
          </div>
        </div>

        <v-btn
          type="submit"
          block
          color="primary"
          size="large"
          class="submit-btn font-weight-bold text-mono tracking-wide mt-2"
          :loading="loading"
          :disabled="!formValid || !passLengthValid || !passwordsMatch"
          elevation="4"
        >
          Initialize Account
          <template v-slot:loader>
            <v-progress-circular indeterminate size="22" width="2" color="white"></v-progress-circular>
          </template>
        </v-btn>
      </v-form>

      <div class="text-center mt-6 text-mono text-caption text-grey">
        Cluster Cryptography: <span class="text-primary font-weight-bold">AES-256 + Bcrypt</span>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const realName = ref('')
const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const formValid = ref(false)
const loading = ref(false)
const errorMessage = ref('')

// Password strength properties
const passLengthValid = computed(() => password.value.length >= 8)
const passwordsMatch = computed(() => password.value !== '' && password.value === confirmPassword.value)

const handleOnboarding = async () => {
  if (!realName.value || !username.value || !password.value || !confirmPassword.value) return
  if (!passLengthValid.value || !passwordsMatch.value) return

  loading.value = true
  errorMessage.value = ''

  try {
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        username: username.value,
        real_name: realName.value,
        password: password.value
      })
    })

    if (!response.ok) {
      const errText = await response.text()
      throw new Error(errText || 'Setup configuration failed.')
    }

    const data = await response.json()

    // Store returned stateless auth tokens to perform seamless login
    localStorage.setItem('halyard_token', data.token)
    localStorage.setItem('halyard_user', JSON.stringify(data.user))

    // Send operator to the dashboard
    router.push('/')
  } catch (err: any) {
    console.error('Onboarding error', err)
    errorMessage.value = err.message || 'Setup request failed. Ensure network is reachable.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-wrapper {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background-color: #030303;
}

/* Beautiful glow background nodes */
.glow-bg {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.15;
  z-index: 0;
}

.glow-purple {
  width: 400px;
  height: 400px;
  background: #8B5CF6;
  top: 15%;
  left: 10%;
}

.glow-blue {
  width: 400px;
  height: 400px;
  background: #3B82F6;
  bottom: 15%;
  right: 10%;
}

.auth-card {
  z-index: 10;
  border-radius: 20px !important;
  background: rgba(15, 23, 42, 0.45) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
}

.logo-avatar {
  border: 2px dashed rgba(139, 92, 246, 0.4);
  padding: 6px;
  background: rgba(0, 0, 0, 0.2);
}

.tracking-widest {
  letter-spacing: 0.12em !important;
}

.tracking-wider {
  letter-spacing: 0.06em !important;
}

.bg-black-opacity {
  background: rgba(0, 0, 0, 0.3);
}

.border-thin {
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.info-alert {
  border-left-width: 4px !important;
  background-color: rgba(59, 130, 246, 0.1) !important;
}

.gap-2 {
  gap: 8px;
}

.submit-btn {
  background: linear-gradient(135deg, #8b5cf6 0%, #3b82f6 100%) !important;
  color: white !important;
  border: none !important;
  height: 48px !important;
  transition: all 0.3s ease;
}

.submit-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 0 20px rgba(139, 92, 246, 0.5) !important;
}

.submit-btn:active {
  transform: translateY(0);
}
</style>
