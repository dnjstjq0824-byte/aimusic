<template>
  <UiModal
    :model-value="true"
    :title="t('legal.mdTitle')"
    :description="trackTitle"
    @close="emit('close')"
  >
    <form class="space-y-4" @submit.prevent="submit">
      <UiSelect v-model="reason" :label="t('legal.mdReason')" required>
        <option value="AI_POLICY_VIOLATION">{{ t('legal.mdR1') }}</option>
        <option value="VOICE_CLONE">{{ t('legal.mdR2') }}</option>
        <option value="UNAUTHORIZED_SAMPLE">{{ t('legal.mdR3') }}</option>
        <option value="OTHER">{{ t('legal.mdR4') }}</option>
      </UiSelect>
      <UiTextarea
        v-model="description"
        :label="t('legal.mdDesc')"
        :placeholder="t('legal.mdDescPh')"
        :rows="3"
        :maxlength="2000"
      />
      <p v-if="error" class="text-xs text-accent-crimson" role="alert">{{ error }}</p>
    </form>
    <template #footer>
      <UiButton variant="ghost" size="sm" @click="emit('close')">{{ t('legal.mdCancel') }}</UiButton>
      <UiButton variant="brand" size="sm" :loading="submitting" @click="submit">{{ t('legal.mdSubmit') }}</UiButton>
    </template>
  </UiModal>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { disputeApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import UiModal from '@/components/ui/UiModal.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';

const { t } = useI18n();
const props = defineProps<{ trackId: string; trackTitle: string }>();
const emit = defineEmits<{ close: []; reported: [id: string] }>();

const reason = ref('AI_POLICY_VIOLATION');
const description = ref('');
const submitting = ref(false);
const error = ref<string | null>(null);

async function submit(): Promise<void> {
  submitting.value = true;
  error.value = null;
  try {
    const d = await disputeApi.report(props.trackId, reason.value, description.value || undefined);
    emit('reported', d.id);
    emit('close');
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    submitting.value = false;
  }
}
</script>
