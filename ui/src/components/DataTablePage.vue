<template>
  <div class="fill-height d-flex flex-column">
    <div class="pa-2 pb-0 d-flex align-center flex-wrap justify-space-between">
      <h1 class="text-h4 font-weight-bold">{{ title }}</h1>
      <div class="d-flex align-center gap-4 mt-2 mt-sm-0">
        <v-text-field
          v-model="searchQuery"
          prepend-inner-icon="mdi-magnify"
          :placeholder="searchPlaceholder"
          variant="solo-filled"
          density="compact"
          flat
          hide-details
          rounded="lg"
          class="search-input glass-input"
          style="width: 280px"
        ></v-text-field>
        <slot name="actions"></slot>
        <v-btn
          icon="mdi-refresh"
          @click="$emit('refresh')"
          :loading="loading"
          size="x-small"
          class="refresh-btn"
          flat
        ></v-btn>
      </div>
    </div>

    <!-- Additional Dialogs / Top Content Slot -->
    <slot name="top"></slot>

    <v-divider class="my-4"></v-divider>
    
    <Loader :loading="loading" />

    <v-row v-if="items.length === 0 && !loading" justify="center" class="mt-8">
      <v-col cols="12" md="6" class="text-center">
        <v-icon size="64" color="grey-lighten-1" class="mb-4">{{ emptyIcon }}</v-icon>
        <h3 class="text-h5 text-grey-darken-1">{{ emptyTitle }}</h3>
        <p class="text-body-1 text-grey-darken-1 mt-2">
          {{ emptyDescription }}
        </p>
      </v-col>
    </v-row>

    <div v-else class="flex-grow-1">
      <v-data-table
        :headers="headers"
        :items="items"
        :search="searchQuery"
        :sort-by="sortBy"
        :row-props="rowProps"
        class="bg-transparent"
        density="comfortable"
        @click:row="(e, row) => $emit('click:row', e, row)"
        items-per-page="25"
      >
        <!-- Pass through all item.* slots -->
        <template v-for="(_, slotName) in $slots" v-slot:[slotName]="slotProps">
          <slot v-if="slotName.startsWith('item.')" :name="slotName" v-bind="slotProps"></slot>
        </template>
      </v-data-table>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import Loader from '@/components/Loader.vue';

const props = defineProps({
  title: String,
  items: {
    type: Array,
    default: () => []
  },
  headers: Array,
  loading: Boolean,
  emptyIcon: String,
  emptyTitle: String,
  emptyDescription: String,
  searchPlaceholder: {
    type: String,
    default: 'Search...'
  },
  sortBy: {
    type: Array,
    default: () => []
  },
  rowProps: {
    type: Function,
    default: undefined
  }
});

defineEmits(['refresh', 'click:row']);

const searchQuery = ref('');
</script>
