// Player controller: bridges UI actions to AiMusicAudioHandler.
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/models.dart';
import '../../core/providers.dart';

/// Play [tracks] from [initialIndex], seeking to a saved resume position.
Future<void> playTracks(
  WidgetRef ref, {
  required List<Track> tracks,
  int initialIndex = 0,
}) async {
  final handler = await ref.read(audioHandlerProvider.future);
  final api = ref.read(apiProvider);
  await handler.setQueue(tracks, initialIndex: initialIndex);
  // Resume from the saved server-side position (best effort).
  if (initialIndex >= 0 && initialIndex < tracks.length) {
    try {
      final state = await api.getPlaybackState(tracks[initialIndex].id);
      if (state != null && state.positionMs > 5000) {
        await handler.seek(Duration(milliseconds: state.positionMs));
      }
    } catch (_) {
      // No saved state yet; start from the beginning.
    }
  }
  await handler.play();
}

/// Convenience: play a single track.
Future<void> playSingle(WidgetRef ref, Track track) =>
    playTracks(ref, tracks: [track]);

/// Queue state exposed to the mini/fullscreen player.
final queueProvider = StreamProvider<List<Track>>((ref) async* {
  final handler = await ref.watch(audioHandlerProvider.future);
  await for (final _ in handler.queue) {
    yield handler.tracks;
  }
});

/// Currently playing track id (from mediaItem stream).
final currentTrackIdProvider = StreamProvider<String?>((ref) async* {
  final handler = await ref.watch(audioHandlerProvider.future);
  await for (final item in handler.mediaItem) {
    yield item?.id;
  }
});
