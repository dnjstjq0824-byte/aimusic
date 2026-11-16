// Store mode screen: pairing claim + scheduled queue playback for tablets.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_locale.dart';
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
    final s = context.l10n;
    final state = ref.watch(storeModeProvider);
    final controller = ref.read(storeModeProvider.notifier);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.storeTitle),
        actions: [
          if (!state.needsClaim)
            IconButton(
              icon: const Icon(Icons.refresh),
              tooltip: s.storeRefreshQueue,
              onPressed: controller.refreshQueue,
            ),
          if (!state.needsClaim)
            IconButton(
              icon: const Icon(Icons.logout),
              tooltip: s.storeUnpair,
              onPressed: () => showDialog<void>(
                context: context,
                builder: (dialogContext) => AlertDialog(
                  title: Text(s.storeUnpairTitle),
                  content: Text(s.storeUnpairBody),
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.of(dialogContext).pop(),
                      child: Text(s.storeCancel),
                    ),
                    TextButton(
                      onPressed: () {
                        Navigator.of(dialogContext).pop();
                        controller.unclaim();
                      },
                      child: Text(s.storeUnpairConfirm),
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
    final s = context.l10n;
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Icon(Icons.storefront, size: 64),
          const SizedBox(height: 16),
          Text(
            s.storePairGuide,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 24),
          TextField(
            controller: codeController,
            decoration: InputDecoration(
              labelText: s.storePairCodeLabel,
              hintText: s.storePairCodeHint,
              border: const OutlineInputBorder(),
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
                : Text(s.storePair),
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
    final s = context.l10n;
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
                  ? (queue.playlistName == null
                      ? s.storeOpenNow(queue.businessName)
                      : s.storeOpenNowPlaylist(
                          queue.businessName, queue.playlistName!))
                  : s.storeClosed(queue.businessName, queue.reason),
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
                  ? Center(child: Text(s.storeQueueEmpty))
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
                    current?.title ?? s.storeNoSelection,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.skip_previous),
                  tooltip: s.storePrev,
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
                  tooltip: state.playing ? s.storePause : s.storePlay,
                  onPressed:
                      (queue?.tracks.isNotEmpty ?? false) && !state.loading
                          ? controller.togglePlay
                          : null,
                ),
                IconButton(
                  icon: const Icon(Icons.skip_next),
                  tooltip: s.storeNext,
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
