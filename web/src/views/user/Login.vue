<template>
  <div class="mx-auto max-w-md py-6 sm:py-12">
    <!-- Brand badge -->
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">AVIBE에 오신 것을 환영합니다</h1>
        <p class="text-xs text-neutral-400 mt-1">상업 이용 조건 확인 절차를 거친 AI 음악 스트리밍</p>
      </div>
    </div>

    <UiErrorBanner :message="oauthError" title="소셜 로그인 실패" class="mb-4" />

    <UiCard class="shadow-2xl">
      <!-- Tabs -->
      <div class="mb-6">
        <UiTabs v-model="activeTab" :tabs="tabOptions" />
      </div>

      <!-- Tab 1: Login -->
      <form v-if="activeTab === 'login'" class="space-y-4" @submit.prevent="submitLogin">
        <UiInput
          v-model="email"
          type="email"
          label="이메일 주소"
          placeholder="name@example.com"
          autocomplete="email"
          required
        />
        <UiInput
          v-model="password"
          type="password"
          label="비밀번호"
          placeholder="비밀번호 입력"
          autocomplete="current-password"
          required
        >
          <template #labelRight>
            <RouterLink to="/reset-password" class="font-medium text-brand-400 hover:underline">
              비밀번호 찾기
            </RouterLink>
          </template>
        </UiInput>
        <UiInput
          v-if="totpStep"
          v-model="totpCode"
          label="2단계 인증 코드"
          placeholder="인증 앱 6자리 코드 또는 복구 코드"
          autocomplete="one-time-code"
          inputmode="text"
          required
        />
        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
        >
          {{ totpStep ? '인증하고 로그인' : '로그인' }}
        </UiButton>

        <!-- Social login -->
        <div v-if="socialLoading" class="space-y-2 pt-2">
          <UiSkeleton height="h-10" width="w-full" />
        </div>
        <div v-else-if="enabledProviders.length > 0" class="space-y-2 pt-2">
          <div class="flex items-center gap-3 text-[11px] text-neutral-500">
            <span class="h-px flex-1 bg-surface-border" aria-hidden="true" />
            <span>간편 로그인</span>
            <span class="h-px flex-1 bg-surface-border" aria-hidden="true" />
          </div>
          <UiButton
            v-if="isProviderEnabled('GOOGLE')"
            variant="secondary"
            size="md"
            full-width
            type="button"
            @click="startSocial('google')"
          >
            Google로 계속하기
          </UiButton>
          <UiButton
            v-if="isProviderEnabled('KAKAO')"
            variant="secondary"
            size="md"
            full-width
            type="button"
            @click="startSocial('kakao')"
          >
            카카오로 계속하기
          </UiButton>
        </div>

        <!-- Test credentials quick pills (dev only; never render in production) -->
        <div v-if="isDev" class="pt-4 border-t border-surface-border text-center">
          <div class="text-[11px] text-neutral-400 mb-2 font-medium">체험용 빠른 로그인 (1-클릭)</div>
          <div class="flex flex-wrap justify-center gap-2">
            <button
              type="button"
              class="rounded-lg bg-brand-500/10 px-3 py-1.5 text-xs text-brand-300 hover:bg-brand-500/20 border border-brand-500/40 transition-colors font-semibold"
              @click="fillCreds('partner@example.com', 'Admin1234!', true)"
            >
              🎵 파트너 계정 (즉시 로그인)
            </button>
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs text-neutral-300 hover:text-white hover:bg-surface-3 border border-surface-border transition-colors font-medium"
              @click="fillCreds('user@example.com', 'Admin1234!', true)"
            >
              🎧 일반 리스너
            </button>
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs text-neutral-400 hover:text-white hover:bg-surface-3 border border-surface-border transition-colors font-medium"
              @click="fillCreds('admin@example.com', 'Admin1234!', true)"
            >
              🛡️ 어드민
            </button>
          </div>
        </div>
      </form>

      <!-- Tab 2: Listener Registration -->
      <form v-else-if="activeTab === 'register'" class="space-y-4" @submit.prevent="submitRegister">
        <UiInput
          v-model="suEmail"
          type="email"
          label="이메일 주소"
          placeholder="name@example.com"
          autocomplete="email"
          required
        />
        <UiInput
          v-model="suName"
          label="닉네임 / 활동명"
          placeholder="음악 애호가"
          autocomplete="nickname"
          required
        />
        <UiInput
          v-model="suPassword"
          type="password"
          label="비밀번호 (8자 이상)"
          placeholder="8자 이상의 안전한 비밀번호"
          :minlength="8"
          autocomplete="new-password"
          required
        />
        <UiInput
          v-model="suBirthYear"
          type="number"
          label="출생연도 (선택)"
          placeholder="예: 1995"
        />
        <UiInput
          v-model="suInviteCode"
          label="초대 코드 (필요 시)"
          placeholder="초대 코드가 있는 경우 입력"
          autocomplete="off"
        />

        <div class="pt-2 space-y-3">
          <UiCheckbox v-model="suAgreeTerms" required>
            <RouterLink to="/legal/TOS" target="_blank" class="text-brand-400 underline">서비스 이용약관</RouterLink> 및
            <RouterLink to="/legal/PRIVACY" target="_blank" class="text-brand-400 underline">개인정보처리방침</RouterLink>에 동의합니다. (필수)
          </UiCheckbox>
          <UiCheckbox v-model="suAgeConfirmed" required>
            만 14세 이상입니다. (필수, 만 14세 미만은 가입 불가)
          </UiCheckbox>
        </div>

        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
          :disabled="!suAgreeTerms || !suAgeConfirmed"
        >
          리스너로 시작하기
        </UiButton>
      </form>

      <!-- Tab 3: Partner Artist Registration -->
      <form v-else class="space-y-4" @submit.prevent="submitPartnerRegister">
        <UiInput
          v-model="ptEmail"
          type="email"
          label="파트너 이메일"
          placeholder="partner@studio.com"
          autocomplete="email"
          required
        />
        <UiInput
          v-model="ptName"
          label="아티스트 / 프로듀서명"
          placeholder="AVIBE Producer"
          required
        />
        <UiInput
          v-model="ptCompany"
          label="회사명 또는 스튜디오명"
          placeholder="개인 크리에이터 또는 법인명"
          required
        />
        <UiInput
          v-model="ptPassword"
          type="password"
          label="비밀번호 (8자 이상)"
          placeholder="8자 이상의 안전한 비밀번호"
          :minlength="8"
          autocomplete="new-password"
          required
        />
        <UiInput
          v-model="ptBirthYear"
          type="number"
          label="출생연도 (선택)"
          placeholder="예: 1995"
        />
        <UiInput
          v-model="ptInviteCode"
          label="초대 코드 (필요 시)"
          placeholder="초대 코드가 있는 경우 입력"
          autocomplete="off"
        />

        <div class="pt-2 space-y-3">
          <UiCheckbox v-model="ptAgreeContract" required>
            <RouterLink to="/legal/PARTNER_CONTRACT" target="_blank" class="text-brand-400 underline">파트너 음원유통·면책 계약서</RouterLink> 및
            <RouterLink to="/legal/COPYRIGHT_POLICY" target="_blank" class="text-brand-400 underline">AI 저작권 준수 확약</RouterLink>에 동의합니다. (필수)
          </UiCheckbox>
          <UiCheckbox v-model="ptAgeConfirmed" required>
            만 14세 이상입니다. (필수, 만 14세 미만은 가입 불가)
          </UiCheckbox>
        </div>

        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
          :disabled="!ptAgreeContract || !ptAgeConfirmed"
        >
          파트너 스튜디오 신청하기
        </UiButton>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { safeRedirect } from '@/router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { authApi } from '@/api/endpoints';
