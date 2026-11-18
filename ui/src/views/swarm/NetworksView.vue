<template>
  <DataTablePage
    title="Swarm Networks"
    subtitle="Overlay and bridge networks across the Docker Swarm cluster."
    :items="networks"
    :headers="headers"
    :loading="loading"
    empty-icon="mdi-network-off-outline"
    empty-title="No networks found"
    empty-description="Networks created for your services will appear here once detected."
    search-placeholder="Search networks..."
    :sort-by="[{ key: 'name', order: 'asc' }]"
    clickable-rows
    @refresh="fetchNetworks"
    @click:row="goToDetail"
  >
    <template #actions>
      <button type="button" class="btn btn-primary" @click="dialog = true">
        + Create Network
      </button>
    </template>

    <template #top>
      <!-- Create Network Modal -->
      <AppModal v-model="dialog" title="Create Swarm Network" max-width="560px" :persistent="saving">
        <div class="form-field">
          <label class="form-label" for="net-name">Network Name</label>
          <input
            id="net-name"
            v-model="newNetwork.name"
            type="text"
            class="form-input"
            placeholder="my-overlay-net"
          />
        </div>

        <div class="form-field">
          <label class="form-label" for="net-driver">Driver</label>
          <select id="net-driver" v-model="newNetwork.driver" class="form-select">
            <option value="overlay">overlay</option>
            <option value="bridge">bridge</option>
          </select>
        </div>

        <div class="config-grid">
          <label class="nested-box flex-between" style="cursor: pointer;">
            <span>
              <strong class="fw-500 text-1" style="display: block;">Attachable</strong>
              <span class="text-2" style="font-size: 12px;">Allow standalone containers</span>
            </span>
            <input v-model="newNetwork.attachable" type="checkbox" role="switch" class="toggle-switch" />
          </label>
          <label class="nested-box flex-between" style="cursor: pointer;">
            <span>
              <strong class="fw-500 text-1" style="display: block;">Internal</strong>
              <span class="text-2" style="font-size: 12px;">Restrict external access</span>
            </span>
            <input v-model="newNetwork.internal" type="checkbox" role="switch" class="toggle-switch" />
          </label>
        </div>

        <div class="config-grid">
          <div class="form-field">
            <label class="form-label" for="net-subnet">Subnet (CIDR, Optional)</label>
            <input
              id="net-subnet"
              v-model="newNetwork.ipam.subnet"
              type="text"
              class="form-input mono"
              placeholder="10.0.10.0/24"
            />
          </div>
          <div class="form-field">
            <label class="form-label" for="net-gateway">Gateway (Optional)</label>
            <input
              id="net-gateway"
              v-model="newNetwork.ipam.gateway"
              type="text"
              class="form-input mono"
              placeholder="10.0.10.1"
            />
          </div>
        </div>

        <label v-if="newNetwork.driver === 'overlay'" class="nested-box flex-between" style="cursor: pointer;">
          <span>
            <strong class="fw-500 text-1" style="display: block;">Enable Encryption (IPSEC)</strong>
            <span class="text-2" style="font-size: 12px;">Encrypt overlay network traffic between nodes</span>
          </span>
          <input v-model="newNetwork.encrypted" type="checkbox" role="switch" class="toggle-switch" />
        </label>

        <template #footer>
          <button type="button" class="btn" :disabled="saving" @click="closeDialog">Cancel</button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="saving || !newNetwork.name.trim()"
            @click="createNetwork"
          >
            {{ saving ? 'Creating...' : 'Create' }}
          </button>
        </template>
      </AppModal>

      <!-- Delete Confirmation Modal -->
      <AppModal v-model="deleteDialog" title="Delete Network?" :persistent="deleting">
        <p class="text-2" style="margin: 0;">
          Are you sure you want to remove the network
          <strong class="text-1">{{ networkToDelete?.name }}</strong>? This action cannot be undone.
        </p>
        <template #footer>
          <button type="button" class="btn" :disabled="deleting" @click="deleteDialog = false">Cancel</button>
          <button type="button" class="btn btn-danger" :disabled="deleting" @click="deleteNetwork">
            {{ deleting ? 'Removing...' : 'Remove Network' }}
          </button>
        </template>
      </AppModal>

      <!-- Error Modal -->
      <AppModal v-model="errorDialog" title="Network Deletion Failed">
        <p class="text-2" style="margin: 0;">
          The network could not be deleted. This usually happens if it is still being used by one or more services or containers.
        </p>
        <pre class="log-viewport text-err" style="margin: 0; white-space: pre-wrap;">{{ errorMessage }}</pre>
        <template #footer>
          <button type="button" class="btn btn-primary" @click="errorDialog = false">Dismiss</button>
        </template>
      </AppModal>
    </template>

    <template #item.name="{ item }">
      <div class="d-flex flex-column" style="gap: 2px;">
        <div class="flex-row">
          <span class="status-dot" :class="item.scope === 'swarm' ? 'dot-ok' : ''"></span>
          <span class="fw-500 text-1">{{ item.name }}</span>
        </div>
        <span class="text-3" style="font-size: 12px; padding-inline-start: 15px;">
          {{ item.stack === '-' ? 'System' : item.stack }}
        </span>
      </div>
    </template>

    <template #item.internal="{ value }">
      <span class="status-badge">{{ value ? 'Yes' : 'No' }}</span>
    </template>

    <template #item.attachable="{ value }">
      <span class="status-badge">{{ value ? 'Yes' : 'No' }}</span>
    </template>

    <template #item.subnet="{ value }">
      <span v-if="value" class="mono text-2">{{ value }}</span>
      <span v-else class="text-3">—</span>
    </template>

    <template #item.ip_usage="{ item }">
      <div v-if="item.total_ips > 0" class="telemetry-cell" style="width: 110px; margin-inline-start: 0;">
        <div class="flex-between w-100" style="font-size: 12px;">
          <span class="mono text-2">{{ item.used_ips }}/{{ item.total_ips }}</span>
          <span class="text-1">{{ Math.round((item.used_ips / item.total_ips) * 100) }}%</span>
        </div>
        <div class="telemetry-track">
          <div
            class="telemetry-fill"
            :class="getIpUsageClass(item.used_ips, item.total_ips)"
            :style="{ width: `${Math.min(100, Math.round((item.used_ips / item.total_ips) * 100))}%` }"
          ></div>
        </div>
      </div>
      <span v-else class="text-3">—</span>
    </template>

    <template #item.created_at="{ value }">
      <div class="text-end">
        <RelativeTime :value="value" />
      </div>
    </template>

    <template #item.actions="{ item }">
      <div class="d-flex justify-center">
        <button
          type="button"
          class="btn btn-icon btn-sm btn-danger-subtle"
          title="Delete Network"
          aria-label="Delete Network"
          @click.stop="confirmDelete(item)"
        >
          <v-icon size="15">mdi-delete-outline</v-icon>
        </button>
      </div>
    </template>

    <!-- Mobile Curated 2-Row Card -->
    <template #mobile-item="{ item }">
      <div
        class="mobile-item-card"
        role="button"
        tabindex="0"
        @click="goToDetail($event, { item })"
        @keydown.enter="goToDetail($event, { item })"
      >
        <div class="mobile-card-top">
          <div class="flex-row" style="min-width: 0;">
            <span class="status-dot" :class="item.scope === 'swarm' ? 'dot-ok' : ''"></span>
            <span class="fw-500 text-1 text-truncate">{{ item.name }}</span>
          </div>
          <span class="status-badge">{{ item.scope }}</span>
        </div>
        <div class="mobile-card-bottom">
          <span class="mono">{{ item.subnet || 'No subnet' }}</span>
          <span><span class="text-3">Stack:</span> {{ item.stack === '-' ? 'System' : item.stack }}</span>
        </div>
      </div>
    </template>
  </DataTablePage>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import DataTablePage from '../../components/DataTablePage.vue';
