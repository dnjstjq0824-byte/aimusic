<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">My Playlists</h1>
    <form class="mb-4 flex gap-2" @submit.prevent="createPlaylist">
      <input v-model="newTitle" required placeholder="새 플레이리스트 제목" class="flex-1 rounded bg-neutral-800 p-2" />
      <label class="flex items-center gap-1 text-sm"><input v-model="newPublic" type="checkbox" /> public</label>
      <button class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold">Create</button>
    </form>
    <p v-if="store.error" class="mb-2 text-sm text-red-400">{{ store.error }}</p>
    <div class="grid gap-4 md:grid-cols-2">
      <div>
        <ul class="space-y-2">
          <li v-for="p in store.playlists" :key="p.id">
            <button
              class="w-full rounded p-3 text-left"
              :class="selected === p.id ? 'bg-brand-600/30' : 'bg-neutral-900'"
              @click="select(p.id)"
            >
              <div class="font-semibold">{{ p.title }}</div>
              <div class="text-xs text-neutral-400">{{ p.isPublic ? 'public' : 'private' }}</div>
            </button>
          </li>
        </ul>
        <p v-if="store.playlists.length === 0" class="text-sm text-neutral-500">플레이리스트가 없습니다.</p>
      </div>
      <div v-if="selected">
        <div class="mb-2 flex items-center justify-between">
          <h2 class="font-bold">{{ selectedTitle }}</h2>
          <div class="flex gap-2 text-xs">
            <button class="rounded bg-neutral-800 px-2 py-1" @click="playSelected">Play all</button>
            <button class="rounded bg-red-800 px-2 py-1" @click="removeSelected">Delete</button>
          </div>
        </div>
        <ul class="space-y-1">
          <li v-for="(t, i) in detailTracks" :key="t.id" class="flex items-center gap-2 rounded bg-neutral-900 px-2 py-1.5 text-sm">
            <span class="w-6 text-neutral-500">{{ i + 1 }}</span>
            <span class="min-w-0 flex-1 truncate">{{ t.title }}</span>
            <button class="text-xs text-neutral-400" @click="move(i, -1)">▲</button>
            <button class="text-xs text-neutral-400" @click="move(i, 1)">▼</button>
            <button class="text-xs text-red-400" @click="removeTrack(t.id)">✕</button>
          </li>
        </ul>
        <p v-if="detailTracks.length === 0" class="text-sm text-neutral-500">트랙이 없습니다. Home에서 추가해 주세요.</p>
        <div class="mt-2 flex gap-2">
          <input v-model="renameTitle" placeholder="새 제목" class="flex-1 rounded bg-neutral-800 p-1.5 text-sm" />
          <button class="rounded bg-neutral-700 px-2 py-1 text-sm" @click="rename">Rename</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { playlistApi } from '@/api/endpoints';
import { usePlaylistStore } from '@/stores/playlistStore';
import { usePlayerStore } from '@/stores/playerStore';
import { errMessageOf } from '@/api/client';

const store = usePlaylistStore();
const player = usePlayerStore();
const route = useRoute();
const router = useRouter();

const newTitle = ref('');
const newPublic = ref(false);
const selected = ref<string>(typeof route.params.id === 'string' ? route.params.id : '');
const renameTitle = ref('');
const detailError = ref<string | null>(null);

const detailTracks = computed(() => (selected.value ? (store.entryTracks[selected.value] ?? []) : []));
const selectedTitle = computed(() => store.playlists.find((p) => p.id === selected.value)?.title ?? '');

async function createPlaylist(): Promise<void> {
  const p = await store.create(newTitle.value, newPublic.value);
  newTitle.value = '';
  newPublic.value = false;
  await router.push(`/playlists/${p.id}`);
  selected.value = p.id;
}

async function select(id: string): Promise<void> {
  selected.value = id;
  await router.push(`/playlists/${id}`);
  await store.fetchTracks(id);
}

function playSelected(): void {
  if (detailTracks.value.length > 0) player.setQueue(detailTracks.value);
}

async function removeSelected(): Promise<void> {
  if (!selected.value) return;
  await store.remove(selected.value);
  selected.value = '';
  await router.push('/playlists/_');
}

async function removeTrack(trackId: string): Promise<void> {
  if (!selected.value) return;
  await store.removeTrack(selected.value, trackId);
}

async function move(index: number, delta: number): Promise<void> {
  if (!selected.value) return;
  const ids = detailTracks.value.map((t) => t.id);
  const j = index + delta;
  if (j < 0 || j >= ids.length) return;
  const tmp = ids[index] as string;
  ids[index] = ids[j] as string;
  ids[j] = tmp;
  await store.reorder(selected.value, ids);
}

async function rename(): Promise<void> {
  if (!selected.value || !renameTitle.value.trim()) return;
  try {
    const current = store.playlists.find((p) => p.id === selected.value);
    await playlistApi.update(selected.value, renameTitle.value.trim(), current?.isPublic ?? false);
    renameTitle.value = '';
    await store.fetchMine();
  } catch (err: unknown) {
    detailError.value = errMessageOf(err);
  }
}

onMounted(async () => {
  await store.fetchMine();
  if (selected.value && selected.value !== '_') {
    await store.fetchTracks(selected.value);
  } else if (store.playlists.length > 0) {
    selected.value = store.playlists[0]?.id ?? '';
    if (selected.value) await store.fetchTracks(selected.value);
  }
});

watch(
  () => route.params.id,
  (id) => {
    if (typeof id === 'string' && id !== '_' && id !== selected.value) {
      selected.value = id;
      void store.fetchTracks(id);
    }
  },
);
</script>
