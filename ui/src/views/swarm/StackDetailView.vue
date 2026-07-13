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
      <div v-if="stack">
        <h1 class="text-h4 font-weight-bold d-flex align-center">
          Stack: {{ stack.name }}
          <v-chip
            :color="stack.status === 'Healthy' ? 'success' : 'warning'"
            size="small"
            label
            class="ms-4 text-uppercase font-weight-bold"
            :class="{ 'status-glow-success': stack.status === 'Healthy' }"
          >
            {{ stack.status }}
          </v-chip>
        </h1>
        <p class="text-subtitle-1 text-grey-lighten-1">
          {{ stack.services_list?.length || 0 }} Services / {{ stack.containers_list?.length || 0 }} Containers
        </p>
      </div>
      <div v-else>
        <h1 class="text-h4 font-weight-bold">Stack Detail</h1>
      </div>
      <v-spacer></v-spacer>
      <v-spacer></v-spacer>
      <v-menu v-if="stack && stack.name !== 'Orphaned/Manual'">
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
        <v-list class="menu-blur rounded-lg elevation-8" density="compact">
          <v-list-item @click="performAction('restart')" class="text-warning">
            <template v-slot:prepend><v-icon size="small">mdi-restart</v-icon></template>
            <v-list-item-title class="text-body-2">Force Restart</v-list-item-title>
          </v-list-item>
          <v-list-item @click="performAction('stop')" class="text-warning">
            <template v-slot:prepend><v-icon size="small">mdi-stop-circle-outline</v-icon></template>
            <v-list-item-title class="text-body-2">Stop (Scale to 0)</v-list-item-title>
          </v-list-item>
          <v-list-item @click="performAction('start')" class="text-success">
            <template v-slot:prepend><v-icon size="small">mdi-play-circle-outline</v-icon></template>
            <v-list-item-title class="text-body-2">Start Services</v-list-item-title>
          </v-list-item>
          <v-list-item @click="performAction('rollback')" class="text-info">
            <template v-slot:prepend><v-icon size="small">mdi-undo</v-icon></template>
            <v-list-item-title class="text-body-2">Rollback Spec</v-list-item-title>
          </v-list-item>
          <v-divider class="my-1"></v-divider>
          <v-list-item @click="confirmDelete" class="text-error">
            <template v-slot:prepend><v-icon size="small">mdi-delete-outline</v-icon></template>
            <v-list-item-title class="text-body-2">Delete Stack</v-list-item-title>
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

    <v-row v-if="stack">
      <!-- Services -->
      <v-col cols="12" md="6">
        <v-card class="glass-card rounded-xl h-100">
          <v-card-title class="pa-6 pb-2 d-flex align-center">
            <v-icon size="20" class="me-2">mdi-server</v-icon>
            <span class="font-weight-bold">Services</span>
          </v-card-title>
          <v-card-text class="pa-0">
            <v-list bg-color="transparent" class="pa-2">
              <v-list-item
                v-for="svc in stack.services_list"
                :key="svc.id"
                class="rounded-lg mb-1"
                @click="$router.push(`/swarm/services/${svc.id}`)"
              >
                <v-list-item-title class="text-body-2 font-weight-bold">{{ svc.name }}</v-list-item-title>
                <v-list-item-subtitle class="text-caption">{{ svc.image.split('@')[0] }}</v-list-item-subtitle>
                <template v-slot:append>
                  <v-chip size="x-small" label :color="svc.running === svc.replicas ? 'success' : 'warning'">
                    {{ svc.running }}/{{ svc.replicas }}
                  </v-chip>
                </template>
              </v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Containers -->
      <v-col cols="12" md="6">
        <v-card class="glass-card rounded-xl h-100">
          <v-card-title class="pa-6 pb-2 d-flex align-center">
            <v-icon size="20" class="me-2">mdi-cube-outline</v-icon>
            <span class="font-weight-bold">Containers</span>
          </v-card-title>
          <v-card-text class="pa-0">
            <v-list bg-color="transparent" class="pa-2">
              <v-list-item
                v-for="cont in stack.containers_list"
                :key="cont.id"
                class="rounded-lg mb-1"
                @click="$router.push(`/swarm/containers/${cont.id}?node=${cont.node}`)"
              >
                <v-list-item-title class="text-body-2 font-weight-bold">{{ cont.names[0].replace('/', '') }}</v-list-item-title>
                <v-list-item-subtitle class="text-caption">
                  {{ cont.node }} // {{ cont.status }}
                </v-list-item-subtitle>
                <template v-slot:append>
                  <v-chip size="x-small" label :color="cont.state === 'running' ? 'success' : 'grey'">
                    {{ cont.state }}
                  </v-chip>
                </template>
              </v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Volumes -->
      <v-col cols="12" md="6">
        <v-card class="glass-card rounded-xl h-100">
          <v-card-title class="pa-6 pb-2 d-flex align-center">
            <v-icon size="20" class="me-2">mdi-database</v-icon>
            <span class="font-weight-bold">Volumes</span>
          </v-card-title>
          <v-card-text class="pa-6">
            <v-row>
              <v-col v-for="vol in stack.volumes_list" :key="vol.name" cols="12" sm="6">
                <div class="pa-4 bg-black bg-opacity-20 rounded-lg d-flex align-center border border-white border-opacity-5">
                  <v-icon :color="vol.external ? 'warning' : 'primary'" class="me-3">{{ vol.external ? 'mdi-link-variant' : 'mdi-plus-circle-outline' }}</v-icon>
                  <div class="flex-grow-1 overflow-hidden">
                    <div class="text-body-2 font-weight-bold text-truncate" :title="vol.name">{{ vol.name }}</div>
                    <div class="text-caption text-grey">{{ vol.external ? 'External' : 'Managed' }}</div>
                  </div>
                </div>
              </v-col>
              <div v-if="!stack.volumes_list?.length" class="pa-6 text-center text-grey text-caption w-100">
                No volumes referenced by this stack.
              </div>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Tasks -->
      <v-col cols="12">
        <v-card class="glass-card rounded-xl">
          <v-card-title class="pa-6 pb-2 d-flex align-center flex-wrap justify-space-between">
            <div class="d-flex align-center">
              <v-icon size="20" class="me-2">mdi-list-status</v-icon>
              <span class="font-weight-bold">Cluster Tasks</span>
            </div>
            <v-text-field
              v-model="searchQuery"
              prepend-inner-icon="mdi-magnify"
              placeholder="Search tasks..."
              variant="solo-filled"
              density="compact"
              flat
              hide-details
              rounded="lg"
              class="search-input glass-input"
              style="width: 240px"
            ></v-text-field>
          </v-card-title>
          <v-card-text class="pa-0">
            <v-data-table
              :headers="taskHeaders"
              :items="stack.tasks_list || []"
              :search="searchQuery"
              class="bg-transparent"
              density="comfortable"
              hover
              items-per-page="25"
              :sort-by="[{ key: 'updated_at', order: 'desc' }]"
            >
              <template v-slot:item.state="{ item }">
                <v-chip
                  size="x-small"
                  label
                  :color="getTaskStateColor(item.state)"
                  class="text-uppercase font-weight-bold"
                >
                  {{ item.state }}
                </v-chip>
              </template>
              <template v-slot:item.node_name="{ value }">
                <code class="text-caption">{{ value || 'Unknown' }}</code>
              </template>
              <template v-slot:item.error="{ value }">
                <span class="text-caption text-error font-mono" v-if="value">{{ value }}</span>
                <span class="text-caption text-grey" v-else>-</span>
              </template>
              <template v-slot:item.updated_at="{ value }">
                <RelativeTime :value="value" />
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="400px">
      <v-card border flat class="bg-surface">
        <v-card-title class="pa-6 pb-2">Delete Stack?</v-card-title>
        <v-card-text class="pa-6 pt-0">
          Are you sure you want to remove the stack
          <strong>{{ stack?.name }}</strong
          >? This will stop and remove all associated services.
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteStack"
            :loading="actionLoading"
            >Remove Stack</v-btn
          >
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import RelativeTime from "../../components/RelativeTime.vue";
import Loader from "../../components/Loader.vue";

