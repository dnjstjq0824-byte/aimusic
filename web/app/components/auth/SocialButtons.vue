<template>
  <div v-if="ordered.length > 0" class="space-y-2">
    <UiButton
      v-for="p in primary"
      :key="p.code"
      variant="secondary"
      size="md"
      full-width
      type="button"
      @click="start(p.slug)"
    >
      {{ t(p.labelKey) }}
    </UiButton>
    <div v-if="secondary.length > 0" class="grid grid-cols-3 gap-2">
      <UiButton
        v-for="p in secondary"
        :key="p.code"
        variant="secondary"
        size="md"
        type="button"
        @click="start(p.slug)"
      >
        {{ p.short }}
      </UiButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { API_BASE_URL } from '@/api/client';
import type { OAuthProviderInfo } from '@/api/types';
import { orderSocialProviders } from '@/utils/social';
import UiButton from '@/components/ui/UiButton.vue';

// Social buttons: engine-enabled providers only, locale order per UI §3.11
// (ordering in utils/social.ts + social.test.ts).
// NOTE: auth.social.* labels resolve via labelKey (dynamic t() calls,
// invisible to check-i18n usage scan — they are used, do not delete).
const props = defineProps<{ providers: OAuthProviderInfo[] }>();

const { t, locale } = useI18n();

const ordered = computed(() =>
  orderSocialProviders(
    props.providers.filter((p) => p.enabled).map((p) => p.provider),
    locale.value,
  ),
);

const primary = computed(() => ordered.value.slice(0, 2));
const secondary = computed(() => ordered.value.slice(2));

function start(slug: string): void {
  window.location.href = `${API_BASE_URL}/auth/oauth/${slug}/start`;
}
</script>
