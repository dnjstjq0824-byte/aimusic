// Background audio handler: gapless queue, takedown auto-skip, heartbeat.
library;

import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:audio_session/audio_session.dart';
import 'package:just_audio/just_audio.dart';

import '../core/env.dart';
import '../core/error.dart';
import '../core/models.dart';
import 'api_service.dart';

/// Periodic playback progress hook -> POST /playback/heartbeat.
typedef HeartbeatHook = Future<void> Function({
  required String trackId,
  required int positionMs,
  required int playMs,
  String? jti,
});

/// Resolves a signed HLS ticket for a track id.
typedef StreamResolver = Future<StreamTicket> Function(String trackId);

/// Position persistence hook -> PUT /playback/state/:trackId.
typedef PositionHook = Future<void> Function(String trackId, int positionMs);

/// Default hooks wired to [ApiService]; overridable in tests.
HeartbeatHook heartbeatHookFor(ApiService api) =>
    ({required String trackId,
      required int positionMs,
      required int playMs,
      String? jti}) =>
        api.heartbeat(
          trackId: trackId,
          positionMs: positionMs,
          playMs: playMs,
          jti: jti,
        );

PositionHook positionHookFor(ApiService api) =>
    (String trackId, int positionMs) =>
        api.setPlaybackState(trackId, positionMs);

/// Builds a lock-screen MediaItem with the AI-generated tag in [MediaItem.album].
MediaItem mediaItemFor(Track track, {String? artUri}) => MediaItem(
      id: track.id,
      title: track.title,
      album: track.subtitle,
      artist: track.aiProvider,
      duration: track.durationMs == null
          ? null
          : Duration(milliseconds: track.durationMs!),
      artUri: artUri == null ? null : Uri.tryParse(artUri),
      extras: {
        'aiGenerated': track.aiGenerated,
        'aiProvider': track.aiProvider,
        'aiDisclosureText': track.aiDisclosureText,
        'status': track.status,
      },
    );

