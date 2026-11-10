import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { authApi, legalApi, type RegisterInput } from '@/api/endpoints';
import { AUTH_TOKENS_EVENT, tokenStore } from '@/api/client';
import type { LegalTerm, Role, User } from '@/api/types';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const role = ref<Role | null>(null);
  const userId = ref<string | null>(null);
  const ageConfirmedAt = ref<string | null>(null);
  const pendingConsents = ref<LegalTerm[]>([]);
  const initialized = ref(false);
  // Reactive mirror of the stored access token. localStorage itself is not
  // reactive, so without this the header kept showing a signed-in user after a
  // failed token refresh had already wiped the tokens (and every protected
  // page then bounced to the login form).
  const accessToken = ref<string | null>(tokenStore.getAccess());

  function syncTokens(): void {
    accessToken.value = tokenStore.getAccess();
    if (!accessToken.value) {
      user.value = null;
      userId.value = null;
      role.value = null;
      ageConfirmedAt.value = null;
      pendingConsents.value = [];
    }
  }
  if (typeof window !== 'undefined') {
    window.addEventListener(AUTH_TOKENS_EVENT, syncTokens);
    // Keep tabs in sync: signing out in one tab signs out the others.
    window.addEventListener('storage', (e) => {
      if (e.key === null || e.key.startsWith('avibe.')) syncTokens();
    });
  }

  const isAuthenticated = computed(() => accessToken.value !== null && userId.value !== null);
  const isPartner = computed(() => role.value === 'PARTNER' || role.value === 'ADMIN');
  const isAdmin = computed(() => role.value === 'ADMIN');
  const needsAgeConfirm = computed(() => isAuthenticated.value && ageConfirmedAt.value == null);
  const needsTermsConsent = computed(() => isAuthenticated.value && pendingConsents.value.length > 0);

  async function refreshPendingConsents(): Promise<void> {
    if (!isAuthenticated.value) {
      pendingConsents.value = [];
      return;
    }
    try {
      pendingConsents.value = (await legalApi.pendingConsents()) ?? [];
    } catch {
      pendingConsents.value = [];
    }
  }

  async function hydrate(): Promise<void> {
    accessToken.value = tokenStore.getAccess();
    if (!accessToken.value) {
      initialized.value = true;
      return;
    }
    try {
      const me = await authApi.me();
      userId.value = me.userId;
      role.value = me.role;
      ageConfirmedAt.value = me.ageConfirmedAt ?? null;
      await refreshPendingConsents();
    } catch {
      tokenStore.clear();
      user.value = null;
      userId.value = null;
      role.value = null;
      ageConfirmedAt.value = null;
      pendingConsents.value = [];
    } finally {
      initialized.value = true;
    }
  }

  async function login(email: string, password: string, totpCode?: string): Promise<void> {
    const { tokens, user: u } = await authApi.login(email, password, totpCode);
    tokenStore.set(tokens.accessToken, tokens.refreshToken);
    user.value = u;
    userId.value = u.id;
    role.value = u.role;
    ageConfirmedAt.value = u.ageConfirmedAt ?? null;
    await refreshPendingConsents();
  }

  // Applies tokens issued out-of-band (social login callback), then resolves
  // the principal from /me instead of trusting redirect parameters.
  async function loginWithTokens(accessToken: string, refreshToken: string): Promise<void> {
    tokenStore.set(accessToken, refreshToken);
    try {
      const me = await authApi.me();
      userId.value = me.userId;
      role.value = me.role;
      ageConfirmedAt.value = me.ageConfirmedAt ?? null;
      await refreshPendingConsents();
    } catch {
      tokenStore.clear();
      userId.value = null;
      role.value = null;
      ageConfirmedAt.value = null;
      pendingConsents.value = [];
      throw new Error('소셜 로그인 세션을 확인할 수 없습니다.');
    }
  }

  async function confirmAge(birthYear?: number): Promise<void> {
    await authApi.confirmAge(true, birthYear);
    ageConfirmedAt.value = new Date().toISOString();
  }

  async function consentTerms(termsId: string): Promise<void> {
    await legalApi.consent(termsId);
    pendingConsents.value = pendingConsents.value.filter((t) => t.id !== termsId);
  }

  async function register(input: RegisterInput): Promise<User> {
    return authApi.register(input);
  }

  async function registerPartner(input: RegisterInput & { company: string; contactEmail?: string }) {
    return authApi.registerPartner(input);
  }

  function logout(): void {
    tokenStore.clear();
    user.value = null;
    userId.value = null;
    role.value = null;
    ageConfirmedAt.value = null;
    pendingConsents.value = [];
  }

  return { user, role, userId, ageConfirmedAt, pendingConsents, initialized, isAuthenticated, isPartner, isAdmin, needsAgeConfirm, needsTermsConsent, hydrate, login, loginWithTokens, register, registerPartner, confirmAge, consentTerms, refreshPendingConsents, logout };
});
