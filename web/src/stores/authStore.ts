import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { authApi } from '@/api/endpoints';
import { tokenStore } from '@/api/client';
import type { Role, User } from '@/api/types';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const role = ref<Role | null>(null);
  const userId = ref<string | null>(null);
  const initialized = ref(false);

  const isAuthenticated = computed(() => tokenStore.getAccess() !== null && userId.value !== null);
  const isPartner = computed(() => role.value === 'PARTNER' || role.value === 'ADMIN');
  const isAdmin = computed(() => role.value === 'ADMIN');

  async function hydrate(): Promise<void> {
    if (!tokenStore.getAccess()) {
      initialized.value = true;
      return;
    }
    try {
      const me = await authApi.me();
      userId.value = me.userId;
      role.value = me.role;
    } catch {
      tokenStore.clear();
      user.value = null;
      userId.value = null;
      role.value = null;
    } finally {
      initialized.value = true;
    }
  }

  async function login(email: string, password: string): Promise<void> {
    const { tokens, user: u } = await authApi.login(email, password);
    tokenStore.set(tokens.accessToken, tokens.refreshToken);
    user.value = u;
    userId.value = u.id;
    role.value = u.role;
  }

  // Applies tokens issued out-of-band (social login callback), then resolves
  // the principal from /me instead of trusting redirect parameters.
  async function loginWithTokens(accessToken: string, refreshToken: string): Promise<void> {
    tokenStore.set(accessToken, refreshToken);
    try {
      const me = await authApi.me();
      userId.value = me.userId;
      role.value = me.role;
    } catch {
      tokenStore.clear();
      userId.value = null;
      role.value = null;
      throw new Error('소셜 로그인 세션을 확인할 수 없습니다.');
    }
  }

  async function register(email: string, password: string, displayName: string): Promise<User> {
    return authApi.register(email, password, displayName);
  }

  async function registerPartner(input: {
    email: string;
    password: string;
    displayName: string;
    company: string;
    contactEmail?: string;
  }) {
    return authApi.registerPartner(input);
  }

  function logout(): void {
    tokenStore.clear();
    user.value = null;
    userId.value = null;
    role.value = null;
  }

  return { user, role, userId, initialized, isAuthenticated, isPartner, isAdmin, hydrate, login, loginWithTokens, register, registerPartner, logout };
});
