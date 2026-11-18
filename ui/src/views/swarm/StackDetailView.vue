<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <div class="status-line">
          <button type="button" class="section-link" @click="$router.push('/swarm/stacks')">
            ← Stacks
          </button>
        </div>
        <h1 class="page-title">
          <span>{{ stack ? `Stack: ${stack.name}` : 'Stack Detail' }}</span>
          <span v-if="stack" class="status-badge">
            <span class="status-dot" :class="stack.status === 'Healthy' ? 'dot-ok' : 'dot-warn'"></span>
            {{ stack.status }}
          </span>
        </h1>
        <p v-if="stack" class="page-subtitle">
          {{ stack.services_list?.length || 0 }} Services ·
          {{ stack.containers_list?.length || 0 }} Containers ·
          {{ stack.volumes_list?.length || 0 }} Volumes
        </p>
      </div>

      <div class="header-actions">
        <v-menu v-if="stack && stack.name !== 'Orphaned/Manual'">
          <template v-slot:activator="{ props }">
            <button type="button" class="btn" v-bind="props" :disabled="actionLoading">
              <v-icon size="15">mdi-dots-vertical</v-icon>
              <span>Actions</span>
            </button>
          </template>
          <v-list class="app-menu-list" density="compact" elevation="0">
            <v-list-item @click="performAction('restart')">
              <template v-slot:prepend><v-icon size="16" class="text-warn">mdi-restart</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Force Restart</v-list-item-title>
            </v-list-item>
            <v-list-item @click="performAction('stop')">
              <template v-slot:prepend><v-icon size="16" class="text-warn">mdi-stop-circle-outline</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Stop (Scale to 0)</v-list-item-title>
            </v-list-item>
            <v-list-item @click="performAction('start')">
              <template v-slot:prepend><v-icon size="16" class="text-ok">mdi-play-circle-outline</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Start Services</v-list-item-title>
            </v-list-item>
            <v-list-item @click="performAction('rollback')">
              <template v-slot:prepend><v-icon size="16" class="text-accent">mdi-undo</v-icon></template>
              <v-list-item-title style="font-size: 12.5px;">Rollback Spec</v-list-item-title>
            </v-list-item>
            <v-divider class="my-1" style="border-color: var(--border-2); opacity: 1;"></v-divider>
            <v-list-item @click="confirmDelete">
              <template v-slot:prepend><v-icon size="16" class="text-err">mdi-delete-outline</v-icon></template>
              <v-list-item-title class="text-err" style="font-size: 12.5px;">Delete Stack</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>

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

    <!-- Main 12-Column Detail Grid -->
    <div v-if="stack" class="detail-grid">
      <!-- Left Column (5 cols): Services & Volumes -->
      <div class="col-5" style="display: flex; flex-direction: column; gap: 20px;">
        <!-- Services Card -->
        <section class="surface-card" aria-labelledby="stack-services-title">
          <div class="card-header">
            <span id="stack-services-title" class="card-title">Services</span>
            <span class="count-pill">{{ stack.services_list?.length || 0 }}</span>
          </div>
          <div v-if="!stack.services_list?.length" class="card-body text-center text-3">
            No services in this stack.
          </div>
          <div v-else>
            <div
              v-for="svc in stack.services_list"
              :key="svc.id"
              class="workload-row"
              role="button"
              tabindex="0"
              @click="$router.push(`/swarm/services/${svc.id}`)"
              @keydown.enter="$router.push(`/swarm/services/${svc.id}`)"
            >
              <div class="workload-info">
                <span class="workload-name">{{ svc.name }}</span>
                <span class="workload-secondary mono">{{ (svc.image || '').split('@')[0] }}</span>
              </div>
              <span class="status-badge">
                <span class="status-dot" :class="svc.running === svc.replicas ? 'dot-ok' : 'dot-warn'"></span>
                {{ svc.running }} / {{ svc.replicas }}
              </span>
            </div>
          </div>
        </section>

        <!-- Volumes Card -->
        <section class="surface-card" aria-labelledby="stack-volumes-title">
          <div class="card-header">
            <span id="stack-volumes-title" class="card-title">Volumes</span>
            <span class="count-pill">{{ stack.volumes_list?.length || 0 }}</span>
          </div>
          <div class="card-body">
            <div v-if="!stack.volumes_list?.length" class="text-center text-3" style="font-size: 12.5px;">
              No volumes referenced by this stack.
            </div>
            <div v-else class="volumes-grid">
              <div v-for="vol in stack.volumes_list" :key="vol.name" class="nested-box">
                <div class="fw-500 text-1 text-truncate" :title="vol.name">{{ vol.name }}</div>
                <div class="text-2" style="font-size: 12px; margin-block-start: 2px;">
                  {{ vol.external ? 'External' : 'Managed' }}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <!-- Right Column (7 cols): Containers -->
      <section class="surface-card col-7" aria-labelledby="stack-containers-title">
        <div class="card-header">
          <span id="stack-containers-title" class="card-title">Containers</span>
          <span class="text-2" style="font-size: 12px;">
            {{ runningContainersCount }} running · {{ exitedContainersCount }} exited
          </span>
        </div>
        <div v-if="!stack.containers_list?.length" class="card-body text-center text-3">
          No containers associated with this stack.
        </div>
        <div v-else>
          <div
            v-for="cont in stack.containers_list"
            :key="cont.id"
            class="workload-row"
            role="button"
            tabindex="0"
            @click="$router.push(`/swarm/containers/${cont.id}?node=${cont.node}`)"
            @keydown.enter="$router.push(`/swarm/containers/${cont.id}?node=${cont.node}`)"
          >
            <div class="workload-info">
              <span
                class="workload-name"
                :class="cont.state === 'running' ? 'text-1' : 'text-2'"
              >
                {{ formatContainerName(cont.names) }}
              </span>
              <span class="workload-secondary">{{ cont.node }} · {{ cont.status }}</span>
            </div>
            <span class="status-badge">
              <span
                class="status-dot"
                :class="cont.state === 'running' ? 'dot-ok' : (cont.state === 'exited' ? 'dot-err' : '')"
              ></span>
              {{ formatStateLabel(cont.state) }}
            </span>
          </div>
        </div>
      </section>

      <!-- Full-Width Bottom Section: Cluster Tasks -->
      <section class="col-12 section-block" aria-labelledby="stack-tasks-title">
        <div class="section-header">
          <h2 id="stack-tasks-title" class="section-title">Cluster Tasks</h2>
          <input
            v-model="searchQuery"
            type="search"
            class="search-input"
            placeholder="Search tasks..."
            aria-label="Search tasks"
          />
        </div>

        <!-- Desktop Table View -->
        <div class="surface-card desktop-table-view">
          <div tabindex="0" role="region" aria-labelledby="stack-tasks-title" class="table-wrapper">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Service</th>
                  <th style="width: 185px;">Node</th>
                  <th style="width: 120px;">State</th>
                  <th style="width: 110px;">Desired</th>
                  <th style="width: 240px;">Error / Status</th>
                  <th class="text-end" style="width: 130px;">Updated ↓</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="filteredTasks.length === 0">
                  <td colspan="6" class="text-center text-3">No matching tasks.</td>
                </tr>
                <tr v-for="task in paginatedTasks" :key="task.id">
                  <th :class="task.state?.toLowerCase() === 'running' ? 'text-1' : 'text-2'">
                    {{ task.service_name }}
                  </th>
                  <td class="text-2">{{ task.node_name || 'Unknown' }}</td>
                  <td>
                    <span class="status-badge">
                      <span class="status-dot" :class="getTaskDotClass(task.state)"></span>
                      {{ formatStateLabel(task.state) }}
                    </span>
                  </td>
                  <td :class="task.desired_state?.toLowerCase() === 'running' ? 'text-2' : 'text-3'">
                    {{ formatStateLabel(task.desired_state) }}
                  </td>
                  <td>
                    <span v-if="task.error" class="mono text-err">{{ task.error }}</span>
                    <span v-else class="text-3">—</span>
                  </td>
                  <td class="text-end text-2">
                    <RelativeTime :value="task.updated_at" />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="table-footer">
            <span>Items per page: <strong class="text-1">25</strong></span>
            <span>
              {{ filteredTasks.length === 0 ? '0 of 0' : `${pageStart + 1}–${pageEnd} of ${filteredTasks.length}` }}
            </span>
            <div v-if="totalPages > 1" class="flex-row" style="gap: 4px;">
              <button
                type="button"
                class="btn btn-icon btn-sm"
                :disabled="page <= 1"
                aria-label="Previous page"
                @click="page--"
              >
                <v-icon size="14">mdi-chevron-left</v-icon>
              </button>
              <button
                type="button"
                class="btn btn-icon btn-sm"
                :disabled="page >= totalPages"
                aria-label="Next page"
                @click="page++"
              >
                <v-icon size="14">mdi-chevron-right</v-icon>
              </button>
            </div>
          </div>
        </div>

        <!-- Mobile Curated 2-Row Cards View -->
        <div class="mobile-cards-view">
          <div v-for="task in filteredTasks" :key="task.id" class="mobile-item-card" style="cursor: default;">
            <div class="mobile-card-top">
              <div class="flex-row" style="min-width: 0;">
                <span class="status-dot" :class="getTaskDotClass(task.state)"></span>
                <span
                  class="fw-500 text-truncate"
                  :class="task.state?.toLowerCase() === 'running' ? 'text-1' : 'text-2'"
                >
                  {{ task.service_name }}
                </span>
              </div>
              <span
                style="font-size: 12px;"
                :class="task.state?.toLowerCase() === 'running' ? 'text-2' : 'text-3'"
              >
                {{ formatStateLabel(task.state) }}
              </span>
            </div>
            <div class="mobile-card-bottom">
              <span><span class="text-3">Node:</span> {{ task.node_name || 'Unknown' }}</span>
              <span><span class="text-3">Updated:</span> {{ formatRelativeTime(task.updated_at) }}</span>
            </div>
            <div
              v-if="task.error"
              class="mono text-err"
              style="padding: 6px 14px; font-size: 11px; border-top: 1px solid var(--border-2);"
            >
              {{ task.error }}
            </div>
          </div>
          <div class="mobile-cards-count">{{ filteredTasks.length }} items</div>
        </div>
      </section>
    </div>

    <!-- Delete Confirmation Dialog -->
    <AppModal v-model="deleteDialog" title="Delete Stack?" :persistent="actionLoading">
      <p class="text-2" style="margin: 0;">
        Are you sure you want to remove the stack
        <strong class="text-1">{{ stack?.name }}</strong>? This will stop and remove all associated services.
      </p>

      <label class="nested-box flex-between" style="cursor: pointer;">
        <span>
          <strong class="fw-500 text-1" style="display: block;">Delete associated volumes</strong>
          <span class="text-2" style="font-size: 12px;">Permanently remove volumes created for this stack</span>
        </span>
        <input v-model="deleteVolumes" type="checkbox" role="switch" class="toggle-switch" />
      </label>

      <template #footer>
        <button type="button" class="btn" :disabled="actionLoading" @click="deleteDialog = false">
          Cancel
        </button>
        <button type="button" class="btn btn-danger" :disabled="actionLoading" @click="deleteStack">
          {{ actionLoading ? 'Removing...' : 'Remove Stack' }}
        </button>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import RelativeTime from '../../components/RelativeTime.vue';
