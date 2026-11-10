<template>
  <div class="mx-auto max-w-2xl py-6 sm:py-10 space-y-6">
    <div>
      <h1 class="font-display text-2xl font-black text-white tracking-tight">계정 보안</h1>
      <p class="text-xs text-neutral-400 mt-1">2단계 인증, 전체 기기 로그아웃, 회원 탈퇴를 관리합니다.</p>
    </div>

    <!-- 2FA card -->
    <UiCard>
      <h2 class="text-sm font-bold text-white mb-1">2단계 인증 (TOTP)</h2>
      <p class="text-xs text-neutral-400 mb-4">
        인증 앱(Google Authenticator 등)에 표시되는 6자리 코드로 로그인을 보호합니다.
      </p>

      <UiErrorBanner :message="totpError" />

      <!-- Enroll: show secret -->
      <div v-if="enrollment && !recoveryCodes" class="space-y-4">
        <div class="rounded-lg bg-surface-2 p-3">
          <div class="text-[11px] text-neutral-400 mb-1">인증 앱에 이 키를 직접 입력하세요</div>
          <code class="block break-all font-mono text-sm text-brand-300">{{ enrollment.secret }}</code>
          <button
            type="button"
            class="mt-2 text-xs text-neutral-400 hover:text-white underline"
            @click="copy(enrollment.secret)"
          >
            키 복사
          </button>
        </div>
        <p class="text-[11px] text-neutral-500 break-all">
          직접 입력이 어려운 경우 앱에서 다음 URI를 여세요:
          <code class="font-mono">{{ enrollment.provisioningUri }}</code>
        </p>
        <form class="space-y-3" @submit.prevent="submitConfirm">
          <UiInput
            v-model="confirmCode"
            label="인증 앱에 표시된 6자리 코드"
            placeholder="000000"
            inputmode="numeric"
            autocomplete="one-time-code"
            required
          />
          <UiButton variant="brand" size="md" full-width type="submit" :loading="busy">
            인증하고 2단계 인증 켜기
          </UiButton>
        </form>
      </div>

      <!-- Recovery codes (shown once) -->
      <div v-else-if="recoveryCodes" class="space-y-4">
        <p class="text-xs text-accent-emerald font-semibold" role="status">
          2단계 인증이 켜졌습니다. 아래 복구 코드는 지금 한 번만 표시됩니다. 안전한 곳에 보관하세요.
        </p>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <code
            v-for="c in recoveryCodes"
            :key="c"
            class="rounded-lg bg-surface-2 px-3 py-2 font-mono text-xs text-white"
          >
            {{ c }}
          </code>
        </div>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="sm" type="button" @click="copy(recoveryCodes.join('\n'))">
            전체 복사
          </UiButton>
          <UiButton variant="brand" size="sm" type="button" @click="dismissRecovery">
            보관했습니다
          </UiButton>
        </div>
      </div>

      <!-- Default state: exactly one action matching the real 2FA state -->
      <div v-else class="space-y-4">
        <p class="text-xs" role="status">
          <span class="text-neutral-400">현재 상태: </span>
          <span v-if="auth.totpEnabled === true" class="font-bold text-accent-emerald">켜짐</span>
          <span v-else-if="auth.totpEnabled === false" class="font-bold text-neutral-200">꺼짐</span>
          <span v-else class="font-bold text-accent-amber">확인 중</span>
        </p>
        <div v-if="auth.totpEnabled === false">
          <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitEnroll">
            2단계 인증 설정 시작
          </UiButton>
        </div>
        <form v-else-if="auth.totpEnabled === true" class="space-y-3" @submit.prevent="submitDisable">
          <p class="text-xs text-neutral-400">비밀번호 확인 후 2단계 인증을 끌 수 있습니다.</p>
          <UiInput
            v-model="disablePassword"
            type="password"
            label="현재 비밀번호"
            placeholder="비밀번호 입력"
            autocomplete="current-password"
            required
          />
          <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitDisable">
            2단계 인증 끄기
          </UiButton>
        </form>
        <div v-else>
          <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="refreshTotpState">
            상태 새로고침
          </UiButton>
        </div>
      </div>
    </UiCard>

    <!-- Sessions card -->
    <UiCard>
      <h2 class="text-sm font-bold text-white mb-1">로그인 세션</h2>
      <p class="text-xs text-neutral-400 mb-4">
        모든 기기에서 로그아웃합니다. 분실한 기기가 있거나 계정 도용이 의심될 때 사용하세요.
      </p>
      <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitLogoutAll">
        모든 기기에서 로그아웃
      </UiButton>
    </UiCard>

    <!-- Data export -->
    <UiCard>
      <h2 class="text-sm font-bold text-white mb-1">내 정보 내려받기</h2>
      <p class="text-xs text-neutral-400 mb-4">
        계정·동의 기록·플레이리스트·재생 기록·주문 내역을 JSON 파일로 내려받습니다.
      </p>
      <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitExport">
        내 정보 다운로드 (JSON)
      </UiButton>
    </UiCard>

    <!-- Danger zone -->
    <UiCard>
      <h2 class="text-sm font-bold text-accent-crimson mb-1">회원 탈퇴</h2>
      <p class="text-xs text-neutral-400 mb-4">
        탈퇴하면 계정이 비활성화되고 모든 세션이 종료됩니다. 이 작업은 되돌릴 수 없습니다.
      </p>
      <UiErrorBanner :message="withdrawError" />
      <div v-if="!withdrawArmed">
        <UiButton variant="secondary" size="md" type="button" @click="withdrawArmed = true">
          회원 탈퇴하기
        </UiButton>
      </div>
      <div v-else class="space-y-3">
        <UiCheckbox v-model="withdrawConfirm">
          <span class="text-xs text-neutral-300">정말로 탈퇴합니다. 되돌릴 수 없음을 이해했습니다.</span>
        </UiCheckbox>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="md" type="button" @click="withdrawArmed = false">
            취소
          </UiButton>
          <UiButton
            variant="brand"
            size="md"
            type="button"
            :loading="busy"
            :disabled="!withdrawConfirm"
            @click="submitWithdraw"
          >
            탈퇴 확정
          </UiButton>
        </div>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { authApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';

const auth = useAuthStore();
const router = useRouter();
const toast = useToast();
const busy = ref(false);

const enrollment = ref<{ secret: string; provisioningUri: string } | null>(null);
const confirmCode = ref('');
const recoveryCodes = ref<string[] | null>(null);
const disablePassword = ref('');
const totpError = ref<string | null>(null);

const withdrawArmed = ref(false);
const withdrawConfirm = ref(false);
const withdrawError = ref<string | null>(null);

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    toast.success('클립보드에 복사했습니다.');
  } catch {
    toast.error('복사에 실패했습니다. 직접 선택해서 복사해 주세요.');
  }
}

