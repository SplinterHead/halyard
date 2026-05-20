<template>
  <div class="auth-wrapper d-flex align-center justify-center fill-height w-100">
    <!-- Glow backgrounds -->
    <div class="glow-bg glow-purple"></div>
    <div class="glow-bg glow-blue"></div>

    <v-card 
      class="glass-panel auth-card pa-8 d-flex flex-column"
      :class="{ 'shake': shakeError }"
      width="440"
      max-width="95%"
      elevation="24"
    >
      <div class="text-center mb-6">
        <v-avatar size="64" class="mb-4 animate-float logo-avatar">
          <v-img src="/logo.png" contain />
        </v-avatar>
        <h1 class="text-h4 font-weight-bold mb-1 text-gradient">Halyard</h1>
        <p class="text-caption text-mono text-grey-darken-1 text-uppercase tracking-widest">
          Docker Swarm GitOps & Observability
        </p>
      </div>

      <v-divider class="mb-6 border-opacity-25" color="white"></v-divider>

      <v-form @submit.prevent="handleLogin" v-model="formValid">
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

        <div class="mb-4">
          <div class="text-caption text-mono font-weight-bold text-primary mb-2 text-uppercase tracking-wider">
            Operator Username
          </div>
          <v-text-field
            v-model="username"
            placeholder="halyard-operator"
            prepend-inner-icon="mdi-account-outline"
            variant="outlined"
            density="comfortable"
            color="primary"
            class="text-mono"
            :rules="[v => !!v || 'Username is required']"
            hide-details="auto"
            bg-color="rgba(0, 0, 0, 0.2)"
            required
            flat
          ></v-text-field>
        </div>

        <div class="mb-6">
          <div class="text-caption text-mono font-weight-bold text-primary mb-2 text-uppercase tracking-wider">
            Cluster Password
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

        <v-btn
          type="submit"
          block
          color="primary"
          size="large"
          class="submit-btn font-weight-bold text-mono tracking-wide mt-2"
          :loading="loading"
          :disabled="!formValid"
          elevation="4"
        >
          Verify Credentials
          <template v-slot:loader>
            <v-progress-circular indeterminate size="22" width="2" color="white"></v-progress-circular>
          </template>
        </v-btn>
      </v-form>

      <div class="text-center mt-6 text-mono text-caption text-grey">
        Cluster Gateway Mode: <span class="text-success font-weight-bold">Zero-Trust Secured</span>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const formValid = ref(false)
const loading = ref(false)
const errorMessage = ref('')
const shakeError = ref(false)

const handleLogin = async () => {
  if (!username.value || !password.value) return

  loading.value = ref(true).value
  errorMessage.value = ''
  shakeError.value = false

  try {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        username: username.value,
        password: password.value
      })
    })

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error('Authentication failed: Invalid credentials.')
      }
      const errText = await response.text()
      throw new Error(errText || 'An unexpected error occurred.')
    }

    const data = await response.json()
    
    // Persist login token and user details
    localStorage.setItem('halyard_token', data.token)
    localStorage.setItem('halyard_user', JSON.stringify(data.user))

    // Successful login: redirect to dashboard
    router.push('/')
  } catch (err: any) {
    console.error('Login error', err)
    errorMessage.value = err.message || 'Connection failed. Ensure manager is online.'
    triggerShake()
  } finally {
    loading.value = false
  }
}

const triggerShake = () => {
  shakeError.value = true
  setTimeout(() => {
    shakeError.value = false
  }, 400)
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

/* High premium credentials-error shaking keyframe */
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20%, 60% { transform: translateX(-8px); }
  40%, 80% { transform: translateX(8px); }
}

.shake {
  animation: shake 0.4s ease-in-out;
}
</style>
