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

  async function register(email: string, password: string, displayName: string): Promise<User> {
    return authApi.register(email, password, displayName);
  }

  function logout(): void {
    tokenStore.clear();
    user.value = null;
    userId.value = null;
    role.value = null;
  }

  return { user, role, userId, initialized, isAuthenticated, isPartner, isAdmin, hydrate, login, register, logout };
});
