<template>
  <div class="fill-height d-flex flex-column pa-4">
    <!-- Header -->
    <div class="d-flex align-center mb-6" v-if="service">
      <v-btn
        icon="mdi-arrow-left"
        variant="text"
        @click="$router.back()"
        class="me-2"
      ></v-btn>
      <div>
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
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-refresh"
        @click="fetchDetails"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

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

    <div v-if="loading && !service" class="fill-height d-flex align-center justify-center">
      <v-progress-circular indeterminate color="primary"></v-progress-circular>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
const service = ref<any>(null);
const loading = ref(false);

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

onMounted(fetchDetails);
</script>

