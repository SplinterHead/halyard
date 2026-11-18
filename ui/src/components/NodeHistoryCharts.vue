<template>
  <section class="section-block" aria-label="Historical Performance">
    <div class="section-header">
      <h2 class="section-title">Historical Performance (Last 24h)</h2>
      <button
        type="button"
        class="btn btn-icon btn-sm"
        :disabled="loading"
        aria-label="Refresh history"
        @click="fetchHistory"
      >
        <v-icon size="15">mdi-refresh</v-icon>
      </button>
    </div>

    <div class="detail-grid">
      <section class="surface-card col-6">
        <div class="card-header">
          <span class="card-title">CPU Usage (%)</span>
        </div>
        <div class="card-body" style="padding: 12px;">
          <apexchart
            type="area"
            height="200"
            :options="cpuChartOptions"
            :series="cpuSeries"
          ></apexchart>
        </div>
      </section>

      <section class="surface-card col-6">
        <div class="card-header">
          <span class="card-title">Memory Usage (GB)</span>
        </div>
        <div class="card-body" style="padding: 12px;">
          <apexchart
            type="area"
            height="200"
            :options="memChartOptions"
            :series="memSeries"
          ></apexchart>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useTheme } from '../theme';

const props = defineProps<{
  nodeId: string;
  totalMemory?: number;
}>();

const { isDark, currentTheme } = useTheme();
const loading = ref(false);
const history = ref<any[]>([]);

const fetchHistory = async () => {
  loading.value = true;
  try {
    const response = await fetch(`/api/nodes/history?id=${props.nodeId}`);
    if (response.ok) {
      history.value = (await response.json()) || [];
    }
  } catch (err) {
    console.error('Failed to fetch history:', err);
  } finally {
    loading.value = false;
  }
};

const cpuSeries = computed(() => [
  {
    name: 'CPU Usage',
    data: history.value.map((h) => ({
      x: new Date(h.timestamp).getTime(),
      y: Math.round(h.cpu_usage * 100) / 100,
    })),
  },
]);

const memSeries = computed(() => [
  {
    name: 'Memory Usage',
    data: history.value.map((h) => ({
      x: new Date(h.timestamp).getTime(),
      y: Math.round((h.memory_usage / (1024 * 1024 * 1024)) * 100) / 100,
    })),
  },
]);

const commonOptions = computed(() => {
  const c = currentTheme.value.colors;
  return {
    chart: {
      toolbar: { show: false },
      animations: { enabled: false },
      background: 'transparent',
      fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, sans-serif',
    },
    dataLabels: { enabled: false },
    theme: { mode: isDark.value ? 'dark' : 'light' },
    stroke: { curve: 'smooth', width: 2 },
    fill: {
      type: 'gradient',
      gradient: {
        shadeIntensity: 1,
        opacityFrom: 0.35,
        opacityTo: 0.05,
      },
    },
    xaxis: {
      type: 'datetime',
      labels: {
        datetimeUTC: false,
        style: { colors: c.textSecondary },
      },
    },
    yaxis: {
      labels: {
        style: { colors: c.textSecondary },
      },
    },
    grid: {
      borderColor: c.borderSecondary,
    },
  };
});

const cpuChartOptions = computed(() => ({
  ...commonOptions.value,
  colors: [currentTheme.value.colors.statusOk],
  yaxis: {
    ...commonOptions.value.yaxis,
    min: 0,
    max: 100,
  },
}));

const memChartOptions = computed(() => {
  const opts: any = {
    ...commonOptions.value,
    colors: [currentTheme.value.colors.accent],
    yaxis: {
      ...commonOptions.value.yaxis,
      min: 0,
    },
  };
  if (props.totalMemory) {
    opts.yaxis.max = Math.round((props.totalMemory / (1024 * 1024 * 1024)) * 100) / 100;
  }
  return opts;
});

const addStats = (stats: any) => {
  history.value.push({
    timestamp: stats.timestamp || new Date().toISOString(),
    cpu_usage: stats.cpu_usage,
    memory_usage: stats.memory_usage,
  });
  if (history.value.length > 500) {
    history.value.shift();
  }
};

defineExpose({ addStats });

onMounted(fetchHistory);
</script>
