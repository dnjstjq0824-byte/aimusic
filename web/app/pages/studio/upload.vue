<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.upload.eyebrow')"
      :title="t('studio.upload.title')"
      :description="t('studio.upload.desc')"
    />

    <Stepper :steps="stepLabels" :current="store.step - 1" />

    <UiErrorBanner :message="loadError" :title="t('studio.upload.loadFail')" />

    <!-- Step 1: audio file -->
    <UiCard v-if="store.step === 1" class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.upload.s1Title') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.upload.s1Desc') }}</p>
      </div>

      <div>
        <label class="block text-xs font-medium text-text-primary mb-1.5" for="studio-upload-file">
          {{ t('studio.upload.fFile') }}
        </label>
        <input
          id="studio-upload-file"
          type="file"
          accept="audio/*"
          class="w-full text-xs text-text-secondary file:mr-3 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-surface-2 file:text-white hover:file:bg-surface-3 cursor-pointer"
          @change="onFile"
        />
      </div>

      <div v-if="store.fileName" class="rounded-xl border border-surface-border bg-surface-0 p-4 text-xs">
        <div class="flex justify-between gap-2">
          <span class="text-text-secondary">{{ t('studio.upload.pickedFile') }}</span>
          <span class="font-mono text-white font-semibold truncate">{{ store.fileName }}</span>
        </div>
      </div>

      <div class="flex items-center justify-end gap-2 pt-2 border-t border-surface-border">
        <UiButton
          variant="brand"
          size="sm"
          :disabled="!store.canLeaveStep1"
          @click="store.step = 2"
        >
          {{ t('studio.upload.next') }}
        </UiButton>
      </div>
    </UiCard>

    <!-- Step 2: origin, plan, proof + pre-check -->
    <UiCard v-else-if="store.step === 2" class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.upload.s2Title') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.upload.s2Desc') }}</p>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <UiSelect
          v-model="store.provider"
          :label="t('studio.upload.fProvider')"
          required
          @change="store.planTierUsed = ''"
        >
          <option value="">{{ t('studio.upload.fProviderPh') }}</option>
          <option v-for="p in store.policies" :key="p.provider" :value="p.provider">
            {{ p.provider }} {{ p.allowsStandaloneStreaming ? t('studio.upload.commercialOk') : t('studio.upload.syncOnly') }}
          </option>
        </UiSelect>

        <UiSelect
          v-model="store.planTierUsed"
          :label="t('studio.upload.fTier')"
          required
        >
          <option value="">{{ t('studio.upload.fTierPh') }}</option>
          <option v-for="tier in store.availablePlanTiers" :key="tier" :value="tier">
            {{ store.provider }} — {{ tier }}
          </option>
        </UiSelect>

        <UiSelect
          v-model="store.subscriptionId"
          :label="t('studio.upload.fSub')"
          required
        >
          <option value="">{{ t('studio.upload.fSubPh') }}</option>
          <option v-for="s in store.subscriptions" :key="s.id" :value="s.id">
            {{ s.provider }} / {{ s.planName }} ({{ subStatus(s.status) }})
          </option>
        </UiSelect>

        <UiInput
          v-model="store.generatedAt"
          type="datetime-local"
          :label="t('studio.upload.fGeneratedAt')"
          required
        />

        <div class="sm:col-span-2">
          <UiInput
            v-model="store.providerTrackUrl"
            :label="t('studio.upload.fTrackUrl')"
            placeholder="https://suno.com/song/..."
          />
        </div>
      </div>

      <div
        v-if="store.instantBlockReason"
        class="rounded-xl border border-red-600 bg-red-950/70 p-4 text-xs font-semibold text-red-200 flex items-start gap-2.5 shadow-lg"
      >
        <span class="text-base" aria-hidden="true">⛔</span>
        <div class="leading-relaxed">
          {{ store.instantBlockReason }}
        </div>
      </div>

      <LicensePrecheckWidget :status="store.gate12" :code="store.gate12Code" />

      <div class="flex items-center justify-between gap-2 pt-2 border-t border-surface-border">
        <UiButton variant="ghost" size="sm" @click="store.step = 1">{{ t('studio.upload.prev') }}</UiButton>
        <div class="flex gap-2">
          <UiButton
            variant="brand"
            size="sm"
            :loading="store.gate12 === 'checking'"
            :disabled="store.instantBlockReason !== null"
            @click="precheck"
          >
            {{ t('studio.upload.runPrecheck') }}
          </UiButton>
          <UiButton
            variant="secondary"
            size="sm"
            :disabled="store.step2Blocked"
            @click="store.step = 3"
          >
            {{ t('studio.upload.next') }}
          </UiButton>
        </div>
      </div>
    </UiCard>

    <!-- Step 3: metadata -->
    <UiCard v-else-if="store.step === 3" class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.upload.s3Title') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.upload.s3Desc') }}</p>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <UiInput
          v-model="store.title"
          :label="t('studio.upload.fTitle')"
          :placeholder="t('studio.upload.fTitlePh')"
          required
        />
        <UiInput
          v-model="store.genre"
          :label="t('studio.upload.fGenre')"
          :placeholder="t('studio.upload.fGenrePh')"
        />
        <UiInput
          v-model="store.moodTagsInput"
          :label="t('studio.upload.fMoods')"
          :placeholder="t('studio.upload.fMoodsPh')"
        />
        <UiSelect
          v-model="store.disclosure"
          :label="t('studio.upload.fDisclosure')"
        >
          <option value="FULL_AI">{{ t('studio.upload.disclosureFull') }}</option>
          <option value="AI_ASSISTED_HUMAN">{{ t('studio.upload.disclosureAssisted') }}</option>
        </UiSelect>

        <div class="sm:col-span-2">
          <UiTextarea
            v-model="store.lyrics"
            :label="t('studio.upload.fLyrics')"
            :placeholder="t('studio.upload.fLyricsPh')"
          />
        </div>

        <div class="sm:col-span-2 rounded-xl border border-surface-border bg-surface-0 p-3">
          <div class="text-xs font-medium text-text-primary mb-1">{{ t('studio.upload.creditTitle') }}</div>
          <p class="text-caption text-text-muted mb-2">{{ t('studio.upload.creditDesc') }}</p>
          <div class="flex flex-wrap gap-x-5 gap-y-2">
            <UiCheckbox
              v-for="opt in AI_CREDIT_PART_OPTIONS"
              :key="opt.code"
              :model-value="store.aiCreditParts.includes(opt.code)"
              @update:model-value="store.toggleAiCreditPart(opt.code)"
            >
              {{ t(opt.key) }}
            </UiCheckbox>
          </div>
        </div>

        <div class="sm:col-span-2 rounded-xl border border-surface-border bg-surface-0 p-3">
          <div class="text-xs font-medium text-text-primary mb-1">{{ t('studio.upload.contribTitle') }}</div>
          <p class="text-caption text-text-muted mb-2">{{ t('studio.upload.contribDesc') }}</p>
          <div class="flex flex-wrap gap-x-5 gap-y-2">
            <UiCheckbox
              v-for="opt in HUMAN_CONTRIBUTION_OPTIONS"
              :key="opt.code"
              :model-value="store.humanContributionTypes.includes(opt.code)"
              @update:model-value="store.toggleHumanContribution(opt.code)"
            >
              {{ t(opt.key) }}
            </UiCheckbox>
          </div>
        </div>

        <div class="sm:col-span-2">
          <UiInput
            v-model="store.aiPrompt"
            :label="t('studio.upload.fPrompt')"
            :placeholder="t('studio.upload.fPromptPh')"
          />
        </div>

        <div class="sm:col-span-2">
          <UiInput
            v-model="store.humanContributionDesc"
            :label="t('studio.upload.fContribDesc')"
            :placeholder="t('studio.upload.fContribDescPh')"
          />
        </div>

        <UiSelect
          v-model="store.collectiveTrust"
          :label="t('studio.upload.fTrust')"
          required
        >
          <option value="NONE">{{ t('studio.upload.trustNone') }}</option>
          <option value="TRUSTED">{{ t('studio.upload.trustTrusted') }}</option>
        </UiSelect>
        <UiInput
          v-model="store.collectiveName"
          :label="t('studio.upload.fTrustName')"
          placeholder="KOMCA"
        />
      </div>

      <div
        v-if="store.collectiveTrust === 'TRUSTED'"
        class="rounded-xl border border-red-600 bg-red-950/70 p-4 text-xs font-semibold text-red-200"
      >
        {{ t('studio.upload.trustedBlock') }}
      </div>

      <div class="flex items-center justify-between gap-2 pt-2 border-t border-surface-border">
        <UiButton variant="ghost" size="sm" @click="store.step = 2">{{ t('studio.upload.prev') }}</UiButton>
        <UiButton
          variant="brand"
          size="sm"
          :disabled="store.step3Blocked"
          @click="store.step = 4"
        >
          {{ t('studio.upload.next') }}
        </UiButton>
      </div>
    </UiCard>

    <!-- Step 4: review + warranties + submit -->
    <UiCard v-else class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.upload.s4Title') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.upload.s4Desc') }}</p>
      </div>

      <div class="rounded-xl border border-surface-border bg-surface-0 p-4 space-y-2 text-xs">
        <div class="flex justify-between gap-2">
          <span class="text-text-secondary">{{ t('studio.upload.revFile') }}</span>
          <span class="font-mono text-white font-semibold truncate">{{ store.fileName }}</span>
        </div>
        <div class="flex justify-between gap-2">
          <span class="text-text-secondary">{{ t('studio.upload.revSha') }}</span>
          <span class="font-mono text-brand-400 break-all text-caption">{{ store.audioSha256 || t('studio.upload.revShaWait') }}</span>
        </div>
        <div class="flex justify-between gap-2">
          <span class="text-text-secondary">{{ t('studio.upload.revOrigin') }}</span>
          <span class="text-white font-semibold">{{ store.provider }} · {{ store.planTierUsed || '—' }}</span>
        </div>
        <div class="flex justify-between gap-2">
          <span class="text-text-secondary">{{ t('studio.upload.revTitle') }}</span>
          <span class="text-white font-semibold truncate">{{ store.title }}</span>
        </div>
        <div class="flex justify-between gap-2">
          <span class="text-text-secondary">{{ t('studio.upload.revPrecheck') }}</span>
          <UiBadge variant="success" size="xs">{{ t('studio.upload.revPass') }}</UiBadge>
        </div>
      </div>

      <UiErrorBanner :message="submitError" :title="t('studio.upload.submitFail')" />

      <div class="space-y-3 rounded-2xl border border-brand-500/30 bg-surface-0 p-4">
        <div class="text-xs font-bold text-white uppercase tracking-wider mb-2">
          {{ t('studio.upload.warrantyTitle') }}
        </div>
        <UiCheckbox v-model="store.declaresOwnership" required>
          {{ t('studio.upload.wOwn') }}
        </UiCheckbox>
        <UiCheckbox v-model="store.declaresTrainingConsent" required>
          {{ t('studio.upload.wNoRetro') }}
        </UiCheckbox>
        <UiCheckbox v-model="store.declaresNoVoiceClone" required>
          {{ t('studio.upload.wNoClone') }}
        </UiCheckbox>
        <UiCheckbox v-model="store.agreesCommercialTerms" required>
          {{ t('studio.upload.wIndemnify') }}
        </UiCheckbox>
      </div>

      <div class="flex items-center justify-between gap-2 pt-2 border-t border-surface-border">
        <UiButton variant="ghost" size="sm" @click="store.step = 3">{{ t('studio.upload.prev') }}</UiButton>
        <UiButton
          variant="brand"
          size="sm"
          :disabled="!store.canFinalize || submitting"
          :loading="submitting"
          @click="submit"
        >
          {{ t('studio.upload.submit') }}
        </UiButton>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
  AI_CREDIT_PART_OPTIONS,
  HUMAN_CONTRIBUTION_OPTIONS,
  usePartnerUploadStore,
} from '@/stores/partnerUploadStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';
import Stepper from '@/components/ui/Stepper.vue';
import LicensePrecheckWidget from '@/components/legal/LicensePrecheckWidget.vue';

