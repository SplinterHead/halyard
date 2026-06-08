<template>
  <div class="fill-height d-flex flex-column pa-4">
    <!-- Header -->
    <div class="d-flex align-center mb-6">
      <v-btn
        icon="mdi-arrow-left"
        variant="text"
        @click="$router.back()"
        class="me-2"
      ></v-btn>
      <div v-if="service">
        <h1 class="text-h4 font-weight-bold d-flex align-center">
          Service: {{ service.name }}
          <v-chip
            :color="service.running === service.replicas ? 'success' : 'warning'"
            size="small"
            label
            class="ms-4 text-uppercase font-weight-bold"
          >
            {{ service.running }}/{{ service.replicas }}
          </v-chip>
        </h1>
        <p class="text-subtitle-1 text-grey-lighten-1">
          Stack: {{ service.stack }} • {{ service.mode }}
        </p>
      </div>
      <div v-else>
        <h1 class="text-h4 font-weight-bold">Service Detail</h1>
      </div>
      <v-spacer></v-spacer>
      <v-spacer></v-spacer>
      <v-menu v-if="service">
        <template v-slot:activator="{ props }">
          <v-btn
            v-bind="props"
            variant="tonal"
            color="primary"
            class="me-2"
            prepend-icon="mdi-dots-vertical"
            size="small"
            :loading="actionLoading"
          >
            Actions
          </v-btn>
        </template>
        <v-list class="bg-surface rounded-lg elevation-4" density="compact">
          <v-list-item @click="performAction('restart')" class="text-warning">
            <template v-slot:prepend><v-icon size="small">mdi-restart</v-icon></template>
            <v-list-item-title class="text-body-2">Force Restart</v-list-item-title>
          </v-list-item>
          <v-list-item @click="performAction('stop')" :disabled="service.mode === 'global'" class="text-warning">
            <template v-slot:prepend><v-icon size="small">mdi-stop-circle-outline</v-icon></template>
            <v-list-item-title class="text-body-2">Stop (Scale to 0)</v-list-item-title>
          </v-list-item>
          <v-list-item @click="performAction('start')" :disabled="service.mode === 'global'" class="text-success">
            <template v-slot:prepend><v-icon size="small">mdi-play-circle-outline</v-icon></template>
            <v-list-item-title class="text-body-2">Start Service</v-list-item-title>
          </v-list-item>
          <v-list-item @click="performAction('rollback')" class="text-info">
            <template v-slot:prepend><v-icon size="small">mdi-undo</v-icon></template>
            <v-list-item-title class="text-body-2">Rollback Spec</v-list-item-title>
          </v-list-item>
          <v-list-item @click="openScaleDialog" :disabled="service.mode === 'global'" class="text-primary">
            <template v-slot:prepend><v-icon size="small">mdi-arrow-up-down</v-icon></template>
            <v-list-item-title class="text-body-2">Scale Service</v-list-item-title>
          </v-list-item>
          <v-divider class="my-1"></v-divider>
          <v-list-item @click="confirmDelete" class="text-error">
            <template v-slot:prepend><v-icon size="small">mdi-delete-outline</v-icon></template>
            <v-list-item-title class="text-body-2">Delete Service</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
      <v-btn
        icon="mdi-refresh"
        @click="fetchDetails"
        :loading="loading"
        size="small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <Loader :loading="loading" />

    <v-row v-if="service">
      <!-- Configuration Overview -->
      <v-col cols="12" md="8">
        <v-card class="glass-card rounded-xl h-100">
          <v-card-title class="pa-6 pb-2 font-weight-bold">Runtime Configuration</v-card-title>
          <v-card-text class="pa-6">
            <v-row>
              <v-col cols="12" sm="6">
                <div class="text-caption text-grey mb-1">Image</div>
                <div class="text-body-2 font-weight-bold break-word">{{ service.image }}</div>
              </v-col>
              <v-col cols="12" sm="6">
                <div class="text-caption text-grey mb-1">Restart Policy</div>
                <div class="text-body-2 font-weight-bold text-uppercase">{{ service.restart_policy }}</div>
              </v-col>
              <v-col cols="12">
                <div class="text-caption text-grey mb-2">Placement Constraints</div>
                <div class="d-flex flex-wrap gap-2">
                  <v-chip v-for="c in service.constraints" :key="c" size="x-small" color="secondary" label variant="tonal">
                    {{ c }}
                  </v-chip>
                  <div v-if="!service.constraints?.length" class="text-caption text-grey">No placement constraints</div>
                </div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Ports -->
      <v-col cols="12" md="4">
        <v-card class="glass-card rounded-xl h-100">
          <v-card-title class="pa-6 pb-2 font-weight-bold">Networking</v-card-title>
          <v-card-text class="pa-6">
            <div class="text-caption text-grey mb-2">Published Ports</div>
            <v-list density="compact" class="bg-transparent pa-0">
              <v-list-item v-for="p in service.ports" :key="p" class="px-0">
                <template v-slot:prepend>
                  <v-icon size="16" color="primary" class="me-2">mdi-lan-connect</v-icon>
                </template>
                <v-list-item-title class="text-caption font-weight-bold">{{ p }}</v-list-item-title>
              </v-list-item>
              <div v-if="!service.ports?.length" class="text-caption text-grey">No ports published</div>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Environment -->
      <v-col cols="12" md="6">
        <v-card class="glass-card rounded-xl">
          <v-card-title class="pa-6 pb-2 font-weight-bold">Environment</v-card-title>
          <v-card-text class="pa-6 pt-2">
            <div class="scroll-y-400 bg-black bg-opacity-20 rounded-lg pa-4">
              <div v-for="(env, idx) in service.env" :key="idx" class="env-item mb-1 d-flex">
                <span class="text-caption font-weight-bold text-primary me-2">{{ env.split('=')[0] }}:</span>
                <span class="text-caption text-grey-lighten-1 break-word">{{ env.split('=')[1] }}</span>
              </div>
              <div v-if="!service.env?.length" class="text-caption text-grey">No environment variables defined</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Labels -->
      <v-col cols="12" md="6">
        <v-card class="glass-card rounded-xl">
          <v-card-title class="pa-6 pb-2 font-weight-bold">Labels</v-card-title>
          <v-card-text class="pa-6 pt-2">
            <div class="scroll-y-400 bg-black bg-opacity-20 rounded-lg pa-4">
              <div v-for="(val, key) in service.labels" :key="key" class="label-item mb-1 d-flex">
                <span class="text-caption font-weight-bold text-secondary me-2" style="min-width: 140px">{{ key }}:</span>
                <span class="text-caption text-grey-lighten-1 break-word">{{ val }}</span>
              </div>
              <div v-if="!Object.keys(service.labels || {}).length" class="text-caption text-grey">No labels found</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Scale Dialog -->
    <v-dialog v-model="scaleDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Scale Service</v-card-title>
        <v-card-text class="pa-6 pt-0">
          <v-text-field
            v-model.number="scaleReplicas"
            type="number"
            label="Desired Replicas"
            min="0"
            variant="outlined"
            density="comfortable"
          ></v-text-field>
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="scaleDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            @click="scaleService"
            :loading="actionLoading"
            >Scale</v-btn
          >
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Delete Service?</v-card-title>
        <v-card-text class="pa-6 pt-0">
          Are you sure you want to remove the service
          <strong>{{ service?.name }}</strong
          >? This action cannot be undone.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteService"
            :loading="actionLoading"
            >Remove Service</v-btn
          >
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import Loader from "../../components/Loader.vue";

