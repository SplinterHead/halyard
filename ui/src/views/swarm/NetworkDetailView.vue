<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <div class="status-line">
          <button type="button" class="section-link" @click="$router.push('/swarm/networks')">
            ← Networks
          </button>
        </div>
        <h1 class="page-title">
          <span>{{ network ? `Network: ${network.name}` : 'Network Detail' }}</span>
          <span v-if="network" class="status-badge">{{ network.driver }}</span>
        </h1>
        <p v-if="network" class="page-subtitle">
          {{ network.scope }} scope · <span class="mono">{{ network.id.substring(0, 12) }}</span>
        </p>
      </div>
      <div class="header-actions">
        <button type="button" class="btn btn-icon" :disabled="loading" aria-label="Refresh" @click="fetchDetail">
          <v-icon size="16">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <Loader :loading="loading" />

    <div v-if="network" class="detail-grid">
      <!-- IPAM Configuration (4 cols) -->
      <section class="surface-card col-4" aria-labelledby="net-ipam-title">
        <div class="card-header">
          <span id="net-ipam-title" class="card-title">IPAM Configuration</span>
        </div>
        <div class="card-body">
          <div v-if="network.ipam_configs && network.ipam_configs.length" style="display: flex; flex-direction: column; gap: 14px;">
            <div
              v-for="(config, idx) in network.ipam_configs"
              :key="idx"
              class="config-grid"
              style="grid-template-columns: 1fr; gap: 12px;"
              :style="idx > 0 ? 'padding-block-start: 14px; border-block-start: 1px solid var(--border-2);' : ''"
            >
              <div class="config-field">
                <span class="config-label">Subnet</span>
                <span class="config-val mono">{{ config.subnet || '—' }}</span>
              </div>
              <div class="config-field">
                <span class="config-label">Gateway</span>
                <span class="config-val mono">{{ config.gateway || '—' }}</span>
              </div>
            </div>
          </div>
          <div v-else class="text-3">No IPAM configuration found.</div>
        </div>
      </section>

      <!-- Properties (4 cols) -->
      <section class="surface-card col-4" aria-labelledby="net-props-title">
        <div class="card-header">
          <span id="net-props-title" class="card-title">Properties</span>
        </div>
        <div class="card-body">
          <div class="config-grid" style="grid-template-columns: 1fr;">
            <div class="flex-between">
              <span class="config-label">Internal</span>
              <span class="status-badge">{{ network.internal ? 'Yes' : 'No' }}</span>
            </div>
            <div class="flex-between">
              <span class="config-label">Attachable</span>
              <span class="status-badge">{{ network.attachable ? 'Yes' : 'No' }}</span>
            </div>
            <div class="flex-between">
              <span class="config-label">Ingress</span>
              <span class="status-badge">{{ network.ingress ? 'Yes' : 'No' }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Metadata (4 cols) -->
      <section class="surface-card col-4" aria-labelledby="net-meta-title">
        <div class="card-header">
          <span id="net-meta-title" class="card-title">Metadata</span>
        </div>
        <div class="card-body" style="display: flex; flex-direction: column; gap: 14px;">
          <div class="config-field">
            <span class="config-label">Created At</span>
            <span class="config-val">{{ new Date(network.created_at).toLocaleString() }}</span>
          </div>
          <div v-if="network.labels && Object.keys(network.labels).length" class="config-field">
            <span class="config-label">Labels</span>
            <div class="env-list" style="margin-block-start: 4px;">
              <div v-for="(v, k) in network.labels" :key="k" class="env-row">
                <span class="env-key" :title="String(k)">{{ k }}</span>
                <span class="env-val mono">{{ v }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Connected Containers (12 cols) -->
      <section class="col-12 section-block" aria-labelledby="net-containers-title">
        <div class="section-header">
          <div id="net-containers-title" class="section-title">
            <span>Connected Containers</span>
            <span class="count-pill">{{ filteredContainers.length }}</span>
          </div>
          <input
            v-model="searchQuery"
            type="search"
            class="search-input"
            placeholder="Search containers..."
            aria-label="Search containers"
          />
        </div>

        <!-- Desktop Table -->
        <div class="surface-card desktop-table-view">
          <div class="table-wrapper">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th style="width: 170px;">IPv4 Address</th>
                  <th style="width: 200px;">IPv6 Address</th>
                  <th style="width: 140px;">Container ID</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="filteredContainers.length === 0">
                  <td colspan="4" class="text-center text-3">No connected containers found.</td>
                </tr>
                <tr
                  v-for="cont in filteredContainers"
                  :key="cont.id"
                  class="clickable-row"
                  @click="goToContainer(cont)"
                >
                  <th>{{ cont.name }}</th>
                  <td class="mono text-2">{{ cont.ipv4 || '—' }}</td>
                  <td class="mono text-2">{{ cont.ipv6 || '—' }}</td>
                  <td class="mono text-3">{{ cont.id.substring(0, 12) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Mobile 2-Row Cards -->
        <div class="mobile-cards-view">
          <div
            v-for="cont in filteredContainers"
            :key="cont.id"
            class="mobile-item-card"
            role="button"
            tabindex="0"
            @click="goToContainer(cont)"
            @keydown.enter="goToContainer(cont)"
          >
            <div class="mobile-card-top">
              <span class="fw-500 text-1 text-truncate">{{ cont.name }}</span>
              <span class="mono text-3">{{ cont.id.substring(0, 12) }}</span>
            </div>
            <div class="mobile-card-bottom">
              <span class="mono"><span class="text-3">IPv4:</span> {{ cont.ipv4 || '—' }}</span>
              <span class="mono"><span class="text-3">IPv6:</span> {{ cont.ipv6 || '—' }}</span>
            </div>
          </div>
          <div class="mobile-cards-count">{{ filteredContainers.length }} items</div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import Loader from '../../components/Loader.vue';

const route = useRoute();
const router = useRouter();
const searchQuery = ref('');
const network = ref<any>(null);
const loading = ref(true);

const filteredContainers = computed(() => {
  const list = network.value?.containers || [];
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return list;
  return list.filter((c: any) => {
    return (
      (c.name || '').toLowerCase().includes(q) ||
      (c.ipv4 || '').toLowerCase().includes(q) ||
      (c.ipv6 || '').toLowerCase().includes(q) ||
      (c.id || '').toLowerCase().includes(q)
    );
  });
});

const fetchDetail = async () => {
  const id = route.params.id as string;
  if (!id) return;

  loading.value = true;
  try {
    const response = await fetch(`/api/networks/detail?id=${encodeURIComponent(id)}`);
    if (response.ok) {
      network.value = await response.json();
    }
  } catch (err) {
    console.error('Error fetching network detail:', err);
  } finally {
    loading.value = false;
  }
};

const goToContainer = (item: any) => {
  router.push(`/swarm/containers/${item.id}`);
};

onMounted(fetchDetail);
</script>
