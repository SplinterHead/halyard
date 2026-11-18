<template>
  <dialog
    ref="dialogRef"
    class="modal"
    :style="dialogStyle"
    :aria-labelledby="titleId"
    :closedby="persistent ? 'none' : 'any'"
    @close="onNativeClose"
    @cancel="onNativeCancel"
    @pointerdown="onPointerDown"
    @pointerup="onPointerUp"
  >
    <div class="modal-header">
      <h2 :id="titleId" class="card-title" style="font-size: 15px; margin: 0;">{{ title }}</h2>
      <button
        v-if="!hideClose"
        type="button"
        class="btn btn-icon btn-sm"
        aria-label="Close modal"
        :disabled="persistent"
        @click="requestClose"
      >
        <v-icon size="16">mdi-close</v-icon>
      </button>
    </div>

    <div class="modal-body">
      <slot></slot>
    </div>

    <div v-if="$slots.footer" class="modal-footer">
      <slot name="footer"></slot>
    </div>
  </dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, useId } from 'vue';

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    title: string;
    maxWidth?: string;
    persistent?: boolean;
    hideClose?: boolean;
  }>(),
  {
    maxWidth: '480px',
    persistent: false,
    hideClose: false,
  }
);

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
}>();

const dialogRef = ref<HTMLDialogElement | null>(null);
const titleId = useId();
let pressedOutside = false;

const dialogStyle = computed(() => ({
  maxInlineSize: props.maxWidth,
}));

const syncOpenState = (open: boolean) => {
  const el = dialogRef.value;
  if (!el) return;
  if (open && !el.open) {
    el.showModal();
  } else if (!open && el.open) {
    el.close();
  }
};

watch(() => props.modelValue, (open) => {
  syncOpenState(open);
});

onMounted(() => {
  if (props.modelValue) {
    syncOpenState(true);
  }
});

const requestClose = () => {
  if (props.persistent) return;
  emit('update:modelValue', false);
};

const onNativeClose = () => {
  if (props.modelValue) {
    emit('update:modelValue', false);
  }
};

const onNativeCancel = (e: Event) => {
  if (props.persistent) {
    e.preventDefault();
  }
};

const isOutside = (e: PointerEvent, dialog: HTMLDialogElement) => {
  const r = dialog.getBoundingClientRect();
  return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
};

const onPointerDown = (e: PointerEvent) => {
  if (props.persistent || !dialogRef.value) {
    pressedOutside = false;
    return;
  }
  pressedOutside = isOutside(e, dialogRef.value);
};

const onPointerUp = (e: PointerEvent) => {
  if (props.persistent || !dialogRef.value) return;
  if (pressedOutside && isOutside(e, dialogRef.value)) {
    emit('update:modelValue', false);
  }
  pressedOutside = false;
};
</script>