import { API_BASE_URL, errCodeOf, errMessageOf } from '@/api/client';
import type { OAuthProviderInfo } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTabs, { type TabOption } from '@/components/ui/UiTabs.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();
const toast = useToast();

const tabOptions: TabOption[] = [
  { label: '로그인', value: 'login' },
  { label: '리스너 회원가입', value: 'register' },
  { label: '파트너 등록', value: 'partner' },
];

const activeTab = ref<string>((route.query.tab as string) || 'login');
const busy = ref(false);
const isDev = import.meta.env.DEV;

// Login state
const email = ref('');
const password = ref('');
const totpCode = ref('');
const totpStep = ref(false);

// Listener sign-up state
const suEmail = ref('');
const suPassword = ref('');
const suName = ref('');
const suAgreeTerms = ref(false);
const suBirthYear = ref('');
const suInviteCode = ref('');
const suAgeConfirmed = ref(false);

// Partner sign-up state
const ptEmail = ref('');
const ptPassword = ref('');
const ptName = ref('');
const ptCompany = ref('');
const ptAgreeContract = ref(false);
const ptBirthYear = ref('');
const ptInviteCode = ref('');
const ptAgeConfirmed = ref(false);

function parseBirthYear(raw: string): number | undefined {
  const n = Number.parseInt(raw.trim(), 10);
  return Number.isFinite(n) ? n : undefined;
}

