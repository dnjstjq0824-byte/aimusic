<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('account.indexTitle')" :description="t('account.indexSub')" />
    <AccountNav />
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import AccountNav from '@/components/account/AccountNav.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const router = useRouter();

// Desktop goes straight to the first section; mobile/tablet-portrait keeps
// this list screen and each section links back here (UI §2.3).
onMounted(() => {
  if (window.matchMedia('(min-width: 768px)').matches) {
    void router.replace(localePath('/account/profile'));
  }
});
</script>
