// Settings: language selection + server sync, legal, logout (WI-R12).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/providers.dart';

class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});

  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  String? _saving;

  @override
  Widget build(BuildContext context) {
    final s = context.l10n;
    final override = ref.watch(localeOverrideProvider);
    final me = ref.watch(authMeProvider).valueOrNull;
    final current =
        override ?? resolveDeviceLanguage(Localizations.localeOf(context));
    return Scaffold(
      appBar: AppBar(title: Text(s.settingsTitle)),
      body: ListView(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Text(s.settingsLanguage,
                style: const TextStyle(fontWeight: FontWeight.bold)),
          ),
          RadioGroup<String>(
            groupValue: current,
            onChanged: (v) {
              if (_saving == null && v != null) _apply(context, ref, v);
            },
            child: Column(
              children: [
                for (final code in appLanguageCodes)
                  RadioListTile<String>(
                    value: code,
                    title: Row(
                      children: [
                        Text(languageName(s, code)),
                        if (appBetaLanguages.contains(code)) ...[
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              borderRadius: BorderRadius.circular(999),
                              border: Border.all(
                                  color: Theme.of(context)
                                      .colorScheme
                                      .outline),
                            ),
                            child: Text(s.commonBeta,
                                style: Theme.of(context)
                                    .textTheme
                                    .bodySmall),
                          ),
                        ],
                      ],
                    ),
                    secondary: _saving == code
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                                strokeWidth: 2),
                          )
                        : null,
                  ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
            child: Text(s.settingsServerNote,
                style: Theme.of(context).textTheme.bodySmall),
          ),
          const Divider(),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: Text(s.settingsLegal),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.go('/legal'),
          ),
          if (me != null)
            ListTile(
              leading: const Icon(Icons.logout),
              title: Text(s.settingsLogout),
              onTap: () async {
                await ref.read(authRepositoryProvider).logout();
                ref.invalidate(authMeProvider);
                if (context.mounted) context.go('/login');
              },
            ),
        ],
      ),
    );
  }

  Future<void> _apply(
      BuildContext context, WidgetRef ref, String code) async {
    final s = context.l10n;
    setState(() => _saving = code);
    try {
      final ok =
          await ref.read(localeOverrideProvider.notifier).apply(code);
      if (!ok) throw const FormatException('persist');
      // Server sync when signed in (A2); local choice wins on failure.
      if (await ref.read(authRepositoryProvider).isLoggedIn) {
        try {
          await ref.read(apiProvider).updateLocale(toServerLocale(code));
        } catch (e) {
          if (context.mounted && e is ApiException) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(friendlyMessage(e, s))),
            );
          }
        }
      }
      ref.invalidate(authMeProvider);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(s.settingsLanguageSaved)),
        );
      }
    } catch (_) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(s.settingsLanguageFailed)),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = null);
    }
  }
}
