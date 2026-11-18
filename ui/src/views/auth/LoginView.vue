<template>
  <section class="surface-card auth-card" :class="{ shake: shakeError }">
    <div class="card-header">
      <div class="flex-row">
        <img src="/logo.png" alt="Halyard" class="rail-brand" />
        <span class="card-title">Halyard</span>
      </div>
      <span class="status-badge">
        <span class="status-dot dot-ok"></span>
        Gateway Ready
      </span>
    </div>

    <form class="card-body" style="display: flex; flex-direction: column; gap: 16px;" @submit.prevent="handleLogin">
      <div>
        <h1 class="page-title" style="font-size: 20px;">Sign in to cluster</h1>
        <p class="page-subtitle" style="margin-block-start: 4px;">
          Docker Swarm GitOps &amp; Observability Platform
        </p>
      </div>

      <div v-if="errorMessage" class="alert-banner alert-err">
        <span>{{ errorMessage }}</span>
      </div>

      <div class="form-field">
        <label class="form-label" for="login-user">Operator Username</label>
        <input
          id="login-user"
          v-model="username"
          type="text"
          class="form-input"
          placeholder="halyard-operator"
          autocomplete="username"
          required
        />
      </div>

      <div class="form-field">
        <label class="form-label" for="login-pass">Cluster Password</label>
        <div class="flex-row">
          <input
            id="login-pass"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            class="form-input"
            placeholder="••••••••"
            autocomplete="current-password"
            required
          />
          <button
            type="button"
            class="btn btn-icon"
            style="height: 36px; width: 36px; flex-shrink: 0;"
            :aria-label="showPassword ? 'Hide password' : 'Show password'"
            @click="showPassword = !showPassword"
          >
            <v-icon size="16">{{ showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline' }}</v-icon>
          </button>
        </div>
      </div>

      <button
        type="submit"
        class="btn btn-primary"
        style="height: 38px; width: 100%; margin-block-start: 4px;"
        :disabled="loading || !username || !password"
      >
        {{ loading ? 'Verifying...' : 'Verify Credentials' }}
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();

const username = ref('');
const password = ref('');
const showPassword = ref(false);
const loading = ref(false);
const errorMessage = ref('');
const shakeError = ref(false);

const handleLogin = async () => {
  if (!username.value || !password.value) return;

  loading.value = true;
  errorMessage.value = '';
  shakeError.value = false;

  try {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value,
        password: password.value,
      }),
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error('Authentication failed: Invalid credentials.');
      }
      const errText = await response.text();
      throw new Error(errText || 'An unexpected error occurred.');
    }

    const data = await response.json();
    localStorage.setItem('halyard_token', data.token);
    localStorage.setItem('halyard_user', JSON.stringify(data.user));
    router.push('/');
  } catch (err: any) {
    errorMessage.value = err.message || 'Connection failed. Ensure manager is online.';
    triggerShake();
  } finally {
    loading.value = false;
  }
};

const triggerShake = () => {
  shakeError.value = true;
  setTimeout(() => {
    shakeError.value = false;
  }, 400);
};
</script>

<style scoped>
.auth-card {
  width: 100%;
  max-width: 420px;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20%, 60% { transform: translateX(-6px); }
  40%, 80% { transform: translateX(6px); }
}

.shake {
  animation: shake 0.35s ease-in-out;
}
</style>