import Loader from '../../components/Loader.vue';
import AppModal from '../../components/AppModal.vue';
import { formatRelativeTime } from '../../utils/time';

const route = useRoute();
const router = useRouter();

const searchQuery = ref('');
const stack = ref<any>(null);
const loading = ref(false);
const actionLoading = ref(false);
const deleteDialog = ref(false);
const deleteVolumes = ref(false);
const page = ref(1);
const perPage = 25;

const runningContainersCount = computed(() => {
  return (stack.value?.containers_list || []).filter((c: any) => c.state === 'running').length;
});

const exitedContainersCount = computed(() => {
  return (stack.value?.containers_list || []).filter((c: any) => c.state !== 'running').length;
});

const sortedTasks = computed(() => {
  const tasks = [...(stack.value?.tasks_list || [])];
  return tasks.sort((a: any, b: any) => {
    return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
  });
});

const filteredTasks = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return sortedTasks.value;
  return sortedTasks.value.filter((t: any) => {
    return (
      (t.service_name || '').toLowerCase().includes(q) ||
      (t.node_name || '').toLowerCase().includes(q) ||
      (t.state || '').toLowerCase().includes(q) ||
      (t.desired_state || '').toLowerCase().includes(q) ||
      (t.error || '').toLowerCase().includes(q)
    );
  });
});