const searchQuery = ref("");

const route = useRoute();
const router = useRouter();
const stack = ref<any>(null);
const loading = ref(false);
const actionLoading = ref(false);
const deleteDialog = ref(false);

const taskHeaders = [
  { title: "Service", key: "service_name", align: "start" as const },
  { title: "Node", key: "node_name", align: "start" as const },
  { title: "State", key: "state", align: "center" as const },
  { title: "Desired", key: "desired_state", align: "center" as const },
  { title: "Error / Status", key: "error", align: "start" as const },
  { title: "Updated", key: "updated_at", align: "end" as const },
];

const getTaskStateColor = (state: string) => {
  switch (state.toLowerCase()) {
    case "running": return "success";
    case "failed": return "error";
    case "shutdown": return "grey";
    case "rejected": return "error";
    case "orphaned": return "warning";
    case "preparing":
    case "starting":
      return "info";
    default: return "warning";
  }
};

const fetchDetails = async () => {
  const name = route.params.name as string;
  if (!name) return;

  loading.value = true;
  try {
    const response = await fetch(`/api/stacks/detail?name=${name}`);
    if (response.ok) {
      const data = await response.json();
      
      // Sort containers by state: running first
      if (data.containers_list) {
        data.containers_list.sort((a: any, b: any) => {
          if (a.state === "running" && b.state !== "running") return -1;
          if (a.state !== "running" && b.state === "running") return 1;
          return 0;
        });
      }
      
      stack.value = data;
    }
  } catch (err) {
    console.error("Failed to fetch stack details:", err);
  } finally {
    loading.value = false;
  }
};

const performAction = async (action: string) => {
  if (!stack.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/stacks/${action}?name=${stack.value.name}`, {
      method: "POST",
    });
    if (!response.ok) {
      const text = await response.text();
      alert(`Failed to ${action} stack: ` + text);
    } else {
      await fetchDetails();
    }
  } catch (err) {
    console.error(`Failed to ${action} stack:`, err);
  } finally {
    actionLoading.value = false;
  }
};

const confirmDelete = () => {
  deleteDialog.value = true;
};

const deleteStack = async () => {
  if (!stack.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/stacks/${stack.value.name}`, {
      method: "DELETE",
    });
    if (response.ok) {
      deleteDialog.value = false;
      router.push("/swarm/stacks");
    } else {
      const text = await response.text();
      alert("Failed to remove stack: " + text);
    }
  } catch (err) {
    console.error("Failed to delete stack:", err);
  } finally {
    actionLoading.value = false;
  }
};

onMounted(fetchDetails);
</script>

