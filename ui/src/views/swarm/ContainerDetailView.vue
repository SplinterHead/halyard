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
      <div v-if="container">
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
      <div v-else>
        <h1 class="text-h4 font-weight-bold">Container Detail</h1>
      </div>
      <v-spacer></v-spacer>
      <div v-if="container" class="d-flex align-center gap-2 me-4">
        <v-btn
          v-if="container.state.toLowerCase() === 'running'"
          prepend-icon="mdi-stop"
          color="warning"
          variant="tonal"
          @click="confirmStop"
          :loading="stopping"
          class="rounded-lg font-weight-bold"
        >Stop</v-btn>
        <v-btn
          v-if="container.state.toLowerCase() !== 'running'"
          prepend-icon="mdi-play"
          color="success"
          variant="tonal"
          @click="startContainer"
          :loading="starting"
          class="rounded-lg font-weight-bold"
        >Start</v-btn>
        <v-btn
          v-if="container.state.toLowerCase() === 'running'"
          prepend-icon="mdi-restart"
          color="info"
          variant="tonal"
          @click="confirmRestart"
          :loading="restarting"
          class="rounded-lg font-weight-bold"
        >Restart</v-btn>
        <v-btn
          prepend-icon="mdi-delete"
          color="error"
          variant="tonal"
          @click="confirmDelete"
          :loading="deleting"
          class="rounded-lg font-weight-bold"
        >Delete</v-btn>
      </div>
      <v-btn
        icon="mdi-refresh"
        @click="fetchDetails"
        :loading="loading"
        size="x-small"
        class="refresh-btn"
        flat
      ></v-btn>
    </div>

    <Loader :loading="loading" />

    <v-tabs v-model="tab" color="primary" class="mb-6 border-b">
      <v-tab value="overview" prepend-icon="mdi-view-dashboard-outline"
        >Overview</v-tab
      >
      <v-tab value="logs" prepend-icon="mdi-text-box-outline">Logs</v-tab>
      <v-tab value="terminal" prepend-icon="mdi-console">Terminal</v-tab>
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
          class="glass-card no-hover-card rounded-xl fill-height d-flex flex-column"
          style="height: calc(100vh - 280px)"
        >
          <v-card-title class="pa-4 pb-2 d-flex align-center">
            <v-icon size="20" class="me-2">mdi-text-box-outline</v-icon>
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

      <!-- Terminal Tab -->
      <v-window-item value="terminal">
        <v-card
          class="glass-card no-hover-card rounded-xl fill-height d-flex flex-column"
          style="height: calc(100vh - 280px)"
        >
          <v-card-title class="pa-4 pb-2 d-flex align-center flex-wrap gap-4">
            <v-icon size="20" class="me-2">mdi-console</v-icon>
            <span class="text-subtitle-2 font-weight-bold me-4">Interactive Console</span>
            
            <div class="d-flex align-center gap-2">
              <v-select
                v-model="selectedShell"
                :items="['/bin/bash', '/bin/sh', '/bin/zsh', '/bin/ash']"
                density="compact"
                hide-details
                variant="solo-filled"
                class="glass-input me-2"
                style="width: 140px;"
                :disabled="isTerminalConnected"
              ></v-select>
              <v-btn
                :color="isTerminalConnected ? 'error' : 'primary'"
                variant="flat"
                @click="toggleTerminalConnection"
                class="rounded-lg font-weight-bold"
              >
                {{ isTerminalConnected ? 'Disconnect' : 'Connect' }}
              </v-btn>
            </div>
          </v-card-title>
          <v-card-text class="pa-0 flex-grow-1 overflow-hidden">
            <div
              ref="terminalContainer"
              class="terminal-container bg-black pa-2 fill-height"
            >
              <div
                v-if="!isTerminalConnected"
                class="d-flex align-center justify-center fill-height text-grey"
              >
                Select a shell and click Connect to start an interactive session.
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-window-item>
    </v-window>

    <!-- Stop Confirmation Dialog -->
    <v-dialog v-model="stopDialog" max-width="450px">
      <v-card border flat class="bg-surface rounded-xl">
        <v-card-title class="pa-6 pb-2 text-h5 font-weight-bold d-flex align-center">
          <v-icon color="warning" class="me-2">mdi-alert-circle-outline</v-icon>
          Stop Container?
        </v-card-title>
        <v-card-text class="pa-6 pt-2">
          Are you sure you want to stop container <strong class="font-mono text-warning">{{ container ? formatName(container.names) : '' }}</strong>?
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="stopDialog = false">Cancel</v-btn>
          <v-btn
            color="warning"
            variant="flat"
            @click="stopContainer"
            :loading="stopping"
            class="rounded-lg"
          >Stop</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Restart Confirmation Dialog -->
    <v-dialog v-model="restartDialog" max-width="450px">
      <v-card border flat class="bg-surface rounded-xl">
        <v-card-title class="pa-6 pb-2 text-h5 font-weight-bold d-flex align-center">
          <v-icon color="info" class="me-2">mdi-restart</v-icon>
          Restart Container?
        </v-card-title>
        <v-card-text class="pa-6 pt-2">
          Are you sure you want to restart container <strong class="font-mono text-info">{{ container ? formatName(container.names) : '' }}</strong>?
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="restartDialog = false">Cancel</v-btn>
          <v-btn
            color="info"
            variant="flat"
            @click="restartContainer"
            :loading="restarting"
            class="rounded-lg"
          >Restart</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="450px">
      <v-card border flat class="bg-surface rounded-xl">
        <v-card-title class="pa-6 pb-2 text-h5 font-weight-bold d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-decagram</v-icon>
          Delete Container?
        </v-card-title>
        <v-card-text class="pa-6 pt-2">
          <p class="mb-4">Are you sure you want to delete container <strong class="font-mono text-error">{{ container ? formatName(container.names) : '' }}</strong>? This action cannot be undone.</p>
          <v-checkbox
            v-model="forceDelete"
            label="Force delete (forcefully kill running container)"
            color="error"
            density="compact"
            hide-details
            class="mt-2"
          ></v-checkbox>
        </v-card-text>
        <v-card-actions class="pa-6 pt-0">
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn
            color="error"
            variant="flat"
            @click="deleteContainer"
            :loading="deleting"
            class="rounded-lg"
          >Remove</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Error/Failure Dialog -->
    <v-dialog v-model="errorDialog" max-width="500px">
      <v-card border flat class="bg-surface rounded-xl">
        <v-card-title class="pa-6 pb-2 text-error d-flex align-center">
          <v-icon color="error" class="me-2">mdi-alert-circle</v-icon>
          Operation Failed
        </v-card-title>
        <v-card-text class="pa-6 pt-2">
          <p class="mb-4">The container lifecycle operation could not be completed successfully.</p>
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
import { ref, onMounted, onUnmounted, watch, nextTick } from "vue";
import { useRoute, useRouter } from "vue-router";
import Loader from "../../components/Loader.vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

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
const router = useRouter();
const container = ref<ContainerDetail | null>(null);
const loading = ref(false);
const tab = ref("overview");

