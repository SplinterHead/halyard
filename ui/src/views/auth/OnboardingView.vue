<template>
  <section class="surface-card auth-card">
    <div class="card-header">
      <div class="flex-row">
        <img src="/logo.png" alt="Halyard" class="rail-brand" />
        <span class="card-title">Halyard Cluster Setup</span>
      </div>
      <span class="status-badge">Initial Setup</span>
    </div>

    <form class="card-body" style="display: flex; flex-direction: column; gap: 16px;" @submit.prevent="handleOnboarding">
      <div>
        <h1 class="page-title" style="font-size: 20px;">Initialize operator account</h1>
        <p class="page-subtitle" style="margin-block-start: 4px;">
          Halyard is running in standalone mode. Create your primary operator credentials below.
        </p>
      </div>

      <div v-if="errorMessage" class="alert-banner alert-err">
        <span>{{ errorMessage }}</span>
      </div>

      <div class="form-field">
        <label class="form-label" for="onb-realname">Operator Real Name</label>
        <input
          id="onb-realname"
          v-model="realName"
          type="text"
          class="form-input"
          placeholder="Jane Doe"
          required
        />
      </div>

      <div class="form-field">
        <label class="form-label" for="onb-username">System Username</label>
        <input
          id="onb-username"
          v-model="username"
          type="text"
          class="form-input"
          placeholder="admin"
          required
        />
        <span v-if="username && !usernameValid" class="form-error">
          Alphanumeric, underscores, and hyphens only.
        </span>
      </div>

      <div class="form-field">
        <label class="form-label" for="onb-password">Master Password</label>
        <div class="flex-row">
          <input
            id="onb-password"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            class="form-input"
            placeholder="••••••••"
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

      <div class="form-field">
        <label class="form-label" for="onb-confirm">Confirm Password</label>
        <input
          id="onb-confirm"
          v-model="confirmPassword"
          :type="showPassword ? 'text' : 'password'"
          class="form-input"
          placeholder="••••••••"
          required
        />
      </div>

      <div class="nested-box" style="display: flex; flex-direction: column; gap: 6px; font-size: 12px;">
        <div class="flex-row">
          <span class="status-dot" :class="passLengthValid ? 'dot-ok' : ''"></span>
          <span :class="passLengthValid ? 'text-ok' : 'text-2'">
            At least 8 characters ({{ password.length }}/8)
          </span>
        </div>
        <div class="flex-row">
          <span class="status-dot" :class="passwordsMatch ? 'dot-ok' : ''"></span>
          <span :class="passwordsMatch ? 'text-ok' : 'text-2'">
            Passwords match
          </span>
        </div>
      </div>

      <button
        type="submit"
        class="btn btn-primary"
        style="height: 38px; width: 100%;"
        :disabled="loading || !canSubmit"
      >
        {{ loading ? 'Initializing...' : 'Initialize Account' }}
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();

const realName = ref('');
const username = ref('');
const password = ref('');
const confirmPassword = ref('');
const showPassword = ref(false);
const loading = ref(false);
const errorMessage = ref('');

const usernameValid = computed(() => /^[a-zA-Z0-9_-]+$/.test(username.value));
const passLengthValid = computed(() => password.value.length >= 8);
const passwordsMatch = computed(() => password.value !== '' && password.value === confirmPassword.value);
const canSubmit = computed(() => {
  return Boolean(
    realName.value.trim() &&
      usernameValid.value &&
      passLengthValid.value &&
      passwordsMatch.value
  );
});

const handleOnboarding = async () => {
  if (!canSubmit.value) return;

  loading.value = true;
  errorMessage.value = '';

  try {
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value,
        real_name: realName.value,
        password: password.value,
      }),
    });

    if (!response.ok) {
      const errText = await response.text();
      throw new Error(errText || 'Setup configuration failed.');
    }

    const data = await response.json();
    localStorage.setItem('halyard_token', data.token);
    localStorage.setItem('halyard_user', JSON.stringify(data.user));
    router.push('/');
  } catch (err: any) {
    errorMessage.value = err.message || 'Setup request failed. Ensure network is reachable.';
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.auth-card {
  width: 100%;
  max-width: 460px;
}
</style>
