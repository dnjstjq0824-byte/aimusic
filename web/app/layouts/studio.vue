<template>
  <ConsoleShell
    id="studio"
    home-to="/studio"
    :home-label="t('studio.nav.home')"
    console-name="Studio"
    :menu="menu"
    :menu-label="t('studio.nav.menu')"
    :open-label="t('studio.nav.open')"
    :close-label="t('studio.nav.close')"
    :listener-home-label="t('studio.nav.listenerHome')"
    :logout-label="t('studio.nav.logout')"
  >
    <slot />
  </ConsoleShell>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ConsoleShell from '@/components/console/ConsoleShell.vue';

const { t } = useI18n();

const menu = computed(() => [
  { to: '/studio', name: 'studio-dashboard', label: t('studio.nav.dashboard') },
  { to: '/studio/create', name: 'studio-create', label: t('studio.nav.create') },
  { to: '/studio/upload', name: 'studio-upload', label: t('studio.nav.upload') },
  { to: '/studio/catalog', name: 'studio-catalog', label: t('studio.nav.catalog') },
  { to: '/studio/proofs', name: 'studio-proofs', label: t('studio.nav.proofs') },
  { to: '/studio/earnings', name: 'studio-earnings', label: t('studio.nav.earnings') },
  { to: '/studio/settings', name: 'studio-settings', label: t('studio.nav.settings') },
]);
</script>
