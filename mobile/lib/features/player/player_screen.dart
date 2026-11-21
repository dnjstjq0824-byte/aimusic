// Mini + fullscreen player: waveform, lyrics, AI transparency, report/block.
library;

import 'dart:convert';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';

/// Persistent mini player bar shown above the bottom navigation.
class MiniPlayer extends ConsumerWidget {
  const MiniPlayer({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final handlerAsync = ref.watch(audioHandlerProvider);
    return handlerAsync.when(
      data: (handler) => StreamBuilder<MediaItem?>(
        stream: handler.mediaItem,
        builder: (context, itemSnap) {
          final item = itemSnap.data;
          if (item == null) return const SizedBox.shrink();
          return StreamBuilder<PlaybackState>(
            stream: handler.playbackState,
            builder: (context, stateSnap) {
              final playing = stateSnap.data?.playing ?? false;
              return GestureDetector(
                onTap: () => context.go('/player'),
                child: Container(
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.surfaceContainerHigh,
                    border: Border(
                      top: BorderSide(
                        color: Theme.of(context).dividerColor,
                      ),
                    ),
                  ),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 8),
                  child: Row(
                    children: [
                      const Icon(Icons.music_note),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              item.title,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                  fontWeight: FontWeight.w600),
                            ),
                            Text(
                              item.album ?? '',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style:
                                  Theme.of(context).textTheme.bodySmall,
                            ),
                          ],
                        ),
                      ),
                      IconButton(
                        icon: Icon(playing
                            ? Icons.pause
                            : Icons.play_arrow),
                        onPressed: () =>
                            playing ? handler.pause() : handler.play(),
                      ),
                      IconButton(
                        icon: const Icon(Icons.skip_next),
                        onPressed: () => handler.skipToNext(),
                      ),
                    ],
                  ),
                ),
              );
            },
          );
        },
      ),
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}

/// Fullscreen player with waveform, lyrics, AI/license, and report menu.
class PlayerScreen extends ConsumerStatefulWidget {
  const PlayerScreen({super.key});

  @override
  ConsumerState<PlayerScreen> createState() => _PlayerScreenState();
}

class _PlayerScreenState extends ConsumerState<PlayerScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
  }

  @override
  void dispose() {
    _tabs.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.l10n;
    final handlerAsync = ref.watch(audioHandlerProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.playerTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.flag_outlined),
            tooltip: s.playerReportBlock,
            onPressed: () => _showReportMenu(context, ref),
          ),
        ],
        bottom: TabBar(
          controller: _tabs,
          tabs: [
            Tab(text: s.playerTabWaveform),
            Tab(text: s.playerTabLyrics),
            Tab(text: s.playerTabAi),
          ],
        ),
      ),
      body: handlerAsync.when(
        data: (handler) => StreamBuilder<MediaItem?>(
          stream: handler.mediaItem,
          builder: (context, itemSnap) {
            final item = itemSnap.data;
            if (item == null) {
              return Center(child: Text(s.playerNoTrack));
            }
            return Column(
              children: [
                Expanded(
                  child: TabBarView(
                    controller: _tabs,
                    children: [
                      _WaveformTab(trackId: item.id),
                      _LyricsTab(trackId: item.id),
                      _AiLicenseTab(item: item),
                    ],
                  ),
                ),
                _TransportControls(item: item),
              ],
            );
          },
        ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(s.playerError(e.toString()))),
      ),
    );
  }

  Future<void> _showReportMenu(BuildContext context, WidgetRef ref) async {
    final handler = await ref.read(audioHandlerProvider.future);
    final item = handler.mediaItem.value;
    if (item == null || !context.mounted) return;
    final s = context.l10n;
    final api = ref.read(apiProvider);
    final reason = await showModalBottomSheet<String>(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(title: Text(s.playerReportReasonTitle)),
            for (final r in DisputeReasons.all)
              ListTile(
                title: Text(DisputeReasons.labelOf(r, s)),
                onTap: () => Navigator.pop(context, r),
              ),
            ListTile(
              leading: const Icon(Icons.person_off_outlined),
              title: Text(s.playerBlockArtist),
              onTap: () => Navigator.pop(context, '__BLOCK_ARTIST__'),
            ),
            ListTile(
              leading: const Icon(Icons.block),
              title: Text(s.playerViewBlocks),
              onTap: () {
                Navigator.pop(context);
                context.go('/legal/blocks');
              },
            ),
          ],
        ),
      ),
    );
    if (reason == null || !context.mounted) return;
    if (reason == '__BLOCK_ARTIST__') {
      try {
        final track = await api.getTrack(item.id);
        await api.blockArtist(
          partnerId: track.partnerId,
          reason: s.playerBlockReasonDefault,
        );
        await handler.removeTracksByPartner(track.partnerId);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(s.playerArtistBlocked)),
          );
        }
      } catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(
                e is ApiException
                    ? friendlyMessage(e, s)
                    : s.playerBlockFailed,
              ),
            ),
          );
        }
      }
      return;
    }
    final descCtrl = TextEditingController();
    final confirm = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(s.playerReportTitle),
        content: TextField(
          controller: descCtrl,
          decoration: InputDecoration(
            labelText: s.playerReportDetailLabel,
            hintText: s.playerReportDetailHint,
          ),
          maxLines: 3,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(s.playerCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(s.playerReport),
          ),
        ],
      ),
    );
    if (confirm != true || !context.mounted) return;
    try {
      final dispute = await api.reportTrack(
        trackId: item.id,
        reason: reason,
        description: descCtrl.text.trim().isEmpty
            ? null
            : descCtrl.text.trim(),
      );
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(s.playerReportReceived(dispute.status))),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e, s)
                  : s.playerReportFailed)),
        );
      }
    }
  }
}

