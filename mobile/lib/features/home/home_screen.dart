// Home feed: READY tracks with AI tags, tap to play.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';

import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../player/player_controller.dart';
import '../player/player_screen.dart';

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final feed = ref.watch(homeTracksProvider);
    return Scaffold(
      appBar: AppBar(
        leading: Padding(
          padding: const EdgeInsets.all(10),
          child: SvgPicture.asset('assets/logo.svg'),
        ),
        title: const Text('AVIBE'),
        actions: [
          IconButton(
            icon: const Icon(Icons.search),
            onPressed: () => context.go('/search'),
          ),
          IconButton(
            icon: const Icon(Icons.info_outline),
            onPressed: () => context.go('/legal'),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: feed.when(
              data: (tracks) => tracks.isEmpty
                  ? const Center(child: Text('아직 공개된 곡이 없습니다.'))
                  : RefreshIndicator(
                      onRefresh: () async =>
                          ref.invalidate(homeTracksProvider),
                      child: ListView.builder(
                        itemCount: tracks.length,
                        itemBuilder: (context, i) =>
                            TrackTile(track: tracks[i], queue: tracks),
                      ),
                    ),
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(
                child: Text(e is ApiException
                    ? friendlyMessage(e)
                    : '피드를 불러오지 못했습니다.'),
              ),
            ),
          ),
          const MiniPlayer(),
        ],
      ),
    );
  }
}

/// Shared track row with the AI-generated tag.
class TrackTile extends ConsumerWidget {
  const TrackTile({super.key, required this.track, this.queue});

  final Track track;
  final List<Track>? queue;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scheme = Theme.of(context).colorScheme;
    return ListTile(
      leading: CircleAvatar(
        backgroundColor: scheme.primaryContainer,
        child: Text(
          track.title.isEmpty ? '?' : track.title[0],
          style: TextStyle(color: scheme.onPrimaryContainer),
        ),
      ),
      title: Text(track.title, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Wrap(
        spacing: 6,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          if (track.aiGenerated)
            Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: scheme.primaryContainer,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                'AI-generated',
                style: TextStyle(
                  fontSize: 11,
                  color: scheme.onPrimaryContainer,
                ),
              ),
            ),
          Text(track.aiProvider,
              style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
      trailing: track.isTakenDown
          ? const Icon(Icons.block, color: Colors.redAccent)
          : Text(
              _formatDuration(track.durationMs),
              style: Theme.of(context).textTheme.bodySmall,
            ),
      enabled: !track.isTakenDown,
      onTap: track.isTakenDown
          ? null
          : () => playTracks(
                ref,
                tracks: queue ?? [track],
                initialIndex: queue == null ? 0 : queue!.indexOf(track),
              ),
    );
  }
}

String _formatDuration(int? ms) {
  if (ms == null || ms <= 0) return '--:--';
  final d = Duration(milliseconds: ms);
  final m = d.inMinutes;
  final s = d.inSeconds % 60;
  return '$m:${s.toString().padLeft(2, '0')}';
}