/// App background audio handler.
///
/// - [ConcatenatingAudioSource] queue with gapless transitions.
/// - Shuffle / repeat wired from [AudioHandler.setShuffleMode] /
///   [AudioHandler.setRepeatMode].
/// - Takedown auto-skip: ERR_NOT_STREAMABLE / 403-410 on resolve skips to
///   the next item instead of surfacing an error state.
/// - Headset + media-button handling via audio_service callbacks and the
///   audio_session noisy-event stream (pause on unplug).
/// - Heartbeat hook: accumulates playMs and fires every
///   [Env.heartbeatIntervalMs] while playing.
class AiMusicAudioHandler extends BaseAudioHandler
    with QueueHandler, SeekHandler {
  AiMusicAudioHandler({
    required AudioPlayer player,
    required StreamResolver resolveStream,
    HeartbeatHook? onHeartbeat,
    PositionHook? onPosition,
    Duration heartbeatInterval = const Duration(
        milliseconds: Env.heartbeatIntervalMs),
  })  : _player = player,
        _resolveStream = resolveStream,
        _onHeartbeat = onHeartbeat,
        _onPosition = onPosition,
        _heartbeatInterval = heartbeatInterval {
    _playlist = ConcatenatingAudioSource(children: []);
    _wire();
  }

  final AudioPlayer _player;
  final StreamResolver _resolveStream;
  final HeartbeatHook? _onHeartbeat;
  final PositionHook? _onPosition;
  final Duration _heartbeatInterval;

  late final ConcatenatingAudioSource _playlist;

  /// Tracks currently in the queue, parallel to [_playlist.children].
  final List<Track> _tracks = [];

  /// Active HLS ticket per queue index (for jti + expiry refresh).
  final Map<int, StreamTicket> _tickets = {};

  /// Indices currently being resolved (re-entrancy guard for auto-skip).
  final Set<int> _resolving = {};

  Timer? _heartbeatTimer;
  int _playMsSinceHeartbeat = 0;
  DateTime? _lastTick;
  String? _lastNotifiedTrackId;

  AudioPlayer get player => _player;
  List<Track> get tracks => List.unmodifiable(_tracks);

  Future<void> _wire() async {
    // Audio session: music config + pause on headset unplug ("noisy").
    final session = await AudioSession.instance;
    await session.configure(const AudioSessionConfiguration.music());
    session.becomingNoisyEventStream.listen((_) {
      if (_player.playing) {
        pause();
      }
    });

    // Broadcast just_audio state as audio_service PlaybackState so the
    // notification, lock screen, and media buttons stay in sync.
    _player.playbackEventStream.listen(
      _broadcastState,
      onError: (Object e, StackTrace st) {
        // Playback errors (e.g. 403 on an expired HLS URL) trigger a
        // ticket refresh, falling back to auto-skip when unstreamable.
        final index = _player.currentIndex;
        if (index != null) {
          unawaited(_recoverPlaybackError(index));
        }
      },
    );
    _player.sequenceStateStream.listen((SequenceState? s) {
      if (s == null) return;
      final index = s.currentIndex;
      if (index >= 0 && index < _tracks.length) {
        final track = _tracks[index];
        mediaItem.add(mediaItemFor(track));
        if (_lastNotifiedTrackId != track.id) {
          _lastNotifiedTrackId = track.id;
          _persistPosition(track.id, _player.position.inMilliseconds);
        }
      }
    });
    _player.processingStateStream.listen((ProcessingState state) async {
      if (state == ProcessingState.completed) {
        await _onTrackCompleted();
      }
    });
  }

  // ---------- Queue management ----------

  /// Replaces the queue with [tracks], resolving the signed HLS URL for the
  /// item at [initialIndex] (others resolve lazily on advance).
  Future<void> setQueue(List<Track> tracks, {int initialIndex = 0}) async {
    await _player.stop();
    _tracks
      ..clear()
      ..addAll(tracks);
    _tickets.clear();
    await _playlist.clear();
    await _playlist.addAll(
      List.generate(tracks.length, (_) => _placeholderSource()),
    );
    queue.add(tracks.map(mediaItemFor).toList());
    try {
      await _player.setAudioSource(
        _playlist,
        initialIndex: initialIndex.clamp(0, tracks.length - 1),
        initialPosition: Duration.zero,
      );
    } catch (_) {
      // Placeholder has no duration; ignore preload errors here since the
      // real source is swapped in below.
    }
    if (tracks.isNotEmpty) {
      await _prepareIndex(initialIndex.clamp(0, tracks.length - 1));
    }
  }

  /// Appends one track to the end of the queue.
  Future<void> appendTrack(Track track) async {
    _tracks.add(track);
    await _playlist.add(_placeholderSource());
    queue.add(_tracks.map(mediaItemFor).toList());
  }

  /// Removes all tracks belonging to [partnerId] from the active queue immediately.
  Future<void> removeTracksByPartner(String partnerId) async {
    for (var i = _tracks.length - 1; i >= 0; i--) {
      if (_tracks[i].partnerId == partnerId) {
        await _skipUnstreamable(i);
      }
    }
  }

  AudioSource _placeholderSource() =>
      SilenceAudioSource(duration: const Duration(milliseconds: 200));

  /// Resolves the signed URL for [index] and swaps it into the playlist.
  /// Auto-skips forward while the item is not streamable (takedown).
  Future<void> _prepareIndex(int index) async {
    if (index < 0 || index >= _tracks.length || _resolving.contains(index)) {
      return;
    }
    _resolving.add(index);
    try {
      final ticket = await _resolveStream(_tracks[index].id);
      _tickets[index] = ticket;
      final source = HlsAudioSource(Uri.parse(ticket.url));
      // Keep playback position stable when swapping the current item.
      final isCurrent = _player.currentIndex == index;
      final pos = isCurrent ? _player.position : Duration.zero;
      await _playlist
          .removeAt(index)
          .then((_) => _playlist.insert(index, source));
      if (isCurrent && pos > Duration.zero) {
        await _player.seek(pos, index: index);
      }
    } on ApiException catch (e) {
      if (e.isNotStreamable) {
        await _skipUnstreamable(index);
      } else {
        rethrow;
      }
    } finally {
      _resolving.remove(index);
    }
  }

  /// Removes an unstreamable (taken down) item and advances playback.
  Future<void> _skipUnstreamable(int index) async {
    if (index < 0 || index >= _tracks.length) return;
    final wasCurrent = _player.currentIndex == index;
    _tracks.removeAt(index);
    _tickets.remove(index);
    if (_playlist.length > index) {
      await _playlist.removeAt(index);
    }
    queue.add(_tracks.map(mediaItemFor).toList());
    if (_tracks.isEmpty) {
      await stop();
      return;
    }
    if (wasCurrent) {
      final next = index.clamp(0, _tracks.length - 1);
      await _prepareIndex(next);
      await _player.seek(Duration.zero, index: next);
      if (_player.playing || playbackState.value.playing) {
        await _player.play();
      }
    }
  }

  Future<void> _recoverPlaybackError(int index) async {
    // Most likely an expired signed URL: refresh the ticket once.
    final fresh = _tickets[index];
    final expired = fresh == null ||
        DateTime.now().isAfter(
          fresh.expiresAt.subtract(
            const Duration(seconds: Env.streamRefreshMarginSecs),
          ),
        );
    if (!expired) {
      await _skipUnstreamable(index);
      return;
    }
    _tickets.remove(index);
    try {
      await _prepareIndex(index);
      await _player.play();
    } on ApiException catch (e) {
      if (e.isNotStreamable) {
        await _skipUnstreamable(index);
      }
    }
  }

  Future<void> _onTrackCompleted() async {
    // Persist final position, then let the playlist advance (or repeat-one).
    final index = _player.currentIndex;
    if (index != null && index < _tracks.length) {
      _flushHeartbeat(_tracks[index].id);
      _persistPosition(
        _tracks[index].id,
        _player.position.inMilliseconds,
      );
    }
    if (_player.loopMode == LoopMode.one) {
      await _player.seek(Duration.zero);
      await _player.play();
      return;
    }
    if (hasNext) {
      await skipToNext();
    } else {
      await stop();
    }
  }

  // ---------- AudioHandler overrides (media buttons / headset) ----------

  @override
  Future<void> play() async {
    final index = _player.currentIndex ?? 0;
    if (_tracks.isNotEmpty && !_tickets.containsKey(index)) {
      await _prepareIndex(index);
    }
    _startHeartbeat();
    await _player.play();
  }

  @override
  Future<void> pause() async {
    await _player.pause();
    _flushHeartbeat(_currentTrackId());
    _persistPosition(
      _currentTrackId(),
      _player.position.inMilliseconds,
    );
  }

  @override
  Future<void> stop() async {
    _stopHeartbeat();
    _flushHeartbeat(_currentTrackId());
    final id = _currentTrackId();
    if (id != null) {
      _persistPosition(id, _player.position.inMilliseconds);
    }
    await _player.stop();
    await playbackState.firstWhere(
      (s) => s.processingState == AudioProcessingState.idle,
      orElse: () => playbackState.value,
    ).timeout(const Duration(milliseconds: 100), onTimeout: () {
      playbackState.add(playbackState.value.copyWith(
        playing: false,
        processingState: AudioProcessingState.idle,
      ));
      return playbackState.value;
    });
  }

  @override
  Future<void> seek(Duration position) => _player.seek(position);

  @override
  Future<void> skipToNext() async {
    if (!hasNext) return;
    final next = (_player.currentIndex ?? -1) + 1;
    if (!_tickets.containsKey(next)) {
      await _prepareIndex(next);
    }
    await _player.seekToNext();
  }

  @override
  Future<void> skipToPrevious() async {
    if (!hasPrevious) return;
    // Within the first 3s go to previous, otherwise restart (standard).
    if (_player.position > const Duration(seconds: 3)) {
      await _player.seek(Duration.zero);
      return;
    }
    final prev = (_player.currentIndex ?? 1) - 1;
    if (prev >= 0 && !_tickets.containsKey(prev)) {
      await _prepareIndex(prev);
    }
    await _player.seekToPrevious();
  }

  @override
  Future<void> skipToQueueItem(int index) async {
    if (index < 0 || index >= _tracks.length) return;
    if (!_tickets.containsKey(index)) {
      await _prepareIndex(index);
    }
    await _player.seek(Duration.zero, index: index);
    await _player.play();
  }

  @override
  Future<void> setShuffleMode(AudioServiceShuffleMode shuffleMode) async {
    final enabled = shuffleMode == AudioServiceShuffleMode.all;
    await _player.setShuffleModeEnabled(enabled);
    _broadcastState(_player.playbackEvent);
  }

  @override
  Future<void> setRepeatMode(AudioServiceRepeatMode repeatMode) async {
    await _player.setLoopMode(switch (repeatMode) {
      AudioServiceRepeatMode.one => LoopMode.one,
      AudioServiceRepeatMode.all => LoopMode.all,
      AudioServiceRepeatMode.none => LoopMode.off,
      _ => LoopMode.off,
    });
    _broadcastState(_player.playbackEvent);
  }

  /// Headset / lock-screen play-pause toggle is routed here by audio_service.
  @override
  Future<void> click([MediaButton button = MediaButton.media]) async {
    if (_player.playing) {
      await pause();
    } else {
      await play();
    }
  }

  bool get hasNext =>
      _player.currentIndex != null &&
      _player.currentIndex! + 1 < _tracks.length;

  bool get hasPrevious =>
      _player.currentIndex != null && _player.currentIndex! > 0;

  // ---------- PlaybackState broadcast ----------

  void _broadcastState(PlaybackEvent event) {
    final playing = _player.playing;
    if (playing) {
      _startHeartbeat();
    }
    playbackState.add(
      PlaybackState(
        controls: [
          if (hasPrevious) MediaControl.skipToPrevious else MediaControl.rewind,
          if (playing) MediaControl.pause else MediaControl.play,
          MediaControl.stop,
          if (hasNext) MediaControl.skipToNext else MediaControl.fastForward,
        ],
        systemActions: const {
          MediaAction.seek,
          MediaAction.seekForward,
          MediaAction.seekBackward,
        },
        androidCompactActionIndices: const [0, 1, 3],
        processingState: switch (_player.processingState) {
          ProcessingState.idle => AudioProcessingState.idle,
          ProcessingState.loading => AudioProcessingState.loading,
          ProcessingState.buffering => AudioProcessingState.buffering,
          ProcessingState.ready => AudioProcessingState.ready,
          ProcessingState.completed => AudioProcessingState.completed,
        },
        playing: playing,
        updatePosition: _player.position,
        bufferedPosition: _player.bufferedPosition,
        speed: _player.speed,
        queueIndex: event.currentIndex,
        shuffleMode: _player.shuffleModeEnabled
            ? AudioServiceShuffleMode.all
            : AudioServiceShuffleMode.none,
        repeatMode: switch (_player.loopMode) {
          LoopMode.one => AudioServiceRepeatMode.one,
          LoopMode.all => AudioServiceRepeatMode.all,
          LoopMode.off => AudioServiceRepeatMode.none,
        },
      ),
    );
  }

  // ---------- Heartbeat + resume position ----------

  String? _currentTrackId() {
    final index = _player.currentIndex;
    if (index == null || index < 0 || index >= _tracks.length) return null;
    return _tracks[index].id;
  }

  void _startHeartbeat() {
    _heartbeatTimer ??= Timer.periodic(_heartbeatInterval, (_) => _tick());
    _lastTick ??= DateTime.now();
  }

  void _stopHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
    _lastTick = null;
    _playMsSinceHeartbeat = 0;
  }

  void _tick() {
    final now = DateTime.now();
    if (_player.playing) {
      _playMsSinceHeartbeat += now.difference(_lastTick ?? now).inMilliseconds;
    }
    _lastTick = now;
    _flushHeartbeat(_currentTrackId());
  }

  void _flushHeartbeat(String? trackId) {
    if (trackId == null || _onHeartbeat == null) {
      _playMsSinceHeartbeat = 0;
      return;
    }
    if (_playMsSinceHeartbeat <= 0) return;
    final playMs = _playMsSinceHeartbeat;
    _playMsSinceHeartbeat = 0;
    final index = _player.currentIndex;
    unawaited(
      _onHeartbeat(
        trackId: trackId,
        positionMs: _player.position.inMilliseconds,
        playMs: playMs,
        jti: index == null ? null : _tickets[index]?.jti,
      ).then((_) {
        _persistPosition(trackId, _player.position.inMilliseconds);
      }).catchError((_) {}),
    );
  }

  void _persistPosition(String? trackId, int positionMs) {
    if (trackId == null || _onPosition == null || positionMs < 0) return;
    unawaited(_onPosition(trackId, positionMs).catchError((_) {}));
  }
}
