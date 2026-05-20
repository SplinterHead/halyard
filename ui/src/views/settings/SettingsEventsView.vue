<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center">
      <h1 class="text-h4 font-weight-bold">Deployment Events</h1>
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-refresh"
        @click="fetchEvents"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <v-divider class="my-4"></v-divider>

    <v-row v-if="events.length === 0 && !loading" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="80" color="grey-lighten-1" class="mb-4">mdi-history</v-icon>
        <h3 class="text-h5 text-grey-darken-1">No Events Found</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          Deployment history and system events will appear here once you start syncing stacks or managing your cluster.
        </p>
      </v-col>
    </v-row>

    <v-data-table
      v-else
      :headers="headers"
      :items="events"
      :loading="loading"
      :row-props="getRowProps"
      class="bg-transparent"
      hover
      items-per-page="25"
    >
      <template v-slot:item.stack_name="{ value }">
        <span class="text-body-2 font-weight-bold">{{ value || '-' }}</span>
      </template>

      <template v-slot:item.sha="{ value }">
        <div v-if="value" class="d-flex align-center">
          <v-icon size="14" class="me-1 text-grey">mdi-source-commit</v-icon>
          <code class="text-caption text-primary font-weight-bold">{{ value.substring(0, 7) }}</code>
        </div>
        <span v-else class="text-caption text-grey">-</span>
      </template>

      <template v-slot:item.timestamp="{ value }">
        <RelativeTime :value="value" />
      </template>

      <template v-slot:item.logs="{ value }">
        <v-tooltip location="start" width="400">
          <template v-slot:activator="{ props }">
            <v-icon
              v-bind="props"
              size="16"
              class="text-grey cursor-help"
            >
              mdi-information-outline
            </v-icon>
          </template>
          <div class="text-caption pa-2">
            <pre style="white-space: pre-wrap">{{ value || 'No logs available' }}</pre>
          </div>
        </v-tooltip>
      </template>
    </v-data-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import RelativeTime from "../../components/RelativeTime.vue";

interface DeploymentEvent {
  id: string;
  sync_id: string;
  stack_name: string;
  sha: string;
  status: string;
  logs: string;
  timestamp: string;
}

const events = ref<DeploymentEvent[]>([]);
const loading = ref(false);

const headers = [
  { title: "Stack", key: "stack_name", align: "start" as const },
  { title: "Commit", key: "sha", width: "150px", align: "start" as const },
  { title: "Time", key: "timestamp", width: "180px", align: "start" as const },
  { title: "Details", key: "logs", width: "80px", align: "center" as const, sortable: false },
];

const getRowProps = ({ item }: any) => {
  let statusClass = "status-grey";
  const status = item.status?.toLowerCase();
  if (status === "success") {
    statusClass = "status-success";
  } else if (status === "failed") {
    statusClass = "status-error";
  }
  return {
    class: `status-bar-row ${statusClass}`
  };
};

const fetchEvents = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/events");
    events.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch events:", error);
  } finally {
    loading.value = false;
  }
};

onMounted(fetchEvents);
</script>
