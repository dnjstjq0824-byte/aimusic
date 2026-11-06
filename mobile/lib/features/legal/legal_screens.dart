// Legal: terms list/detail, AI notice, strike blocks, withdrawal guidance.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../player/player_screen.dart';

class LegalListScreen extends ConsumerWidget {
  const LegalListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final terms = ref.watch(legalTermsProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('약관 및 안내')),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              children: [
                ListTile(
                  leading: const Icon(Icons.smart_toy_outlined),
                  title: const Text('AI 생성물 안내'),
                  subtitle: const Text('AI 음원 표시 및 라이선스 고지'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/legal/ai-notice'),
                ),
                ListTile(
                  leading: const Icon(Icons.block),
                  title: const Text('내 차단 내역'),
                  subtitle: const Text('신고 스트라이크에 따른 이용 제한'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/legal/blocks'),
                ),
                ListTile(
                  leading: const Icon(Icons.person_remove_outlined),
                  title: const Text('회원 탈퇴 안내'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/legal/withdrawal'),
                ),
                const Divider(),
                const Padding(
                  padding: EdgeInsets.fromLTRB(16, 8, 16, 4),
                  child: Text('이용약관',
                      style: TextStyle(fontWeight: FontWeight.bold)),
                ),
                terms.when(
                  data: (items) => Column(
                    children: [
                      for (final t in items)
                        ListTile(
                          title: Text(t.titleKo),
                          subtitle: Text('${t.code} · v${t.version}'
                              '${t.isRequired ? ' · 필수' : ''}'),
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
                        ? friendlyMessage(e)
                        : '약관을 불러오지 못했습니다.'),
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
              Text('버전 ${t.version}${t.isRequired ? ' · 필수 동의' : ''}',
                  style: Theme.of(context).textTheme.bodySmall),
              const Divider(height: 24),
              SelectableText(t.bodyMd),
              if (t.isRequired) ...[
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: () => _consent(context, ref, t),
                  child: const Text('동의하기'),
                ),
              ],
            ],
          ),
        ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Text(e is ApiException
              ? friendlyMessage(e)
              : '약관을 불러오지 못했습니다.'),
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
          const SnackBar(content: Text('동의가 기록되었습니다.')),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '동의 처리에 실패했습니다.')),
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
    return Scaffold(
      appBar: AppBar(title: const Text('AI 생성물 안내')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('AI 음원 표시 안내',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 12),
            const Text(
              '이 플랫폼의 모든 음원은 AI 생성 여부가 표시됩니다. 재생 화면의 ‘AI·라이선스’ 탭에서 '
              '제공자(SUNO, UDIO, AIVA 등), 고지 문구, 비트레이트를 확인할 수 있습니다.',
            ),
            const SizedBox(height: 12),
            const Text(
              'AI 음원의 상업적 이용 범위는 각 제공자의 라이선스 정책과 플랫폼 이용약관을 따릅니다. '
              '무단 보이스 클론, 무단 샘플 사용이 의심되는 곡은 재생 화면의 신고 메뉴로 제보할 수 있습니다.',
            ),
            const SizedBox(height: 24),
            OutlinedButton(
              onPressed: () => context.go('/legal/terms/${TermCodes.tos}'),
              child: const Text('이용약관(TOS) 보기'),
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
    final blocks = ref.watch(myBlocksProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('내 차단 내역')),
      body: blocks.when(
        data: (items) => items.isEmpty
            ? const Center(child: Text('차단 내역이 없습니다.'))
            : ListView.builder(
                itemCount: items.length,
                itemBuilder: (context, i) {
                  final b = items[i];
                  return ListTile(
                    leading: const Icon(Icons.block, color: Colors.redAccent),
                    title: Text('스트라이크 ${b.strikeNumber} · ${b.reason}'),
                    subtitle: Text(b.expiresAt == null
                        ? '만료일 없음'
                        : '만료: ${b.expiresAt}'),
                  );
                },
              ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Text(e is ApiException
              ? friendlyMessage(e)
              : '차단 내역을 불러오지 못했습니다.'),
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
    return Scaffold(
      appBar: AppBar(title: const Text('회원 탈퇴 안내')),
      body: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('탈퇴 전 확인하세요',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 12),
            const Text(
              '· 플레이리스트와 재생 기록이 즉시 비활성화 및 삭제 처리됩니다.\n'
              '· 진행 중인 신고·분쟁이 있으면 관련 법령에 따라 보존 후 파기됩니다.\n'
              '· 아래 [즉시 회원 탈퇴] 버튼을 누르면 별도 고객센터 문의 없이 앱 내에서 즉시 탈퇴됩니다.',
            ),
            const SizedBox(height: 24),
            Row(
              children: [
                OutlinedButton(
                  onPressed: () async {
                    await ref.read(authRepositoryProvider).logout();
                    ref.invalidate(authMeProvider);
                    if (context.mounted) context.go('/login');
                  },
                  child: const Text('로그아웃'),
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
                  child: const Text('즉시 회원 탈퇴'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
