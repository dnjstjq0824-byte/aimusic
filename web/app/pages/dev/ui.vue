<template>
  <div class="mx-auto max-w-5xl space-y-12 px-4 py-10">
    <div>
      <p class="text-caption font-semibold uppercase tracking-widest text-brand-400">Design System v2 · Signal &amp; Seal</p>
      <h1 class="font-display text-h1 font-extrabold text-text-primary">UI Showcase (dev only)</h1>
    </div>

    <section class="space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Type scale</h2>
      <p class="font-display text-display font-extrabold text-text-primary">Display 바로 써도 되는 AI 음악</p>
      <p class="font-display text-h1 font-extrabold text-text-primary">H1 페이지 제목</p>
      <p class="font-display text-h2 font-bold text-text-primary">H2 섹션 제목</p>
      <p class="text-h3 font-bold text-text-primary">H3 카드 제목</p>
      <p class="text-body text-text-primary">Body 본문 16/24 — 모든 곡은 상업적 이용권 보유 인증을 거쳐 공개됩니다.</p>
      <p class="text-body-sm text-text-secondary">Body-sm 보조 14/20 — 무료로 듣고, 매장과 영상에는 라이선스로 쓰세요.</p>
      <p class="text-caption text-text-muted">Caption 메타 12/16 (최소 크기)</p>
      <p class="text-body-sm tnum text-text-secondary">Tabular 0123456789 ₩5,900 3:47</p>
    </section>

    <section class="space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Buttons</h2>
      <div class="flex flex-wrap items-center gap-3">
        <UiButton variant="primary">Primary</UiButton>
        <UiButton variant="secondary">Secondary</UiButton>
        <UiButton variant="ghost">Ghost</UiButton>
        <UiButton variant="danger">Danger</UiButton>
        <UiButton variant="success">Success</UiButton>
        <UiButton variant="outline">Outline</UiButton>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <UiButton size="sm">sm 36</UiButton>
        <UiButton size="md">md 44</UiButton>
        <UiButton size="lg">lg 52</UiButton>
        <UiButton size="xs">xs</UiButton>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <UiButton loading>Loading</UiButton>
        <UiButton disabled>Disabled</UiButton>
      </div>
    </section>

    <section class="max-w-md space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Inputs</h2>
      <UiInput v-model="demoText" label="이메일" placeholder="you@example.com" hint="안내 문구" />
      <UiInput v-model="demoText" label="비밀번호" type="password" placeholder="••••••••" />
      <UiInput v-model="demoText" label="오류 상태" error="올바른 값을 입력하세요." />
      <UiInput v-model="demoText" label="비활성" disabled placeholder="disabled" />
    </section>

    <section class="space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Verified Seal</h2>
      <div class="flex flex-wrap items-center gap-6">
        <VerifiedSeal :size="16" cert-number="AVB-DEMO" />
        <VerifiedSeal :size="20" cert-number="AVB-DEMO" />
        <VerifiedSeal :size="28" cert-number="AVB-DEMO" />
        <VerifiedSeal :size="20" :show-label="false" />
      </div>
    </section>

    <section class="space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Covers + Mosaic</h2>
      <div class="flex flex-wrap items-end gap-4">
        <UiCoverArt seed="track-a" title="곡 A" size="xs" />
        <UiCoverArt seed="track-b" title="곡 B" size="sm" />
        <UiCoverArt seed="track-c" title="곡 C" size="md" :verified="true" />
        <UiCoverArt seed="track-d" title="곡 D" size="lg" />
        <CoverMosaic :seeds="['m1', 'm2', 'm3', 'm4']" title="믹스" size="sm" />
        <CoverMosaic :seeds="['m1', 'm2']" title="믹스" size="md" />
      </div>
    </section>

    <section class="max-w-xl space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Stepper</h2>
      <Stepper :steps="['계정', '활동 정보', '약관 동의']" :current="step" />
      <div class="flex gap-2">
        <UiButton size="sm" variant="secondary" @click="step = Math.max(0, step - 1)">이전</UiButton>
        <UiButton size="sm" variant="secondary" @click="step = Math.min(2, step + 1)">다음</UiButton>
      </div>
    </section>

    <section class="space-y-3">
      <h2 class="font-display text-h2 font-bold text-text-primary">Menu + Sheet + Locale</h2>
      <div class="flex flex-wrap items-center gap-4">
        <UiMenu :items="menuItems" @select="menuPick = $event" />
        <span class="text-caption text-text-muted">선택: {{ menuPick === null ? '없음' : menuItems[menuPick]?.label }}</span>
        <UiButton variant="secondary" @click="sheetOpen = true">시트 열기</UiButton>
        <LocaleSwitcher v-model="locale" />
        <span class="text-caption text-text-muted">locale: {{ locale }}</span>
      </div>
      <BottomSheet v-model="sheetOpen" title="바텀시트 (모바일) / 모달 (데스크톱)">
        <p class="text-body-sm text-text-secondary">필터·⋯ 메뉴·플레이리스트 선택·문의 폼에 쓰는 공용 시트.</p>
        <div class="mt-4 flex justify-end gap-2">
          <UiButton variant="secondary" @click="sheetOpen = false">닫기</UiButton>
          <UiButton @click="sheetOpen = false">확인</UiButton>
        </div>
      </BottomSheet>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import Stepper from '@/components/ui/Stepper.vue';
import UiMenu from '@/components/ui/UiMenu.vue';
import BottomSheet from '@/components/ui/BottomSheet.vue';
import LocaleSwitcher from '@/components/ui/LocaleSwitcher.vue';

if (!import.meta.dev) {
  throw createError({ statusCode: 404, statusMessage: 'Not found' });
}

definePageMeta({ layout: 'user' });

const demoText = ref('');
const step = ref(1);
const sheetOpen = ref(false);
const locale = ref('ko');
const menuPick = ref<number | null>(null);
const menuItems = [
  { label: '다음에 재생' },
  { label: '큐에 추가' },
  { label: '플레이리스트에 추가' },
  { label: '공유' },
  { label: '신고', danger: true },
];
</script>
