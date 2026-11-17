<template>
  <nav :aria-label="t('account.navLabel')" class="md:w-60 md:shrink-0">
    <ul class="flex gap-2 overflow-x-auto pb-1 md:flex-col md:gap-1 md:pb-0">
      <li v-for="item in items" :key="item.to" class="shrink-0 md:shrink">
        <RouterLink
          :to="localePath(item.to)"
          class="flex min-h-11 items-center rounded-xl px-3 text-sm font-semibold transition-colors"
          :class="isActive(item.to)
            ? 'bg-surface-2 text-white'
            : 'text-text-secondary hover:bg-surface-2 hover:text-white'"
          :aria-current="isActive(item.to) ? 'page' : undefined"
        >
          {{ t(item.key) }}
        </RouterLink>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();

const items = [
  { to: '/account/profile', key: 'account.navProfile' },
  { to: '/account/security', key: 'account.navSecurity' },
  { to: '/account/billing', key: 'account.navBilling' },
  { to: '/account/language', key: 'account.navLanguage' },
  { to: '/account/privacy', key: 'account.navPrivacy' },
  { to: '/account/connections', key: 'account.navConnections' },
];

function isActive(to: string): boolean {
  return route.path === to;
}
</script>
