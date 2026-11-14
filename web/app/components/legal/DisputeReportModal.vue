<template>
  <UiModal
    :model-value="true"
    title="저작권 분쟁 신고"
    :description="trackTitle"
    @close="emit('close')"
  >
    <form class="space-y-4" @submit.prevent="submit">
      <UiSelect v-model="reason" label="신고 사유" required>
        <option value="AI_POLICY_VIOLATION">비상업·무료 요금제 생성물 의심</option>
        <option value="VOICE_CLONE">음성·퍼블리시티권 침해 (무단 딥보이스)</option>
        <option value="UNAUTHORIZED_SAMPLE">저작권 침해·무단 샘플링</option>
        <option value="OTHER">기타 (상표권·불법 유해 콘텐츠 등)</option>
      </UiSelect>
      <UiTextarea
        v-model="description"
        label="상세 설명"
        placeholder="권리 관계를 구체적으로 적어 주세요"
        :rows="3"
        :maxlength="2000"
      />
      <p v-if="error" class="text-xs text-accent-crimson" role="alert">{{ error }}</p>
    </form>
    <template #footer>
      <UiButton variant="ghost" size="sm" @click="emit('close')">취소</UiButton>
      <UiButton variant="brand" size="sm" :loading="submitting" @click="submit">신고하기</UiButton>
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
