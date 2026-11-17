<template>
  <footer class="break-keep border-t border-white/[0.06] bg-surface-0 text-text-secondary text-sm">
    <div class="mx-auto max-w-7xl px-4 py-12 sm:px-6 sm:py-16 lg:px-8">
      <div class="grid grid-cols-2 gap-x-6 gap-y-8 md:grid-cols-4">
        <!-- Brand & Mission -->
        <div class="col-span-2 space-y-3 md:col-span-1">
          <BrandLogo :size="28" tagline="" />
          <p class="max-w-xs text-sm leading-relaxed text-text-muted">
            {{ t('common.footer.tagline') }}
          </p>
          <div class="inline-flex items-center gap-2 rounded-full border border-white/10 px-3 py-1 text-caption text-text-secondary">
            <span class="h-1.5 w-1.5 rounded-full bg-verified"></span>
            {{ t('common.footer.aiBadge') }}
          </div>
        </div>

        <!-- Navigation Links -->
        <div>
          <h4 class="mb-4 text-sm font-medium text-white">{{ t('common.footer.colStream') }}</h4>
          <ul class="space-y-2.5 text-sm text-text-muted">
            <li><RouterLink to="/explore?sort=popular" class="hover:text-white transition-colors">{{ t('common.footer.popular') }}</RouterLink></li>
            <li><RouterLink to="/explore" class="hover:text-white transition-colors">{{ t('common.footer.byModel') }}</RouterLink></li>
            <li><RouterLink to="/explore?mood=Focus" class="hover:text-white transition-colors">{{ t('common.footer.focus') }}</RouterLink></li>
            <li><RouterLink to="/explore?mood=Chill" class="hover:text-white transition-colors">{{ t('common.footer.chill') }}</RouterLink></li>
          </ul>
        </div>

        <!-- Partner & Studio -->
        <div>
          <h4 class="mb-4 text-sm font-medium text-white">{{ t('common.footer.colPartner') }}</h4>
          <ul class="space-y-2.5 text-sm text-text-muted">
            <li><RouterLink to="/studio/upload" class="hover:text-white transition-colors">{{ t('common.footer.gate') }}</RouterLink></li>
            <li><RouterLink to="/studio/proofs" class="hover:text-white transition-colors">{{ t('common.footer.proof') }}</RouterLink></li>
            <li><RouterLink to="/studio/earnings" class="hover:text-white transition-colors">{{ t('common.footer.settlement') }}</RouterLink></li>
            <li><RouterLink to="/rights-center" class="hover:text-white transition-colors">{{ t('common.footer.disputes') }}</RouterLink></li>
          </ul>
        </div>

        <!-- Legal & Compliance -->
        <div>
          <h4 class="mb-4 text-sm font-medium text-white">{{ t('common.footer.colLegal') }}</h4>
          <ul class="space-y-2.5 text-sm text-text-muted">
            <li><RouterLink to="/legal/TOS" class="hover:text-white transition-colors">{{ t('common.footer.tos') }}</RouterLink></li>
            <li><RouterLink to="/legal/PRIVACY" class="hover:text-white transition-colors text-text-secondary">{{ t('common.footer.privacy') }}</RouterLink></li>
            <li><RouterLink to="/legal/COPYRIGHT_POLICY" class="hover:text-white transition-colors">{{ t('common.footer.copyright') }}</RouterLink></li>
            <li><RouterLink to="/legal/PARTNER_CONTRACT" class="hover:text-white transition-colors">{{ t('common.footer.contract') }}</RouterLink></li>
          </ul>
        </div>
      </div>

      <!-- Business identity block (C5): values come from VITE_BUSINESS_* env, hidden until registered -->
      <div v-if="businessLines.length > 0" class="mt-8 border-t border-white/[0.06] pt-6 text-caption leading-relaxed text-text-muted">
        <div class="font-bold text-text-primary mb-1">{{ t('common.footer.bizInfo') }}</div>
        <p v-for="(line, i) in businessLines" :key="i">{{ line }}</p>
        <p v-if="ospContact">
          {{ t('common.footer.ospContact') }}: <a :href="`mailto:${ospContact}`" class="text-brand-400 hover:underline">{{ ospContact }}</a>
        </p>
      </div>
      <div v-else-if="ospContact" class="mt-8 border-t border-white/[0.06] pt-6 text-caption text-text-muted">
        {{ t('common.footer.ospContact') }}: <a :href="`mailto:${ospContact}`" class="text-brand-400 hover:underline">{{ ospContact }}</a>
      </div>

      <!-- Bottom strip -->
      <div class="mt-10 border-t border-white/[0.06] pt-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-center sm:text-left text-caption text-text-muted">
        <div>
          {{ t('common.footer.rights') }}
        </div>
        <div class="flex flex-wrap items-center justify-center gap-x-4 gap-y-2">
          <RouterLink to="/pricing" class="hover:text-white transition-colors">{{ t('common.footer.pricing') }}</RouterLink>
          <LocaleSwitcher />
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import BrandLogo from '@/components/BrandLogo.vue';
import LocaleSwitcher from './LocaleSwitcher.vue';
const { t } = useI18n();

const env = import.meta.env as Record<string, string | undefined>;
const ospContact = (env.VITE_OSP_CONTACT_EMAIL ?? '').trim();

// Electronic Commerce Act §10 business identity (C5). Each line renders only
// when its env value is set, so nothing fictitious is ever displayed.
const businessLines = computed(() => {
  const lines: string[] = [];
  const name = (env.VITE_BUSINESS_NAME ?? '').trim();
  const ceo = (env.VITE_BUSINESS_CEO ?? '').trim();
  const addr = (env.VITE_BUSINESS_ADDRESS ?? '').trim();
  const tel = (env.VITE_BUSINESS_TEL ?? '').trim();
  const email = (env.VITE_BUSINESS_EMAIL ?? '').trim();
  const bizNo = (env.VITE_BUSINESS_REG_NO ?? '').trim();
  const mailOrderNo = (env.VITE_BUSINESS_MAILORDER_NO ?? '').trim();
  const hosting = (env.VITE_BUSINESS_HOSTING ?? '').trim();
  if (name) lines.push(`${t('common.footer.biz.company')}: ${name}${ceo ? ` · ${t('common.footer.biz.ceo')}: ${ceo}` : ''}`);
  if (addr) lines.push(`${t('common.footer.biz.address')}: ${addr}`);
  if (tel || email) lines.push(`${t('common.footer.biz.contact')}: ${[tel, email].filter(Boolean).join(' · ')}`);
  if (bizNo) lines.push(`${t('common.footer.biz.regNo')}: ${bizNo}`);
  if (mailOrderNo) lines.push(`${t('common.footer.biz.mailOrderNo')}: ${mailOrderNo}`);
  if (hosting) lines.push(`${t('common.footer.biz.hosting')}: ${hosting}`);
  return lines;
});
</script>
