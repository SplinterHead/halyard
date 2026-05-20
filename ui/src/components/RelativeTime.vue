<template>
  <v-tooltip location="top">
    <template v-slot:activator="{ props }">
      <span v-bind="props" class="text-caption text-grey cursor-help">
        {{ formatRelativeTime(value) }}
      </span>
    </template>
    {{ new Date(value).toLocaleString() }}
  </v-tooltip>
</template>

<script setup lang="ts">
import { formatRelativeTime } from "../utils/time";

defineProps<{
  value: string | Date;
}>();
</script>

<style scoped>
.cursor-help {
  cursor: help;
}
</style>
