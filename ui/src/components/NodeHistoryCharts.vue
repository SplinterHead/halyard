<template>
  <div class="mt-4 mb-4">
    <div class="d-flex align-center mb-4 px-1">
      <h3 class="text-subtitle-2 font-weight-bold text-grey">Historical Performance (Last 24h)</h3>
      <v-spacer></v-spacer>
      <v-btn icon="mdi-refresh" size="x-small" class="refresh-btn" flat @click="fetchHistory" :loading="loading"></v-btn>
    </div>

    <v-row>
      <v-col cols="12" md="6">
        <v-card class="glass-card pa-2" elevation="0">
          <apexchart
            type="area"
            height="200"
            :options="cpuChartOptions"
            :series="cpuSeries"
          ></apexchart>
        </v-card>
      </v-col>
      <v-col cols="12" md="6">
        <v-card class="glass-card pa-2" elevation="0">
          <apexchart
            type="area"
            height="200"
            :options="memChartOptions"
            :series="memSeries"
          ></apexchart>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

const props = defineProps<{
  nodeId: string
}>()

const loading = ref(false)
const history = ref<any[]>([])

const fetchHistory = async () => {
  loading.value = true
  try {
    const response = await fetch(`/api/nodes/history?id=${props.nodeId}`)
    if (response.ok) {
      history.value = await response.json()
    }
  } catch (err) {
    console.error('Failed to fetch history:', err)
  } finally {
    loading.value = false
  }
}

const cpuSeries = computed(() => [{
  name: 'CPU Usage',
  data: history.value.map(h => ({
    x: new Date(h.timestamp).getTime(),
    y: Math.round(h.cpu_usage * 100) / 100
  }))
}])

const memSeries = computed(() => [{
  name: 'Memory Usage',
  data: history.value.map(h => ({
    x: new Date(h.timestamp).getTime(),
    y: Math.round((h.memory_usage / (1024 * 1024 * 1024)) * 100) / 100 // Convert to GB
  }))
}])

const commonOptions = {
  chart: {
    toolbar: { show: false },
    animations: { enabled: false },
    background: 'transparent',
  },
  dataLabels: { enabled: false },
  theme: { mode: 'dark' },
  stroke: { curve: 'smooth', width: 2 },
  fill: {
    type: 'gradient',
    gradient: {
      shadeIntensity: 1,
      opacityFrom: 0.4,
      opacityTo: 0.1,
    }
  },
  xaxis: {
    type: 'datetime',
    labels: {
      datetimeUTC: false,
      style: { colors: '#757575' }
    }
  },
  yaxis: {
    labels: {
      style: { colors: '#757575' }
    }
  },
  grid: {
    borderColor: 'rgba(255,255,255,0.05)',
  }
}

const cpuChartOptions = computed(() => ({
  ...commonOptions,
  colors: ['#4CAF50'],
  yaxis: {
    ...commonOptions.yaxis,
    min: 0,
    max: 100,
    title: { text: 'CPU %', style: { color: '#757575' } }
  }
}))

const memChartOptions = computed(() => ({
  ...commonOptions,
  colors: ['#2196F3'],
  yaxis: {
    ...commonOptions.yaxis,
    min: 0,
    title: { text: 'Memory (GB)', style: { color: '#757575' } }
  }
}))

const addStats = (stats: any) => {
  // Only add if it's for this node (though parent should filter)
  history.value.push({
    timestamp: stats.timestamp || new Date().toISOString(),
    cpu_usage: stats.cpu_usage,
    memory_usage: stats.memory_usage
  })
  
  // Keep a reasonable number of points for the chart
  if (history.value.length > 500) {
    history.value.shift()
  }
}

defineExpose({ addStats })

onMounted(fetchHistory)
</script>

<style scoped>
</style>
