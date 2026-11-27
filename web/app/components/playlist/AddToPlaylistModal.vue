<template>
  <UiModal
    :model-value="Boolean(track)"
    :title="t('playlist.picker.title')"
    :description="track ? t('playlist.picker.descWithTrack', { title: track.title }) : t('playlist.picker.desc')"
    @close="close"
  >
    <div v-if="loading" class="space-y-2" aria-busy="true">
      <UiSkeleton v-for="i in 3" :key="i" height="h-14" width="w-full" rounded="xl" />
    </div>

    <p v-else-if="loadError" class="rounded-xl border border-accent-crimson/40 bg-accent-crimson/10 p-3 text-xs text-accent-crimson" role="alert">
      {{ loadError }}
      <button type="button" class="ml-2 font-bold underline" @click="load">{{ t('track.retry') }}</button>
    </p>

    <div v-else class="space-y-2">
      <p v-if="list.length === 0 && !creating" class="py-2 text-center text-xs text-text-secondary">
        {{ t('playlist.picker.empty') }}
      </p>
      <ul v-if="list.length > 0" class="max-h-[50vh] space-y-2 overflow-y-auto pr-1">
        <li v-for="p in list" :key="p.id">
          <button
            type="button"
            class="flex w-full items-center gap-3 rounded-xl border border-surface-border bg-surface-2 p-2 text-left text-xs transition-colors enabled:hover:bg-surface-3 disabled:cursor-default"
            :disabled="p.containsTrack || busyId !== null"
            :aria-label="p.containsTrack ? t('playlist.picker.alreadyAria', { title: p.title }) : t('playlist.picker.addAria', { title: p.title })"
            @click="add(p)"
          >
            <CoverMosaic :seeds="p.coverSeeds ?? []" :title="p.title" size="xs" />
            <span class="min-w-0 flex-1">
              <span class="block truncate font-bold text-white">{{ p.title }}</span>
              <span class="mt-0.5 block text-caption text-text-muted">
                {{ p.isPublic ? t('playlist.picker.public') : t('playlist.picker.private') }}
                · {{ t('playlist.metaTracks', { count: fmt.number(p.trackCount ?? 0) }) }}
              </span>
            </span>
            <span v-if="busyId === p.id" class="shrink-0 text-text-secondary">…</span>
            <span v-else-if="p.containsTrack" class="shrink-0 font-semibold text-verified">✓ {{ t('playlist.picker.already') }}</span>
            <span v-else class="shrink-0 text-text-secondary">{{ t('playlist.picker.add') }}</span>
          </button>
        </li>
      </ul>

      <button
        v-if="!creating"
        type="button"
        class="flex w-full items-center justify-center gap-1.5 rounded-xl border border-dashed border-surface-border p-3 text-xs font-semibold text-text-secondary transition-colors hover:border-brand-500/50 hover:text-brand-300"
        @click="creating = true"
      >
        {{ list.length === 0 ? t('playlist.picker.createNew') : t('playlist.picker.createMore') }}
      </button>
      <form v-else class="space-y-2 rounded-xl border border-brand-500/40 bg-brand-950/20 p-2" @submit.prevent="createAndAdd">
        <UiInput
          v-model="newTitle"
          :placeholder="t('playlist.picker.namePh')"
          :maxlength="100"
          :label="t('playlist.createName')"
        />
        <UiCheckbox v-model="newPublic">{{ t('playlist.createPublic') }}</UiCheckbox>
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="creating = false">{{ t('playlist.createCancel') }}</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="createBusy" :disabled="newTitle.trim() === ''">
            {{ t('playlist.picker.createAndAdd') }}
          </UiButton>
        </div>
      </form>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useTrackActionsStore } from '@/stores/trackActionsStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import type { Playlist } from '@/api/types';
import UiModal from '@/components/ui/UiModal.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';

const { t } = useI18n();
const dialogs = useTrackActionsStore();
const playlists = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

const track = computed(() => dialogs.pickerTrack);
const list = computed<Playlist[]>(() => playlists.playlists);
const loading = ref(false);
const loadError = ref<string | null>(null);
const busyId = ref<string | null>(null);
const creating = ref(false);
const createBusy = ref(false);
const newTitle = ref('');
const newPublic = ref(false);

async function load(): Promise<void> {
  if (!track.value) return;
  loading.value = true;
  loadError.value = null;
  await playlists.fetchMine({ containsTrack: track.value.id });
  loadError.value = playlists.error ? t('playlist.errTitle') : null;
  loading.value = false;
}

watch(
  track,
  (tr) => {
    creating.value = false;
    newTitle.value = '';
    newPublic.value = false;
    if (tr) void load();
  },
  { immediate: true },
);

function close(): void {
  dialogs.closePicker();
}

async function add(p: Playlist): Promise<void> {
  const tr = track.value;
  if (!tr || p.containsTrack) return;
  busyId.value = p.id;
  try {
    const res = await playlists.addTrack(p.id, tr);
    if (res.duplicate) toast.info(t('playlist.toastDuplicate', { title: p.title }));
    else toast.success(t('playlist.toastAddedTo', { title: p.title }));
    close();
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busyId.value = null;
  }
}

async function createAndAdd(): Promise<void> {
  const tr = track.value;
  const title = newTitle.value.trim();
  if (!tr || !title) return;
  createBusy.value = true;
  try {
    const created = await playlists.create(title, newPublic.value);
    await playlists.addTrack(created.id, tr);
    toast.success(t('playlist.toastCreatedAdded', { title }));
    close();
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    createBusy.value = false;
  }
}
</script>
