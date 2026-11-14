<template>
  <UiModal
    :model-value="modelValue"
    title="AVIBE 상업 라이선스 검증 플랫폼 안내"
    description="AI 생성 표시 및 권리보호 안내"
    max-width="lg"
    @close="$emit('update:modelValue', false)"
  >
    <div class="space-y-4 py-1 text-xs text-text-primary">
      <div class="rounded-xl border border-brand-500/30 bg-brand-950/30 p-3.5 flex items-start gap-3">
        <svg class="h-6 w-6 shrink-0 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M12 3 5 6v5c0 4.4 3 8.4 7 10 4-1.6 7-5.6 7-10V6l-7-3z" />
          <path d="m9 12 2 2 4-4" />
        </svg>
        <div>
          <div class="font-bold text-white text-sm mb-1">상업 이용 조건 확인 절차를 거친 AI 음원을 유통합니다</div>
          <p class="leading-relaxed text-text-primary">
            AVIBE는 Suno Pro/Premier, AIVA Pro 등 <strong>유료 상업용 요금제에서 생성된 음원만</strong> 4단계 게이트키퍼(Gate 1~4)를 거쳐 정식 서비스합니다. 무료 플랜이나 BGM 전용(Sync-only)으로 확인된 음원은 등록 단계에서 차단합니다.
          </p>
        </div>
      </div>

      <div class="space-y-2.5">
        <div class="flex items-start gap-2.5">
          <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-brand-400 font-bold text-caption">1</div>
          <p><strong>AI 투명성 배지:</strong> 모든 트랙에 AI 생성물 여부와 제공자(Suno, Udio, AIVA 등)가 명확히 고지됩니다.</p>
        </div>
        <div class="flex items-start gap-2.5">
          <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-brand-400 font-bold text-caption">2</div>
          <p><strong>이어지는 재생:</strong> 웹에서는 탭을 열어 두면 다른 화면을 보는 동안에도 계속 재생됩니다. 화면을 끈 상태의 재생은 모바일 앱에서 지원합니다.</p>
        </div>
        <div class="flex items-start gap-2.5">
          <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-brand-400 font-bold text-caption">3</div>
          <p><strong>권리자 보호(OSP):</strong> 보이스 클로닝이나 타인 저작물 침해가 의심되면 트랙 화면에서 신고할 수 있습니다. 접수된 신고는 검토 후 저작권법 제103조 절차에 따라 임시조치합니다.</p>
        </div>
      </div>
    </div>

    <template #footer>
      <UiButton variant="brand" size="sm" @click="$emit('update:modelValue', false)">
        시작하기
      </UiButton>
    </template>
  </UiModal>
</template>

<script setup lang="ts">
import UiModal from '@/components/ui/UiModal.vue';
import UiButton from '@/components/ui/UiButton.vue';

defineProps<{
  modelValue: boolean;
}>();

defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();
</script>
