<template>
  <footer class="border-t border-surface-border bg-surface-0 text-neutral-400 text-xs">
    <div class="mx-auto max-w-7xl px-4 py-12 sm:px-6">
      <div class="grid gap-8 sm:grid-cols-2 md:grid-cols-4">
        <!-- Brand & Mission -->
        <div class="space-y-3">
          <div class="flex items-center gap-2">
            <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-tr from-brand-600 to-amber-400 text-white font-black text-xs">
              AI
            </div>
            <span class="font-bold text-white text-sm">AIVIBE</span>
          </div>
          <p class="text-xs leading-relaxed text-neutral-400">
            상업적 라이선스가 검증된 정식 유료 플랜 생성 AI 음원만을 유통하는 상용 스트리밍 플랫폼입니다.
          </p>
          <div class="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-950/40 px-2.5 py-1 text-[11px] text-emerald-300">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            인공지능기본법 제31조 준수 플랫폼
          </div>
        </div>

        <!-- Navigation Links -->
        <div>
          <h4 class="font-bold text-white uppercase tracking-wider text-[11px] mb-3">스트리밍 &amp; 탐색</h4>
          <ul class="space-y-2 text-xs">
            <li><RouterLink to="/" class="hover:text-white transition-colors">실시간 Top 음원</RouterLink></li>
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
            <li><RouterLink to="/partner" class="hover:text-white transition-colors">파트너 정산 대시보드 (70:30)</RouterLink></li>
            <li><RouterLink to="/disputes" class="hover:text-white transition-colors">권리침해 신고 &amp; 소명 센터</RouterLink></li>
          </ul>
        </div>

        <!-- Legal & Compliance -->
        <div>
          <h4 class="font-bold text-white uppercase tracking-wider text-[11px] mb-3">법적 고지 &amp; 거버넌스</h4>
          <ul class="space-y-2 text-xs">
            <li><RouterLink to="/legal/TOS" class="hover:text-white transition-colors">서비스 이용약관</RouterLink></li>
            <li><RouterLink to="/legal/PRIVACY" class="hover:text-white transition-colors font-medium text-neutral-300">개인정보처리방침</RouterLink></li>
            <li><RouterLink to="/legal/AI_COPYRIGHT_POLICY" class="hover:text-white transition-colors">AI 저작권 및 OSP 면책 정책</RouterLink></li>
            <li><RouterLink to="/legal/PARTNER_CONTRACT" class="hover:text-white transition-colors">파트너 음원유통·면책 계약서</RouterLink></li>
          </ul>
        </div>
      </div>

      <!-- Bottom Compliance strip -->
      <div class="mt-10 border-t border-surface-border pt-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-[11px] text-neutral-500">
        <div>
          © 2026 AIVIBE Technologies Co., Ltd. All rights reserved. 대한민국 법령 준수 플랫폼.
        </div>
        <div class="flex items-center gap-4">
          <span>암호화: AES-256-GCM</span>
          <span>포렌식: FFprobe v7.0</span>
          <span>HLS 적응형 스트리밍</span>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
// Complete Footer component
</script>
