<template>
  <UiModal
    v-if="auth.needsTermsConsent && !auth.needsAgeConfirm"
    :model-value="true"
    title="필수 약관 동의"
    @update:model-value="noop"
  >
    <div class="space-y-4">
      <p class="text-xs leading-relaxed text-text-primary">
        개정된 필수 약관이 있습니다. 스트리밍·결제·업로드를 계속 이용하려면 아래 약관에 동의해 주세요.
      </p>
      <ul class="space-y-2">
        <li
          v-for="t in auth.pendingConsents"
          :key="t.id"
          class="flex items-center justify-between gap-3 rounded-lg bg-surface-2 px-3 py-2"
        >
          <div class="min-w-0">
            <div class="truncate text-sm font-semibold text-white">{{ t.titleKo }}</div>
            <div class="text-caption text-text-secondary">{{ t.code }} · {{ t.version }}</div>
          </div>
          <div class="flex shrink-0 gap-2">
            <RouterLink :to="`/legal/${t.code}`" target="_blank">
              <UiButton variant="secondary" size="xs" type="button">전문 보기</UiButton>
            </RouterLink>
            <UiButton
              variant="brand"
              size="xs"
              type="button"
              :loading="busyId === t.id"
              @click="consentOne(t.id)"
            >
              동의하기
            </UiButton>
          </div>
        </li>
      </ul>
      <UiErrorBanner :message="error" />
      <UiButton variant="secondary" size="md" type="button" full-width @click="auth.logout()">
        로그아웃
      </UiButton>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { errMessageOf } from '@/api/client';
import UiButton from '@/components/ui/UiButton.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiModal from '@/components/ui/UiModal.vue';

const auth = useAuthStore();
const toast = useToast();

const busyId = ref<string | null>(null);
const error = ref<string | null>(null);

// The gate cannot be dismissed except by consenting or signing out.
function noop(): void {}

async function consentOne(termsId: string): Promise<void> {
  busyId.value = termsId;
  error.value = null;
  try {
    await auth.consentTerms(termsId);
    if (auth.pendingConsents.length === 0) {
      toast.success('약관 동의가 완료되었습니다.');
    }
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busyId.value = null;
  }
}
</script>
