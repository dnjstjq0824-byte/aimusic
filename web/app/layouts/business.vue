<template>
  <ConsoleShell
    id="business"
    home-to="/business/console"
    :home-label="t('business.console.navHome')"
    console-name="Business"
    :menu="menu"
    :menu-label="t('business.console.navMenu')"
    :open-label="t('business.console.navOpen')"
    :close-label="t('business.console.navClose')"
    :listener-home-label="t('business.console.navListenerHome')"
    :logout-label="t('business.console.navLogout')"
  >
    <slot />
  </ConsoleShell>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ConsoleShell from '@/components/console/ConsoleShell.vue';

const { t } = useI18n();

const menu = computed(() => [
  { to: '/business/console', name: 'business-console', label: t('business.console.navOverview') },
  { to: '/business/console/stores', name: 'business-console-stores', label: t('business.console.navStores') },
  { to: '/business/console/players', name: 'business-console-players', label: t('business.console.navPlayers') },
  { to: '/business/console/licenses', name: 'business-console-licenses', label: t('business.console.navLicenses') },
  { to: '/business/console/billing', name: 'business-console-billing', label: t('business.console.navBilling') },
]);
</script>
