<template>
  <div
    class="grid shrink-0 grid-cols-2 gap-0.5 overflow-hidden rounded-xl border border-surface-border"
    :class="boxClass"
    role="img"
    :aria-label="`${title} 커버 모자이크`"
  >
    <UiCoverArt
      v-for="(seed, i) in cells"
      :key="`${seed}-${i}`"
      :seed="seed"
      :title="title"
      size="xs"
      rounded="md"
      class="!h-full !w-full !rounded-none !border-0"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';

// 2x2 mosaic of generative covers (playlist cards, UI §3.4).
const props = withDefaults(
  defineProps<{
    seeds: string[];
    title?: string;
    size?: 'sm' | 'md' | 'lg';
  }>(),
  { title: '플레이리스트', size: 'md' },
);

const cells = computed(() => {
  const out = [...props.seeds].slice(0, 4);
  while (out.length < 4) out.push(`${props.title}-fill-${out.length}`);
  return out;
});

const boxClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'h-20 w-20';
    case 'lg': return 'h-52 w-52';
    default: return 'h-40 w-40';
  }
});
</script>
