// Legal: terms list/detail, AI notice, strike blocks, withdrawal guidance.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../player/player_screen.dart';

class LegalListScreen extends ConsumerWidget {
  const LegalListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final terms = ref.watch(legalTermsProvider);
    return Scaffold(
      appBar: AppBar(title: Text(s.legalTitle)),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              children: [
                ListTile(
                  leading: const Icon(Icons.smart_toy_outlined),
                  title: Text(s.legalAiTitle),
                  subtitle: Text(s.legalAiDesc),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/legal/ai-notice'),
                ),
                ListTile(
                  leading: const Icon(Icons.block),
                  title: Text(s.legalBlocksTitle),
                  subtitle: Text(s.legalBlocksDesc),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/legal/blocks'),
                ),
                ListTile(
                  leading: const Icon(Icons.person_remove_outlined),
                  title: Text(s.legalWithdrawalTitle),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/legal/withdrawal'),
                ),
                const Divider(),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                  child: Text(s.legalTermsHeader,
                      style: const TextStyle(fontWeight: FontWeight.bold)),
                ),
                terms.when(
                  data: (items) => Column(
                    children: [
                      for (final t in items)
                        ListTile(
                          title: Text(t.titleKo),
                          subtitle: Text('${t.code} · v${t.version}'
                              '${t.isRequired ? ' · ${s.legalRequired}' : ''}'),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => context.go('/legal/terms/${t.code}'),
                        ),
                    ],
                  ),
                  loading: () => const Padding(
                    padding: EdgeInsets.all(24),
                    child: Center(child: CircularProgressIndicator()),
                  ),
                  error: (e, _) => Padding(
                    padding: const EdgeInsets.all(24),
                    child: Text(e is ApiException
                        ? friendlyMessage(e, s)
                        : s.legalLoadFailed),
                  ),
                ),
              ],
            ),
          ),
          const MiniPlayer(),
        ],
      ),
    );
  }
}

class TermDetailScreen extends ConsumerWidget {
  const TermDetailScreen({super.key, required this.code});

  final String code;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final term = ref.watch(_termProvider(code));
    return Scaffold(
      appBar: AppBar(title: Text(code)),
      body: term.when(
        data: (t) => SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(t.titleKo,
                  style: Theme.of(context).textTheme.headlineSmall),
              const SizedBox(height: 4),
              Text(
                  s.legalVersion(t.version) +
                      (t.isRequired ? s.legalConsentRequired : ''),
                  style: Theme.of(context).textTheme.bodySmall),
              const Divider(height: 24),
              SelectableText(t.bodyMd),
              if (t.isRequired) ...[
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: () => _consent(context, ref, t),
                  child: Text(s.legalConsent),
                ),
              ],
            ],
          ),
        ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Text(e is ApiException
              ? friendlyMessage(e, s)
              : s.legalLoadFailed),
        ),
      ),
    );
  }

  Future<void> _consent(
      BuildContext context, WidgetRef ref, LegalTerm t) async {
    try {
      await ref.read(apiProvider).createConsent(t.id);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(context.l10n.legalConsentDone)),
        );
      }
    } catch (e) {
      if (context.mounted) {
        final s = context.l10n;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e, s)
                  : s.legalConsentFailed)),
        );
      }
    }
  }
}

final _termProvider = FutureProvider.family<LegalTerm, String>((ref, code) {
  return ref.watch(apiProvider).getTerm(code);
});

/// Static AI disclosure notice (platform-level guidance).
class AiNoticeScreen extends StatelessWidget {
  const AiNoticeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final s = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(s.legalAiTitle)),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(s.legalAiHeading,
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 12),
            Text(s.legalAiP1),
            const SizedBox(height: 12),
            Text(s.legalAiP2),
            const SizedBox(height: 24),
            OutlinedButton(
              onPressed: () => context.go('/legal/terms/${TermCodes.tos}'),
              child: Text(s.legalViewTos),
            ),
          ],
        ),
      ),
    );
  }
}

/// Strike blocks from GET /disputes/blocks.
class BlocksScreen extends ConsumerWidget {
  const BlocksScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final blocks = ref.watch(myBlocksProvider);
    return Scaffold(
      appBar: AppBar(title: Text(s.legalBlocksTitle)),
      body: blocks.when(
        data: (items) => items.isEmpty
            ? Center(child: Text(s.legalNoBlocks))
            : ListView.builder(
                itemCount: items.length,
                itemBuilder: (context, i) {
                  final b = items[i];
                  return ListTile(
                    leading:
                        const Icon(Icons.block, color: Colors.redAccent),
                    title: Text(s.legalStrikeItem(
                        b.strikeNumber.toString(), b.reason)),
                    subtitle: Text(b.expiresAt == null
                        ? s.legalNoExpiry
                        : s.legalExpires(
                            b.expiresAt!.toIso8601String().split('T').first)),
                  );
                },
              ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Text(e is ApiException
              ? friendlyMessage(e, s)
              : s.legalBlocksFailed),
        ),
      ),
    );
  }
}

/// Immediate self-withdrawal screen (App Store / Google Play store compliance).
class WithdrawalScreen extends ConsumerWidget {
  const WithdrawalScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(s.legalWithdrawalTitle)),
      body: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(s.legalWithdrawalHeading,
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 12),
            Text(s.legalWithdrawalBody),
            const SizedBox(height: 24),
            Row(
              children: [
                OutlinedButton(
                  onPressed: () async {
                    await ref.read(authRepositoryProvider).logout();
                    ref.invalidate(authMeProvider);
                    if (context.mounted) context.go('/login');
                  },
                  child: Text(s.legalLogout),
                ),
                const SizedBox(width: 12),
                FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: Colors.redAccent,
                  ),
                  onPressed: () async {
                    try {
                      await ref.read(apiProvider).withdrawAccount();
                    } catch (_) {}
                    await ref.read(authRepositoryProvider).logout();
                    ref.invalidate(authMeProvider);
                    if (context.mounted) context.go('/login');
                  },
                  child: Text(s.legalWithdrawNow),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