watch(searchQuery, () => {
  page.value = 1;
});

const totalPages = computed(() => Math.max(1, Math.ceil(filteredTasks.value.length / perPage)));
const pageStart = computed(() => (page.value - 1) * perPage);
const pageEnd = computed(() => Math.min(filteredTasks.value.length, pageStart.value + perPage));
const paginatedTasks = computed(() => filteredTasks.value.slice(pageStart.value, pageEnd.value));

const formatContainerName = (names: string[]) => {
  if (!names || names.length === 0) return 'Unknown';
  return names[0].replace(/^\//, '');
};

const formatStateLabel = (state: string) => {
  if (!state) return 'Unknown';
  return state.charAt(0).toUpperCase() + state.slice(1).toLowerCase();
};

const getTaskDotClass = (state: string) => {
  switch ((state || '').toLowerCase()) {
    case 'running':
      return 'dot-ok';
    case 'failed':
    case 'rejected':
      return 'dot-err';
    case 'shutdown':
      return '';
    default:
      return 'dot-warn';
  }
};

const fetchDetails = async () => {
  const name = route.params.name as string;
  if (!name) return;

  loading.value = true;
  try {
    const response = await fetch(`/api/stacks/detail?name=${encodeURIComponent(name)}`);
    if (response.ok) {
      const data = await response.json();
      if (data.containers_list) {
        data.containers_list.sort((a: any, b: any) => {
          if (a.state === 'running' && b.state !== 'running') return -1;
          if (a.state !== 'running' && b.state === 'running') return 1;
          return 0;
        });
      }
      stack.value = data;
    }
  } catch (err) {
    console.error('Failed to fetch stack details:', err);
  } finally {
    loading.value = false;
  }
};

const performAction = async (action: string) => {
  if (!stack.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(`/api/stacks/${action}?name=${encodeURIComponent(stack.value.name)}`, {
      method: 'POST',
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
  deleteVolumes.value = false;
  deleteDialog.value = true;
};

const deleteStack = async () => {
  if (!stack.value) return;
  actionLoading.value = true;
  try {
    const response = await fetch(
      `/api/stacks/${encodeURIComponent(stack.value.name)}?deleteVolumes=${deleteVolumes.value}`,
      { method: 'DELETE' }
    );
    if (response.ok) {
      deleteDialog.value = false;
      router.push('/swarm/stacks');
    } else {
      const text = await response.text();
      alert('Failed to remove stack: ' + text);
    }
  } catch (err) {
    console.error('Failed to delete stack:', err);
  } finally {
    actionLoading.value = false;
  }
};

onMounted(fetchDetails);
</script>
