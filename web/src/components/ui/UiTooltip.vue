<template>
  <span class="group/tooltip relative inline-flex" @mouseenter="open = true" @mouseleave="open = false">
    <span
      :aria-describedby="open ? tipId : undefined"
      class="inline-flex focus:outline-none"
      tabindex="0"
      @focus="open = true"
      @blur="open = false"
    >
      <slot />
    </span>
    <span
      v-if="open"
      :id="tipId"
      role="tooltip"
      class="pointer-events-none absolute z-50 whitespace-nowrap rounded-lg border border-surface-border bg-surface-2 px-2.5 py-1.5 text-xs text-neutral-100 shadow-2xl"
      :class="positionClass"
    >
      {{ text }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue';

interface Props {
  text: string;
  position?: 'top' | 'bottom' | 'left' | 'right';
}

const props = withDefaults(defineProps<Props>(), {
  position: 'top',
});

const open = ref(false);
const tipId = `tooltip-${useId()}`;

const positionClass = computed(() => {
  switch (props.position) {
    case 'bottom':
      return 'top-full left-1/2 -translate-x-1/2 mt-2';
    case 'left':
      return 'right-full top-1/2 -translate-y-1/2 mr-2';
    case 'right':
      return 'left-full top-1/2 -translate-y-1/2 ml-2';
    default:
      return 'bottom-full left-1/2 -translate-x-1/2 mb-2';
  }
});
</script>
