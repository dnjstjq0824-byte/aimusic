<template>
  <div class="mx-auto max-w-4xl py-6">
    <!-- Header -->
    <div class="mb-6 flex flex-wrap items-center justify-between gap-4 border-b border-surface-border pb-4">
      <div>
        <h1 class="text-2xl font-black text-white tracking-tight">{{ currentTerm?.titleKo || '법적 고지 및 약관' }}</h1>
        <p class="text-xs text-neutral-400 mt-1">
          최종 개정일: {{ currentTerm?.effectiveAt ? new Date(currentTerm.effectiveAt).toLocaleDateString() : '2026. 11. 01' }}
          · 버전: {{ currentTerm?.version || '1.0.0' }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UiButton
          v-if="auth.isAuthenticated && !consented"
          variant="brand"
          size="sm"
          :loading="consenting"
          @click="consent"
        >
          ✓ 본 약관 동의하기
        </UiButton>
        <span v-else-if="consented" class="rounded-full bg-emerald-950/80 border border-emerald-600/40 px-3 py-1 text-xs font-semibold text-emerald-300">
          ✓ 동의 완료됨
        </span>
      </div>
    </div>

    <!-- Navigation Pills -->
    <div class="mb-6 flex flex-wrap gap-2">
      <button
        v-for="code in termCodes"
        :key="code.id"
        type="button"
        class="rounded-xl px-3 py-1.5 text-xs font-semibold transition-all duration-150"
        :class="activeCode === code.id ? 'bg-brand-500 text-white shadow-glow-brand' : 'bg-surface-2 text-neutral-300 hover:bg-surface-3 hover:text-white border border-surface-border'"
        @click="selectCode(code.id)"
      >
        {{ code.name }}
      </button>
    </div>

    <!-- Loading & Error -->
    <div v-if="loading" class="space-y-4">
      <UiSkeleton height="h-8" width="w-2/3" />
      <UiSkeleton height="h-4" width="w-full" />
      <UiSkeleton height="h-4" width="w-5/6" />
      <UiSkeleton height="h-24" width="w-full" />
    </div>

    <div v-else-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <!-- Markdown Content Viewer -->
    <UiCard v-else class="prose prose-invert max-w-none text-xs leading-relaxed text-neutral-300">
      <div class="whitespace-pre-wrap font-sans space-y-3" v-html="renderedContent" />
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { legalApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { LegalTerm } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const termCodes = [
  { id: 'TOS', name: '서비스 이용약관' },
  { id: 'PRIVACY', name: '개인정보처리방침' },
  { id: 'AI_COPYRIGHT_POLICY', name: 'AI 저작권 & OSP 면책' },
  { id: 'PARTNER_CONTRACT', name: '파트너 음원유통 계약서' },
];

const activeCode = ref<string>((route.params.code as string) || 'TOS');
const terms = ref<LegalTerm[]>([]);
const currentTerm = ref<LegalTerm | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const consenting = ref(false);
const consented = ref(false);

const renderedContent = computed(() => {
  if (!currentTerm.value?.bodyMd) return '<p>내용이 등록되지 않았습니다.</p>';
  return currentTerm.value.bodyMd
    .replace(/^### (.*$)/gim, '<h3 class="text-sm font-bold text-white mt-4 mb-2">$1</h3>')
    .replace(/^## (.*$)/gim, '<h2 class="text-base font-extrabold text-white mt-6 mb-3 border-b border-surface-border pb-1">$1</h2>')
    .replace(/^# (.*$)/gim, '<h1 class="text-lg font-black text-brand-400 mt-6 mb-4">$1</h1>')
    .replace(/\*\*(.*?)\*\*/gim, '<strong class="font-bold text-white">$1</strong>')
    .replace(/^\* (.*$)/gim, '<li class="ml-4 list-disc text-neutral-300">$1</li>');
});

async function loadTerms() {
  loading.value = true;
  error.value = null;
  try {
    const list = await legalApi.listTerms();
    terms.value = list;
    syncCurrentTerm();
  } catch {
    error.value = '약관 정보를 불러오지 못했습니다.';
  } finally {
    loading.value = false;
  }
}

function syncCurrentTerm() {
  const match = terms.value.find((t) => t.code === activeCode.value || t.id === activeCode.value);
  if (match) {
    currentTerm.value = match;
  } else if (terms.value.length > 0) {
    currentTerm.value = terms.value[0];
  }
}

function selectCode(code: string) {
  activeCode.value = code;
  syncCurrentTerm();
  void router.replace(`/legal/${code}`);
}

async function consent() {
  if (!currentTerm.value) return;
  consenting.value = true;
  try {
    await legalApi.consent(currentTerm.value.id);
    consented.value = true;
    toast.success('약관에 동의하셨습니다.');
  } catch (err: unknown) {
    toast.error('약관 동의 처리에 실패했습니다.');
  } finally {
    consenting.value = false;
  }
}

watch(() => route.params.code, (newCode) => {
  if (newCode && typeof newCode === 'string') {
    activeCode.value = newCode;
    syncCurrentTerm();
  }
});

onMounted(() => {
  void loadTerms();
});
</script>
