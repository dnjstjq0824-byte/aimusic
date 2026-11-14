<template>
  <footer class="break-keep border-t border-surface-border bg-surface-0 text-neutral-400 text-xs">
    <div class="mx-auto max-w-7xl px-4 py-10 sm:px-6 sm:py-12 lg:px-8">
      <div class="grid grid-cols-2 gap-x-6 gap-y-8 md:grid-cols-4">
        <!-- Brand & Mission -->
        <div class="col-span-2 space-y-3 md:col-span-1">
          <BrandLogo :size="32" />
          <p class="text-xs leading-relaxed text-neutral-400">
            상업 이용 조건 확인 절차를 거친 AI 음원을 유통하는 스트리밍 플랫폼입니다.
          </p>
          <div class="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-950/40 px-2.5 py-1 text-[11px] text-emerald-300">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            AI 생성 표시 제공
          </div>
        </div>

        <!-- Navigation Links -->
        <div>
          <h4 class="font-bold text-white uppercase tracking-wider text-[11px] mb-3">스트리밍 &amp; 탐색</h4>
          <ul class="space-y-2 text-xs">
            <li><RouterLink to="/explore?sort=popular" class="hover:text-white transition-colors">인기 음원</RouterLink></li>
            <li><RouterLink to="/explore" class="hover:text-white transition-colors">AI 모델별 탐색 (Suno, Udio, AIVA)</RouterLink></li>
            <li><RouterLink to="/explore?mood=Focus" class="hover:text-white transition-colors">집중 &amp; 작업용 사운드</RouterLink></li>
            <li><RouterLink to="/explore?mood=Chill" class="hover:text-white transition-colors">칠 &amp; 로우파이 라운지</RouterLink></li>
          </ul>
        </div>

        <!-- Partner & Studio -->
        <div>
          <h4 class="font-bold text-white uppercase tracking-wider text-[11px] mb-3">파트너 &amp; 크리에이터</h4>
          <ul class="space-y-2 text-xs">
            <li><RouterLink to="/partner/upload" class="hover:text-white transition-colors">4단계 상업 라이선스 업로드 게이트</RouterLink></li>
            <li><RouterLink to="/partner/subscriptions" class="hover:text-white transition-colors">유료 AI 구독 증빙 등록</RouterLink></li>
            <li><RouterLink to="/partner" class="hover:text-white transition-colors">파트너 정산 대시보드</RouterLink></li>
            <li><RouterLink to="/disputes" class="hover:text-white transition-colors">권리침해 신고 &amp; 소명 센터</RouterLink></li>
          </ul>
        </div>

        <!-- Legal & Compliance -->
        <div>
          <h4 class="font-bold text-white uppercase tracking-wider text-[11px] mb-3">법적 고지 &amp; 거버넌스</h4>
          <ul class="space-y-2 text-xs">
            <li><RouterLink to="/legal/TOS" class="hover:text-white transition-colors">서비스 이용약관</RouterLink></li>
            <li><RouterLink to="/legal/PRIVACY" class="hover:text-white transition-colors font-medium text-neutral-300">개인정보처리방침</RouterLink></li>
            <li><RouterLink to="/legal/COPYRIGHT_POLICY" class="hover:text-white transition-colors">AI 저작권 및 OSP 면책 정책</RouterLink></li>
            <li><RouterLink to="/legal/PARTNER_CONTRACT" class="hover:text-white transition-colors">파트너 음원유통·면책 계약서</RouterLink></li>
          </ul>
        </div>
      </div>

      <!-- Business identity block (C5): values come from VITE_BUSINESS_* env, hidden until registered -->
      <div v-if="businessLines.length > 0" class="mt-8 border-t border-surface-border pt-6 text-[11px] leading-relaxed text-neutral-500">
        <div class="font-bold text-neutral-300 mb-1">사업자 정보</div>
        <p v-for="(line, i) in businessLines" :key="i">{{ line }}</p>
        <p v-if="ospContact">
          저작권 침해 중단 요구(수령인): <a :href="`mailto:${ospContact}`" class="text-brand-400 hover:underline">{{ ospContact }}</a>
        </p>
      </div>
      <div v-else-if="ospContact" class="mt-8 border-t border-surface-border pt-6 text-[11px] text-neutral-500">
        저작권 침해 중단 요구(수령인): <a :href="`mailto:${ospContact}`" class="text-brand-400 hover:underline">{{ ospContact }}</a>
      </div>

      <!-- Bottom Compliance strip -->
      <div class="mt-10 border-t border-surface-border pt-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-center sm:text-left text-[11px] text-neutral-500">
        <div>
          © 2026 AVIBE — Beyond Your Music. All rights reserved.
        </div>
        <div class="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
          <span>암호화: AES-256-GCM</span>
          <span>포렌식: FFprobe v7.0</span>
          <span>HLS 적응형 스트리밍</span>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import BrandLogo from '@/components/BrandLogo.vue';

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
  if (name) lines.push(`상호: ${name}${ceo ? ` · 대표자: ${ceo}` : ''}`);
  if (addr) lines.push(`주소: ${addr}`);
  if (tel || email) lines.push(`연락처: ${[tel, email].filter(Boolean).join(' · ')}`);
  if (bizNo) lines.push(`사업자등록번호: ${bizNo}`);
  if (mailOrderNo) lines.push(`통신판매업 신고번호: ${mailOrderNo}`);
  if (hosting) lines.push(`호스팅 제공자: ${hosting}`);
  return lines;
});
</script>
