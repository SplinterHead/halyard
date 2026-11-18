<template>
  <div class="section-block" style="gap: 24px;">
    <!-- Standardized Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">{{ title }}</h1>
        <p v-if="subtitle" class="page-subtitle">{{ subtitle }}</p>
      </div>
      <div class="header-actions">
        <input
          v-model="searchQuery"
          type="search"
          class="search-input"
          :placeholder="searchPlaceholder"
          :aria-label="searchPlaceholder"
        />
        <slot name="actions"></slot>
        <button
          type="button"
          class="btn btn-icon"
          :disabled="loading"
          aria-label="Refresh"
          title="Refresh"
          @click="$emit('refresh')"
        >
          <v-icon size="16" :class="{ 'spin-icon': loading }">mdi-refresh</v-icon>
        </button>
      </div>
    </div>

    <!-- Additional Dialogs / Top Content Slot -->
    <slot name="top"></slot>

    <Loader :loading="loading" />

    <!-- Empty State -->
    <div v-if="items.length === 0 && !loading" class="surface-card card-body text-center" style="padding: 48px 24px;">
      <v-icon v-if="emptyIcon" size="42" class="text-3 mb-3">{{ emptyIcon }}</v-icon>
      <div class="card-title" style="font-size: 16px;">{{ emptyTitle }}</div>
      <p class="page-subtitle" style="margin-block-start: 6px;">{{ emptyDescription }}</p>
    </div>

    <template v-else>
      <!-- Desktop Table View -->
      <div class="surface-card" :class="{ 'desktop-table-view': hasMobileSlot }">
        <v-data-table
          :headers="headers"
          :items="items"
          :search="searchQuery"
          :sort-by="sortBy"
          :row-props="rowProps"
          :class="{ 'clickable-rows': clickableRows }"
          density="comfortable"
          :items-per-page="itemsPerPage"
          @click:row="(e: any, row: any) => $emit('click:row', e, row)"
        >
          <template v-for="(_, slotName) in $slots" v-slot:[slotName]="slotProps">
            <slot v-if="String(slotName).startsWith('item.')" :name="slotName" v-bind="slotProps"></slot>
          </template>
        </v-data-table>
      </div>

      <!-- Mobile Curated 2-Row Cards View -->
      <div v-if="hasMobileSlot" class="mobile-cards-view">
        <template v-for="(item, index) in filteredItems" :key="index">
          <slot name="mobile-item" :item="item" :index="index"></slot>
        </template>
        <div class="mobile-cards-count">{{ filteredItems.length }} items</div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, useSlots } from 'vue';
import Loader from './Loader.vue';

const props = withDefaults(
  defineProps<{
    title: string;
    subtitle?: string;
    items?: any[];
    headers?: any[];
    loading?: boolean;
    emptyIcon?: string;
    emptyTitle?: string;
    emptyDescription?: string;
    searchPlaceholder?: string;
    sortBy?: any[];
    rowProps?: (data: any) => any;
    itemsPerPage?: number | string;
    clickableRows?: boolean;
  }>(),
  {
    subtitle: '',
    items: () => [],
    headers: () => [],
    loading: false,
    emptyIcon: '',
    emptyTitle: 'No items found',
    emptyDescription: '',
    searchPlaceholder: 'Search...',
    sortBy: () => [],
    rowProps: undefined,
    itemsPerPage: 25,
    clickableRows: false,
  }
);

defineEmits<{
  refresh: [];
  'click:row': [event: any, row: any];
}>();

const slots = useSlots();
const searchQuery = ref('');

const hasMobileSlot = computed(() => Boolean(slots['mobile-item']));

const flattenValue = (val: unknown): string => {
  if (val == null) return '';
  if (Array.isArray(val)) return val.map(flattenValue).join(' ');
  if (typeof val === 'object') return Object.values(val as Record<string, unknown>).map(flattenValue).join(' ');
  return String(val);
};

const filteredItems = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  let list = props.items;
  if (q) {
    list = list.filter((item) => flattenValue(item).toLowerCase().includes(q));
  } else {
    list = [...list];
  }

  if (props.sortBy && props.sortBy.length > 0) {
    const primary = props.sortBy[0];
    const key = primary?.key;
    const desc = primary?.order === 'desc';
    if (key) {
      list.sort((a: any, b: any) => {
        const av = a?.[key];
        const bv = b?.[key];
        const cmp = flattenValue(av).localeCompare(flattenValue(bv), undefined, { numeric: true });
        return desc ? -cmp : cmp;
      });
    }
  }

  return list;
});
</script>

<style scoped>
.spin-icon {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