const route = useRoute();
const router = useRouter();
const service = ref<any>(null);
const loading = ref(false);
const actionLoading = ref(false);
const deleteDialog = ref(false);
const scaleDialog = ref(false);
const scaleReplicas = ref(0);

const fetchDetails = async () => {
  const id = route.params.id as string;
  if (!id) return;

  loading.value = true;
  try {
    const response = await fetch(`/api/services/detail?id=${id}`);
    if (response.ok) {
      service.value = await response.json();
    }
  } catch (err) {
    console.error("Failed to fetch service details:", err);
  } finally {
    loading.value = false;
  }
};

const performAction = async (action: string) => {
  if (!service.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/services/${action}?id=${service.value.id}`, {
      method: "POST",
    });
    if (!response.ok) {
      const text = await response.text();
      alert(`Failed to ${action} service: ` + text);
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
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ replicas: scaleReplicas.value }),
    });
    if (!response.ok) {
      const text = await response.text();
      alert("Failed to scale service: " + text);
    } else {
      scaleDialog.value = false;
      await fetchDetails();
    }
  } catch (err) {
    console.error("Failed to scale service:", err);
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
      method: "DELETE",
    });
    if (response.ok) {
      deleteDialog.value = false;
      router.push("/swarm/stacks");
    } else {
      const text = await response.text();
      alert("Failed to remove service: " + text);
    }
  } catch (err) {
    console.error("Failed to delete service:", err);
  } finally {
    actionLoading.value = false;
  }
};

onMounted(fetchDetails);
</script>