async function submitEnroll(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    enrollment.value = await authApi.enrollTOTP();
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function submitConfirm(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    const res = await authApi.confirmTOTPEnrollment(confirmCode.value);
    recoveryCodes.value = res.recoveryCodes;
    confirmCode.value = '';
    auth.totpEnabled = true;
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

function dismissRecovery(): void {
  recoveryCodes.value = null;
  enrollment.value = null;
}

async function refreshTotpState(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    await auth.hydrate();
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function submitDisable(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    await authApi.disableTOTP(disablePassword.value);
    disablePassword.value = '';
    auth.totpEnabled = false;
    toast.success('2단계 인증을 껐습니다.');
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function submitLogoutAll(): Promise<void> {
  busy.value = true;
  try {
    await authApi.logoutAll();
    auth.logout();
    toast.success('모든 기기에서 로그아웃했습니다. 다시 로그인해 주세요.');
    await router.push('/login');
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '로그아웃 실패');
  } finally {
    busy.value = false;
  }
}

async function submitExport(): Promise<void> {
  busy.value = true;
  try {
    const bundle = await authApi.exportData();
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `avibe-my-data-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    toast.success('내 정보를 다운로드했습니다.');
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '내보내기 실패');
  } finally {
    busy.value = false;
  }
}

async function submitWithdraw(): Promise<void> {
  busy.value = true;
  withdrawError.value = null;
  try {
    await authApi.withdraw();
    auth.logout();
    toast.success('탈퇴가 완료되었습니다. 이용해 주셔서 감사합니다.');
    await router.push('/');
  } catch (err: unknown) {
    withdrawError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}
</script>
