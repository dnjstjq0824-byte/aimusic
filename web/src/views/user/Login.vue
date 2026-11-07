<template>
  <div class="mx-auto max-w-md py-6 sm:py-12">
    <!-- Brand badge -->
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">AVIBE에 오신 것을 환영합니다</h1>
        <p class="text-xs text-neutral-400 mt-1">상업 라이선스가 검증된 차세대 AI 음악 스트리밍</p>
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
        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
        >
          로그인
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

        <!-- Test credentials quick pills (dev only) -->
        <div v-if="isDev" class="pt-4 border-t border-surface-border text-center">
          <div class="text-[11px] text-neutral-500 mb-2 font-medium">로컬 테스트 빠른 채우기 (개발 전용)</div>
          <div class="flex flex-wrap justify-center gap-1.5">
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-2.5 py-1 text-[11px] text-neutral-300 hover:text-white hover:bg-surface-3 border border-surface-border transition-colors"
              @click="fillCreds('admin@avibe.internal', 'admin1234!')"
            >
              어드민
            </button>
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-2.5 py-1 text-[11px] text-brand-400 hover:bg-surface-3 border border-surface-border transition-colors"
              @click="fillCreds('partner@avibe.internal', 'partner1234!')"
            >
              파트너
            </button>
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-2.5 py-1 text-[11px] text-neutral-300 hover:text-white hover:bg-surface-3 border border-surface-border transition-colors"
              @click="fillCreds('user@avibe.internal', 'user1234!')"
            >
              일반 유저
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

        <div class="pt-2">
          <UiCheckbox v-model="suAgreeTerms" required>
            <RouterLink to="/legal/TOS" target="_blank" class="text-brand-400 underline">서비스 이용약관</RouterLink> 및
            <RouterLink to="/legal/PRIVACY" target="_blank" class="text-brand-400 underline">개인정보처리방침</RouterLink>에 동의합니다. (필수)
          </UiCheckbox>
        </div>

        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
          :disabled="!suAgreeTerms"
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

        <div class="pt-2">
          <UiCheckbox v-model="ptAgreeContract" required>
            <RouterLink to="/legal/PARTNER_CONTRACT" target="_blank" class="text-brand-400 underline">파트너 음원유통·면책 계약서</RouterLink> 및
            <RouterLink to="/legal/COPYRIGHT_POLICY" target="_blank" class="text-brand-400 underline">AI 저작권 준수 확약</RouterLink>에 동의합니다. (필수)
          </UiCheckbox>
        </div>

        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
          :disabled="!ptAgreeContract"
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
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { authApi } from '@/api/endpoints';
import { API_BASE_URL, errMessageOf } from '@/api/client';
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

// Listener sign-up state
const suEmail = ref('');
const suPassword = ref('');
const suName = ref('');
const suAgreeTerms = ref(false);

// Partner sign-up state
const ptEmail = ref('');
const ptPassword = ref('');
const ptName = ref('');
const ptCompany = ref('');
const ptAgreeContract = ref(false);

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

function fillCreds(e: string, p: string) {
  email.value = e;
  password.value = p;
  activeTab.value = 'login';
}

async function submitLogin(): Promise<void> {
  busy.value = true;
  try {
    await auth.login(email.value, password.value);
    toast.success('성공적으로 로그인되었습니다.');
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    await router.push(redirect);
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '로그인 실패');
  } finally {
    busy.value = false;
  }
}

async function submitRegister(): Promise<void> {
  busy.value = true;
  try {
    await auth.register(suEmail.value, suPassword.value, suName.value);
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
    providers.value = await authApi.oauthProviders();
  } catch {
    providers.value = [];
  } finally {
    socialLoading.value = false;
  }
});
</script>
