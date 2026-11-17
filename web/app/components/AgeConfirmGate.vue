<template>
  <UiModal
    v-if="auth.needsAgeConfirm"
    :model-value="true"
    :title="t('auth.ageGate.title')"
    @update:model-value="noop"
  >
    <div class="space-y-4">
      <p class="text-xs leading-relaxed text-text-primary">
        {{ t('auth.ageGate.desc') }}
      </p>
      <UiCheckbox v-model="confirmed" required>
        <span class="text-xs text-text-primary">{{ t('auth.ageGate.confirm') }}</span>
      </UiCheckbox>
      <UiInput
        v-model="birthYear"
        :label="t('auth.common.birthYear')"
        :placeholder="t('auth.common.birthYearPh')"
        inputmode="numeric"
      />
      <UiErrorBanner :message="error" />
      <div class="flex gap-2">
        <UiButton variant="secondary" size="md" type="button" @click="auth.logout()">
          {{ t('common.actions.logout') }}
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
          {{ t('auth.ageGate.start') }}
        </UiButton>
      </div>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
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
const { t } = useI18n();

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
    toast.success(t('auth.ageGate.done'));
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}
</script>
