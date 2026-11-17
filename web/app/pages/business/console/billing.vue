<template>
  <div class="mx-auto max-w-2xl space-y-6">
    <UiPageHeader
      :eyebrow="t('business.console.biEyebrow')"
      :title="t('business.console.biTitle')"
      :description="t('business.console.biDesc')"
    />

    <UiErrorBanner :message="error" :title="t('business.console.biLoadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiCard v-if="loading" class="space-y-2">
      <UiSkeleton height="h-4" width="w-1/3" />
      <UiSkeleton height="h-4" width="w-2/3" />
    </UiCard>

    <UiCard v-else-if="!profile" class="space-y-2">
      <p class="text-xs text-text-secondary">{{ t('business.console.biNoProfile') }}</p>
    </UiCard>

    <UiCard v-else-if="profile" class="space-y-3">
      <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">{{ t('business.console.biContract') }}</h2>
      <dl class="space-y-2 text-xs">
        <div class="flex justify-between gap-2">
          <dt class="text-text-secondary">{{ t('business.console.biCompany') }}</dt>
          <dd class="font-bold text-white">{{ profile.companyName }}</dd>
        </div>
        <div class="flex justify-between gap-2">
          <dt class="text-text-secondary">{{ t('business.console.biCountry') }}</dt>
          <dd class="text-text-primary">{{ profile.country }}</dd>
        </div>
        <div v-if="profile.industry" class="flex justify-between gap-2">
          <dt class="text-text-secondary">{{ t('business.console.biIndustry') }}</dt>
          <dd class="text-text-primary">{{ profile.industry }}</dd>
        </div>
        <div class="flex justify-between gap-2">
          <dt class="text-text-secondary">{{ t('business.console.biStatus') }}</dt>
          <dd>
            <UiBadge :variant="profile.status === 'APPROVED' ? 'success' : profile.status === 'SUSPENDED' ? 'danger' : 'warning'" size="xs">
              {{ profileStatus(profile.status) }}
            </UiBadge>
          </dd>
        </div>
        <div class="flex justify-between gap-2">
          <dt class="text-text-secondary">{{ t('business.console.biStores') }}</dt>
          <dd class="font-mono text-text-primary">{{ stores.length }}</dd>
        </div>
        <div class="flex justify-between gap-2">
          <dt class="text-text-secondary">{{ t('business.console.biLicenses') }}</dt>
          <dd class="font-mono text-text-primary">{{ totalLicenses }}</dd>
        </div>
      </dl>
    </UiCard>

    <UiCard v-if="!loading">
      <p class="text-xs text-text-secondary">{{ t('business.console.biNote') }}</p>
      <RouterLink to="/business" class="mt-2 inline-block text-xs font-semibold text-brand-400 hover:underline">
        {{ t('business.console.biCta') }}
      </RouterLink>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { consoleApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { BusinessProfile, StoreBusiness } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, te } = useI18n();

function profileStatus(s: string): string {
  const key = `business.console.profileStatus.${s}`;
  return te(key) ? String(t(key)) : s;
}

const profile = ref<BusinessProfile | null>(null);
const stores = ref<StoreBusiness[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const totalLicenses = computed(() => stores.value.reduce((acc, s) => acc + (s.licenseCount || 0), 0));

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    // Migrated creator accounts hold no business profile; stores still list.
    const [p, list] = await Promise.all([
      consoleApi.profile().catch(() => null),
      consoleApi.listStores(),
    ]);
    profile.value = p;
    stores.value = list ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => void load());
</script>
