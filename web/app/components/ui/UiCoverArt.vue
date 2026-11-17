<template>
  <div
    class="relative shrink-0 overflow-hidden bg-surface-2"
    :class="[sizeClass, roundedClass]"
    role="img"
    :aria-label="t('player.coverArt', { title })"
  >
    <!-- v3 flow: photo-led covers. Tracks have no artwork yet, so each seed
         maps deterministically onto a licensed abstract photo (see
         public/images/stock/CREDITS.md) with a seed-based tint. -->
    <img
      :src="src"
      alt=""
      :loading="eager ? 'eager' : 'lazy'"
      decoding="async"
      class="absolute inset-0 h-full w-full object-cover"
      :style="{ filter: `hue-rotate(${hueShift}deg) saturate(1.1)` }"
    >
    <div class="absolute inset-0 bg-gradient-to-t from-black/35 via-transparent to-transparent" aria-hidden="true" />
    <span
      v-if="verified"
      class="absolute flex items-center justify-center rounded-full bg-verified text-surface-0 shadow"
      :class="badgeClass"
      :title="$t('player.verified')"
    >
      <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M20 6 9 17l-5-5" />
      </svg>
    </span>
  </div>
</template>

<script lang="ts">
// Licensed abstract cover pool (Unsplash License). Order is stable: a seed
// always lands on the same image, so covers never shuffle between visits.
export const COVER_POOL = [
  '/images/stock/cover-1.webp',
  '/images/stock/cover-2.webp',
  '/images/stock/cover-3.webp',
  '/images/stock/cover-4.webp',
  '/images/stock/cover-5.webp',
  '/images/stock/cover-6.webp',
  '/images/stock/cover-7.webp',
  '/images/stock/cover-8.webp',
];

export function seedHash(seed: string): number {
  let h = 2166136261;
  for (const ch of seed) {
    h ^= ch.codePointAt(0) ?? 0;
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

export function coverFor(seed: string): string {
  return COVER_POOL[seedHash(seed) % COVER_POOL.length] as string;
}
</script>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  seed: string;
  title?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'fill';
  rounded?: 'md' | 'lg' | 'xl';
  verified?: boolean;
  eager?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: 'AVIBE',
  size: 'md',
  rounded: 'lg',
  verified: false,
  eager: false,
});

const { t } = useI18n();

const sizeClass = computed(() => {
  switch (props.size) {
    case 'xs': return 'h-9 w-9';
    case 'sm': return 'h-11 w-11';
    case 'md': return 'h-12 w-12';
    case 'lg': return 'h-20 w-20';
    case 'xl': return 'h-40 w-40';
    case 'fill': return 'aspect-square w-full';
    default: return 'h-12 w-12';
  }
});

const roundedClass = computed(() => {
  switch (props.rounded) {
    case 'md': return 'rounded-lg';
    case 'lg': return 'rounded-xl';
    case 'xl': return 'rounded-2xl';
    default: return 'rounded-xl';
  }
});

const badgeClass = computed(() =>
  props.size === 'fill' || props.size === 'xl' ? 'bottom-2 right-2 h-5 w-5' : 'bottom-1 right-1 h-4 w-4',
);

const hash = computed(() => seedHash(props.seed));
const src = computed(() => COVER_POOL[hash.value % COVER_POOL.length] as string);
// Second-order variation so two seeds on the same photo still differ.
const hueShift = computed(() => ((hash.value >>> 8) % 7) * 18 - 54);
</script>
