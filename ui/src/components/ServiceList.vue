<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Swarm Services</h1>
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-refresh"
        @click="fetchServices"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>
    
    <v-row v-if="loading" justify="center" class="mt-8">
      <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
    </v-row>

    <v-row v-else-if="services.length === 0" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="64" color="grey-lighten-1" class="mb-4">mdi-server-off</v-icon>
        <h3 class="text-h5 text-grey-darken-1">No services found</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          Deploy services to your cluster to manage and monitor them here.
        </p>
      </v-col>
    </v-row>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="services"
        :loading="loading"
        :row-props="getRowProps"
        class="bg-transparent clickable-rows"
        hover
        density="comfortable"
        @click:row="goToDetail"
        items-per-page="25"
      >
        <template v-slot:item.name="{ item }">
          <div class="d-flex flex-column">
            <span class="text-body-2 font-weight-bold">{{ item.name }}</span>
            <span class="text-caption text-grey">{{ item.stack }}</span>
          </div>
        </template>

        <template v-slot:item.replicas="{ item }">
          <div class="d-flex align-center" style="min-width: 120px">
            <v-chip
              :color="item.running < item.replicas ? 'warning' : 'success'"
              size="x-small"
              label
              class="font-weight-bold"
            >
              {{ item.running }} / {{ item.replicas }}
            </v-chip>
            <span class="text-caption text-grey ms-2">{{ item.mode }}</span>
          </div>
        </template>

        <template v-slot:item.image="{ value }">
          <div class="d-flex align-center">
            <v-icon
              size="18"
              color="primary"
              class="me-2"
              :icon="getRegistryIcon(value)"
            ></v-icon>
            <span
              class="text-caption font-mono text-primary truncate-image"
              :title="value"
            >
              {{ formatImage(value) }}
            </span>
          </div>
        </template>

        <template v-slot:item.ports="{ value }">
          <div class="d-flex flex-wrap gap-1">
            <v-chip
              v-for="port in value"
              :key="port"
              size="x-small"
              variant="tonal"
              color="info"
            >
              {{ port }}
            </v-chip>
            <span
              v-if="!value || value.length === 0"
              class="text-caption text-grey-lighten-1"
              >-</span
            >
          </div>
        </template>

        <template v-slot:item.updated_at="{ value }">
          <RelativeTime :value="value" />
        </template>

        <template v-slot:item.actions="{ item }">
          <div class="d-flex justify-center align-center">
            <v-btn
              icon="mdi-delete-outline"
              size="x-small"
              variant="text"
              color="error"
              @click.stop="confirmDelete(item)"
            ></v-btn>
          </div>
        </template>

      </v-data-table>
    </div>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="450px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2 text-h5 font-weight-bold d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-decagram</v-icon>
          Delete Service?
        </v-card-title>
        <v-card-text class="pa-6 pt-2">
          Are you sure you want to remove the service <strong class="font-mono text-error">{{ serviceToDelete?.name }}</strong>? This will permanently stop and remove all tasks for this service.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteService"
            :loading="deleting"
          >Remove Service</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Error Dialog -->
    <v-dialog v-model="errorDialog" max-width="500px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2 text-error d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-circle</v-icon>
          Service Deletion Failed
        </v-card-title>
        <v-card-text class="pa-6 pt-0">
          <p class="mb-4">The service could not be deleted.</p>
          <div class="bg-black bg-opacity-20 pa-4 rounded-lg font-mono text-caption text-error border border-error border-opacity-20">
            {{ errorMessage }}
          </div>
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="flat" color="primary" @click="errorDialog = false">Dismiss</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import RelativeTime from "./RelativeTime.vue";

const router = useRouter();
const goToDetail = (_: any, { item }: any) => {
  router.push(`/swarm/services/${item.id}`);
};

interface Service {
  id: string;
  name: string;
  stack: string;
  image: string;
  mode: string;
  replicas: number;
  running: number;
  ports: string[];
  updated_at: string;
}

const services = ref<Service[]>([]);
const loading = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const serviceToDelete = ref<Service | null>(null);
const errorDialog = ref(false);
const errorMessage = ref("");

const getRowProps = ({ item }: any) => {
  let colorClass = "status-success";
  if (item.running === 0 && item.replicas > 0) colorClass = "status-error";
  else if (item.running < item.replicas) colorClass = "status-warning";

  return {
    class: `status-bar-row ${colorClass}`,
  };
};

const headers = [
  {
    title: "Service Name",
    key: "name",
    sortable: true,
    align: "start" as const,
  },
  {
    title: "Replicas",
    key: "replicas",
    width: "150px",
    align: "start" as const,
  },
  { title: "Image", key: "image", sortable: true, align: "start" as const },
  { title: "Ports", key: "ports", align: "start" as const },
  {
    title: "Updated",
    key: "updated_at",
    width: "200px",
    align: "start" as const,
  },
  {
    title: "Actions",
    key: "actions",
    width: "80px",
    align: "center" as const,
    sortable: false,
  },
];

const getRegistryIcon = (repo: string): string => {
  if (!repo) return 'mdi-docker'
  const lower = repo.toLowerCase()
  if (lower.includes('ghcr.io')) return 'mdi-github'
  if (lower.includes('gcr.io') || lower.includes('pkg.dev')) return 'mdi-google'
  if (lower.includes('quay.io')) return 'mdi-redhat'
  if (lower.includes('public.ecr.aws') || lower.includes('.dkr.ecr.')) return 'mdi-aws'
  if (lower.includes('azurecr.io')) return 'mdi-microsoft-azure'
  if (lower.includes('gitlab.com')) return 'mdi-gitlab'
  if (lower.includes('lscr.io')) return 'mdi-linux'
  return 'mdi-docker' // Default docker hub whale
}

const formatImage = (image: string) => {
  if (!image) return "-";
  // Remove digest if present
  let clean = image.split("@")[0];
  
  // Strip tag (everything after the last colon, excluding any registry port)
  const slashParts = clean.split("/");
  const lastPart = slashParts[slashParts.length - 1];
  
  if (lastPart.includes(":")) {
    const colonParts = lastPart.split(":");
    slashParts[slashParts.length - 1] = colonParts.slice(0, -1).join(":");
    clean = slashParts.join("/");
  }
  
  return clean;
};

const fetchServices = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/services");
    services.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch services:", error);
  } finally {
    loading.value = false;
  }
};

const confirmDelete = (service: Service) => {
  serviceToDelete.value = service;
  deleteDialog.value = true;
};

const deleteService = async () => {
  if (!serviceToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/services?id=${encodeURIComponent(serviceToDelete.value.id)}`, {
      method: "DELETE",
    });

    if (response.ok) {
      await fetchServices();
      deleteDialog.value = false;
    } else {
      const err = await response.text();
      errorMessage.value = err;
      errorDialog.value = true;
    }
  } catch (err) {
    console.error("Failed to delete service:", err);
    errorMessage.value = String(err);
    errorDialog.value = true;
  } finally {
    deleting.value = false;
    serviceToDelete.value = null;
  }
};

onMounted(fetchServices);
</script>

<style scoped>
.truncate-image {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gap-1 {
  gap: 4px;
}
</style>
