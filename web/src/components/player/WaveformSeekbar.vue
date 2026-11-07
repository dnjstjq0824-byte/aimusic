<template>
  <div
    class="h-10 w-full cursor-pointer rounded bg-surface-2"
    role="slider"
    :aria-valuenow="Math.round(positionMs)"
    :aria-valuemin="0"
    :aria-valuemax="Math.round(durationMs)"
    :aria-valuetext="`${fmt(positionMs)} / ${fmt(durationMs)}`"
    aria-label="재생 위치"
    tabindex="0"
    @click="onClick"
    @keydown.left.prevent="nudge(-5000)"
    @keydown.right.prevent="nudge(5000)"
  >
    <svg viewBox="0 0 100 10" preserveAspectRatio="none" class="h-full w-full">
      <rect
        v-for="(h, i) in bars"
        :key="i"
        :x="(i * 100) / bars.length"
        :y="10 - h"
        :width="100 / bars.length - 0.4"
        :height="h"
        :fill="(i / bars.length) * 100 <= progressPct ? 'var(--color-brand)' : 'var(--color-surface-3)'"
      />
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ positionMs: number; durationMs: number; seed?: string }>();
const emit = defineEmits<{ seek: [ms: number] }>();

// Deterministic pseudo-waveform derived from the track id seed.
const bars = computed<number[]>(() => {
  const seed = props.seed ?? 'wave';
  let h = 2166136261;
  for (const ch of seed) {
    h ^= ch.codePointAt(0) ?? 0;
    h = Math.imul(h, 16777619);
  }
  const out: number[] = [];
  let x = h >>> 0;
  for (let i = 0; i < 64; i++) {
    x = (Math.imul(x, 1664525) + 1013904223) >>> 0;
    const wave = 0.5 + 0.5 * Math.sin(i / 5);
    out.push(2 + ((x % 70) / 100) * 6 * wave + 2 * wave);
  }
  return out;
});

const progressPct = computed(() =>
  props.durationMs > 0 ? Math.min(100, (props.positionMs / props.durationMs) * 100) : 0,
);

function onClick(e: MouseEvent): void {
  const el = e.currentTarget as HTMLElement;
  const rect = el.getBoundingClientRect();
  const ratio = (e.clientX - rect.left) / rect.width;
  emit('seek', Math.max(0, Math.min(props.durationMs, ratio * props.durationMs)));
}

function nudge(delta: number): void {
  emit('seek', Math.max(0, Math.min(props.durationMs, props.positionMs + delta)));
}

function fmt(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000));
  return `${Math.floor(s / 60)}분 ${String(s % 60).padStart(2, '0')}초`;
}
</script>
