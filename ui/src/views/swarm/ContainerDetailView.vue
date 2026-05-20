<template>
  <div class="fill-height d-flex flex-column pa-4">
    <!-- Header -->
    <div class="d-flex align-center mb-6" v-if="container">
      <v-btn
        icon="mdi-arrow-left"
        variant="text"
        @click="$router.back()"
        class="me-2"
      ></v-btn>
      <div>
        <h1 class="text-h4 font-weight-bold d-flex align-center">
          {{ formatName(container.names) }}
          <v-chip
            :color="getStateColor(container.state)"
            size="small"
            label
            class="ms-4 text-uppercase font-weight-bold"
          >
            {{ container.state }}
          </v-chip>
        </h1>
        <p class="text-subtitle-1 text-grey-lighten-1">
          {{ container.id.substring(0, 12) }}
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

    <v-tabs v-model="tab" color="primary" class="mb-6 border-b">
      <v-tab value="overview" prepend-icon="mdi-view-dashboard-outline"
        >Overview</v-tab
      >
      <v-tab value="logs" prepend-icon="mdi-console">Logs</v-tab>
    </v-tabs>

    <v-window v-model="tab" class="flex-grow-1">
      <!-- Overview Tab -->
      <v-window-item value="overview">
        <div v-if="container">
          <v-row>
            <!-- Basic Info -->
            <v-col cols="12" md="8">
              <v-card class="glass-card rounded-xl h-100">
                <v-card-title class="pa-6 pb-2 font-weight-bold"
                  >Configuration</v-card-title
                >
                <v-card-text class="pa-6">
                  <v-row>
                    <v-col cols="12" sm="6">
                      <div class="text-caption text-grey mb-1">Image</div>
                      <div class="text-body-2 font-weight-bold break-word">
                        {{ container.image }}
                      </div>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <div class="text-caption text-grey mb-1">Image ID</div>
                      <div class="text-body-2 font-weight-bold break-word">
                        {{ container.image_id }}
                      </div>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <div class="text-caption text-grey mb-1">Service</div>
                      <div class="text-body-2 font-weight-bold">
                        {{ container.service }}
                      </div>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <div class="text-caption text-grey mb-1">Stack</div>
                      <div class="text-body-2 font-weight-bold">
                        {{ container.stack }}
                      </div>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <div class="text-caption text-grey mb-1">Node</div>
                      <div class="text-body-2 font-weight-bold">
                        {{ container.node }}
                      </div>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <div class="text-caption text-grey mb-1">Created At</div>
                      <div class="text-body-2 font-weight-bold">
                        {{ new Date(container.created_at).toLocaleString() }}
                      </div>
                    </v-col>
                  </v-row>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- Network Info -->
            <v-col cols="12" md="4">
              <v-card class="glass-card rounded-xl h-100">
                <v-card-title class="pa-6 pb-2 font-weight-bold"
                  >Networking</v-card-title
                >
                <v-card-text class="pa-6">
                  <div class="mb-4">
                    <div class="text-caption text-grey mb-2">Networks</div>
                    <div class="d-flex flex-wrap gap-2">
                      <v-chip
                        v-for="net in container.networks"
                        :key="net"
                        size="x-small"
                        color="primary"
                        label
                        variant="tonal"
                      >
                        {{ net }}
                      </v-chip>
                    </div>
                  </div>
                  <div>
                    <div class="text-caption text-grey mb-2">Ports</div>
                    <v-list density="compact" class="bg-transparent pa-0">
                      <v-list-item
                        v-for="(port, idx) in container.ports"
                        :key="idx"
                        class="px-0"
                      >
                        <template v-slot:prepend>
                          <v-icon size="16" class="me-2"
                            >mdi-lan-connect</v-icon
                          >
                        </template>
                        <v-list-item-title class="text-caption">
                          {{ port.public_port }}:{{ port.private_port }}/{{
                            port.type
                          }}
                        </v-list-item-title>
                      </v-list-item>
                      <div
                        v-if="!container.ports?.length"
                        class="text-caption text-grey-darken-1"
                      >
                        No ports exposed
                      </div>
                    </v-list>
                  </div>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- Environment Variables -->
            <v-col cols="12" md="6">
              <v-card class="glass-card rounded-xl">
                <v-card-title class="pa-6 pb-2 font-weight-bold"
                  >Environment</v-card-title
                >
                <v-card-text class="pa-6 pt-2">
                  <div class="scroll-y-400 font-mono bg-black bg-opacity-20 rounded-lg pa-4">
                    <div
                      v-for="(env, idx) in container.env"
                      :key="idx"
                      class="env-item mb-1 d-flex"
                    >
                      <span
                        class="text-caption font-weight-bold text-primary me-2"
                        >{{ env.split("=")[0] }}:</span
                      >
                      <span
                        class="text-caption text-grey-lighten-1 break-word"
                        >{{ env.split("=")[1] }}</span
                      >
                    </div>
                    <div
                      v-if="!container.env?.length"
                      class="text-caption text-grey"
                    >
                      No environment variables
                    </div>
                  </div>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- Mounts -->
            <v-col cols="12" md="6">
              <v-card class="glass-card rounded-xl">
                <v-card-title class="pa-6 pb-2 font-weight-bold"
                  >Mounts</v-card-title
                >
                <v-card-text class="pa-0">
                  <v-table density="compact" class="bg-transparent">
                    <thead>
                      <tr>
                        <th class="text-caption font-weight-bold">Source</th>
                        <th class="text-caption font-weight-bold">
                          Destination
                        </th>
                        <th class="text-caption font-weight-bold">Type</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(mount, idx) in container.mounts" :key="idx">
                        <td
                          class="text-caption text-truncate-200"
                          :title="mount.source"
                        >
                          {{ mount.source }}
                        </td>
                        <td
                          class="text-caption text-truncate-200"
                          :title="mount.destination"
                        >
                          {{ mount.destination }}
                        </td>
                        <td>
                          <v-chip size="x-small" label>{{ mount.type }}</v-chip>
                        </td>
                      </tr>
                    </tbody>
                  </v-table>
                  <div
                    v-if="!container.mounts?.length"
                    class="pa-6 text-center text-caption text-grey"
                  >
                    No mounts configured
                  </div>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- Labels -->
            <v-col cols="12">
              <v-card class="glass-card rounded-xl">
                <v-card-title class="pa-6 pb-2 font-weight-bold"
                  >Labels</v-card-title
                >
                <v-card-text class="pa-6 pt-2">
                  <div class="scroll-y-400 bg-black bg-opacity-20 rounded-lg pa-4">
                    <div
                      v-for="(value, key) in container.labels"
                      :key="key"
                      class="label-item mb-1 d-flex"
                    >
                      <span
                        class="text-caption font-weight-bold text-primary me-2"
                        style="min-width: 150px"
                        >{{ key }}:</span
                      >
                      <span
                        class="text-caption text-grey-lighten-1 break-word"
                        >{{ value }}</span
                      >
                    </div>
                    <div
                      v-if="!Object.keys(container.labels || {}).length"
                      class="text-caption text-grey"
                    >
                      No labels found
                    </div>
                  </div>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>
        </div>
      </v-window-item>

      <!-- Logs Tab -->
      <v-window-item value="logs">
        <v-card
          class="glass-card rounded-xl fill-height d-flex flex-column"
          style="height: calc(100vh - 280px)"
        >
          <v-card-title class="pa-4 pb-2 d-flex align-center">
            <v-icon size="20" class="me-2">mdi-console</v-icon>
            <span class="text-subtitle-2 font-weight-bold">Live Logs</span>
            <v-spacer></v-spacer>
            <div class="d-flex align-center gap-4 me-4">
              <v-switch
                v-model="showIndex"
                label="Line Numbers"
                density="compact"
                hide-details
                color="primary"
                class="mt-0 ml-4"
              ></v-switch>
              <v-switch
                v-model="showTimestamp"
                label="Timestamps"
                density="compact"
                hide-details
                color="primary"
                class="mt-0 ml-4"
              ></v-switch>
            </div>
            <v-btn
              icon="mdi-delete-sweep-outline"
              variant="text"
              size="small"
              @click="clearLogs"
              title="Clear View"
              class="me-1"
            ></v-btn>
            <v-btn
              :icon="
                autoScroll ? 'mdi-chevron-double-down' : 'mdi-chevron-down'
              "
              variant="text"
              size="small"
              @click="autoScroll = !autoScroll"
              :color="autoScroll ? 'primary' : 'grey'"
              title="Toggle Auto-scroll"
            ></v-btn>
          </v-card-title>
          <v-card-text class="pa-0 flex-grow-1 overflow-hidden">
            <div
              ref="logContainer"
              class="log-viewer bg-black pa-4 fill-height overflow-y-auto"
            >
              <div v-for="(log, idx) in logs" :key="idx" class="log-line">
                <span
                  v-if="showIndex"
                  class="text-grey-darken-1 me-2 text-caption opacity-50"
                  >[{{ idx + 1 }}]</span
                >
                <span
                  v-if="showTimestamp && log.timestamp"
                  class="log-timestamp me-2 text-caption opacity-80"
                  >{{ formatLogTime(log.timestamp) }}</span
                >
                <span class="log-content text-body-2 pre-wrap">{{
                  log.content
                }}</span>
              </div>
              <div
                v-if="logs.length === 0"
                class="d-flex align-center justify-center fill-height text-grey"
              >
                {{
                  loadingLogs
                    ? "Connecting to log stream..."
                    : "No logs available"
                }}
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-window-item>
    </v-window>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from "vue";
