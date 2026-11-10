// Login / register screen (POST /auth/login, POST /auth/register).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/error.dart';
import '../../core/providers.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _displayName = TextEditingController();
  final _birthYear = TextEditingController();
  final _inviteCode = TextEditingController();
  bool _registerMode = false;
  bool _busy = false;
  bool _ageConfirmed = false;
  bool _totpStep = false;
  final _totpCode = TextEditingController();

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _totpCode.dispose();
    _displayName.dispose();
    _birthYear.dispose();
    _inviteCode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('AVIBE 로그인')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              decoration: const InputDecoration(labelText: '이메일'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _password,
              obscureText: true,
              decoration: const InputDecoration(
                labelText: '비밀번호 (8자 이상)',
              ),
            ),
            if (_registerMode) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _displayName,
                decoration: const InputDecoration(labelText: '표시 이름'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _birthYear,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: '출생연도 (선택)',
                  hintText: '예: 1995',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _inviteCode,
                decoration: const InputDecoration(
                  labelText: '초대 코드 (필요 시)',
                ),
              ),
              const SizedBox(height: 8),
              CheckboxListTile(
                value: _ageConfirmed,
                contentPadding: EdgeInsets.zero,
                controlAffinity: ListTileControlAffinity.leading,
                title: const Text(
                  '만 14세 이상입니다. (필수, 만 14세 미만은 가입 불가)',
                  style: TextStyle(fontSize: 13),
                ),
                onChanged: (v) => setState(() => _ageConfirmed = v ?? false),
              ),
            ],
            if (_totpStep && !_registerMode) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _totpCode,
                keyboardType: TextInputType.text,
                decoration: const InputDecoration(
                  labelText: '2단계 인증 코드',
                  hintText: '인증 앱 6자리 또는 복구 코드',
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
                  : Text(_registerMode
                      ? '회원가입'
                      : (_totpStep ? '인증하고 로그인' : '로그인')),
            ),
            TextButton(
              onPressed: _busy
                  ? null
                  : () => setState(() => _registerMode = !_registerMode),
              child: Text(_registerMode ? '이미 계정이 있어요' : '계정 만들기'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    if (_registerMode && !_ageConfirmed) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('만 14세 이상 확인이 필요합니다.')),
      );
      return;
    }
    setState(() => _busy = true);
    try {
      final repo = ref.read(authRepositoryProvider);
      if (_registerMode) {
        await repo.register(
          email: _email.text.trim(),
          password: _password.text,
          displayName: _displayName.text.trim(),
          birthYear: int.tryParse(_birthYear.text.trim()),
          ageConfirmed: _ageConfirmed,
          inviteCode: _inviteCode.text.trim().isEmpty ? null : _inviteCode.text.trim(),
        );
      } else {
        await repo.login(
          email: _email.text.trim(),
          password: _password.text,
          totpCode: _totpStep ? _totpCode.text.trim() : null,
        );
      }
      ref.invalidate(authMeProvider);
      if (mounted) {
        // New/updated required terms gate commerce: route to consent first.
        try {
          final pending = await ref.read(apiProvider).pendingConsents();
          if (mounted) context.go(pending.isEmpty ? '/' : '/legal');
        } catch (_) {
          if (mounted) context.go('/');
        }
      }
    } catch (e) {
      if (e is ApiException &&
          e.code == ApiCodes.totpRequired &&
          !_totpStep &&
          mounted) {
        setState(() => _totpStep = true);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('인증 앱의 6자리 코드를 입력하세요.')),
        );
      } else if (mounted) {
        if (e is ApiException && e.code == ApiCodes.totpInvalid) {
          _totpCode.clear();
        }
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '인증에 실패했습니다.')),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}
