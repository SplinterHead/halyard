<template>
  <div class="section-block" style="gap: 32px;">
    <!-- Hero Status Header -->
    <div class="page-header">
      <div class="page-header-left">
        <div class="status-line" :class="isClusterHealthy ? 'text-ok' : 'text-warn'">
          <span class="status-dot" :class="isClusterHealthy ? 'dot-ok' : 'dot-warn'"></span>
          <span>{{ isClusterHealthy ? 'All cluster systems healthy' : 'Cluster issues require attention' }}</span>
        </div>
        <h1 class="page-title">
          {{ isClusterHealthy ? 'Cluster is operating normally.' : 'One or more Swarm workloads require attention.' }}
        </h1>
        <p class="page-subtitle">
          {{ nodes.length }} Swarm nodes connected. {{ stacks.length }} stacks and {{ services.length }} active services running.
        </p>
      </div>
      <div class="header-actions">
        <button type="button" class="btn" :disabled="pruning" @click="pruneSwarm">
          Prune Swarm
        </button>
        <button type="button" class="btn" :disabled="loading" @click="fetchAllStats">
          <v-icon size="15">mdi-refresh</v-icon>
          <span>Refresh</span>
        </button>
      </div>
    </div>

    <Loader :loading="loading && nodes.length === 0" />

    <!-- 4 Key Metric Cards -->
    <section class="metrics-row" aria-label="Cluster Summary">
      <div class="metric-box" role="button" tabindex="0" @click="$router.push('/swarm/nodes')" @keydown.enter="$router.push('/swarm/nodes')">
        <span class="metric-label">Nodes</span>
        <span class="metric-number">{{ nodes.length }}</span>
        <span class="metric-meta">
          {{ managersCount }} {{ managersCount === 1 ? 'manager' : 'managers' }},
          {{ workersCount }} {{ workersCount === 1 ? 'worker' : 'workers' }}
          <span v-if="offlineCount > 0" class="text-err"> · {{ offlineCount }} offline</span>
        </span>
      </div>

      <div class="metric-box" role="button" tabindex="0" @click="$router.push('/swarm/stacks')" @keydown.enter="$router.push('/swarm/stacks')">
        <span class="metric-label">Stacks</span>
        <span class="metric-number">{{ stacks.length }}</span>
        <span class="metric-meta">
          {{ healthyStacksCount }} healthy,
          <span :class="unhealthyStacksCount > 0 ? 'text-warn' : ''">{{ unhealthyStacksCount }} issues</span>
        </span>
      </div>

      <div class="metric-box" role="button" tabindex="0" @click="$router.push('/swarm/containers')" @keydown.enter="$router.push('/swarm/containers')">
        <span class="metric-label">Replicas</span>
        <span class="metric-number">{{ runningContainersCount }}</span>
        <span class="metric-meta">{{ stoppedContainersCount }} stopped</span>
      </div>

      <div class="metric-box" role="button" tabindex="0" @click="$router.push('/swarm/nodes')" @keydown.enter="$router.push('/swarm/nodes')">
        <span class="metric-label">Cluster Memory</span>
        <span class="metric-number">{{ avgMem }}%</span>
        <span class="metric-meta">CPU: {{ avgCpu }}% avg</span>
      </div>
    </section>

    <!-- Active Workloads Section -->
    <section class="section-block" aria-labelledby="workloads-title">
      <div class="section-header">
        <h2 id="workloads-title" class="section-title">Active Workloads</h2>
        <router-link to="/swarm/stacks" class="section-link">
          View all {{ stacks.length }} stacks →
        </router-link>
      </div>

      <div v-if="featuredStacks.length === 0" class="surface-card card-body text-center text-3">
        No active stacks deployed on the cluster.
      </div>

      <div v-else class="surface-card">
        <div
          v-for="stack in featuredStacks"
          :key="stack.name"
          class="workload-row"
          role="button"
          tabindex="0"
          @click="$router.push(`/swarm/stacks/${stack.name}`)"
          @keydown.enter="$router.push(`/swarm/stacks/${stack.name}`)"
        >
          <div class="workload-info">
            <div class="workload-primary">
              <span class="status-dot" :class="stack.status === 'Healthy' ? 'dot-ok' : 'dot-warn'"></span>
              <span class="workload-name">{{ stack.name }}</span>
              <template v-if="getStackComposePath(stack.name)">
                <span class="text-3">·</span>
                <span class="workload-path">{{ getStackComposePath(stack.name) }}</span>
              </template>
            </div>
            <div class="workload-secondary">
              Services: <span class="text-2">{{ getStackServicesText(stack.name, stack.services) }}</span>
            </div>
          </div>

          <div class="workload-meta">
            <span
              class="fw-500"
              :class="stack.running_replicas === stack.total_replicas ? 'text-1' : 'text-warn'"
            >
              {{ stack.running_replicas }}/{{ stack.total_replicas }} tasks
            </span>
            <span v-if="getStackCommit(stack.name)" class="status-badge mono">
              {{ getStackCommit(stack.name) }}
            </span>
            <span class="text-3">{{ formatRelativeTime(stack.updated_at) }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Swarm Nodes Section -->
    <section class="section-block" aria-labelledby="nodes-section-title">
      <div class="section-header">
        <div id="nodes-section-title" class="section-title">
          <span>Swarm Nodes</span>
          <span class="count-pill">{{ nodes.length }}</span>
        </div>
        <div class="flex-row" style="gap: 10px;">
          <div v-if="nodes.length > 3" class="flex-row" style="gap: 4px;">
            <button
              type="button"
              class="btn btn-icon btn-sm"
              aria-label="Previous nodes"
              :disabled="nodeOffset === 0"
              @click="prevNodes"
            >
              <v-icon size="15">mdi-chevron-left</v-icon>
            </button>
            <button
              type="button"
              class="btn btn-icon btn-sm"
              aria-label="Next nodes"
              :disabled="nodeOffset + 3 >= nodes.length"
              @click="nextNodes"
            >
              <v-icon size="15">mdi-chevron-right</v-icon>
            </button>
          </div>
          <router-link to="/swarm/nodes" class="section-link">
            View node details →
          </router-link>
        </div>
      </div>

      <div v-if="nodes.length === 0" class="surface-card card-body text-center text-3">
        No Swarm nodes reported yet.
      </div>

      <div v-else class="nodes-grid">
        <div
          v-for="node in visibleNodes"
          :key="node.node_id"
          class="node-card"
          role="button"
          tabindex="0"
          @click="$router.push(`/swarm/nodes/${node.node_id}`)"
          @keydown.enter="$router.push(`/swarm/nodes/${node.node_id}`)"
        >
          <div class="node-card-top">
            <div class="flex-row" style="min-width: 0;">
              <span class="status-dot" :class="node.status === 'ready' ? 'dot-ok' : 'dot-err'"></span>
              <span class="fw-500 text-truncate">{{ node.hostname }}</span>
            </div>
            <span :class="node.role === 'manager' ? 'text-2 fw-500' : 'text-3'" style="font-size: 12px;">
              {{ formatRole(node.role) }}
            </span>
          </div>
          <div class="node-card-bottom">
            <span>CPU <strong class="text-1">{{ Math.round(node.cpu_usage || 0) }}%</strong></span>
            <span>RAM <strong class="text-1">{{ formatNodeMem(node) }}</strong></span>
            <span class="mono text-3">{{ node.ip }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- GitOps Synchronization Bar -->
    <div
      class="gitops-bar"
      role="button"
      tabindex="0"
      @click="$router.push('/git/syncs')"
      @keydown.enter="$router.push('/git/syncs')"
    >
      <div class="flex-row" style="flex-wrap: wrap;">
        <span class="fw-500 text-1">GitOps Synchronization</span>
        <span class="text-2" style="font-size: 12.5px;">
          {{ syncedCount }} of {{ syncs.length }} stacks in sync with remote branches
        </span>
      </div>
      <div class="flex-row" style="gap: 16px;">
        <span v-if="driftCount === 0" class="text-ok fw-500" style="font-size: 12.5px;">
          All up to date
        </span>
        <span v-else class="text-warn fw-500" style="font-size: 12.5px;">
          {{ driftCount }} drift detected
        </span>
        <span class="section-link">View Git Syncs →</span>
      </div>
    </div>

    <!-- Prune Swarm Modal -->
    <AppModal v-model="pruneDialog" title="Prune Swarm Resources" :persistent="pruning">
      <p class="text-2" style="font-size: 13px; margin: 0;">
        Select unused resources to reclaim across all {{ nodes.length }} cluster nodes:
      </p>

      <label class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">Unused Containers</strong>
          <span class="text-2" style="font-size: 12px;">Remove stopped containers not backing active services</span>
        </span>
        <input v-model="pruneOptions.containers" type="checkbox" role="switch" class="toggle-switch" />
      </label>

      <label class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">Unused Networks</strong>
          <span class="text-2" style="font-size: 12px;">Remove networks not used by any active container</span>
        </span>
        <input v-model="pruneOptions.networks" type="checkbox" role="switch" class="toggle-switch" />
      </label>

      <label class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">Dangling Images</strong>
          <span class="text-2" style="font-size: 12px;">Remove untagged image layers across nodes</span>
        </span>
        <input v-model="pruneOptions.images" type="checkbox" role="switch" class="toggle-switch" />
      </label>

      <label v-if="pruneOptions.images" class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">All Unused Images (-a)</strong>
          <span class="text-2" style="font-size: 12px;">Also remove tagged images with no active container</span>
        </span>
        <input v-model="pruneOptions.imagesAll" type="checkbox" role="switch" class="toggle-switch" />
      </label>

      <label class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">
            Unused Volumes
            <span class="text-warn" style="font-size: 11.5px; margin-inline-start: 4px;">Caution</span>
          </strong>
          <span class="text-2" style="font-size: 12px;">Remove local volumes not attached to any container</span>
        </span>
        <input v-model="pruneOptions.volumes" type="checkbox" role="switch" class="toggle-switch" />
      </label>

      <template #footer>
        <button type="button" class="btn" :disabled="pruning" @click="pruneDialog = false">
          Cancel
        </button>
        <button type="button" class="btn btn-primary" :disabled="pruning" @click="confirmPrune">
          {{ pruning ? 'Pruning...' : 'Confirm Prune' }}
        </button>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import Loader from '../components/Loader.vue';
import AppModal from '../components/AppModal.vue';
import { formatRelativeTime } from '../utils/time';

const nodes = ref<any[]>([]);
const stacks = ref<any[]>([]);
const containers = ref<any[]>([]);
const services = ref<any[]>([]);
const syncs = ref<any[]>([]);
const loading = ref(false);
const pruning = ref(false);
const pruneDialog = ref(false);
const nodeOffset = ref(0);

const pruneOptions = ref({
  containers: true,
  networks: true,
  volumes: false,
  images: true,
  imagesAll: false,
});

const managersCount = computed(() => nodes.value.filter((n) => n.role === 'manager').length);
const workersCount = computed(() => nodes.value.filter((n) => n.role === 'worker').length);
const offlineCount = computed(() => nodes.value.filter((n) => n.status !== 'ready').length);

const healthyStacksCount = computed(() => stacks.value.filter((s) => s.status === 'Healthy').length);
const unhealthyStacksCount = computed(() => stacks.value.filter((s) => s.status !== 'Healthy').length);

const runningContainersCount = computed(
  () => containers.value.filter((c) => c.state === 'running').length
);
const stoppedContainersCount = computed(
  () => containers.value.filter((c) => c.state !== 'running').length
);

const outOfDateSyncsCount = computed(
  () => syncs.value.filter((s) => s.last_status === 'Out of Date').length
);
const failedSyncsCount = computed(
  () => syncs.value.filter((s) => s.last_status === 'Failed').length
);
const driftCount = computed(() => outOfDateSyncsCount.value + failedSyncsCount.value);
const syncedCount = computed(() => Math.max(0, syncs.value.length - driftCount.value));

const isClusterHealthy = computed(() => {
  return offlineCount.value === 0 && unhealthyStacksCount.value === 0 && failedSyncsCount.value === 0;
});

const avgCpu = computed(() => {
  if (nodes.value.length === 0) return 0;
  const sum = nodes.value.reduce((acc, n) => acc + (n.cpu_usage || 0), 0);
  return Math.round(sum / nodes.value.length);
});

const avgMem = computed(() => {
  if (nodes.value.length === 0) return 0;
  const sum = nodes.value.reduce((acc, n) => {
    if (!n.memory_total) return acc;
    return acc + (n.memory_usage / n.memory_total) * 100;
  }, 0);
  return Math.round(sum / nodes.value.length);
});

const featuredStacks = computed(() => {
  const sorted = [...stacks.value].sort((a, b) => {
    const aHealthy = a.status === 'Healthy' ? 1 : 0;
    const bHealthy = b.status === 'Healthy' ? 1 : 0;
    if (aHealthy !== bHealthy) return aHealthy - bHealthy;
    return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
  });
  return sorted.slice(0, 6);
});

const visibleNodes = computed(() => {
  return nodes.value.slice(nodeOffset.value, nodeOffset.value + 3);
});

const prevNodes = () => {
  nodeOffset.value = Math.max(0, nodeOffset.value - 3);
};

const nextNodes = () => {
  if (nodeOffset.value + 3 < nodes.value.length) {
    nodeOffset.value += 3;
  }
};

const getStackSync = (stackName: string) => {
  return syncs.value.find((s) => s.stack_name === stackName);
};

const getStackComposePath = (stackName: string) => {
  return getStackSync(stackName)?.path || '';
};

const getStackCommit = (stackName: string) => {
  const sha = getStackSync(stackName)?.last_applied_sha;
  return sha ? sha.substring(0, 7) : '';
};

const getStackServicesText = (stackName: string, fallbackCount: number) => {
  const names = services.value
    .filter((svc) => svc.stack === stackName)
    .map((svc) => svc.name);
  if (names.length > 0) return names.join(', ');
  return `${fallbackCount || 0} service${fallbackCount === 1 ? '' : 's'}`;
};

const formatRole = (role: string) => {
  if (!role) return 'Worker';
  return role.charAt(0).toUpperCase() + role.slice(1).toLowerCase();
};

const formatNodeMem = (node: any) => {
  if (!node.memory_total) return '0 / 0 GB';
  const usedGb = (node.memory_usage / (1024 ** 3)).toFixed(1);
  const totalGb = Math.round(node.memory_total / (1024 ** 3));
  return `${usedGb} / ${totalGb} GB`;
};

const fetchAllStats = async () => {
  loading.value = true;
  try {
    const [nRes, sRes, cRes, svcRes, syncRes] = await Promise.all([
      fetch('/api/nodes'),
      fetch('/api/stacks'),
      fetch('/api/containers'),
      fetch('/api/services'),
      fetch('/api/syncs'),
    ]);
    if (nRes.ok) nodes.value = (await nRes.json()) || [];
    if (sRes.ok) stacks.value = (await sRes.json()) || [];
    if (cRes.ok) containers.value = (await cRes.json()) || [];
    if (svcRes.ok) services.value = (await svcRes.json()) || [];
    if (syncRes.ok) syncs.value = (await syncRes.json()) || [];
  } catch (err) {
    console.error('Failed to fetch dashboard stats:', err);
  } finally {
    loading.value = false;
  }
};

const pruneSwarm = () => {
  pruneDialog.value = true;
};

const confirmPrune = async () => {
  pruning.value = true;
  try {
    const response = await fetch('/api/swarm/prune', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        containers: pruneOptions.value.containers,
        networks: pruneOptions.value.networks,
        volumes: pruneOptions.value.volumes,
        images: pruneOptions.value.images,
        images_all: pruneOptions.value.imagesAll,
      }),
    });
    if (response.ok) {
      pruneDialog.value = false;
      await fetchAllStats();
    } else {
      console.error('Failed to prune swarm:', await response.text());
    }
  } catch (err) {
    console.error('Failed to prune swarm:', err);
  } finally {
    pruning.value = false;
  }
};

onMounted(fetchAllStats);
</script>