/// Transport: seek bar, play/pause, next/prev, shuffle/repeat.
class _TransportControls extends ConsumerWidget {
  const _TransportControls({required this.item});

  final MediaItem item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final handlerAsync = ref.watch(audioHandlerProvider);
    return handlerAsync.when(
      data: (handler) => StreamBuilder<PlaybackState>(
        stream: handler.playbackState,
        builder: (context, snap) {
          final state = snap.data;
          final pos = state?.updatePosition ?? Duration.zero;
          final total = item.duration ?? Duration.zero;
          final playing = state?.playing ?? false;
          final shuffle =
              state?.shuffleMode == AudioServiceShuffleMode.all;
          final repeat = state?.repeatMode ?? AudioServiceRepeatMode.none;
          return Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(item.title,
                    style:
                        Theme.of(context).textTheme.titleLarge),
                const SizedBox(height: 4),
                Text(item.album ?? '',
                    style: Theme.of(context).textTheme.bodyMedium),
                Slider(
                  value: total.inMilliseconds == 0
                      ? 0
                      : pos.inMilliseconds
                          .clamp(0, total.inMilliseconds)
                          .toDouble(),
                  max: (total.inMilliseconds <= 0
                          ? 1
                          : total.inMilliseconds)
                      .toDouble(),
                  onChanged: (v) => handler
                      .seek(Duration(milliseconds: v.toInt())),
                ),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                  children: [
                    IconButton(
                      icon: Icon(
                        Icons.shuffle,
                        color: shuffle
                            ? Theme.of(context).colorScheme.primary
                            : null,
                      ),
                      onPressed: () => handler.setShuffleMode(
                        shuffle
                            ? AudioServiceShuffleMode.none
                            : AudioServiceShuffleMode.all,
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.skip_previous, size: 32),
                      onPressed: () => handler.skipToPrevious(),
                    ),
                    IconButton.filled(
                      icon: Icon(
                          playing ? Icons.pause : Icons.play_arrow,
                          size: 32),
                      onPressed: () =>
                          playing ? handler.pause() : handler.play(),
                    ),
                    IconButton(
                      icon: const Icon(Icons.skip_next, size: 32),
                      onPressed: () => handler.skipToNext(),
                    ),
                    IconButton(
                      icon: Icon(
                        repeat == AudioServiceRepeatMode.one
                            ? Icons.repeat_one
                            : Icons.repeat,
                        color: repeat == AudioServiceRepeatMode.none
                            ? null
                            : Theme.of(context).colorScheme.primary,
                      ),
                      onPressed: () => handler.setRepeatMode(
                        switch (repeat) {
                          AudioServiceRepeatMode.none =>
                            AudioServiceRepeatMode.all,
                          AudioServiceRepeatMode.all =>
                            AudioServiceRepeatMode.one,
                          _ => AudioServiceRepeatMode.none,
                        },
                      ),
                    ),
                  ],
                ),
              ],
            ),
          );
        },
      ),
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}

/// Real waveform from track.waveformJson with deterministic fallback.
class _WaveformTab extends ConsumerWidget {
  const _WaveformTab({required this.trackId});

