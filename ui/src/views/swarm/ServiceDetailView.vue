<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <div class="status-line">
          <button type="button" class="section-link" @click="$router.back()">
            ← {{ service?.stack ? `Stack: ${service.stack}` : 'Services' }}
          </button>
        </div>
        <h1 class="page-title">
          <span>{{ service ? `Service: ${service.name}` : 'Service Detail' }}</span>
          <span v-if="service" class="status-badge">
            <span class="status-dot" :class="service.running === service.replicas ? 'dot-ok' : 'dot-warn'"></span>
            {{ service.running }} / {{ service.replicas }}
          </span>
        </h1>
        <p v-if="service" class="page-subtitle">
          Stack: {{ service.stack || '—' }} · {{ service.mode }}
        </p>
      </div>

      <div class="header-actions">
        <v-menu v-if="service">
          <template v-slot:activator="{ props }">
            <button type="button" class="btn" v-bind="props" :disabled="actionLoading">
              <v-icon size="15">mdi-dots-vertical</v-icon>
              <span>Actions</span>
            </button>
          </template>
          <v-list class="app-menu-list" density="compact" elevation="0">
            <v-list-item @click="performAction('restart')">
              <template v-slot:prepend><v-icon size="16" class="text-warn">mdi-restart</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Force Restart</v-list-item-title>
            </v-list-item>
            <v-list-item :disabled="service.mode === 'global'" @click="performAction('stop')">
              <template v-slot:prepend><v-icon size="16" class="text-warn">mdi-stop-circle-outline</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Stop (Scale to 0)</v-list-item-title>
            </v-list-item>
            <v-list-item :disabled="service.mode === 'global'" @click="performAction('start')">
              <template v-slot:prepend><v-icon size="16" class="text-ok">mdi-play-circle-outline</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Start Service</v-list-item-title>
            </v-list-item>
            <v-list-item @click="performAction('rollback')">
              <template v-slot:prepend><v-icon size="16" class="text-accent">mdi-undo</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Rollback Spec</v-list-item-title>
            </v-list-item>
            <v-list-item :disabled="service.mode === 'global'" @click="openScaleDialog">
              <template v-slot:prepend><v-icon size="16" class="text-accent">mdi-arrow-up-down</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Scale Service</v-list-item-title>
            </v-list-item>
            <v-divider class="my-1" style="border-color: var(--border-2); opacity: 1;"></v-divider>
            <v-list-item @click="confirmDelete">
              <template v-slot:prepend><v-icon size="16" class="text-err">mdi-delete-outline</v-icon></template>
              <v-list-item-title class="text-err" style="font-size: 12.5px;">Delete Service</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>

        <button type="button" class="btn btn-icon" :disabled="loading" aria-label="Refresh" @click="fetchDetails">
          <v-icon size="16">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <Loader :loading="loading" />

    <div v-if="service" class="detail-grid">
      <!-- Runtime Configuration (8 cols) -->
      <section class="surface-card col-8" aria-labelledby="svc-config-title">
        <div class="card-header">
          <span id="svc-config-title" class="card-title">Runtime Configuration</span>
        </div>
        <div class="card-body">
          <div class="config-grid">
            <div class="config-field">
              <span class="config-label">Image</span>
              <span class="config-val mono text-1">{{ parsedImage.name }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">SHA</span>
              <span class="config-val mono text-1">{{ parsedImage.sha }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">Restart Policy</span>
              <span class="config-val fw-500">{{ service.restart_policy || '—' }}</span>
            </div>
            <div class="config-field" style="grid-column: 1 / -1;">
              <span class="config-label">Placement Constraints</span>
              <div v-if="service.constraints?.length" class="flex-row" style="flex-wrap: wrap; gap: 6px; margin-block-start: 4px;">
                <span v-for="c in service.constraints" :key="c" class="status-badge mono">{{ c }}</span>
              </div>
              <span v-else class="text-3" style="font-size: 13px;">No placement constraints</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Networking (4 cols) -->
      <section class="surface-card col-4" aria-labelledby="svc-net-title">
        <div class="card-header">
          <span id="svc-net-title" class="card-title">Networking</span>
        </div>
        <div class="card-body">
          <div class="config-field">
            <span class="config-label">Published Ports</span>
            <div v-if="service.ports?.length" style="display: flex; flex-direction: column; gap: 6px; margin-block-start: 6px;">
              <span v-for="p in service.ports" :key="p" class="status-badge mono" style="width: fit-content;">{{ p }}</span>
            </div>
            <span v-else class="text-3" style="font-size: 13px;">No ports published</span>
          </div>
        </div>
      </section>

      <!-- Environment (6 cols) -->
      <section class="surface-card col-6" aria-labelledby="svc-env-title">
        <div class="card-header">
          <span id="svc-env-title" class="card-title">Environment</span>
          <span class="count-pill">{{ service.env?.length || 0 }}</span>
        </div>
        <div class="card-body">
          <div v-if="!service.env?.length" class="text-center text-3">No environment variables defined</div>
          <div v-else class="env-list">
            <div v-for="(env, idx) in service.env" :key="idx" class="env-row">
              <span class="env-key" :title="splitEnv(env).key">{{ splitEnv(env).key }}</span>
              <span class="env-val mono">{{ splitEnv(env).val }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Labels (6 cols) -->
      <section class="surface-card col-6" aria-labelledby="svc-labels-title">
        <div class="card-header">
          <span id="svc-labels-title" class="card-title">Labels</span>
          <span class="count-pill">{{ Object.keys(service.labels || {}).length }}</span>
        </div>
        <div class="card-body">
          <div v-if="!Object.keys(service.labels || {}).length" class="text-center text-3">No labels found</div>
          <div v-else class="env-list">
            <div v-for="(val, key) in service.labels" :key="key" class="env-row">
              <span class="env-key" :title="String(key)">{{ key }}</span>
              <span class="env-val mono">{{ val }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Scale Modal -->
    <AppModal v-model="scaleDialog" title="Scale Service" max-width="400px" :persistent="actionLoading">
      <div class="form-field">
        <label class="form-label" for="scale-replicas">Desired Replicas</label>
        <input
          id="scale-replicas"
          v-model.number="scaleReplicas"
          type="number"
          min="0"
          class="form-input"
        />
      </div>
      <template #footer>
        <button type="button" class="btn" :disabled="actionLoading" @click="scaleDialog = false">Cancel</button>
        <button type="button" class="btn btn-primary" :disabled="actionLoading" @click="scaleService">
          {{ actionLoading ? 'Scaling...' : 'Scale' }}
        </button>
      </template>
    </AppModal>

    <!-- Delete Confirmation Modal -->
    <AppModal v-model="deleteDialog" title="Delete Service?" max-width="400px" :persistent="actionLoading">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to remove the service
        <strong class="text-1">{{ service?.name }}</strong>? This action cannot be undone.
      </p>
      <template #footer>
        <button type="button" class="btn" :disabled="actionLoading" @click="deleteDialog = false">Cancel</button>
        <button type="button" class="btn btn-danger" :disabled="actionLoading" @click="deleteService">
          {{ actionLoading ? 'Removing...' : 'Remove Service' }}
        </button>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import Loader from '../../components/Loader.vue';
import AppModal from '../../components/AppModal.vue';
import { splitImageReference } from '../../utils/image';

const route = useRoute();
const router = useRouter();
const service = ref<any>(null);
const parsedImage = computed(() => splitImageReference(service.value?.image));
const loading = ref(false);
const actionLoading = ref(false);
const deleteDialog = ref(false);
const scaleDialog = ref(false);
const scaleReplicas = ref(0);

const splitEnv = (entry: string) => {
  const idx = entry.indexOf('=');
  if (idx === -1) return { key: entry, val: '' };
  return { key: entry.slice(0, idx), val: entry.slice(idx + 1) };
};

const fetchDetails = async () => {
  const id = route.params.id as string;
  if (!id) return;

  loading.value = true;
  try {
    const response = await fetch(`/api/services/detail?id=${encodeURIComponent(id)}`);
    if (response.ok) {
      service.value = await response.json();
    }
  } catch (err) {
    console.error('Failed to fetch service details:', err);
  } finally {
    loading.value = false;
  }
};

const performAction = async (action: string) => {
  if (!service.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/services/${action}?id=${service.value.id}`, {
      method: 'POST',
    });
    if (!response.ok) {
      alert(`Failed to ${action} service: ` + (await response.text()));
    } else {
      await fetchDetails();
    }
  } catch (err) {
    console.error(`Failed to ${action} service:`, err);
  } finally {
    actionLoading.value = false;
  }
};

const openScaleDialog = () => {
  scaleReplicas.value = service.value.replicas;
  scaleDialog.value = true;
};

const scaleService = async () => {
  if (!service.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/services/scale?id=${service.value.id}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ replicas: scaleReplicas.value }),
    });
    if (!response.ok) {
      alert('Failed to scale service: ' + (await response.text()));
    } else {
      scaleDialog.value = false;
      await fetchDetails();
    }
  } catch (err) {
    console.error('Failed to scale service:', err);
  } finally {
    actionLoading.value = false;
  }
};

const confirmDelete = () => {
  deleteDialog.value = true;
};

const deleteService = async () => {
  if (!service.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/services?id=${service.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      deleteDialog.value = false;
      router.push('/swarm/services');
    } else {
      alert('Failed to remove service: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to delete service:', err);
  } finally {
    actionLoading.value = false;
  }
};

onMounted(fetchDetails);
</script>
