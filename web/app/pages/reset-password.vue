<template>
  <div class="mx-auto max-w-md py-6 sm:py-12">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">비밀번호 찾기</h1>
        <p class="text-xs text-text-secondary mt-1">
          {{ token ? '새 비밀번호를 입력하세요.' : '가입한 이메일로 재설정 링크를 보내드립니다.' }}
        </p>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <!-- Request stage -->
      <form v-if="!token" class="space-y-4" @submit.prevent="submitRequest">
        <UiInput
          v-model="email"
          type="email"
          label="가입 이메일 주소"
          placeholder="name@example.com"
          autocomplete="email"
          required
        />
        <UiButton variant="brand" size="md" full-width type="submit" :loading="busy">
          재설정 링크 보내기
        </UiButton>
        <p v-if="requested" class="text-xs leading-relaxed text-success" role="status">
          입력한 이메일이 가입되어 있다면 재설정 링크를 발송했습니다. 메일함을 확인하세요.
          (링크는 30분간 1회만 유효합니다.)
        </p>
        <div class="text-center">
          <RouterLink to="/login" class="text-xs text-text-secondary hover:text-white hover:underline">
            로그인으로 돌아가기
          </RouterLink>
        </div>
      </form>

      <!-- Confirm stage -->
      <form v-else class="space-y-4" @submit.prevent="submitConfirm">
        <UiInput
          v-model="newPassword"
          type="password"
          label="새 비밀번호 (8자 이상)"
          placeholder="8자 이상의 안전한 비밀번호"
          :minlength="8"
          autocomplete="new-password"
          required
        />
        <UiInput
          v-model="confirmPassword"
          type="password"
          label="새 비밀번호 확인"
          placeholder="다시 입력"
          autocomplete="new-password"
          required
        />
        <UiErrorBanner :message="mismatch" />
        <UiButton variant="brand" size="md" full-width type="submit" :loading="busy" :disabled="mismatch !== null">
          비밀번호 변경하기
        </UiButton>
        <div class="text-center">
          <RouterLink to="/login" class="text-xs text-text-secondary hover:text-white hover:underline">
            로그인으로 돌아가기
          </RouterLink>
        </div>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { authApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const route = useRoute();
const router = useRouter();
const toast = useToast();

const token = computed(() => {
  const t = route.query.token;
  return typeof t === 'string' && t !== '' ? t : null;
});

const email = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const busy = ref(false);
const requested = ref(false);

const mismatch = computed(() => {
  if (confirmPassword.value === '') return null;
  return newPassword.value === confirmPassword.value ? null : '두 비밀번호가 일치하지 않습니다.';
});

async function submitRequest(): Promise<void> {
  busy.value = true;
  try {
    await authApi.requestPasswordReset(email.value);
    requested.value = true;
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '요청 실패');
  } finally {
    busy.value = false;
  }
}

async function submitConfirm(): Promise<void> {
  if (!token.value || mismatch.value) return;
  busy.value = true;
  try {
    await authApi.confirmPasswordReset(token.value, newPassword.value);
    toast.success('비밀번호가 변경되었습니다. 새 비밀번호로 로그인하세요.');
    await router.push('/login');
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '변경 실패');
  } finally {
    busy.value = false;
  }
}
</script>
