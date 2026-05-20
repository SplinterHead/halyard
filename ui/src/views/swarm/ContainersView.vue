<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Cluster Containers</h1>
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-refresh"
        @click="fetchContainers"
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

    <v-row v-else-if="containers.length === 0" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="64" color="grey-lighten-1" class="mb-4">mdi-package-variant-closed-remove</v-icon>
        <h3 class="text-h5 text-grey-darken-1">No containers found</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          Your cluster is currently empty. Deploy services or stacks to see containers here.
        </p>
      </v-col>
    </v-row>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="containers"
        :loading="loading"
        :sort-by="[{ key: 'names', order: 'asc' }]"
        :row-props="getRowProps"
        class="bg-transparent"
        density="comfortable"
        @click:row="goToDetail"
        items-per-page="25"
      >
        <template v-slot:item.names="{ item }">
          <div class="d-flex flex-column">
            <span
              class="text-body-2 font-weight-bold truncate-text"
              :title="formatNames(item.names)"
            >
              {{ formatNames(item.names) }}
            </span>
            <span
              class="text-caption text-primary truncate-text me-1"
              :title="item.image"
            >
              {{ formatImage(item.image) }}
            </span>
          </div>
        </template>

        <template v-slot:item.up_to_date="{ item }">
          <div class="text-center">
            <v-chip
              v-if="!item.up_to_date"
              color="warning"
              size="x-small"
              label
              class="text-uppercase font-weight-bold"
            >
              Out of Date
            </v-chip>
            <v-chip
              v-else
              color="success"
              variant="tonal"
              size="x-small"
              label
              class="text-uppercase font-weight-bold"
            >
              Current
            </v-chip>
          </div>
        </template>

        <template v-slot:item.service="{ item }">
          <div class="d-flex flex-column">
            <span class="text-caption font-weight-bold">{{
              item.service
            }}</span>
            <span class="text-caption text-grey">{{ item.stack }}</span>
          </div>
        </template>

        <template v-slot:item.node="{ value }">
          <code class="text-caption">{{ value }}</code>
        </template>

        <template v-slot:item.status="{ value }">
          <span class="text-caption text-grey">{{ value }}</span>
        </template>

        <template v-slot:item.created_at="{ value }">
          <RelativeTime :value="value" />
        </template>

      </v-data-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import RelativeTime from "../../components/RelativeTime.vue";

const router = useRouter();

interface Container {
  id: string;
  names: string[];
  image: string;
  state: string;
  status: string;
  node: string;
  service: string;
  stack: string;
  up_to_date: boolean;
  created_at: string;
}

const containers = ref<Container[]>([]);
const loading = ref(false);

const getRowProps = ({ item }: any) => {
  const state = item.state.toLowerCase();
  let colorClass = "status-grey";
  if (state === "running") colorClass = "status-success";
  else if (state === "exited") colorClass = "status-error";
  else if (state === "created") colorClass = "status-info";
  else if (state === "paused" || state === "restarting")
    colorClass = "status-warning";

  return {
    class: `status-bar-row ${colorClass}`,
  };
};

const headers = [
  {
    title: "Container / Image",
    key: "names",
    sortable: true,
    align: "start" as const,
  },
  {
    title: "Update",
    key: "up_to_date",
    width: "120px",
    align: "center" as const,
  },
  {
    title: "Service / Stack",
    key: "service",
    sortable: true,
    align: "start" as const,
  },
  { title: "Status", key: "status", align: "start" as const },
  { title: "Node", key: "node", width: "150px", align: "center" as const },
  { title: "Created", key: "created_at", align: "start" as const },
];

const getStateColor = (state: string) => {
  switch (state.toLowerCase()) {
    case "running":
      return "success";
    case "exited":
      return "error";
    case "created":
      return "info";
    case "paused":
      return "warning";
    case "restarting":
      return "warning";
    default:
      return "grey";
  }
};

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

const formatNames = (names: string[]) => {
  if (!names || names.length === 0) return "-";
  return names[0].replace(/^\//, "");
};

const fetchContainers = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/containers");
    containers.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch containers:", error);
  } finally {
    loading.value = false;
  }
};

const goToDetail = (_event: any, { item }: any) => {
  router.push({
    name: "container-detail",
    params: { id: item.id },
    query: { node: item.node },
  });
};

onMounted(fetchContainers);
</script>

<style scoped>
.truncate-text {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.clickable-rows :deep(tbody tr) {
  cursor: pointer;
}
</style>
