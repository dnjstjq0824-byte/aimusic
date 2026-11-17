<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('playlist.meTitle')">
      <template #actions>
        <UiButton variant="brand" size="sm" @click="createOpen = true">
          {{ t('playlist.meNew') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('playlist.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-xs text-text-secondary" role="status">
        {{ t('playlist.metaTracks', { count: sorted.length }) }}
      </p>
      <div class="flex items-center gap-2">
        <UiSelect v-model="sortBy" :label="t('playlist.meSort')" class="w-36">
          <option value="recent">{{ t('playlist.meSortRecent') }}</option>
          <option value="title">{{ t('playlist.meSortTitle') }}</option>
        </UiSelect>
        <div class="flex gap-1 rounded-lg border border-surface-border bg-surface-1 p-1" role="group" :aria-label="t('playlist.meGrid')">
          <button
            type="button"
            class="whitespace-nowrap rounded-md px-2.5 py-1 text-xs font-bold transition-colors"
            :class="view === 'grid' ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
            :aria-pressed="view === 'grid'"
            @click="view = 'grid'"
          >
            {{ t('playlist.meGrid') }}
          </button>
          <button
            type="button"
            class="whitespace-nowrap rounded-md px-2.5 py-1 text-xs font-bold transition-colors"
            :class="view === 'list' ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
            :aria-pressed="view === 'list'"
            @click="view = 'list'"
          >
            {{ t('playlist.meList') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <UiSkeleton v-for="i in 4" :key="i" height="h-40" width="w-full" rounded="xl" />
    </div>

    <div v-else-if="!error && sorted.length === 0" class="py-10">
      <UiEmptyState :title="t('playlist.meEmptyT')" :description="t('playlist.meEmptyD')">
        <template #action>
          <UiButton variant="brand" size="sm" @click="createOpen = true">{{ t('playlist.meEmptyCta') }}</UiButton>
        </template>
      </UiEmptyState>
    </div>

    <div v-else-if="view === 'grid'" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard v-for="pl in sorted" :key="pl.id" class="group relative">
        <RouterLink :to="localePath({ name: 'playlist-detail', params: { id: pl.id } })" class="block">
          <CoverMosaic :seeds="[pl.id]" :title="pl.title" size="md" />
          <span class="mt-2 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
          <span class="mt-0.5 block text-caption text-text-muted">
            {{ pl.isPublic ? t('playlist.detailPublic') : t('playlist.detailPrivate') }}
          </span>
        </RouterLink>
        <div class="absolute right-2 top-2">
          <UiMenu :items="cardMenu(pl)" :label="t('track.moreActions')" @select="(i) => onCardMenu(pl, i)" />
        </div>
      </UiCard>
    </div>

    <ul v-else class="space-y-1">
      <li
        v-for="pl in sorted"
        :key="pl.id"
        class="flex items-center gap-3 rounded-xl border border-transparent px-2 py-2 transition-colors hover:border-surface-border hover:bg-surface-1"
      >
        <RouterLink
          :to="localePath({ name: 'playlist-detail', params: { id: pl.id } })"
          class="min-w-0 flex-1 truncate text-sm font-bold text-white hover:text-brand-400"
        >
          {{ pl.title }}
        </RouterLink>
        <span class="shrink-0 text-caption text-text-muted">
          {{ pl.isPublic ? t('playlist.detailPublic') : t('playlist.detailPrivate') }}
        </span>
        <UiMenu :items="cardMenu(pl)" :label="t('track.moreActions')" @select="(i) => onCardMenu(pl, i)" />
      </li>
    </ul>

    <UiModal v-model="createOpen" :title="t('playlist.createTitle')" max-width="sm">
      <div class="space-y-4 p-5">
        <UiInput v-model="createTitle" :label="t('playlist.createName')" :placeholder="t('playlist.createNamePh')" required />
        <UiCheckbox v-model="createPublic">{{ t('playlist.createPublic') }}</UiCheckbox>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="md" class="flex-1" @click="createOpen = false">
            {{ t('playlist.createCancel') }}
          </UiButton>
          <UiButton variant="brand" size="md" class="flex-1" :loading="busy" :disabled="!createTitle.trim()" @click="submitCreate">
            {{ t('playlist.createSubmit') }}
          </UiButton>
        </div>
      </div>
    </UiModal>

    <UiModal v-model="renameOpen" :title="t('playlist.meRename')" max-width="sm">
      <div class="space-y-4 p-5">
        <UiInput v-model="renameTitle" :label="t('playlist.editTitle')" required />
        <UiCheckbox v-model="renamePublic">{{ t('playlist.editVisibility') }}</UiCheckbox>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="md" class="flex-1" @click="renameOpen = false">
            {{ t('playlist.editCancel') }}
          </UiButton>
          <UiButton variant="brand" size="md" class="flex-1" :loading="busy" :disabled="!renameTitle.trim()" @click="submitRename">
            {{ t('playlist.editSave') }}
          </UiButton>
        </div>
      </div>
    </UiModal>

    <UiConfirmDialog
      v-model="deleteOpen"
      :title="t('playlist.delTitle')"
      :body="t('playlist.delBody')"
      :confirm-label="t('playlist.delConfirm')"
      danger
      :loading="busy"
      @confirm="submitDelete"
    />
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { playlistApi } from '@/api/endpoints';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { Playlist } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiMenu, { type MenuItem } from '@/components/ui/UiMenu.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const toast = useToast();
const apiError = useApiError();

const items = ref<Playlist[]>([]);
const loading = ref(true);
const busy = ref(false);
const error = ref<string | null>(null);
const sortBy = ref<'recent' | 'title'>('recent');
const view = ref<'grid' | 'list'>('grid');

const createOpen = ref(false);
const createTitle = ref('');
const createPublic = ref(false);
const renameOpen = ref(false);
const renameTarget = ref<Playlist | null>(null);
const renameTitle = ref('');
const renamePublic = ref(false);
const deleteOpen = ref(false);
const deleteTarget = ref<Playlist | null>(null);

const sorted = computed(() => {
  const list = [...items.value];
  if (sortBy.value === 'title') {
    list.sort((a, b) => a.title.localeCompare(b.title));
  } else {
    list.sort((a, b) => +new Date(b.updatedAt) - +new Date(a.updatedAt));
  }
  return list;
});

function cardMenu(pl: Playlist): MenuItem[] {
  return [
    { label: t('playlist.meRename') },
    { label: pl.isPublic ? t('playlist.meMakePrivate') : t('playlist.meMakePublic') },
    { label: t('playlist.meDelete'), danger: true },
  ];
}

function onCardMenu(pl: Playlist, index: number): void {
  if (index === 0) {
    renameTarget.value = pl;
    renameTitle.value = pl.title;
    renamePublic.value = pl.isPublic;
    renameOpen.value = true;
  } else if (index === 1) {
    void toggleVisibility(pl);
  } else {
    deleteTarget.value = pl;
    deleteOpen.value = true;
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    items.value = (await playlistApi.listMine()) ?? [];
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

async function submitCreate(): Promise<void> {
  const title = createTitle.value.trim();
  if (!title) return;
  busy.value = true;
  try {
    const created = await playlistApi.create(title, createPublic.value);
    items.value.unshift(created);
    toast.success(t('playlist.toastCreated'));
    createOpen.value = false;
    createTitle.value = '';
    createPublic.value = false;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}

async function submitRename(): Promise<void> {
  if (!renameTarget.value) return;
  const title = renameTitle.value.trim();
  if (!title) return;
  busy.value = true;
  try {
    const updated = await playlistApi.update(renameTarget.value.id, title, renamePublic.value);
    const idx = items.value.findIndex((p) => p.id === updated.id);
    if (idx >= 0) items.value[idx] = updated;
    toast.success(t('playlist.toastUpdated'));
    renameOpen.value = false;
    renameTarget.value = null;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}

async function toggleVisibility(pl: Playlist): Promise<void> {
  try {
    const updated = await playlistApi.update(pl.id, pl.title, !pl.isPublic);
    const idx = items.value.findIndex((p) => p.id === updated.id);
    if (idx >= 0) items.value[idx] = updated;
    toast.success(t('playlist.toastUpdated'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function submitDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  busy.value = true;
  try {
    await playlistApi.remove(deleteTarget.value.id);
    items.value = items.value.filter((p) => p.id !== deleteTarget.value?.id);
    toast.success(t('playlist.toastDeleted'));
    deleteOpen.value = false;
    deleteTarget.value = null;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
