<template>
  <v-progress-linear
    v-if="loading"
    indeterminate
    color="primary"
    height="4"
    class="mb-4 rounded-sm"
  ></v-progress-linear>
</template>

<script setup lang="ts">
defineProps<{
  loading: boolean
}>()
</script>
