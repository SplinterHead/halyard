<template>
  <span class="text-2 cursor-help" :title="formattedFull">
    {{ formatRelativeTime(valueStr) }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { formatRelativeTime } from '../utils/time';

const props = defineProps<{
  value: string | Date;
}>();

const valueStr = computed(() => {
  if (!props.value) return '';
  return props.value instanceof Date ? props.value.toISOString() : String(props.value);
});

const formattedFull = computed(() => {
  if (!valueStr.value || valueStr.value.startsWith('0001')) return '';
  const d = new Date(valueStr.value);
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString();
});
</script>

<style scoped>
.cursor-help {
  cursor: help;
}
</style>