const { t, te } = useI18n();
const store = usePartnerUploadStore();
const router = useRouter();
const toast = useToast();
const apiError = useApiError();

const stepLabels = computed(() => [
  t('studio.upload.step1'),
  t('studio.upload.step2'),
  t('studio.upload.step3'),
  t('studio.upload.step4'),
]);

function subStatus(status: string): string {
  const key = `studio.proofs.subStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

const loadError = ref<string | null>(null);
const submitError = ref<string | null>(null);
const submitting = ref(false);

function onFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  store.pickFile(file);
}

async function precheck() {
  const ok = await store.runPrecheck();
  if (ok) {
    toast.success(t('studio.upload.precheckPass'));
  } else {
    toast.error(store.gate12Message || t('studio.upload.precheckFail'));
  }
}

async function submit() {
  submitting.value = true;
  submitError.value = null;
  try {
    const staged = await store.uploadStaged();
    if (!staged) {
      submitError.value = store.gate3Message || t('studio.upload.uploadFail');
      return;
    }
    const ok = await store.finalize();
    if (ok) {
      toast.success(t('studio.upload.submitDone'));
      store.reset();
      await router.push('/studio/catalog');
    } else {
      submitError.value = store.finalizeError || t('studio.upload.submitFail');
    }
  } catch (err: unknown) {
    submitError.value = apiError(err);
  } finally {
    submitting.value = false;
  }
}

onMounted(async () => {
  try {
    await store.loadReference();
  } catch (err: unknown) {
    loadError.value = apiError(err);
  }
});
</script>