const logs = ref<{ timestamp: string; content: string }[]>([]);
const logContainer = ref<HTMLElement | null>(null);
const autoScroll = ref(true);
const showIndex = ref(true);
const showTimestamp = ref(true);
const loadingLogs = ref(false);

const starting = ref(false);
const stopping = ref(false);
const restarting = ref(false);
const deleting = ref(false);

const stopDialog = ref(false);
const restartDialog = ref(false);
const deleteDialog = ref(false);
const forceDelete = ref(false);

const errorDialog = ref(false);
const errorMessage = ref("");

const selectedShell = ref("/bin/sh");
const isTerminalConnected = ref(false);
const terminalContainer = ref<HTMLElement | null>(null);
let term: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let termSocket: WebSocket | null = null;

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
  const token = localStorage.getItem("halyard_token") || "";
  const wsUrl = `${protocol}//${window.location.host}/api/containers/logs?id=${id}&node=${node}&token=${encodeURIComponent(token)}`;

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

const startContainer = async () => {
  if (!container.value) return;
  starting.value = true;
  try {
    const response = await fetch(
      `/api/containers/start?id=${container.value.id}&node=${container.value.node}`,
      { method: "POST" }
    );
    if (response.ok) {
      await fetchDetails();
    } else {
      const err = await response.text();
      errorMessage.value = err;
      errorDialog.value = true;
    }
  } catch (error) {
    console.error("Failed to start container:", error);
    errorMessage.value = String(error);
    errorDialog.value = true;
  } finally {
    starting.value = false;
  }
};

const confirmStop = () => {
  stopDialog.value = true;
};

const stopContainer = async () => {
  if (!container.value) return;
  stopDialog.value = false;
  stopping.value = true;
  try {
    const response = await fetch(
      `/api/containers/stop?id=${container.value.id}&node=${container.value.node}`,
      { method: "POST" }
    );
    if (response.ok) {
      await fetchDetails();
    } else {
      const err = await response.text();
      errorMessage.value = err;
      errorDialog.value = true;
    }
  } catch (error) {
    console.error("Failed to stop container:", error);
    errorMessage.value = String(error);
    errorDialog.value = true;
  } finally {
    stopping.value = false;
  }
};

const confirmRestart = () => {
  restartDialog.value = true;
};