import { useRoute } from "vue-router";

interface Port {
  ip: string;
  private_port: number;
  public_port: number;
  type: string;
}

interface Mount {
  source: string;
  destination: string;
  type: string;
  rw: boolean;
}

interface ContainerDetail {
  id: string;
  names: string[];
  image: string;
  image_id: string;
  state: string;
  status: string;
  node: string;
  service: string;
  stack: string;
  up_to_date: boolean;
  created_at: string;
  env: string[];
  labels: Record<string, string>;
  mounts: Mount[];
  networks: string[];
  ports: Port[];
}

const route = useRoute();
const container = ref<ContainerDetail | null>(null);
const loading = ref(false);
const tab = ref("overview");

const logs = ref<{ timestamp: string; content: string }[]>([]);
const logContainer = ref<HTMLElement | null>(null);
const autoScroll = ref(true);
const showIndex = ref(true);
const showTimestamp = ref(true);
const loadingLogs = ref(false);
let abortController: AbortController | null = null;

const fetchDetails = async () => {
  const id = route.params.id as string;
  const node = route.query.node as string;

  if (!id || !node) return;

  loading.value = true;
  try {
    const response = await fetch(
      `/api/containers/detail?id=${id}&node=${node}`,
    );
    if (response.ok) {
      container.value = await response.json();
    }
  } catch (error) {
    console.error("Failed to fetch container details:", error);
  } finally {
    loading.value = false;
  }
};

