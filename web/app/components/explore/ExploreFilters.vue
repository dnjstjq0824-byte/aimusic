<template>
  <div class="space-y-4">
    <UiSelect :model-value="model.mood" :label="t('explore.moodLabel')" @change="set('mood', $event)">
      <option value="">{{ t('explore.moodAll') }}</option>
      <option v-for="m in MOODS" :key="m" :value="m">{{ tagName(m) }}</option>
    </UiSelect>
    <UiSelect :model-value="model.genre" :label="t('explore.genreLabel')" @change="set('genre', $event)">
      <option value="">{{ t('explore.genreAll') }}</option>
      <option v-for="g in GENRES" :key="g" :value="g">{{ tagName(g) }}</option>
    </UiSelect>
    <UiSelect :model-value="model.provider" :label="t('explore.providerLabel')" @change="set('provider', $event)">
      <option value="">{{ t('explore.providerAll') }}</option>
      <option v-for="p in PROVIDERS" :key="p" :value="p">{{ providerDisplayName(p) }}</option>
    </UiSelect>
    <div>
      <span class="mb-1.5 block text-xs font-semibold text-text-secondary">{{ t('explore.certLabel') }}</span>
      <div class="flex gap-2">
        <button
          type="button"
          class="flex-1 rounded-lg border px-2 py-1.5 text-xs font-semibold transition-colors"
          :class="!model.certified ? 'border-brand-500 bg-brand-500/15 text-brand-300' : 'border-surface-border text-text-secondary'"
          @click="setCert(false)"
        >
          {{ t('explore.certAny') }}
        </button>
        <button
          type="button"
          class="flex-1 rounded-lg border px-2 py-1.5 text-xs font-semibold transition-colors"
          :class="model.certified ? 'border-brand-500 bg-brand-500/15 text-brand-300' : 'border-surface-border text-text-secondary'"
          @click="setCert(true)"
        >
          {{ t('explore.certOnly') }}
        </button>
      </div>
    </div>
    <div>
      <span class="mb-1.5 block text-xs font-semibold text-text-secondary">{{ t('explore.bpmLabel') }}</span>
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="opt in bpmOpts"
          :key="opt.id"
          type="button"
          class="rounded-full px-2.5 py-1 text-xs font-semibold transition-colors"
          :class="model.bpm === opt.id ? 'bg-brand-500 text-on-brand' : 'bg-surface-2 text-text-secondary hover:text-white'"
          @click="set('bpm', opt.id)"
        >
          {{ t(opt.labelKey) }}
        </button>
      </div>
    </div>
    <div>
      <span class="mb-1.5 block text-xs font-semibold text-text-secondary">{{ t('explore.durLabel') }}</span>
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="opt in durOpts"
          :key="opt.id"
          type="button"
          class="rounded-full px-2.5 py-1 text-xs font-semibold transition-colors"
          :class="model.dur === opt.id ? 'bg-brand-500 text-on-brand' : 'bg-surface-2 text-text-secondary hover:text-white'"
          @click="set('dur', opt.id)"
        >
          {{ t(opt.labelKey) }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import UiSelect from '@/components/ui/UiSelect.vue';
import { providerDisplayName } from '@/utils/providerDisplay';
import { GENRES, tagDisplayName } from '@/utils/tags';

export interface ExploreFilterModel {
  mood: string;
  genre: string;
  provider: string;
  certified: boolean;
  bpm: '' | 'slow' | 'mid' | 'fast';
  dur: '' | 'short' | 'mid' | 'long';
}

defineProps<{ model: ExploreFilterModel }>();
const emit = defineEmits<{
  (e: 'update', patch: Partial<ExploreFilterModel>): void;
}>();

const { t, te } = useI18n();
const tagName = (key: string) => tagDisplayName(key, t, te);

const MOODS = ['focus', 'chill', 'cafe', 'workout', 'drive', 'sleep', 'cinematic', 'cyberpunk'];
const PROVIDERS = ['SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'ELEVENLABS', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'];

// NOTE: option labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const bpmOpts = [
  { id: '', labelKey: 'explore.bpmAny' },
  { id: 'slow', labelKey: 'explore.bpmSlow' },
  { id: 'mid', labelKey: 'explore.bpmMid' },
  { id: 'fast', labelKey: 'explore.bpmFast' },
] as const;
const durOpts = [
  { id: '', labelKey: 'explore.durAny' },
  { id: 'short', labelKey: 'explore.durShort' },
  { id: 'mid', labelKey: 'explore.durMid' },
  { id: 'long', labelKey: 'explore.durLong' },
] as const;

function set(key: 'mood' | 'genre' | 'provider', value: string): void;
function set(key: 'bpm', value: '' | 'slow' | 'mid' | 'fast'): void;
function set(key: 'dur', value: '' | 'short' | 'mid' | 'long'): void;
function set(key: string, value: string): void {
  emit('update', { [key]: value } as Partial<ExploreFilterModel>);
}

function setCert(value: boolean): void {
  emit('update', { certified: value });
}
</script>