const restartContainer = async () => {
  if (!container.value) return;
  restartDialog.value = false;
  restarting.value = true;
  try {
    const response = await fetch(
      `/api/containers/restart?id=${container.value.id}&node=${container.value.node}`,
      { method: "POST" }
    );
    if (response.ok) {
      await fetchDetails();
    } else {
      const err = await response.text();
      errorMessage.value = err;
      errorDialog.value = true;
    }
  } catch (error) {
    console.error("Failed to restart container:", error);
    errorMessage.value = String(error);
    errorDialog.value = true;
  } finally {
    restarting.value = false;
  }
};

const confirmDelete = () => {
  deleteDialog.value = true;
};

const deleteContainer = async () => {
  if (!container.value) return;
  deleteDialog.value = false;
  deleting.value = true;
  try {
    const response = await fetch(
      `/api/containers?id=${container.value.id}&node=${container.value.node}&force=${forceDelete.value}`,
      { method: "DELETE" }
    );
    if (response.ok) {
      router.back();
    } else {
      const err = await response.text();
      errorMessage.value = err;
      errorDialog.value = true;
    }
  } catch (error) {
    console.error("Failed to delete container:", error);
    errorMessage.value = String(error);
    errorDialog.value = true;
  } finally {
    deleting.value = false;
  }
};

const toggleTerminalConnection = () => {
  if (isTerminalConnected.value) {
    disconnectTerminal();
  } else {
    connectTerminal();
  }
};

const connectTerminal = () => {
  const id = route.params.id as string;
  const node = route.query.node as string;

  if (!id || !node) return;

  disconnectTerminal();

  isTerminalConnected.value = true;

  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: "Roboto Mono, Courier New, monospace",
    theme: {
      background: "#000000",
      foreground: "#ffffff",
      cursor: "#ffffff",
    },
  });

  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);

  if (terminalContainer.value) {
    terminalContainer.value.innerHTML = "";
    term.open(terminalContainer.value);
    fitAddon.fit();
  }

  term.write("Connecting to shell " + selectedShell.value + "...\r\n");

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const token = localStorage.getItem("halyard_token") || "";
  const wsUrl = `${protocol}//${window.location.host}/api/containers/exec?id=${id}&node=${node}&shell=${encodeURIComponent(selectedShell.value)}&token=${encodeURIComponent(token)}`;

  termSocket = new WebSocket(wsUrl);
  termSocket.binaryType = "arraybuffer";

  termSocket.onopen = () => {
    term?.write("\r\n--- Shell Connected ---\r\n\r\n");
    sendTerminalSize();
  };

  termSocket.onmessage = (event) => {
    if (term) {
      term.write(new Uint8Array(event.data));
    }
  };

  termSocket.onclose = () => {
    term?.write("\r\n--- Shell Disconnected ---\r\n");
    isTerminalConnected.value = false;
  };

  termSocket.onerror = (err) => {
    console.error("Terminal WebSocket error:", err);
    term?.write("\r\nError: Connection failed\r\n");
    isTerminalConnected.value = false;
  };

  term.onData((data) => {
    if (termSocket && termSocket.readyState === WebSocket.OPEN) {
      termSocket.send(data);
    }
  });

  window.addEventListener("resize", handleWindowResize);
};

const disconnectTerminal = () => {
  window.removeEventListener("resize", handleWindowResize);

  if (termSocket) {
    termSocket.close();
    termSocket = null;
  }

  if (term) {
    term.dispose();
    term = null;
  }

  fitAddon = null;
  isTerminalConnected.value = false;
};

const sendTerminalSize = () => {
  if (termSocket && termSocket.readyState === WebSocket.OPEN && term && fitAddon) {
    fitAddon.fit();
    termSocket.send(JSON.stringify({
      type: "resize",
      cols: term.cols,
      rows: term.rows
    }));
  }
};

const handleWindowResize = () => {
  sendTerminalSize();
};

watch(tab, (newTab) => {
  if (newTab === "logs") {
    startLogStream();
    disconnectTerminal();
  } else if (newTab === "terminal") {
    stopLogStream();
    // Don't auto-connect, let the user choose
  } else {
    stopLogStream();
    disconnectTerminal();
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

onUnmounted(() => {
  stopLogStream();
  disconnectTerminal();
});
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

.terminal-container {
  background-color: #000000 !important;
  height: 100%;
}

.terminal-container :deep(.xterm) {
  padding: 8px;
  height: 100%;
}

.terminal-container :deep(.xterm-viewport) {
  background-color: #000000 !important;
}

.glass-card.no-hover-card:hover {
  transform: none !important;
  background: rgba(30, 41, 59, 0.4) !important;
  border: 1px solid rgba(255, 255, 255, 0.05) !important;
  box-shadow: none !important;
}
</style>

