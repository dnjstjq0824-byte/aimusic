import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { authApi, legalApi, type RegisterInput } from '@/api/endpoints';
import { AUTH_TOKENS_EVENT, tKey, tokenStore } from '@/api/client';
import type { LegalTerm, Role, User } from '@/api/types';
import { shouldApplyServerLocale } from '@/utils/locale';
import { trackEvent } from '@/utils/analytics';
import { usePlayerStore } from './playerStore';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const role = ref<Role | null>(null);
  const roles = ref<Role[]>([]);
  const userId = ref<string | null>(null);
  const displayName = ref<string | null>(null);
  const email = ref<string | null>(null);
  const ageConfirmedAt = ref<string | null>(null);
  // null = unknown (not yet resolved from /me); never guess the 2FA state.
  const totpEnabled = ref<boolean | null>(null);
  // Server locale + verification state (A2). null until /me resolves.
  const locale = ref<string | null>(null);
  const emailVerified = ref<boolean | null>(null);
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
      roles.value = [];
      displayName.value = null;
      email.value = null;
      ageConfirmedAt.value = null;
      locale.value = null;
      emailVerified.value = null;
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
  const displayLabel = computed(() => displayName.value || email.value || '');
  const hasRole = (r: Role): boolean => role.value === r || roles.value.includes(r);
  const isPartner = computed(() => hasRole('PARTNER') || hasRole('ADMIN'));
  const isBusiness = computed(() => hasRole('BUSINESS') || hasRole('ADMIN'));
  const isAdmin = computed(() => hasRole('ADMIN'));
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
      roles.value = me.roles ?? [];
      displayName.value = me.displayName;
      email.value = me.email;
      ageConfirmedAt.value = me.ageConfirmedAt ?? null;
      totpEnabled.value = me.totpEnabled ?? null;
      locale.value = me.locale ?? null;
      emailVerified.value = me.emailVerified ?? null;
      await refreshPendingConsents();
    } catch {
      tokenStore.clear();
      user.value = null;
      userId.value = null;
      role.value = null;
      roles.value = [];
      displayName.value = null;
      email.value = null;
      ageConfirmedAt.value = null;
      totpEnabled.value = null;
      locale.value = null;
      emailVerified.value = null;
      pendingConsents.value = [];
    } finally {
      initialized.value = true;
    }
  }

  // A2: after sign-in the server locale wins (detection order ①) on
  // unprefixed non-public screens. Best-effort: any failure keeps local.
  function applyServerLocale(): void {
    if (typeof window === 'undefined') return;
    try {
      const route = useRoute();
      const { $i18n } = useNuxtApp();
      const base = String(route.name ?? '').replace(/___.+$/, '');
      const apply = shouldApplyServerLocale(locale.value, $i18n.locale.value, route.path, base);
      if (apply) void $i18n.setLocale(apply);
    } catch {
      // Locale stays local; the switcher still works.
    }
  }

  async function login(loginEmail: string, password: string, totpCode?: string): Promise<void> {
    const { tokens, user: u } = await authApi.login(loginEmail, password, totpCode);
    tokenStore.set(tokens.accessToken, tokens.refreshToken);
    user.value = u;
    userId.value = u.id;
    role.value = u.role;
    displayName.value = u.displayName;
    email.value = u.email;
    ageConfirmedAt.value = u.ageConfirmedAt ?? null;
    // The login response carries no 2FA state; resolve it from /me so the
    // security screen never shows the wrong 2FA action after login.
    await hydrate();
    applyServerLocale();
    trackEvent({ name: 'login_succeeded', props: { means: 'password' } });
  }

  // Applies tokens issued out-of-band (social login callback), then resolves
  // the principal from /me instead of trusting redirect parameters.
  async function loginWithTokens(accessToken: string, refreshToken: string): Promise<void> {
    tokenStore.set(accessToken, refreshToken);
    try {
      const me = await authApi.me();
      trackEvent({ name: 'login_succeeded', props: { means: 'token' } });
      userId.value = me.userId;
      role.value = me.role;
      roles.value = me.roles ?? [];
      displayName.value = me.displayName;
      email.value = me.email;
      ageConfirmedAt.value = me.ageConfirmedAt ?? null;
      totpEnabled.value = me.totpEnabled ?? null;
      locale.value = me.locale ?? null;
      emailVerified.value = me.emailVerified ?? null;
      await refreshPendingConsents();
      applyServerLocale();
    } catch {
      tokenStore.clear();
      userId.value = null;
      role.value = null;
      roles.value = [];
      displayName.value = null;
      email.value = null;
      ageConfirmedAt.value = null;
      totpEnabled.value = null;
      locale.value = null;
      emailVerified.value = null;
      pendingConsents.value = [];
      throw new Error(tKey('auth.social.sessionErr', "Couldn't verify the social login session."));
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

  async function registerBusiness(input: RegisterInput & { company: string; bizNo?: string; country?: string; industry?: string }) {
    return authApi.registerBusiness(input);
  }

  function logout(): void {
    // Stop playback and drop the queue FIRST, while tokens are still valid
    // so the final resume position can persist; otherwise the detached audio
    // element keeps playing after logout.
    try {
      usePlayerStore().clearQueue();
    } catch {
      // Player store unavailable in this context; auth cleanup continues.
    }
    tokenStore.clear();
    user.value = null;
    userId.value = null;
    role.value = null;
    roles.value = [];
    displayName.value = null;
    email.value = null;
    ageConfirmedAt.value = null;
    totpEnabled.value = null;
    locale.value = null;
    emailVerified.value = null;
    pendingConsents.value = [];
  }

  return { user, role, roles, hasRole, userId, displayName, email, displayLabel, ageConfirmedAt, totpEnabled, locale, emailVerified, pendingConsents, initialized, isAuthenticated, isPartner, isBusiness, isAdmin, needsAgeConfirm, needsTermsConsent, hydrate, login, loginWithTokens, register, registerPartner, registerBusiness, confirmAge, consentTerms, refreshPendingConsents, logout };
});
