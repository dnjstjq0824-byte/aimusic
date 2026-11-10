// Blocking 14+ age confirmation gate (social signups land without it).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/error.dart';
import '../../core/providers.dart';

bool _gateOpen = false;

/// Shows the gate exactly once per app state change; call from a listener.
Future<void> maybeShowAgeGate(
  BuildContext context,
  WidgetRef ref,
  bool needsConfirm,
) async {
  if (!needsConfirm || !context.mounted || _gateOpen) return;
  _gateOpen = true;
  try {
    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => const _AgeGateDialog(),
    );
  } finally {
    _gateOpen = false;
  }
}

class _AgeGateDialog extends ConsumerStatefulWidget {
  const _AgeGateDialog();

  @override
  ConsumerState<_AgeGateDialog> createState() => _AgeGateDialogState();
}

class _AgeGateDialogState extends ConsumerState<_AgeGateDialog> {
  bool _confirmed = false;
  bool _busy = false;
  final _birthYear = TextEditingController();

  @override
  void dispose() {
    _birthYear.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() => _busy = true);
    try {
      await ref.read(authRepositoryProvider).confirmAge(
            birthYear: int.tryParse(_birthYear.text.trim()),
          );
      ref.invalidate(authMeProvider);
      if (mounted) Navigator.of(context).pop();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '연령 확인에 실패했습니다.')),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: false,
      child: AlertDialog(
        title: const Text('만 14세 이상 확인'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                '소셜 계정으로 처음 로그인했습니다. 스트리밍·결제를 이용하려면 연령 확인이 필요합니다.',
                style: TextStyle(fontSize: 13),
              ),
              const SizedBox(height: 12),
              CheckboxListTile(
                value: _confirmed,
                contentPadding: EdgeInsets.zero,
                controlAffinity: ListTileControlAffinity.leading,
                title: const Text('만 14세 이상입니다. (필수)',
                    style: TextStyle(fontSize: 13)),
                onChanged: (v) => setState(() => _confirmed = v ?? false),
              ),
              TextField(
                controller: _birthYear,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: '출생연도 (선택)',
                  hintText: '예: 1995',
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: _busy
                ? null
                : () async {
                    await ref.read(authRepositoryProvider).logout();
                    ref.invalidate(authMeProvider);
                    if (context.mounted) Navigator.of(context).pop();
                  },
            child: const Text('로그아웃'),
          ),
          FilledButton(
            onPressed: (_confirmed && !_busy) ? _submit : null,
            child: _busy
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('확인하고 시작하기'),
          ),
        ],
      ),
    );
  }
}
