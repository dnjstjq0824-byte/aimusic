<template>
  <UiModal
    :model-value="modelValue"
    title="AIVIBE 상업 라이선스 검증 플랫폼 안내"
    description="인공지능기본법 제31조 및 저작권법 준수 가이드"
    max-width="lg"
    @close="$emit('update:modelValue', false)"
  >
    <div class="space-y-4 py-1 text-xs text-neutral-300">
      <div class="rounded-xl border border-brand-500/30 bg-brand-950/30 p-3.5 flex items-start gap-3">
        <span class="text-xl">🛡️</span>
        <div>
          <div class="font-bold text-white text-sm mb-1">상업적 권리가 보장된 AI 음원만 유통합니다</div>
          <p class="leading-relaxed text-neutral-300">
            AIVIBE는 Suno Pro/Premier, AIVA Pro 등 <strong>유료 상업용 요금제에서 생성된 음원만</strong> 4단계 게이트키퍼(Gate 1~4)를 거쳐 정식 서비스합니다. 무료 플랜이나 BGM 전용(Sync-only) 음원은 원천 차단됩니다.
          </p>
        </div>
      </div>

      <div class="space-y-2.5">
        <div class="flex items-start gap-2.5">
          <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-brand-400 font-bold text-[10px]">1</div>
          <p><strong>AI 투명성 배지:</strong> 모든 트랙에 AI 생성물 여부와 제공자(Suno, Udio, AIVA 등)가 명확히 고지됩니다.</p>
        </div>
        <div class="flex items-start gap-2.5">
          <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-brand-400 font-bold text-[10px]">2</div>
          <p><strong>끊김 없는 백그라운드 재생:</strong> 웹 탭을 닫거나 스마트폰 화면을 꺼도 음악이 멈추지 않고 연속 재생됩니다.</p>
        </div>
        <div class="flex items-start gap-2.5">
          <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-brand-400 font-bold text-[10px]">3</div>
          <p><strong>권리자 즉시 보호(OSP):</strong> 보이스 클로닝이나 타인 저작물 침해 의심 시 클릭 한 번으로 즉시 임시조치(Takedown)가 실행됩니다.</p>
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