import AppModal from '../../components/AppModal.vue';
import RelativeTime from '../../components/RelativeTime.vue';

const router = useRouter();

interface Network {
  id: string;
  name: string;
  scope: string;
  subnet: string;
  stack: string;
  created_at: string;
  internal: boolean;
  attachable: boolean;
  used_ips: number;
  total_ips: number;
}

const networks = ref<Network[]>([]);
const loading = ref(false);
const dialog = ref(false);
const saving = ref(false);
const deleteDialog = ref(false);
const deleting = ref(false);
const networkToDelete = ref<Network | null>(null);
const errorDialog = ref(false);
const errorMessage = ref('');

const newNetwork = ref({
  name: '',
  driver: 'overlay',
  attachable: true,
  internal: false,
  encrypted: false,
  ipam: {
    subnet: '',
    gateway: '',
  },
});

const headers = [
  { title: 'Network / Stack', key: 'name', sortable: true, align: 'start' as const },
  { title: 'Internal', key: 'internal', width: '95px', align: 'start' as const },
  { title: 'Attachable', key: 'attachable', width: '105px', align: 'start' as const },
  { title: 'Subnet', key: 'subnet', width: '165px', align: 'start' as const },
  {
    title: 'IP Usage',
    key: 'ip_usage',
    width: '140px',
    align: 'start' as const,
    value: (item: Network) => (item.total_ips > 0 ? item.used_ips / item.total_ips : 0),
  },
  { title: 'Created', key: 'created_at', width: '130px', align: 'end' as const },
  { title: 'Actions', key: 'actions', width: '80px', align: 'center' as const, sortable: false },
];

