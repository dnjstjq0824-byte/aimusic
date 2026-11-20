<template>
  <section :aria-labelledby="headingId">
    <h2 :id="headingId" class="mb-5 text-xl font-medium text-white sm:text-2xl">{{ t('home.startTitle') }}</h2>
    <div class="scrollbar-none -mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2 sm:mx-0 sm:gap-4 sm:px-0 lg:grid lg:grid-cols-6 lg:overflow-visible lg:pb-0">
      <RouterLink
        v-for="s in STARTERS"
        :key="s.key"
        :to="localePath({ path: '/explore', query: { mood: s.mood } })"
        class="group relative aspect-[4/5] w-52 shrink-0 snap-start overflow-hidden rounded-2xl bg-surface-2 sm:w-60 lg:w-auto"
        :class="compact ? 'lg:aspect-[4/3]' : ''"
      >
        <img
          :src="s.img"
          alt=""
          loading="lazy"
          decoding="async"
          class="absolute inset-0 h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
        >
        <div class="absolute inset-0 bg-gradient-to-t from-black/75 via-black/10 to-transparent" aria-hidden="true" />
        <span class="absolute inset-x-4 bottom-4 text-sm font-medium leading-snug text-white sm:text-base">{{ t(s.key) }}</span>
      </RouterLink>
    </div>
  </section>
</template>

<script lang="ts">
// "Get started" photo cards -> explore by mood (licensed stock, see
// public/images/stock/CREDITS.md). Shared by home and explore (v3 flow).
export const STARTERS = [
  { key: 'home.startStore', mood: 'cafe', img: '/images/stock/start-store.webp' },
  { key: 'home.startVideo', mood: 'cinematic', img: '/images/stock/start-video.webp' },
  { key: 'home.startFocus', mood: 'focus', img: '/images/stock/start-focus.webp' },
  { key: 'home.startWorkout', mood: 'workout', img: '/images/stock/start-workout.webp' },
  { key: 'home.startDrive', mood: 'drive', img: '/images/stock/start-drive.webp' },
  { key: 'home.startRest', mood: 'sleep', img: '/images/stock/start-rest.webp' },
];
</script>

<script setup lang="ts">
withDefaults(defineProps<{ headingId?: string; compact?: boolean }>(), {
  headingId: 'start-h',
  compact: false,
});

const { t } = useI18n();
const localePath = useLocalePath();
</script>
