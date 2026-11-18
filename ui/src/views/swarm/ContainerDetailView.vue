<template>
  <div class="section-block" :class="{ 'viewport-fit-active': tab === 'logs' || tab === 'terminal' }" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header" style="border-block-end: 0; padding-block-end: 4px;">
      <div class="page-header-left">
        <div class="status-line">
          <button type="button" class="section-link" @click="$router.back()">
            ← {{ container?.stack ? `Stack: ${container.stack}` : 'Containers' }}
          </button>
        </div>
        <h1 class="page-title">
          <span>{{ container ? formatName(container.names) : 'Container Detail' }}</span>
          <span v-if="container" class="status-badge">
            <span class="status-dot" :class="getStateDotClass(container.state)"></span>
            {{ formatStateLabel(container.state) }}
          </span>
        </h1>
        <p v-if="container" class="page-subtitle mono">
          {{ container.id.substring(0, 12) }}
        </p>
      </div>

      <div class="header-actions">
        <template v-if="container">
          <button
            v-if="container.state.toLowerCase() === 'running'"
            type="button"
            class="btn"
            :disabled="stopping"
            @click="confirmStop"
          >
            {{ stopping ? 'Stopping...' : 'Stop' }}
          </button>
          <button
            v-else
            type="button"
            class="btn btn-primary"
            :disabled="starting"
            @click="startContainer"
          >
            {{ starting ? 'Starting...' : 'Start' }}
          </button>
          <button
            v-if="container.state.toLowerCase() === 'running'"
            type="button"
            class="btn"
            :disabled="restarting"
            @click="confirmRestart"
          >
            {{ restarting ? 'Restarting...' : 'Restart' }}
          </button>
          <button
            type="button"
            class="btn btn-danger-subtle"
            :disabled="deleting"
            @click="confirmDelete"
          >
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </template>
        <button
          type="button"
          class="btn btn-icon"
          :disabled="loading"
          aria-label="Refresh"
          title="Refresh"
          @click="fetchDetails"
        >
          <v-icon size="16">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <Loader :loading="loading" />

    <!-- Tab Bar -->
    <div class="tabs-bar" role="tablist" aria-label="Container views">
      <button
        type="button"
        role="tab"
        class="tab-btn"
        :aria-selected="tab === 'overview'"
        @click="tab = 'overview'"
      >
        Overview
      </button>
      <button
        type="button"
        role="tab"
        class="tab-btn"
        :aria-selected="tab === 'logs'"
        @click="tab = 'logs'"
      >
        Logs
      </button>
      <button
        type="button"
        role="tab"
        class="tab-btn"
        :aria-selected="tab === 'terminal'"
        @click="tab = 'terminal'"
      >
        Terminal
      </button>
    </div>

    <!-- TAB 1: OVERVIEW -->
    <div v-if="tab === 'overview' && container" role="tabpanel" class="detail-grid">
      <!-- Configuration (8 cols) -->
      <section class="surface-card col-8" aria-labelledby="cont-config-title">
        <div class="card-header">
          <span id="cont-config-title" class="card-title">Configuration</span>
        </div>
        <div class="card-body">
          <div class="config-grid">
            <div class="config-field">
              <span class="config-label">Service</span>
              <span class="config-val fw-500">{{ container.service || '-' }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">Stack</span>
              <span class="config-val fw-500">{{ container.stack || '-' }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">Node</span>
              <span class="config-val">{{ container.node || '-' }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">Created At</span>
              <span class="config-val">{{ new Date(container.created_at).toLocaleString() }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">Image</span>
              <span class="config-val mono text-1">{{ parsedImage.name }}</span>
            </div>
            <div class="config-field">
              <span class="config-label">SHA</span>
              <span class="config-val mono text-1">{{ parsedImage.sha }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Networking (4 cols) -->
      <section class="surface-card col-4" aria-labelledby="cont-net-title">
        <div class="card-header">
          <span id="cont-net-title" class="card-title">Networking</span>
        </div>
        <div class="card-body" style="display: flex; flex-direction: column; gap: 16px;">
          <div class="config-field">
            <span class="config-label">Attached Networks</span>
            <div v-if="container.networks?.length" class="flex-row" style="flex-wrap: wrap; gap: 6px; margin-block-start: 4px;">
              <span v-for="net in container.networks" :key="net" class="status-badge">
                {{ net }}
              </span>
            </div>
            <span v-else class="text-3" style="font-size: 13px;">No networks attached</span>
          </div>

          <div class="config-field">
            <span class="config-label">Exposed Ports</span>
            <div v-if="container.ports?.length" style="display: flex; flex-direction: column; gap: 4px; margin-block-start: 4px;">
              <span v-for="(port, idx) in container.ports" :key="idx" class="mono text-1">
                {{ port.public_port }}:{{ port.private_port }}/{{ port.type }}
              </span>
            </div>
            <span v-else class="text-3" style="font-size: 13px;">No host ports exposed</span>
          </div>
        </div>
      </section>

      <!-- Mounts (6 cols — Desktop Table + Mobile 2-Row Cards) -->
      <section class="surface-card col-6" aria-labelledby="cont-mounts-title">
        <div class="card-header">
          <span id="cont-mounts-title" class="card-title">Mounts</span>
          <span class="count-pill">{{ container.mounts?.length || 0 }}</span>
        </div>

        <div v-if="!container.mounts?.length" class="card-body text-center text-3">
          No mounts configured
        </div>

        <template v-else>
          <!-- Desktop Table -->
          <div tabindex="0" role="region" aria-labelledby="cont-mounts-title" class="table-wrapper desktop-table-view">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Source</th>
                  <th>Destination</th>
                  <th class="text-end">Type</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(mount, idx) in container.mounts" :key="idx">
                  <th class="mono text-2" :title="mount.source">{{ mount.source }}</th>
                  <td class="mono" :title="mount.destination">{{ mount.destination }}</td>
                  <td class="text-end"><span class="status-badge">{{ mount.type }}</span></td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Mobile 2-Row Cards -->
          <div class="card-body mobile-cards-view">
            <div v-for="(mount, idx) in container.mounts" :key="idx" class="mobile-item-card" style="cursor: default;">
              <div class="mobile-card-top">
                <span class="mono fw-500 text-truncate" :title="mount.destination">{{ mount.destination }}</span>
                <span class="status-badge">{{ mount.type }}</span>
              </div>
              <div class="mobile-card-bottom">
                <span class="mono text-2 text-truncate" :title="mount.source">{{ mount.source }}</span>
                <span>{{ mount.rw ? 'RW' : 'RO' }}</span>
              </div>
            </div>
          </div>
        </template>
      </section>

      <!-- Environment (6 cols) -->
      <section class="surface-card col-6" aria-labelledby="cont-env-title">
        <div class="card-header">
          <span id="cont-env-title" class="card-title">Environment</span>
          <span class="count-pill">{{ container.env?.length || 0 }}</span>
        </div>
        <div class="card-body">
          <div v-if="!container.env?.length" class="text-center text-3">
            No environment variables
          </div>
          <div v-else class="env-list">
            <div v-for="(env, idx) in container.env" :key="idx" class="env-row">
              <span class="env-key" :title="splitEnv(env).key">{{ splitEnv(env).key }}</span>
              <span class="env-val mono">{{ splitEnv(env).val }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Labels (12 cols) -->
      <section class="surface-card col-12" aria-labelledby="cont-labels-title">
        <div class="card-header">
          <span id="cont-labels-title" class="card-title">Labels</span>
          <span class="count-pill">{{ Object.keys(container.labels || {}).length }}</span>
        </div>
        <div class="card-body">
          <div v-if="!Object.keys(container.labels || {}).length" class="text-center text-3">
            No labels found
          </div>
          <div v-else class="env-list">
            <div v-for="(value, key) in container.labels" :key="key" class="env-row">
              <span class="env-key" :title="String(key)">{{ key }}</span>
              <span class="env-val mono">{{ value }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- TAB 2: LOGS -->
    <div v-else-if="tab === 'logs'" role="tabpanel" class="console-tabpanel">
      <section class="surface-card console-card">
        <div class="card-header">
          <div class="flex-row">
            <span class="status-dot" :class="loadingLogs ? 'dot-warn' : 'dot-ok'"></span>
            <span class="card-title">Live Container Logs</span>
          </div>
          <div class="flex-row" style="gap: 12px; flex-wrap: wrap;">
            <label style="font-size: 12px; display: inline-flex; align-items: center; gap: 8px; cursor: pointer;">
              <input v-model="showIndex" type="checkbox" role="switch" class="toggle-switch" /> Line Numbers
            </label>
            <label style="font-size: 12px; display: inline-flex; align-items: center; gap: 8px; cursor: pointer;">
              <input v-model="showTimestamp" type="checkbox" role="switch" class="toggle-switch" /> Timestamps
            </label>
            <button
              type="button"
              class="btn btn-sm"
              :class="{ 'btn-primary': autoScroll }"
              @click="autoScroll = !autoScroll"
            >
              Auto-scroll
            </button>
            <button type="button" class="btn btn-sm" @click="clearLogs">Clear</button>
          </div>
        </div>

        <div class="card-body console-card-body">
          <div ref="logContainer" class="log-viewport console-viewport">
            <div v-for="(log, idx) in logs" :key="idx" class="log-line">
              <span v-if="showIndex" class="log-num">{{ idx + 1 }}</span>
              <span v-if="showTimestamp && log.timestamp" class="log-ts">{{ formatLogTime(log.timestamp) }}</span>
              <span v-html="log.htmlContent"></span>
            </div>
            <div v-if="logs.length === 0" class="text-center text-3" style="padding: 48px 16px;">
              {{ loadingLogs ? 'Connecting to log stream...' : 'No logs available' }}
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- TAB 3: TERMINAL -->
    <div v-else-if="tab === 'terminal'" role="tabpanel" class="console-tabpanel">
      <section class="surface-card console-card">
        <div class="card-header">
          <div class="flex-row" style="flex-wrap: wrap;">
            <span class="card-title">Interactive Console</span>
            <select
              v-model="selectedShell"
              class="form-select"
              style="height: 28px; width: 128px; font-size: 12px;"
              aria-label="Select shell"
              :disabled="isTerminalConnected"
            >
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/zsh">/bin/zsh</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button
              type="button"
              class="btn btn-sm"
              :class="isTerminalConnected ? 'btn-danger-subtle' : 'btn-primary'"
              @click="toggleTerminalConnection"
            >
              {{ isTerminalConnected ? 'Disconnect' : 'Connect' }}
            </button>
          </div>
          <span v-if="container" class="text-3" style="font-size: 12px;">{{ container.node }}</span>
        </div>

        <div class="card-body console-card-body">
          <div
            ref="terminalContainer"
            class="log-viewport terminal-viewport console-viewport"
            style="padding: 8px; overflow: hidden;"
          >
            <div v-if="!isTerminalConnected" class="text-center text-3" style="padding: 48px 16px;">
              Select a shell and click Connect to start an interactive session.
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Stop Modal -->
    <AppModal v-model="stopDialog" title="Stop Container?" :persistent="stopping">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to stop container
        <strong class="text-1 mono">{{ container ? formatName(container.names) : '' }}</strong>?
      </p>
      <template #footer>
        <button type="button" class="btn" :disabled="stopping" @click="stopDialog = false">Cancel</button>
        <button type="button" class="btn btn-primary" :disabled="stopping" @click="stopContainer">
          {{ stopping ? 'Stopping...' : 'Stop' }}
        </button>
      </template>
    </AppModal>

    <!-- Restart Modal -->
    <AppModal v-model="restartDialog" title="Restart Container?" :persistent="restarting">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to restart container
        <strong class="text-1 mono">{{ container ? formatName(container.names) : '' }}</strong>?
      </p>
      <template #footer>
        <button type="button" class="btn" :disabled="restarting" @click="restartDialog = false">Cancel</button>
        <button type="button" class="btn btn-primary" :disabled="restarting" @click="restartContainer">
          {{ restarting ? 'Restarting...' : 'Restart' }}
        </button>
      </template>
    </AppModal>

    <!-- Delete Modal -->
    <AppModal v-model="deleteDialog" title="Delete Container?" :persistent="deleting">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to delete container
        <strong class="text-1 mono">{{ container ? formatName(container.names) : '' }}</strong>?
        This action cannot be undone.
      </p>
      <label class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">Force delete</strong>
          <span class="text-2" style="font-size: 12px;">Forcefully kill the container if it is still running</span>
        </span>
        <input v-model="forceDelete" type="checkbox" role="switch" class="toggle-switch" />
      </label>
      <template #footer>
        <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
        <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteContainer">
          {{ deleting ? 'Removing...' : 'Remove' }}
        </button>
      </template>
    </AppModal>

    <!-- Error Modal -->
    <AppModal v-model="errorDialog" title="Operation Failed">
      <p class="text-2" style="margin: 0;">
        The container lifecycle operation could not be completed successfully.
      </p>
      <pre class="log-viewport text-err" style="margin: 0; white-space: pre-wrap;">{{ errorMessage }}</pre>
      <template #footer>
        <button type="button" class="btn btn-primary" @click="errorDialog = false">Dismiss</button>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import Loader from '../../components/Loader.vue';
import AppModal from '../../components/AppModal.vue';
import { useTheme } from '../../theme';
import { splitImageReference } from '../../utils/image';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { AnsiUp } from 'ansi_up';

const ansi = new AnsiUp();
ansi.use_classes = true;

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
const { isDark, currentTheme } = useTheme();

const container = ref<ContainerDetail | null>(null);
const parsedImage = computed(() =>
  splitImageReference(container.value?.image, container.value?.image_id)
);
const loading = ref(false);
const tab = ref<'overview' | 'logs' | 'terminal'>('overview');

const logs = ref<{ timestamp: string; content: string; htmlContent: string }[]>([]);
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
const errorMessage = ref('');

const selectedShell = ref('/bin/sh');
const isTerminalConnected = ref(false);
const terminalContainer = ref<HTMLElement | null>(null);
let term: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let termSocket: WebSocket | null = null;
let termResizeObserver: ResizeObserver | null = null;
let logSocket: WebSocket | null = null;

const splitEnv = (entry: string) => {
  const idx = entry.indexOf('=');
  if (idx === -1) return { key: entry, val: '' };
  return { key: entry.slice(0, idx), val: entry.slice(idx + 1) };
};

const fetchDetails = async () => {
  const id = route.params.id as string;
  const node = route.query.node as string;
  if (!id || !node) return;

  loading.value = true;
  try {
    const response = await fetch(`/api/containers/detail?id=${encodeURIComponent(id)}&node=${encodeURIComponent(node)}`);
    if (response.ok) {
      container.value = await response.json();
    }
  } catch (error) {
    console.error('Failed to fetch container details:', error);
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

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('halyard_token') || '';
  const wsUrl = `${protocol}//${window.location.host}/api/containers/logs?id=${encodeURIComponent(id)}&node=${encodeURIComponent(node)}&token=${encodeURIComponent(token)}`;

  const socket = new WebSocket(wsUrl);
  logSocket = socket;

  socket.onopen = () => {
    loadingLogs.value = false;
  };

  socket.onmessage = (event) => {
    const lines = event.data.split('\n');
    lines.forEach((line: string) => {
      if (line.trim()) {
        const tsMatch = line.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s(.*)$/);
        if (tsMatch) {
          logs.value.push({
            timestamp: tsMatch[1],
            content: tsMatch[2],
            htmlContent: ansi.ansi_to_html(tsMatch[2]),
          });
        } else {
          logs.value.push({
            timestamp: '',
            content: line,
            htmlContent: ansi.ansi_to_html(line),
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
    console.error('WebSocket error:', error);
    loadingLogs.value = false;
  };

  socket.onclose = () => {
    loadingLogs.value = false;
  };
};

const stopLogStream = () => {
  if (logSocket) {
    logSocket.close();
    logSocket = null;
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
      { method: 'POST' }
    );
    if (response.ok) {
      await fetchDetails();
    } else {
      errorMessage.value = await response.text();
      errorDialog.value = true;
    }
  } catch (error) {
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
      { method: 'POST' }
    );
    if (response.ok) {
      await fetchDetails();
    } else {
      errorMessage.value = await response.text();
      errorDialog.value = true;
    }
  } catch (error) {
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
      { method: 'POST' }
    );
    if (response.ok) {
      await fetchDetails();
    } else {
      errorMessage.value = await response.text();
      errorDialog.value = true;
    }
  } catch (error) {
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
      { method: 'DELETE' }
    );
    if (response.ok) {
      router.back();
    } else {
      errorMessage.value = await response.text();
      errorDialog.value = true;
    }
  } catch (error) {
    errorMessage.value = String(error);
    errorDialog.value = true;
  } finally {
    deleting.value = false;
  }
};

const getTerminalTheme = () => {
  const c = currentTheme.value.colors;
  return {
    background: c.background,
    foreground: c.textPrimary,
    cursor: c.accent,
  };
};

watch(isDark, () => {
  if (term) {
    term.options.theme = getTerminalTheme();
  }
});

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
    fontSize: 13,
    fontFamily: 'JetBrains Mono, SFMono-Regular, Consolas, monospace',
    theme: getTerminalTheme(),
  });

  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);

  if (terminalContainer.value) {
    terminalContainer.value.innerHTML = '';
    term.open(terminalContainer.value);
    fitAddon.fit();

    if (typeof ResizeObserver !== 'undefined') {
      termResizeObserver = new ResizeObserver(() => {
        sendTerminalSize();
      });
      termResizeObserver.observe(terminalContainer.value);
    }
  }

  term.write(`Connecting to shell ${selectedShell.value}...\r\n`);

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('halyard_token') || '';
  const wsUrl = `${protocol}//${window.location.host}/api/containers/exec?id=${encodeURIComponent(id)}&node=${encodeURIComponent(node)}&shell=${encodeURIComponent(selectedShell.value)}&token=${encodeURIComponent(token)}`;

  termSocket = new WebSocket(wsUrl);
  termSocket.binaryType = 'arraybuffer';

  termSocket.onopen = () => {
    term?.write('\r\n--- Shell Connected ---\r\n\r\n');
    sendTerminalSize();
  };

  termSocket.onmessage = (event) => {
    if (term) {
      term.write(new Uint8Array(event.data));
    }
  };

  termSocket.onclose = () => {
    term?.write('\r\n--- Shell Disconnected ---\r\n');
    isTerminalConnected.value = false;
  };

  termSocket.onerror = (err) => {
    console.error('Terminal WebSocket error:', err);
    term?.write('\r\nError: Connection failed\r\n');
    isTerminalConnected.value = false;
  };

  term.onData((data) => {
    if (termSocket && termSocket.readyState === WebSocket.OPEN) {
      termSocket.send(data);
    }
  });

  window.addEventListener('resize', handleWindowResize);
};

const disconnectTerminal = () => {
  window.removeEventListener('resize', handleWindowResize);
  if (termResizeObserver) {
    termResizeObserver.disconnect();
    termResizeObserver = null;
  }
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
    termSocket.send(
      JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows,
      })
    );
  }
};

const handleWindowResize = () => {
  sendTerminalSize();
};

watch(tab, (newTab) => {
  if (newTab === 'logs') {
    startLogStream();
    disconnectTerminal();
  } else if (newTab === 'terminal') {
    stopLogStream();
  } else {
    stopLogStream();
    disconnectTerminal();
  }
});

const formatLogTime = (ts: string) => {
  if (!ts) return '';
  try {
    const date = new Date(ts);
    return date.toLocaleTimeString([], {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return ts.substring(11, 19);
  }
};

const formatName = (names: string[]) => {
  if (!names || names.length === 0) return 'Unknown';
  return names[0].replace(/^\//, '');
};

const formatStateLabel = (state: string) => {
  if (!state) return 'Unknown';
  return state.charAt(0).toUpperCase() + state.slice(1).toLowerCase();
};

const getStateDotClass = (state: string) => {
  switch ((state || '').toLowerCase()) {
    case 'running':
      return 'dot-ok';
    case 'exited':
      return 'dot-err';
    case 'paused':
      return 'dot-warn';
    default:
      return '';
  }
};

onMounted(() => {
  fetchDetails();
  if (tab.value === 'logs') {
    startLogStream();
  }
});

onUnmounted(() => {
  stopLogStream();
  disconnectTerminal();
});
</script>

<style scoped>
.viewport-fit-active {
  flex: 1 1 0;
  min-height: 0;
}

.console-tabpanel,
.console-card,
.console-card-body {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.console-viewport {
  flex: 1 1 0;
  min-height: 0;
}

.terminal-viewport :deep(.xterm) {
  height: 100%;
}

.terminal-viewport :deep(.xterm-viewport) {
  background-color: var(--surface-0) !important;
}
</style>