const getIpUsageClass = (used: number, total: number) => {
  if (!total) return '';
  const pct = (used / total) * 100;
  if (pct >= 90) return 'is-err';
  if (pct >= 75) return 'is-warn';
  return '';
};

const goToDetail = (_: any, { item }: any) => {
  router.push({ name: 'network-detail', params: { id: item.id } });
};

const confirmDelete = (network: Network) => {
  networkToDelete.value = network;
  deleteDialog.value = true;
};

const deleteNetwork = async () => {
  if (!networkToDelete.value) return;
  deleting.value = true;
  try {
    const response = await fetch(`/api/networks?id=${networkToDelete.value.id}`, {
      method: 'DELETE',
    });
    if (response.ok) {
      await fetchNetworks();
      deleteDialog.value = false;
    } else {
      errorMessage.value = await response.text();
      errorDialog.value = true;
    }
  } catch (err) {
    errorMessage.value = String(err);
    errorDialog.value = true;
  } finally {
    deleting.value = false;
    networkToDelete.value = null;
  }
};

const fetchNetworks = async () => {
  loading.value = true;
  try {
    const response = await fetch('/api/networks');
    if (response.ok) {
      networks.value = (await response.json()) || [];
    }
  } catch (error) {
    console.error('Failed to fetch networks:', error);
  } finally {
    loading.value = false;
  }
};

const createNetwork = async () => {
  if (!newNetwork.value.name.trim()) return;
  saving.value = true;
  try {
    const options: Record<string, string> = {};
    if (newNetwork.value.encrypted) {
      options['encrypted'] = 'true';
    }

    const payload = {
      name: newNetwork.value.name.trim(),
      driver: newNetwork.value.driver,
      attachable: newNetwork.value.attachable,
      internal: newNetwork.value.internal,
      options,
      ipam: newNetwork.value.ipam.subnet
        ? {
            subnet: newNetwork.value.ipam.subnet,
            gateway: newNetwork.value.ipam.gateway,
          }
        : null,
    };

    const response = await fetch('/api/networks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (response.ok) {
      await fetchNetworks();
      closeDialog();
    } else {
      alert('Failed to create network: ' + (await response.text()));
    }
  } catch (err) {
    console.error('Failed to create network:', err);
  } finally {
    saving.value = false;
  }
};

const closeDialog = () => {
  dialog.value = false;
  newNetwork.value = {
    name: '',
    driver: 'overlay',
    attachable: true,
    internal: false,
    encrypted: false,
    ipam: { subnet: '', gateway: '' },
  };
};

onMounted(fetchNetworks);
</script>
