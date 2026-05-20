<template>
  <div class="home-dashboard">
    <!-- Header Section -->
    <div class="d-flex align-center mb-8">
      <div>
        <h1 class="text-h3 font-weight-bold text-gradient mb-1">
          Cluster Overview
        </h1>
        <p class="text-subtitle-1 text-grey-lighten-1">
          Real-time status of your Docker Swarm cluster.
        </p>
      </div>
      <v-spacer></v-spacer>
      <div class="d-flex gap-2">
        <v-btn
          prepend-icon="mdi-broom"
          color="error"
          variant="tonal"
          @click="pruneSwarm"
          :loading="pruning"
          rounded="lg"
        >
          Prune Swarm
        </v-btn>
        <v-btn
          icon="mdi-refresh"
          @click="fetchAllStats"
          :loading="loading"
          size="x-small"
          class="refresh-btn"
          flat
        ></v-btn>
      </div>
    </div>

    <!-- Bento Grid -->
    <v-row class="bento-grid">
      <!-- Nodes Box -->
      <v-col cols="12" md="4">
        <v-card
          class="glass-card bento-item h-100 rounded-xl"
          @click="$router.push('/swarm/nodes')"
        >
          <v-card-text class="pa-6 h-100 d-flex flex-column">
            <div class="d-flex align-center mb-6">
              <v-avatar
                color="success"
                size="48"
                variant="tonal"
                class="rounded-lg"
              >
                <v-icon size="24">mdi-monitor-dashboard</v-icon>
              </v-avatar>
              <div class="ms-4">
                <div class="text-h6 font-weight-bold line-height-1">Nodes</div>
                <div class="text-caption font-weight-bold text-success">
                  {{ managersCount }} Managers / {{ workersCount }} Workers
                </div>
              </div>
            </div>

            <v-row no-gutters class="flex-grow-1 align-center">
              <v-col
                cols="6"
                class="border-e border-opacity-10 d-flex flex-column align-center justify-center"
              >
                <div class="text-h3 font-weight-bold text-success">
                  {{ onlineCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-green me-2"></div>
                  <span class="text-overline font-weight-bold text-grey"
                    >Online</span
                  >
                </div>
              </v-col>
              <v-col
                cols="6"
                class="d-flex flex-column align-center justify-center"
              >
                <div class="text-h3 font-weight-bold text-error">
                  {{ offlineCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-red me-2"></div>
                  <span class="text-overline font-weight-bold text-grey"
                    >Offline</span
                  >
                </div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Stacks Box -->
      <v-col cols="12" md="4">
        <v-card
          class="glass-card bento-item h-100 rounded-xl"
          @click="$router.push('/swarm/stacks')"
        >
          <v-card-text class="pa-6 h-100 d-flex flex-column">
            <div class="d-flex align-center mb-6">
              <v-avatar
                color="primary"
                size="48"
                variant="tonal"
                class="rounded-lg"
              >
                <v-icon size="24">mdi-layers-outline</v-icon>
              </v-avatar>
              <div class="ms-4">
                <div class="text-h6 font-weight-bold line-height-1">Stacks</div>
                <div class="text-caption font-weight-bold text-primary">
                  Orchestrated Deployments
                </div>
              </div>
            </div>

            <v-row no-gutters class="flex-grow-1 align-center">
              <v-col
                cols="6"
                class="border-e border-opacity-10 d-flex flex-column align-center justify-center"
              >
                <div class="text-h3 font-weight-bold text-success">
                  {{ healthyStacksCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-green me-2"></div>
                  <span class="text-overline font-weight-bold text-grey"
                    >Healthy</span
                  >
                </div>
              </v-col>
              <v-col
                cols="6"
                class="d-flex flex-column align-center justify-center"
              >
                <div class="text-h3 font-weight-bold text-warning">
                  {{ unhealthyStacksCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-orange me-2"></div>
                  <span class="text-overline font-weight-bold text-grey"
                    >Issue</span
                  >
                </div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Containers Box -->
      <v-col cols="12" md="4">
        <v-card
          class="glass-card bento-item h-100 rounded-xl"
          @click="$router.push('/swarm/containers')"
        >
          <v-card-text class="pa-6 h-100 d-flex flex-column">
            <div class="d-flex align-center mb-6">
              <v-avatar color="info" size="48" variant="tonal" class="rounded-lg">
                <v-icon size="24">mdi-docker</v-icon>
              </v-avatar>
              <div class="ms-4">
                <div class="text-h6 font-weight-bold line-height-1">
                  Containers
                </div>
                <div class="text-caption font-weight-bold text-info">
                  Active Runtime
                </div>
              </div>
            </div>

            <v-row no-gutters class="flex-grow-1 align-center">
              <v-col
                cols="6"
                class="border-e border-opacity-10 d-flex flex-column align-center justify-center"
              >
                <div class="text-h3 font-weight-bold text-info">
                  {{ runningContainersCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-blue me-2"></div>
                  <span class="text-overline font-weight-bold text-grey"
                    >Running</span
                  >
                </div>
              </v-col>
              <v-col
                cols="6"
                class="d-flex flex-column align-center justify-center"
              >
                <div class="text-h3 font-weight-bold text-grey">
                  {{ stoppedContainersCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-grey me-2"></div>
                  <span class="text-overline font-weight-bold text-grey"
                    >Stopped</span
                  >
                </div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Services List (Wider Box) -->
      <v-col cols="12" md="8">
        <v-card class="glass-card h-100 rounded-xl">
          <v-card-title class="pa-6 pb-2 d-flex align-center">
            <span class="font-weight-bold">Recent Services</span>
            <v-spacer></v-spacer>
            <v-btn
              to="/swarm/services"
              variant="text"
              size="small"
              color="primary"
              >View All</v-btn
            >
          </v-card-title>
          <v-card-text class="pa-0">
            <v-list bg-color="transparent" class="pa-2">
              <v-list-item
                v-for="service in services.slice(0, 5)"
                :key="service.id"
                class="rounded-lg mb-1"
              >
                <template v-slot:prepend>
                  <v-avatar
                    size="32"
                    color="primary"
                    variant="tonal"
                    class="me-3"
                  >
                    <v-icon size="16">mdi-server</v-icon>
                  </v-avatar>
                </template>
                <v-list-item-title class="text-body-2 font-weight-bold">{{
                  service.name
                }}</v-list-item-title>
                <v-list-item-subtitle class="text-caption">{{
                  service.stack || "No Stack"
                }}</v-list-item-subtitle>
                <template v-slot:append>
                  <v-chip
                    size="x-small"
                    :color="
                      service.running === service.replicas
                        ? 'success'
                        : 'warning'
                    "
                    label
                  >
                    {{ service.running }}/{{ service.replicas }}
                  </v-chip>
                </template>
              </v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Git Syncs Box -->
      <v-col cols="12" md="4">
        <v-card
          class="glass-card bento-item h-100 rounded-xl"
          :class="{ 'sync-pulsing': isSyncingAny }"
          @click="$router.push('/git/syncs')"
        >
          <v-card-text class="pa-6 h-100 d-flex flex-column">
            <div class="d-flex align-center mb-6">
              <v-avatar
                color="secondary"
                size="48"
                variant="tonal"
                class="rounded-lg"
              >
                <v-icon size="24">mdi-sync</v-icon>
              </v-avatar>
              <div class="ms-4">
                <div class="text-h6 font-weight-bold line-height-1">Git Syncs</div>
                <div class="text-caption font-weight-bold text-secondary">
                  GitOps Reconciler
                </div>
              </div>
            </div>

            <v-row no-gutters class="flex-grow-1 align-center">
              <v-col
                cols="4"
                class="border-e border-opacity-10 d-flex flex-column align-center justify-center"
              >
                <div class="text-h4 font-weight-bold text-success">
                  {{ syncs.length - outOfDateSyncsCount - failedSyncsCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-green me-2"></div>
                  <span class="text-overline font-weight-bold text-grey" style="font-size: 0.6rem !important"
                    >Synced</span
                  >
                </div>
              </v-col>
              <v-col
                cols="4"
                class="border-e border-opacity-10 d-flex flex-column align-center justify-center"
              >
                <div class="text-h4 font-weight-bold text-warning">
                  {{ outOfDateSyncsCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-orange me-2"></div>
                  <span class="text-overline font-weight-bold text-grey" style="font-size: 0.6rem !important"
                    >Pending</span
                  >
                </div>
              </v-col>
              <v-col
                cols="4"
                class="d-flex flex-column align-center justify-center"
              >
                <div class="text-h4 font-weight-bold text-error">
                  {{ failedSyncsCount }}
                </div>
                <div class="d-flex align-center mt-1">
                  <div class="led-dot led-red me-2"></div>
                  <span class="text-overline font-weight-bold text-grey" style="font-size: 0.6rem !important"
                    >Failed</span
                  >
                </div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Prune Confirmation Dialog -->
    <v-dialog v-model="pruneDialog" max-width="500" persistent>
      <v-card class="rounded-xl border-0" color="#121212">
        <v-card-title class="pa-6 pb-2 d-flex align-center">
          <v-avatar color="error" size="36" variant="tonal" class="me-3">
            <v-icon size="20">mdi-broom</v-icon>
          </v-avatar>
          <span class="font-weight-bold">Prune Swarm Cluster</span>
        </v-card-title>
        
        <v-card-text class="pa-6 pt-2 text-body-2 text-grey-lighten-1">
          <p class="mb-4 text-body-1">Select the resources to prune across all nodes in the cluster:</p>
          
          <v-switch
            v-model="pruneOptions.containers"
            color="error"
            label="Unused Containers"
            messages="Removes all stopped containers"
            density="comfortable"
            hide-details
            class="mb-3"
          ></v-switch>

          <v-switch
            v-model="pruneOptions.networks"
            color="error"
            label="Unused Networks"
            messages="Removes all networks not used by at least one container"
            density="comfortable"
            hide-details
            class="mb-3"
          ></v-switch>

          <v-switch
            v-model="pruneOptions.volumes"
            color="error"
            messages="Removes all local volumes not used by at least one container"
            density="comfortable"
            hide-details
            class="mb-3"
          >
            <template v-slot:label>
              <div class="d-flex align-center">
                <span>Unused Volumes</span>
                <v-chip size="x-small" color="warning" class="ms-2 font-weight-bold" variant="tonal">Caution</v-chip>
              </div>
            </template>
          </v-switch>

          <v-switch
            v-model="pruneOptions.images"
            color="error"
            label="Unused Images"
            messages="Removes unused images"
            density="comfortable"
            hide-details
            class="mb-1"
          ></v-switch>

          <v-expand-transition>
            <div v-if="pruneOptions.images" class="ms-8 ps-3 border-s-2 border-grey-darken-3 mb-3">
              <v-checkbox
                v-model="pruneOptions.imagesAll"
                color="error"
                label="Prune All Unused Images (-a)"
                messages="Removes all images without at least one container, not just dangling ones"
                density="compact"
                hide-details
              ></v-checkbox>
            </div>
          </v-expand-transition>
        </v-card-text>

        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" color="grey" @click="pruneDialog = false" :disabled="pruning">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="confirmPrune"
            :loading="pruning"
            class="rounded-lg px-6"
          >
            Prune Cluster
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue";

const nodes = ref([]);
const stacks = ref([]);
const containers = ref([]);
const services = ref([]);
const syncs = ref([]);
const loading = ref(false);
const pruning = ref(false);
const pruneDialog = ref(false);
const pruneOptions = ref({
  containers: true,
  networks: true,
  volumes: false,
  images: true,
  imagesAll: false,
});

const managersCount = computed(
  () => nodes.value.filter((n: any) => n.role === "manager").length,
);
const workersCount = computed(
  () => nodes.value.filter((n: any) => n.role === "worker").length,
);
const onlineCount = computed(
  () => nodes.value.filter((n: any) => n.status === "ready").length,
);
const offlineCount = computed(
  () => nodes.value.filter((n: any) => n.status !== "ready").length,
);
const healthyStacksCount = computed(
  () => stacks.value.filter((s: any) => s.status === "Healthy").length,
);
const unhealthyStacksCount = computed(
  () => stacks.value.filter((s: any) => s.status !== "Healthy").length,
);
const runningContainersCount = computed(
  () => containers.value.filter((c: any) => c.state === "running").length,
);
const stoppedContainersCount = computed(
  () => containers.value.filter((c: any) => c.state !== "running").length,
);
const runningContainers = computed(
  () => containers.value.filter((c: any) => c.state === "running").length,
);

const outOfDateSyncsCount = computed(
  () => syncs.value.filter((s: any) => s.last_status === "Out of Date").length,
);
const failedSyncsCount = computed(
  () => syncs.value.filter((s: any) => s.last_status === "Failed").length,
);
const isSyncingAny = computed(
  () => syncs.value.some((s: any) => s.last_status === "Syncing"),
);

const avgCpu = computed(() => {
  if (nodes.value.length === 0) return 0;
  const sum = nodes.value.reduce((acc: number, n: any) => acc + n.cpu_usage, 0);
  return Math.round(sum / nodes.value.length);
});

const avgMem = computed(() => {
  if (nodes.value.length === 0) return 0;
  const sum = nodes.value.reduce(
    (acc: number, n: any) => acc + (n.memory_usage / n.memory_total) * 100,
    0,
  );
  return Math.round(sum / nodes.value.length);
});

const fetchAllStats = async () => {
  loading.value = true;
  try {
    const [nRes, sRes, cRes, svcRes, syncRes] = await Promise.all([
      fetch("/api/nodes"),
      fetch("/api/stacks"),
      fetch("/api/containers"),
      fetch("/api/services"),
      fetch("/api/syncs"),
    ]);
    nodes.value = await nRes.json();
    stacks.value = await sRes.json();
    containers.value = await cRes.json();
    services.value = await svcRes.json();
    syncs.value = await syncRes.json();
  } catch (err) {
    console.error("Failed to fetch dashboard stats:", err);
  } finally {
    loading.value = false;
  }
};

const pruneSwarm = () => {
  pruneDialog.value = true;
};

const confirmPrune = async () => {
  pruning.value = true;
  try {
    const response = await fetch("/api/swarm/prune", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        containers: pruneOptions.value.containers,
        networks: pruneOptions.value.networks,
        volumes: pruneOptions.value.volumes,
        images: pruneOptions.value.images,
        images_all: pruneOptions.value.imagesAll,
      }),
    });
    if (response.ok) {
      pruneDialog.value = false;
      await fetchAllStats();
    } else {
      const err = await response.text();
      console.error("Failed to prune swarm:", err);
    }
  } catch (err) {
    console.error("Failed to prune swarm:", err);
  } finally {
    pruning.value = false;
  }
};

onMounted(fetchAllStats);
</script>

<style scoped>
.bento-item {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.bento-item:hover {
  transform: translateY(-4px);
  background: rgba(255, 255, 255, 0.08) !important;
  border-color: rgba(139, 92, 246, 0.3);
}

.bg-gradient-primary {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%) !important;
}

.bg-gradient-purple {
  background: linear-gradient(135deg, #8b5cf6 0%, #d946ef 100%) !important;
}

.line-height-1 {
  line-height: 1;
}

.animate-pulse-glow {
  animation: pulse-glow 4s infinite alternate;
}

@keyframes pulse-glow {
  0% {
    opacity: 0.3;
    transform: scale(1);
  }
  100% {
    opacity: 0.6;
    transform: scale(1.1);
  }
}
</style>
