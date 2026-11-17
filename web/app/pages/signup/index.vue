<template>
  <div class="mx-auto max-w-2xl">
    <div class="mb-8 text-center">
      <h1 class="type-display text-3xl text-white sm:text-4xl">{{ t('auth.signup.title') }}</h1>
      <p class="text-xs text-text-secondary mt-2">{{ t('auth.signup.subtitle') }}</p>
    </div>

    <div class="grid gap-4 sm:grid-cols-3">
      <UiCard
        v-for="card in cards"
        :key="card.to"
        class="flex flex-col items-center gap-2 p-6 text-center shadow-2xl transition-colors hover:border-brand-500/50"
      >
        <h2 class="text-lg font-bold text-white">{{ t(card.titleKey) }}</h2>
        <p class="min-h-10 text-xs leading-relaxed text-text-secondary">{{ t(card.descKey) }}</p>
        <NuxtLink :to="localePath(card.to)" class="mt-2 w-full">
          <UiButton variant="brand" size="md" full-width type="button">
            {{ t(card.ctaKey) }}
          </UiButton>
        </NuxtLink>
      </UiCard>
    </div>

    <p class="mt-8 text-center text-xs text-text-secondary">
      {{ t('auth.signup.hasAccount') }}
      <NuxtLink :to="localePath('/login')" class="font-medium text-brand-400 hover:underline">
        {{ t('auth.signup.loginLink') }}
      </NuxtLink>
    </p>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';

const { t } = useI18n();
const localePath = useLocalePath();

// NOTE: card title/desc/cta keys resolve dynamically (invisible to
// check-i18n usage scan — they are used, do not delete).
const cards = [
  { to: '/signup/listener', titleKey: 'auth.signup.listenerTitle', descKey: 'auth.signup.listenerDesc', ctaKey: 'auth.signup.listenerCta' },
  { to: '/signup/creator', titleKey: 'auth.signup.creatorTitle', descKey: 'auth.signup.creatorDesc', ctaKey: 'auth.signup.creatorCta' },
  { to: '/signup/business', titleKey: 'auth.signup.businessTitle', descKey: 'auth.signup.businessDesc', ctaKey: 'auth.signup.businessCta' },
];
</script>
