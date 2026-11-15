<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">{{ t('auth.verify.title') }}</h1>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <div v-if="!token" class="space-y-4 text-center">
        <h2 class="text-lg font-bold text-white">{{ t('auth.verify.noTokenTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.verify.noTokenBody') }}</p>
        <div>
          <RouterLink :to="localePath('/login')" class="text-xs text-text-secondary hover:text-white hover:underline">
            {{ t('auth.common.backToLogin') }}
          </RouterLink>
        </div>
      </div>
      <div v-else class="space-y-4 text-center">
        <h2 class="text-lg font-bold text-white">{{ t('auth.verify.soonTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.verify.soonBody') }}</p>
        <div>
          <RouterLink :to="localePath('/login')" class="text-xs text-text-secondary hover:text-white hover:underline">
            {{ t('auth.common.backToLogin') }}
          </RouterLink>
        </div>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import UiCard from '@/components/ui/UiCard.vue';
import BrandLogo from '@/components/BrandLogo.vue';

// TODO(WI-A2): call the verify endpoint with the token when the engine lands it.
const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();

const token = computed(() => {
  const tok = route.query.token;
  return typeof tok === 'string' && tok !== '' ? tok : null;
});
</script>
