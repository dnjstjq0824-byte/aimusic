<template>
  <nav
    class="pb-safe fixed inset-x-0 bottom-0 z-40 border-t border-surface-border bg-surface-0/95 backdrop-blur-xl lg:hidden"
    :aria-label="t('common.tabs.label')"
  >
    <div class="grid grid-cols-5">
      <NuxtLink
        v-for="tab in tabs"
        :key="tab.to"
        :to="hrefFor(tab.to)"
        class="flex min-h-14 flex-col items-center justify-center gap-0.5 px-1 py-1.5 text-caption transition-colors"
        :class="isActive(tab) ? 'text-brand-400 font-bold' : 'text-text-muted font-medium'"
        :aria-current="isActive(tab) ? 'page' : undefined"
      >
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path v-if="tab.icon === 'home'" d="M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z" />
          <g v-else-if="tab.icon === 'compass'">
            <circle cx="12" cy="12" r="9" />
            <path d="m15.5 8.5-2 5-5 2 2-5z" />
          </g>
          <g v-else-if="tab.icon === 'list'">
            <path d="M4 6h16M4 12h16M4 18h9" />
            <circle cx="19" cy="18" r="2.2" />
          </g>
          <g v-else-if="tab.icon === 'heart'">
            <path d="M12 20.5C7 16.5 3.5 13.3 3.5 9.6 3.5 7 5.5 5 8 5c1.6 0 3.1.9 4 2.3C12.9 5.9 14.4 5 16 5c2.5 0 4.5 2 4.5 4.6 0 3.7-3.5 6.9-8.5 10.9z" />
          </g>
          <g v-else>
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 20c1.4-3.6 4.2-5.4 7.5-5.4s6.1 1.8 7.5 5.4" />
          </g>
        </svg>
        <span class="whitespace-nowrap leading-tight">{{ t(tab.labelKey) }}</span>
      </NuxtLink>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();

// App-only routes (/library, /account) have no locale variants:
// localePath() returns '' for them, so keep the raw target (UiFooter.hrefFor).
const hrefFor = (to: string): string => localePath(to) || to;

const routeName = computed(() => String(route.name ?? '').replace(/___.+$/, ''));

// Static targets only (SSR-safe): the auth guard redirects guests to /login.
const tabs = [
  { to: '/', icon: 'home', labelKey: 'common.tabs.home', match: ['home'] },
  { to: '/explore', icon: 'compass', labelKey: 'common.tabs.explore', match: ['explore'] },
  { to: '/playlists', icon: 'list', labelKey: 'common.tabs.playlists', match: ['playlists', 'playlist-detail'] },
  { to: '/library', icon: 'heart', labelKey: 'common.tabs.library', match: ['library'] },
  { to: '/account', icon: 'user', labelKey: 'common.tabs.account', match: ['account-index', 'account-profile', 'account-security', 'account-billing', 'account-language', 'account-privacy', 'account-connections', 'login', 'signup', 'signup-listener', 'signup-creator', 'signup-business'] },
];

function isActive(tab: { match: string[] }): boolean {
  return tab.match.includes(routeName.value);
}
</script>
