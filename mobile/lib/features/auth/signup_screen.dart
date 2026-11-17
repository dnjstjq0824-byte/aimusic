// Signup screen (POST /auth/register). Split from login (WI-R12).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/providers.dart';

class SignupScreen extends ConsumerStatefulWidget {
  const SignupScreen({super.key});

  @override
  ConsumerState<SignupScreen> createState() => _SignupScreenState();
}

class _SignupScreenState extends ConsumerState<SignupScreen> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _displayName = TextEditingController();
  final _birthYear = TextEditingController();
  final _inviteCode = TextEditingController();
  bool _busy = false;
  bool _ageConfirmed = false;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _displayName.dispose();
    _birthYear.dispose();
    _inviteCode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(s.signupTitle)),
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
            const SizedBox(height: 12),
            TextField(
              controller: _displayName,
              decoration: InputDecoration(labelText: s.signupDisplayName),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _birthYear,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(
                labelText: s.signupBirthYear,
                hintText: s.signupBirthYearHint,
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _inviteCode,
              decoration: InputDecoration(
                labelText: s.signupInviteCode,
              ),
            ),
            const SizedBox(height: 8),
            CheckboxListTile(
              value: _ageConfirmed,
              contentPadding: EdgeInsets.zero,
              controlAffinity: ListTileControlAffinity.leading,
              title: Text(
                s.signupAgeConfirm,
                style: const TextStyle(fontSize: 13),
              ),
              onChanged: (v) => setState(() => _ageConfirmed = v ?? false),
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(s.signupSubmit),
            ),
            TextButton(
              onPressed: _busy ? null : () => context.go('/login'),
              child: Text(s.loginHaveAccount),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final s = context.l10n;
    final deviceLocale = Localizations.localeOf(context);
    if (!_ageConfirmed) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(s.loginNeedAgeConfirm)),
      );
      return;
    }
    setState(() => _busy = true);
    try {
      final repo = ref.read(authRepositoryProvider);
      await repo.register(
        email: _email.text.trim(),
        password: _password.text,
        displayName: _displayName.text.trim(),
        birthYear: int.tryParse(_birthYear.text.trim()),
        ageConfirmed: _ageConfirmed,
        inviteCode: _inviteCode.text.trim().isEmpty
            ? null
            : _inviteCode.text.trim(),
      );
      // New accounts inherit the app language on the server (A2).
      try {
        await ref.read(apiProvider).updateLocale(
              toServerLocale(
                  ref.read(localeOverrideProvider) ??
                      resolveDeviceLanguage(deviceLocale)),
            );
      } catch (_) {}
      ref.invalidate(authMeProvider);
      if (mounted) {
        try {
          final pending = await ref.read(apiProvider).pendingConsents();
          if (mounted) context.go(pending.isEmpty ? '/' : '/legal');
        } catch (_) {
          if (mounted) context.go('/');
        }
      }
    } catch (e) {
      if (mounted) {
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
}
