// Social-login email fallback (WI-N9, web /signup/complete-email parity).
// The provider gave no email: collect one, file it with the one-time
// signup token (POST /auth/oauth/complete-email), then show the
// verification-mail notice. The account is created when the link clicks.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/providers.dart';

class CompleteEmailScreen extends ConsumerStatefulWidget {
  const CompleteEmailScreen({super.key, required this.token});

  final String token;

  @override
  ConsumerState<CompleteEmailScreen> createState() =>
      _CompleteEmailScreenState();
}

class _CompleteEmailScreenState extends ConsumerState<CompleteEmailScreen> {
  final _email = TextEditingController();
  bool _busy = false;
  String? _submittedEmail;

  @override
  void dispose() {
    _email.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(s.completeEmailTitle)),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(s.completeEmailSubtitle,
                style: Theme.of(context).textTheme.bodyMedium),
            const SizedBox(height: 24),
            if (_submittedEmail != null) ...[
              Text(s.completeEmailPendingTitle,
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 8),
              Text(s.completeEmailPendingBody(_submittedEmail!)),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: () => context.go('/login'),
                child: Text(s.completeEmailBackToLogin),
              ),
            ] else ...[
              TextField(
                controller: _email,
                keyboardType: TextInputType.emailAddress,
                decoration: InputDecoration(
                  labelText: s.loginEmail,
                  hintText: s.completeEmailHint,
                ),
              ),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _busy ? null : _submit,
                child: _busy
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child:
                            CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(s.completeEmailSubmit),
              ),
              TextButton(
                onPressed: _busy ? null : () => context.go('/login'),
                child: Text(s.completeEmailBackToLogin),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final s = context.l10n;
    final email = _email.text.trim();
    if (email.isEmpty || !email.contains('@')) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(s.completeEmailInvalid)),
      );
      return;
    }
    setState(() => _busy = true);
    try {
      final social = ref.read(socialAuthProvider);
      final locale = Localizations.localeOf(context).languageCode;
      final filed = await social.completeEmail(
        token: widget.token,
        email: email,
        locale: locale,
      );
      if (mounted) setState(() => _submittedEmail = filed);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e, s)
                  : s.completeEmailFailed)),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}
