<template>
  <section :aria-labelledby="headingId" class="space-y-3">
    <h3 :id="headingId" class="text-sm font-bold text-white">
      {{ t('track.comments.title') }}
      <span v-if="total > 0" class="ml-1 font-mono text-xs text-text-muted">{{ fmt.number(total) }}</span>
    </h3>

    <ClientOnly>
      <form v-if="auth.isAuthenticated" class="space-y-2" @submit.prevent="submit">
        <UiTextarea
          v-model="draft"
          :label="t('track.comments.label')"
          :placeholder="t('track.comments.placeholder')"
          :maxlength="MAX"
          :rows="2"
        />
        <div class="flex items-center justify-between gap-2">
          <span class="font-mono text-caption text-text-muted">{{ draft.length }}/{{ MAX }}</span>
          <UiButton variant="brand" size="sm" type="submit" :loading="posting" :disabled="draft.trim() === ''">
            {{ t('track.comments.submit') }}
          </UiButton>
        </div>
      </form>
      <p v-else class="rounded-xl border border-surface-border bg-surface-1 p-3 text-xs text-text-secondary">
        {{ t('track.comments.loginHint') }}
        <RouterLink
          :to="localePath({ name: 'login', query: { redirect: route.fullPath } })"
          class="ml-1 font-bold text-brand-400 hover:underline"
        >
          {{ t('track.comments.loginCta') }}
        </RouterLink>
      </p>
    </ClientOnly>

    <p v-if="error" class="text-xs text-accent-crimson" role="alert">
      {{ error }}
      <button type="button" class="ml-2 font-bold underline" @click="load(false)">{{ t('track.retry') }}</button>
    </p>

    <div v-if="loading && items.length === 0" class="space-y-2">
      <UiSkeleton v-for="i in 2" :key="i" height="h-12" width="w-full" rounded="lg" />
    </div>
    <p v-else-if="!error && items.length === 0" class="py-2 text-xs text-text-muted">{{ t('track.comments.empty') }}</p>

    <ul v-if="items.length > 0" class="space-y-2">
      <li v-for="c in items" :key="c.id" class="rounded-xl border border-surface-border bg-surface-1 px-3 py-2">
        <div class="flex items-center justify-between gap-2">
          <span class="truncate text-xs font-bold text-white">{{ c.authorName }}</span>
          <span class="flex shrink-0 items-center gap-2">
            <time :datetime="c.createdAt" class="text-caption text-text-muted">{{ fmt.date(c.createdAt) }}</time>
            <button
              v-if="canDelete(c)"
              type="button"
              class="text-caption text-text-muted hover:text-accent-crimson hover:underline"
              :disabled="deletingId === c.id"
              @click="remove(c)"
            >
              {{ t('track.comments.delete') }}
            </button>
          </span>
        </div>
        <p class="mt-1 whitespace-pre-line break-words text-xs leading-relaxed text-text-primary">{{ c.body }}</p>
      </li>
    </ul>

    <UiButton v-if="nextBefore" variant="ghost" size="sm" :loading="loading" @click="load(true)">
      {{ t('track.comments.more') }}
    </UiButton>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref, useId, watch } from 'vue';
import { useRoute } from 'vue-router';
import { commentApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import type { TrackComment } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';

const props = defineProps<{ trackId: string }>();

const MAX = 1000;
const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();
const headingId = `comments-${useId()}`;

const items = ref<TrackComment[]>([]);
const total = ref(0);
const nextBefore = ref<string | undefined>(undefined);
const loading = ref(false);
const error = ref<string | null>(null);
const draft = ref('');
const posting = ref(false);
const deletingId = ref<string | null>(null);

function canDelete(c: TrackComment): boolean {
  return auth.isAuthenticated && (c.userId === auth.userId || auth.isAdmin);
}

async function load(more: boolean): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const page = await commentApi.list(props.trackId, more ? nextBefore.value : undefined);
    items.value = more ? [...items.value, ...(page.items ?? [])] : (page.items ?? []);
    total.value = page.total;
    nextBefore.value = page.nextBefore || undefined;
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

async function submit(): Promise<void> {
  const body = draft.value.trim();
  if (!body) return;
  posting.value = true;
  try {
    const c = await commentApi.create(props.trackId, body);
    items.value = [c, ...items.value];
    total.value += 1;
    draft.value = '';
    toast.success(t('track.comments.posted'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    posting.value = false;
  }
}

async function remove(c: TrackComment): Promise<void> {
  deletingId.value = c.id;
  try {
    await commentApi.remove(props.trackId, c.id);
    items.value = items.value.filter((x) => x.id !== c.id);
    total.value = Math.max(0, total.value - 1);
    toast.info(t('track.comments.deleted'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    deletingId.value = null;
  }
}

// Client-side only: comments are personal-ish, change often, and must not
// bloat the cached SSR page.
onMounted(() => void load(false));
watch(
  () => props.trackId,
  () => {
    items.value = [];
    void load(false);
  },
);
</script>