const startLogStream = () => {
  const id = route.params.id as string;
  const node = route.query.node as string;

  if (!id || !node) return;

  stopLogStream();

  loadingLogs.value = true;
  logs.value = [];

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/api/containers/logs?id=${id}&node=${node}`;

  const socket = new WebSocket(wsUrl);

  socket.onopen = () => {
    loadingLogs.value = false;
  };

  socket.onmessage = (event) => {
    const lines = event.data.split("\n");
    lines.forEach((line: string) => {
      if (line.trim()) {
        const tsMatch = line.match(
          /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s(.*)$/,
        );
        if (tsMatch) {
          logs.value.push({
            timestamp: tsMatch[1],
            content: tsMatch[2],
          });
        } else {
          logs.value.push({
            timestamp: "",
            content: line,
          });
        }

        if (logs.value.length > 1000) {
          logs.value.shift();
        }
      }
    });

    if (autoScroll.value) {
      nextTick(() => {
        if (logContainer.value) {
          logContainer.value.scrollTop = logContainer.value.scrollHeight;
        }
      });
    }
  };

  socket.onerror = (error) => {
    console.error("WebSocket error:", error);
    loadingLogs.value = false;
  };

  socket.onclose = () => {
    console.log("WebSocket connection closed");
    loadingLogs.value = false;
  };

  (window as any)._logSocket = socket;
};

const stopLogStream = () => {
  const socket = (window as any)._logSocket;
  if (socket) {
    socket.close();
    (window as any)._logSocket = null;
  }
};

const clearLogs = () => {
  logs.value = [];
};

watch(tab, (newTab) => {
  if (newTab === "logs") {
    startLogStream();
  } else {
    stopLogStream();
  }
});

const formatLogTime = (ts: string) => {
  if (!ts) return "";
  try {
    const date = new Date(ts);
    return date.toLocaleTimeString([], {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return ts.substring(11, 19);
  }
};

const formatName = (names: string[]) => {
  if (!names || names.length === 0) return "Unknown";
  return names[0].replace(/^\//, "");
};

const getStateColor = (state: string) => {
  switch (state.toLowerCase()) {
    case "running":
      return "success";
    case "exited":
      return "error";
    case "paused":
      return "warning";
    default:
      return "grey";
  }
};

onMounted(() => {
  fetchDetails();
  if (tab.value === "logs") {
    startLogStream();
  }
});

onUnmounted(stopLogStream);
</script>

<style scoped>
.min-h-400 {
  min-height: 400px;
}

.log-viewer {
  background-color: #000 !important;
}

.log-line,
.log-line span {
  font-family: var(--font-mono) !important;
  font-size: 0.85rem !important;
  line-height: 1.4 !important;
  font-weight: 700 !important;
}

.log-timestamp {
  color: #22d3ee !important; /* Cyan-400 */
}

.log-content {
  color: #f8fafc !important; /* Slate-50 (off-white) */
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.gap-2 {
  gap: 8px;
}
</style>

