<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">{{ t('auth.completeEmail.title') }}</h1>
        <p class="text-xs text-text-secondary mt-1">{{ t('auth.completeEmail.subtitle') }}</p>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <form class="space-y-4" @submit.prevent="submit">
        <UiInput
          v-model="email"
          type="email"
          :label="t('auth.common.email')"
          :placeholder="t('auth.common.emailPh')"
          autocomplete="email"
          required
        />
        <UiButton variant="brand" size="md" full-width type="submit">
          {{ t('auth.completeEmail.submit') }}
        </UiButton>
        <div class="text-center">
          <RouterLink :to="localePath('/login')" class="text-xs text-text-secondary hover:text-white hover:underline">
            {{ t('auth.common.backToLogin') }}
          </RouterLink>
        </div>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { ref } from 'vue';
import { useToast } from '@/composables/useToast';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const toast = useToast();

const email = ref('');

function submit(): void {
  // TODO(WI-A1): wire to the social-link endpoint when the engine lands it.
  toast.info(t('auth.completeEmail.comingSoon'));
}
</script>
