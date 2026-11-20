<template>
  <ConsoleShell
    id="admin"
    home-to="/admin"
    :home-label="t('admin.nav.home')"
    console-name="Admin"
    :menu="menu"
    :menu-label="t('admin.nav.menu')"
    :open-label="t('admin.nav.open')"
    :close-label="t('admin.nav.close')"
    :listener-home-label="t('admin.nav.listenerHome')"
    :logout-label="t('admin.nav.logout')"
  >
    <template #badge>
      <span class="hidden rounded-full border border-white/10 px-3 py-1 text-xs text-text-secondary md:inline">
        {{ t('admin.nav.badge') }}
      </span>
    </template>
    <slot />
  </ConsoleShell>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import ConsoleShell from '@/components/console/ConsoleShell.vue';

const { t, locale, setLocale } = useI18n();
const route = useRoute();

// R-2: the admin console ships ko/en only. Any other saved locale falls
// back to English while inside /admin/**.
function enforceAdminLocale(): void {
  if (locale.value !== 'ko' && locale.value !== 'en') {
    void setLocale('en');
  }
}
watch(() => route.fullPath, () => enforceAdminLocale());
// Also on an in-console switch (footer menu), not only on the next navigation.
watch(locale, () => enforceAdminLocale());
onMounted(() => enforceAdminLocale());

// v3 flow (phase 3): 13 sections moved from a crowded top bar into the
// shared sidebar, same as studio and business.
const menu = computed(() => [
  { to: '/admin', name: 'admin-overview', label: t('admin.nav.overview') },
  { to: '/admin/reviews', name: 'admin-reviews', label: t('admin.nav.reviews') },
  { to: '/admin/policies', name: 'admin-policies', label: t('admin.nav.policies') },
  { to: '/admin/disputes', name: 'admin-disputes', label: t('admin.nav.disputes') },
  { to: '/admin/curation', name: 'admin-curation', label: t('admin.nav.curation') },
  { to: '/admin/audit', name: 'admin-audit', label: t('admin.nav.audit') },
  { to: '/admin/users', name: 'admin-users', label: t('admin.nav.users') },
  { to: '/admin/stores', name: 'admin-stores', label: t('admin.nav.stores') },
  { to: '/admin/distribution', name: 'admin-distribution', label: t('admin.nav.distribution') },
  { to: '/admin/settlements', name: 'admin-settlements', label: t('admin.nav.settlements') },
  { to: '/admin/inquiries', name: 'admin-inquiries', label: t('admin.nav.inquiries') },
  { to: '/admin/role-grants', name: 'admin-role-grants', label: t('admin.nav.roles') },
  { to: '/admin/social', name: 'admin-social', label: t('admin.nav.social') },
]);
</script>
