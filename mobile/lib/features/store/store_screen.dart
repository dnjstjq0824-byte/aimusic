// Store mode screen: pairing claim + scheduled queue playback for tablets.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'store_mode_controller.dart';

/// Dedicated store-tablet mode: claim by pairing code, then play the
/// schedule-gated queue with royalty attribution heartbeats.
class StoreScreen extends ConsumerStatefulWidget {
  const StoreScreen({super.key});

  @override
  ConsumerState<StoreScreen> createState() => _StoreScreenState();
}

class _StoreScreenState extends ConsumerState<StoreScreen> {
  final _codeController = TextEditingController();

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(storeModeProvider);
    final controller = ref.read(storeModeProvider.notifier);
    return Scaffold(
      appBar: AppBar(
        title: const Text('매장 모드'),
        actions: [
          if (!state.needsClaim)
            IconButton(
              icon: const Icon(Icons.refresh),
              tooltip: '큐 새로고침',
              onPressed: controller.refreshQueue,
            ),
          if (!state.needsClaim)
            IconButton(
              icon: const Icon(Icons.logout),
              tooltip: '페어링 해제',
              onPressed: () => showDialog<void>(
                context: context,
                builder: (dialogContext) => AlertDialog(
                  title: const Text('페어링 해제'),
                  content: const Text('이 태블릿의 매장 페어링을 해제하고 재생을 중지합니다.'),
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.of(dialogContext).pop(),
                      child: const Text('취소'),
                    ),
                    TextButton(
                      onPressed: () {
                        Navigator.of(dialogContext).pop();
                        controller.unclaim();
                      },
                      child: const Text('해제'),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
      body: state.needsClaim
          ? _ClaimView(
              codeController: _codeController,
              loading: state.loading,
              error: state.error,
              onClaim: () => controller.claim(_codeController.text),
            )
          : _QueueView(state: state),
    );
  }
}

class _ClaimView extends StatelessWidget {
  const _ClaimView({
    required this.codeController,
    required this.loading,
    required this.error,
    required this.onClaim,
  });

  final TextEditingController codeController;
  final bool loading;
  final String? error;
  final VoidCallback onClaim;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Icon(Icons.storefront, size: 64),
          const SizedBox(height: 16),
          const Text(
            '파트너 콘솔의 매장 관리에서 발급된 8자리 페어링 코드를 입력하세요.',
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 24),
          TextField(
            controller: codeController,
            decoration: const InputDecoration(
              labelText: '페어링 코드',
              hintText: '예: A1B2C3D4',
              border: OutlineInputBorder(),
            ),
            textCapitalization: TextCapitalization.characters,
            inputFormatters: [
              FilteringTextInputFormatter.allow(RegExp('[A-Za-z0-9]')),
              LengthLimitingTextInputFormatter(8),
            ],
            onSubmitted: (_) => onClaim(),
          ),
          if (error != null) ...[
            const SizedBox(height: 12),
            Text(
              error!,
              textAlign: TextAlign.center,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 16),
          FilledButton(
            onPressed: loading ? null : onClaim,
            child: loading
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('페어링'),
          ),
        ],
      ),
    );
  }
}

class _QueueView extends ConsumerWidget {
  const _QueueView({required this.state});

  final StoreModeState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final controller = ref.read(storeModeProvider.notifier);
    final queue = state.queue;
    final current = state.currentTrack;
    return Column(
      children: [
        if (queue != null)
          Container(
            width: double.infinity,
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            color: queue.open
                ? Colors.green.withValues(alpha: 0.12)
                : Colors.orange.withValues(alpha: 0.12),
            child: Text(
              queue.open
                  ? '${queue.businessName} · 영업 중${queue.playlistName == null ? '' : ' · ${queue.playlistName}'}'
                  : '${queue.businessName} · ${queue.reason}',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        if (state.error != null)
          Container(
            width: double.infinity,
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            color: Theme.of(context).colorScheme.errorContainer,
            child: Text(
              state.error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.onErrorContainer,
              ),
            ),
          ),
        Expanded(
          child: queue == null
              ? const Center(child: CircularProgressIndicator())
              : queue.tracks.isEmpty
                  ? const Center(child: Text('재생할 곡이 없습니다.'))
                  : ListView.builder(
                      itemCount: queue.tracks.length,
                      itemBuilder: (context, i) {
                        final t = queue.tracks[i];
                        final selected =
                            queue.tracks.isNotEmpty &&
                            i == state.trackIndex % queue.tracks.length;
                        return ListTile(
                          selected: selected,
                          leading: Icon(
                            selected && state.playing
                                ? Icons.volume_up
                                : Icons.music_note,
                          ),
                          title: Text(t.title),
                          subtitle: t.durationMs == null
                              ? null
                              : Text(_formatDuration(t.durationMs!)),
                          onTap: () => controller.playIndex(i),
                        );
                      },
                    ),
        ),
        SafeArea(
          child: Container(
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              border: Border(
                top: BorderSide(color: Theme.of(context).dividerColor),
              ),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    current?.title ?? '선택된 곡 없음',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.skip_previous),
                  tooltip: '이전 곡',
                  onPressed:
                      (queue?.tracks.isNotEmpty ?? false) && !state.loading
                          ? controller.previous
                          : null,
                ),
                IconButton(
                  icon: state.loading
                      ? const SizedBox(
                          width: 24,
                          height: 24,
                          child:
                              CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Icon(state.playing
                          ? Icons.pause_circle_filled
                          : Icons.play_circle_filled),
                  iconSize: 40,
                  tooltip: state.playing ? '일시정지' : '재생',
                  onPressed:
                      (queue?.tracks.isNotEmpty ?? false) && !state.loading
                          ? controller.togglePlay
                          : null,
                ),
                IconButton(
                  icon: const Icon(Icons.skip_next),
                  tooltip: '다음 곡',
                  onPressed:
                      (queue?.tracks.isNotEmpty ?? false) && !state.loading
                          ? controller.next
                          : null,
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

String _formatDuration(int durationMs) {
  final totalSecs = durationMs ~/ 1000;
  final minutes = totalSecs ~/ 60;
  final seconds = totalSecs % 60;
  return '$minutes:${seconds.toString().padLeft(2, '0')}';
}
