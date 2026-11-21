// Login screen (POST /auth/login + social). Registration lives on
// the separate signup screen (WI-R12: login/signup split).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/providers.dart';
import '../../services/social_auth_service.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  bool _busy = false;
  bool _totpStep = false;
  final _totpCode = TextEditingController();
  String? _socialBusy;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _totpCode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.l10n;
    final locale = Localizations.localeOf(context).languageCode;
    final social = ref.watch(_socialProvidersProvider(locale));
    return Scaffold(
      appBar: AppBar(title: Text(s.loginTitle)),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              decoration: InputDecoration(labelText: s.loginEmail),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _password,
              obscureText: true,
              decoration: InputDecoration(
                labelText: s.loginPassword,
              ),
            ),
            if (_totpStep) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _totpCode,
                keyboardType: TextInputType.text,
                decoration: InputDecoration(
                  labelText: s.loginTotp,
                  hintText: s.loginTotpHint,
                ),
              ),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(_totpStep ? s.loginSubmitTotp : s.loginSubmit),
            ),
            TextButton(
              onPressed: _busy ? null : () => context.go('/signup'),
              child: Text(s.loginCreateAccount),
            ),
            social.when(
              data: (items) {
                final enabled =
                    items.where((p) => p.enabled).toList(growable: false);
                if (enabled.isEmpty) return const SizedBox.shrink();
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const SizedBox(height: 8),
                    Text(s.loginOrSocial,
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.bodySmall),
                    const SizedBox(height: 8),
                    for (final p in enabled)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 8),
                        child: OutlinedButton(
                          onPressed: (_busy || _socialBusy != null)
                              ? null
                              : () => _socialLogin(p.provider),
                          child: _socialBusy == p.provider
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2),
                                )
                              : Text(providerDisplayName(s, p.provider)),
                        ),
                      ),
                  ],
                );
              },
              loading: () => const SizedBox.shrink(),
              error: (_, __) => const SizedBox.shrink(),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final s = context.l10n;
    setState(() => _busy = true);
    try {
      final repo = ref.read(authRepositoryProvider);
      await repo.login(
        email: _email.text.trim(),
        password: _password.text,
        totpCode: _totpStep ? _totpCode.text.trim() : null,
      );
      await _afterAuth();
    } catch (e) {
      if (e is ApiException &&
          e.code == ApiCodes.totpRequired &&
          !_totpStep &&
          mounted) {
        setState(() => _totpStep = true);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(s.loginTotpPrompt)),
        );
      } else if (mounted) {
        if (e is ApiException && e.code == ApiCodes.totpInvalid) {
          _totpCode.clear();
        }
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e, s)
                  : s.loginFailed)),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _socialLogin(String provider) async {
    final s = context.l10n;
    setState(() => _socialBusy = provider);
    try {
      final social = ref.read(socialAuthProvider);
      final code = await social.signIn(provider);
      await social.exchange(code);
      await _afterAuth();
    } on SocialAuthCancelled {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(s.loginSocialCancelled)),
        );
      }
    } on SocialEmailRequired catch (e) {
      // WI-N9: email-less provider -> collect the email with the token.
      if (mounted) {
        context.go(
            '/complete-email?token=${Uri.encodeComponent(e.emailToken)}');
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e, s)
                  : s.loginSocialFailed(
                      providerDisplayName(s, provider)))),
        );
      }
    } finally {
      if (mounted) setState(() => _socialBusy = null);
    }
  }

  /// Post-login routing shared by password + social: adopt the server
  /// locale when the user has no local override, then route to consent
  /// when required terms are pending.
  Future<void> _afterAuth() async {
    ref.invalidate(authMeProvider);
    final me = await ref.read(authMeProvider.future);
    if (me?.locale != null && me!.locale!.isNotEmpty) {
      final override = ref.read(localeOverrideProvider);
      if (override == null) {
        await ref
            .read(localeOverrideProvider.notifier)
            .apply(normalizeAppLanguage(me.locale));
      }
    }
    if (!mounted) return;
    try {
      final pending = await ref.read(apiProvider).pendingConsents();
      if (mounted) context.go(pending.isEmpty ? '/' : '/legal');
    } catch (_) {
      if (mounted) context.go('/');
    }
  }
}

final _socialProvidersProvider =
    FutureProvider.family<List<OAuthProviderInfo>, String>((ref, locale) {
  return ref.watch(socialAuthProvider).providers(locale);
});
