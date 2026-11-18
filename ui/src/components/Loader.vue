<template>
  <v-progress-linear
    v-if="loading"
    indeterminate
    color="primary"
    height="2"
    class="mb-2"
  ></v-progress-linear>
</template>

<script setup lang="ts">
defineProps<{
  loading: boolean
}>()
</script>
