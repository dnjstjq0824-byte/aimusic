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
        {{ t('playlist.meCount', { count: sorted.length }) }}
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
        <RouterLink :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })" class="block">
          <CoverMosaic :seeds="pl.coverSeeds ?? []" :title="pl.title" size="md" class="!h-auto !w-full aspect-square" />
          <span class="mt-2 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
          <span class="mt-0.5 block text-caption text-text-muted">
            {{ pl.isPublic ? t('playlist.detailPublic') : t('playlist.detailPrivate') }} · {{ statLine(pl) }}
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
        <CoverMosaic :seeds="pl.coverSeeds ?? []" :title="pl.title" size="xs" />
        <RouterLink
          :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })"
          class="min-w-0 flex-1 truncate text-sm font-bold text-white hover:text-brand-400"
        >
          {{ pl.title }}
        </RouterLink>
        <span class="hidden shrink-0 text-caption text-text-muted sm:inline">{{ statLine(pl) }}</span>
        <span class="shrink-0 text-caption text-text-muted">
          {{ pl.isPublic ? t('playlist.detailPublic') : t('playlist.detailPrivate') }}
        </span>
        <UiMenu :items="cardMenu(pl)" :label="t('track.moreActions')" @select="(i) => onCardMenu(pl, i)" />
      </li>
    </ul>

    <UiModal v-model="createOpen" :title="t('playlist.createTitle')" max-width="sm">
      <div class="space-y-4 p-5">
        <UiInput v-model="createTitle" :label="t('playlist.createName')" :placeholder="t('playlist.createNamePh')" :maxlength="100" required />
        <UiTextarea
          v-model="createDescription"
          :label="t('playlist.editDescription')"
          :placeholder="t('playlist.editDescriptionPh')"
          :maxlength="500"
          :rows="3"
        />
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

    <UiModal v-model="renameOpen" :title="t('playlist.editInfo')" max-width="sm">
      <div class="space-y-4 p-5">
        <UiInput v-model="renameTitle" :label="t('playlist.editTitle')" :maxlength="100" required />
        <UiTextarea
          v-model="renameDescription"
          :label="t('playlist.editDescription')"
          :placeholder="t('playlist.editDescriptionPh')"
          :maxlength="500"
          :rows="3"
        />
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
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useFormat } from '@/composables/useFormat';
import { detailParam } from '@/utils/seo';
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
import UiTextarea from '@/components/ui/UiTextarea.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();
const store = usePlaylistStore();

// The store is the single copy: picker adds, detail edits and deletes made
// elsewhere show up here without a reload.
const items = computed<Playlist[]>(() => store.playlists);
const loading = ref(true);
const busy = ref(false);
const error = ref<string | null>(null);
const sortBy = ref<'recent' | 'title'>('recent');
const view = ref<'grid' | 'list'>('grid');

const createOpen = ref(false);
const createTitle = ref('');
const createPublic = ref(false);
const createDescription = ref('');
const renameOpen = ref(false);
const renameTarget = ref<Playlist | null>(null);
const renameTitle = ref('');
const renamePublic = ref(false);
const renameDescription = ref('');
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

function statLine(pl: Playlist): string {
  const count = fmt.number(pl.trackCount ?? 0);
  const mins = Math.floor((pl.totalDurationMs ?? 0) / 60000);
  if (mins >= 60) {
    return t('playlist.metaHM', { count, h: fmt.number(Math.floor(mins / 60)), m: fmt.number(mins % 60) });
  }
  return t('playlist.metaM', { count, m: fmt.number(mins) });
}

function cardMenu(pl: Playlist): MenuItem[] {
  return [
    { label: t('playlist.editInfo') },
    { label: pl.isPublic ? t('playlist.meMakePrivate') : t('playlist.meMakePublic') },
    { label: t('playlist.meDelete'), danger: true },
  ];
}

function onCardMenu(pl: Playlist, index: number): void {
  if (index === 0) {
    renameTarget.value = pl;
    renameTitle.value = pl.title;
    renamePublic.value = pl.isPublic;
    renameDescription.value = pl.description ?? '';
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
  await store.fetchMine();
  error.value = store.error ? t('playlist.errTitle') : null;
  loading.value = false;
}

async function submitCreate(): Promise<void> {
  const title = createTitle.value.trim();
  if (!title) return;
  busy.value = true;
  try {
    await store.create(title, createPublic.value, createDescription.value.trim() || undefined);
    toast.success(t('playlist.toastCreated'));
    createOpen.value = false;
    createTitle.value = '';
    createDescription.value = '';
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
    await store.update(renameTarget.value.id, {
      title,
      isPublic: renamePublic.value,
      description: renameDescription.value.trim(),
    });
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
    await store.update(pl.id, { title: pl.title, isPublic: !pl.isPublic });
    toast.success(t('playlist.toastUpdated'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function submitDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  busy.value = true;
  try {
    await store.remove(deleteTarget.value.id);
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