// Social login providers
const providers = ref<OAuthProviderInfo[]>([]);
const socialLoading = ref(true);
const enabledProviders = computed(() => providers.value.filter((p) => p.enabled));

function isProviderEnabled(provider: string): boolean {
  return providers.value.some((p) => p.provider === provider && p.enabled);
}

function startSocial(provider: 'google' | 'kakao'): void {
  window.location.href = `${API_BASE_URL}/auth/oauth/${provider}/start`;
}

const oauthError = computed(() => {
  const code = route.query.oauthError;
  if (typeof code !== 'string' || code === '') return null;
  return errMessageOf({ response: { data: { error: { code, message: code } } } });
});

function fillCreds(e: string, p: string, autoSubmit = false) {
  email.value = e;
  password.value = p;
  activeTab.value = 'login';
  if (autoSubmit) {
    void submitLogin();
  }
}

async function submitLogin(): Promise<void> {
  busy.value = true;
  try {
    await auth.login(email.value, password.value, totpStep.value ? totpCode.value : undefined);
    toast.success('성공적으로 로그인되었습니다.');
    await router.push(safeRedirect(route.query.redirect));
  } catch (err: unknown) {
    if (errCodeOf(err) === 'ERR_TOTP_REQUIRED' && !totpStep.value) {
      totpStep.value = true;
      toast.info('인증 앱에 표시된 6자리 코드를 입력해 주세요.');
    } else {
      toast.error(errMessageOf(err), '로그인 실패');
      if (errCodeOf(err) === 'ERR_TOTP_INVALID') totpCode.value = '';
    }
  } finally {
    busy.value = false;
  }
}

async function submitRegister(): Promise<void> {
  busy.value = true;
  try {
    await auth.register({
      email: suEmail.value,
      password: suPassword.value,
      displayName: suName.value,
      birthYear: parseBirthYear(suBirthYear.value),
      ageConfirmed: suAgeConfirmed.value,
      inviteCode: suInviteCode.value.trim() || undefined,
    });
    toast.success('회원가입이 완료되었습니다. 로그인해 주세요.');
    email.value = suEmail.value;
    password.value = suPassword.value;
    activeTab.value = 'login';
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '회원가입 실패');
  } finally {
    busy.value = false;
  }
}

async function submitPartnerRegister(): Promise<void> {
  busy.value = true;
  try {
    await auth.registerPartner({
      email: ptEmail.value,
      password: ptPassword.value,
      displayName: ptName.value,
      company: ptCompany.value,
      birthYear: parseBirthYear(ptBirthYear.value),
      ageConfirmed: ptAgeConfirmed.value,
      inviteCode: ptInviteCode.value.trim() || undefined,
    });
    toast.success('파트너 계정이 등록되었습니다. 로그인해 주세요.');
    email.value = ptEmail.value;
    password.value = ptPassword.value;
    activeTab.value = 'login';
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '파트너 등록 실패');
  } finally {
    busy.value = false;
  }
}

onMounted(async () => {
  try {
    providers.value = (await authApi.oauthProviders()) ?? [];
  } catch {
    providers.value = [];
  } finally {
    socialLoading.value = false;
  }
});
</script>