  final String trackId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final detail = ref.watch(_trackDetailProvider(trackId));
    final bars = detail.maybeWhen(
      data: (track) => _parseWaveform(track.waveformJson) ?? _barsFor(trackId, 64),
      orElse: () => _barsFor(trackId, 64),
    );
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          SizedBox(
            height: 120,
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                for (var i = 0; i < bars.length; i++)
                  Expanded(
                    child: Container(
                      margin: const EdgeInsets.symmetric(horizontal: 1),
                      height: 12 + bars[i].clamp(0.05, 1.0) * 108,
                      decoration: BoxDecoration(
                        color: Theme.of(context)
                            .colorScheme
                            .primary
                            .withValues(alpha: 0.25 + bars[i].clamp(0.05, 1.0) * 0.6),
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Text(s.playerWaveformTitle,
              style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
    );
  }

  List<double>? _parseWaveform(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is List && decoded.isNotEmpty) {
        return decoded
            .map((e) => (e is num) ? e.toDouble() : 0.3)
            .toList();
      }
    } catch (_) {}
    return null;
  }

  List<double> _barsFor(String seed, int n) {
    var h = 0x811c9dc5;
    for (final c in seed.codeUnits) {
      h ^= c;
      h = (h * 0x01000193) & 0xffffffff;
    }
    return List.generate(n, (i) {
      h = (h * 1103515245 + 12345 + i * 97) & 0x7fffffff;
      return ((h % 1000) / 1000 * 0.85) + 0.15;
    });
  }
}

/// Lyrics & AI Prompt tab.
class _LyricsTab extends ConsumerWidget {
  const _LyricsTab({required this.trackId});

  final String trackId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final detail = ref.watch(_trackDetailProvider(trackId));
    return detail.when(
      data: (track) => SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(track.title,
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 8),
            Text(
              track.lyrics != null && track.lyrics!.trim().isNotEmpty
                  ? track.lyrics!
                  : (track.isInstrumental
                      ? s.playerInstrumental
                      : s.playerNoLyrics(track.aiProvider)),
              style: Theme.of(context).textTheme.bodyLarge,
            ),
            if (track.aiPrompt != null &&
                track.aiPrompt!.trim().isNotEmpty &&
                track.isPromptPublic) ...[
              const SizedBox(height: 16),
              Text(s.playerPromptLabel(track.aiPrompt!),
                  style: Theme.of(context).textTheme.bodySmall),
            ],
            if (track.aiDisclosureText != null) ...[
              const SizedBox(height: 16),
              Text(track.aiDisclosureText!,
                  style: Theme.of(context).textTheme.bodyMedium),
            ],
          ],
        ),
      ),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (_, __) => Center(child: Text(s.playerTrackFailed)),
    );
  }
}

/// AI transparency + license tab from the MediaItem extras / track detail.
class _AiLicenseTab extends ConsumerWidget {
  const _AiLicenseTab({required this.item});

  final MediaItem item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final detail = ref.watch(_trackDetailProvider(item.id));
    return detail.when(
      data: (track) => ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _row(context, s.playerAiLabel,
              track.aiGenerated ? s.playerAiYes : s.playerAiNo),
          _row(context, s.playerProviderLabel, track.aiProvider),
          _row(context, s.playerDisclosureLabel,
              track.aiDisclosureText ?? s.playerNoDisclosure),
          _row(context, s.playerBitrateLabel,
              track.bitrateKbps == null ? '-' : '${track.bitrateKbps} kbps'),
          _row(context, 'BPM', track.bpm == null ? '-' : '${track.bpm}'),
          _row(context, s.playerKeyLabel, track.musicalKey ?? '-'),
          _row(context, 'ISRC', track.isrc ?? '-'),
          _row(context, s.playerPlayCountLabel, '${track.playCount}'),
          _row(context, s.playerStatusLabel, track.status),
          if (track.certNumber != null)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                children: [
                  SizedBox(
                    width: 120,
                    child: Text(s.playerCertificateLabel,
                        style: Theme.of(context).textTheme.bodySmall),
                  ),
                  Expanded(
                    child: GestureDetector(
                      onTap: () => context.go(
                          '/certificate/${Uri.encodeComponent(track.certNumber!)}'),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            Icons.verified,
                            size: 18,
                            color:
                                Theme.of(context).colorScheme.primary,
                          ),
                          const SizedBox(width: 6),
                          Flexible(
                            child: Text(
                              track.certNumber!,
                              style: TextStyle(
                                fontWeight: FontWeight.w600,
                                color: Theme.of(context)
                                    .colorScheme
                                    .primary,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          const SizedBox(height: 12),
          Text(
            s.playerAiNotice,
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 8),
          OutlinedButton(
            onPressed: () => context.go('/legal'),
            child: Text(s.playerViewTerms),
          ),
        ],
      ),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (_, __) => Center(child: Text(s.playerTrackFailed)),
    );
  }

  Widget _row(BuildContext context, String label, String value) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 120,
              child: Text(label,
                  style: Theme.of(context).textTheme.bodySmall),
            ),
            Expanded(
              child: Text(value,
                  style: const TextStyle(fontWeight: FontWeight.w600)),
            ),
          ],
        ),
      );
}

final _trackDetailProvider =
    FutureProvider.family<Track, String>((ref, id) async {
  return ref.watch(apiProvider).getTrack(id);
});
