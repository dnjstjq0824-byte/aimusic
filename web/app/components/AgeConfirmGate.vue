<template>
  <UiModal
    v-if="auth.needsAgeConfirm"
    :model-value="true"
    title="만 14세 이상 확인"
    @update:model-value="noop"
  >
    <div class="space-y-4">
      <p class="text-xs leading-relaxed text-text-primary">
        소셜 계정으로 처음 로그인했습니다. AVIBE는 만 14세 이상만 이용할 수 있습니다.
        스트리밍·결제·업로드를 이용하려면 연령 확인이 필요합니다.
      </p>
      <UiCheckbox v-model="confirmed" required>
        <span class="text-xs text-text-primary">만 14세 이상입니다. (필수)</span>
      </UiCheckbox>
      <UiInput
        v-model="birthYear"
        label="출생연도 (선택)"
        placeholder="예: 1995"
        inputmode="numeric"
      />
      <UiErrorBanner :message="error" />
      <div class="flex gap-2">
        <UiButton variant="secondary" size="md" type="button" @click="auth.logout()">
          로그아웃
        </UiButton>
        <UiButton
          variant="brand"
          size="md"
          type="button"
          full-width
          :loading="busy"
          :disabled="!confirmed"
          @click="submit"
        >
          확인하고 시작하기
        </UiButton>
      </div>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { errMessageOf } from '@/api/client';
import UiButton from '@/components/ui/UiButton.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';

const auth = useAuthStore();
const toast = useToast();

const confirmed = ref(false);
const birthYear = ref('');
const busy = ref(false);
const error = ref<string | null>(null);

// The gate cannot be dismissed except by confirming or signing out.
function noop(): void {}

async function submit(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    const year = Number.parseInt(birthYear.value.trim(), 10);
    await auth.confirmAge(Number.isFinite(year) ? year : undefined);
    toast.success('연령 확인이 완료되었습니다.');
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}
</script>
