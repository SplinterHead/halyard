<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Swarm Stacks</h1>
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-refresh"
        @click="fetchStacks"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <v-row v-if="loading" justify="center" class="mt-8">
      <v-progress-circular
        indeterminate
        color="primary"
        size="64"
      ></v-progress-circular>
    </v-row>

    <v-row v-else-if="stacks.length === 0" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="64" color="grey-lighten-1" class="mb-4"
          >mdi-layers-off-outline</v-icon
        >
        <h3 class="text-h5 text-grey-darken-1">No stacks found</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          Deploy a stack using the GitOps workflow or the Docker CLI to see it
          here.
        </p>
      </v-col>
    </v-row>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="stacks"
        :loading="loading"
        :sort-by="[{ key: 'name', order: 'asc' }]"
        :row-props="getRowProps"
        class="bg-transparent"
        density="comfortable"
        @click:row="goToDetail"
        items-per-page="25"
      >
        <template v-slot:item.name="{ value }">
          <span class="text-body-1 font-weight-medium">{{ value }}</span>
        </template>

        <template v-slot:item.services="{ value }">
          <code class="text-caption">{{ value }}</code>
        </template>

        <template v-slot:item.updated_at="{ value }">
          <RelativeTime :value="value" />
        </template>

        <template v-slot:item.actions="{ item }">
          <div class="d-flex justify-center">
            <v-btn
              v-if="item.name !== 'Orphaned/Manual'"
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
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Delete Stack?</v-card-title>
        <v-card-text class="pa-6 pt-0">
          Are you sure you want to remove the stack
          <strong>{{ stackToDelete?.name }}</strong
          >? This will stop and remove all associated services.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteStack"
            :loading="deleting"
            >Remove Stack</v-btn
          >
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import RelativeTime from "../../components/RelativeTime.vue";

const router = useRouter();
const goToDetail = (_: any, { item }: any) => {
  router.push(`/swarm/stacks/${item.name}`);
};

interface Stack {
  name: string;
  services: number;
  total_replicas: number;
  running_replicas: number;
  status: string;
  updated_at: string;
}

const stacks = ref<Stack[]>([]);
const loading = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const stackToDelete = ref<Stack | null>(null);

const getRowProps = ({ item }: any) => {
  const status = item.status;
  let colorClass = "status-grey";
  if (status === "Healthy") colorClass = "status-success";
  else if (status === "Degraded") colorClass = "status-warning";
  else if (status === "Updating") colorClass = "status-info";

  return {
    class: `status-bar-row ${colorClass}`,
  };
};

const headers = [
  { title: "Stack Name", key: "name", sortable: true, align: "start" as const },
  {
    title: "Services",
    key: "services",
    width: "150px",
    align: "start" as const,
  },
  { title: "Updated", key: "updated_at", width: "150px", align: "start" as const },
  {
    title: "Actions",
    key: "actions",
    width: "100px",
    align: "center" as const,
    sortable: false,
  },
];

const getStatusColor = (status: string) => {
  switch (status) {
    case "Healthy":
      return "success";
    case "Degraded":
      return "warning";
    case "Updating":
      return "info";
    default:
      return "grey";
  }
};

const fetchStacks = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/stacks");
    stacks.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch stacks:", error);
  } finally {
    loading.value = false;
  }
};

const confirmDelete = (stack: Stack) => {
  stackToDelete.value = stack;
  deleteDialog.value = true;
};

const deleteStack = async () => {
  if (!stackToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/stacks/${stackToDelete.value.name}`, {
      method: "DELETE",
    });
    if (response.ok) {
      await fetchStacks();
      deleteDialog.value = false;
    } else {
      const text = await response.text();
      alert("Failed to remove stack: " + text);
    }
  } catch (err) {
    console.error("Failed to delete stack:", err);
  } finally {
    deleting.value = false;
  }
};

onMounted(fetchStacks);
</script>

<style scoped>
</style>
