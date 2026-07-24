<template>
  <DataTablePage
    title="Cluster Containers"
    :items="containers"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-package-variant-closed-remove"
    empty-title="No containers found"
    empty-description="Your cluster is currently empty. Deploy services or stacks to see containers here."
    search-placeholder="Search containers..."
    :sort-by="[{ key: 'names', order: 'asc' }]"
    :row-props="getRowProps"
    @refresh="fetchContainers"
    @click:row="goToDetail"
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
              <code class="font-mono text-caption">{{ value }}</code>
            </template>

            <template v-slot:item.status="{ value }">
              <span class="text-caption text-grey">{{ value }}</span>
            </template>

            <template v-slot:item.created_at="{ value }">
              <RelativeTime :value="value" />
            </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import DataTablePage from "../../components/DataTablePage.vue";
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
  { title: "Node", key: "node", width: "150px", align: "start" as const },
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
