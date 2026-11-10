// Store mode: pairing claim + scheduled queue playback + attribution heartbeat.
// Mirrors the web storePlayerStore; playback runs on a dedicated just_audio
// player so listener audio (audio_service singleton) stays untouched.
library;

import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';

import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';

/// Immutable store-mode UI state.
class StoreModeState {
  const StoreModeState({
    this.device,
    this.queue,
    this.trackIndex = 0,
    this.playing = false,
    this.loading = false,
    this.error,
    this.needsClaim = false,
  });

  final StoreDevice? device;
  final DeviceQueue? queue;
  final int trackIndex;
  final bool playing;
  final bool loading;
  final String? error;
  final bool needsClaim;

  DeviceQueueTrack? get currentTrack {
    final q = queue;
    if (q == null || q.tracks.isEmpty) return null;
    return q.tracks[trackIndex % q.tracks.length];
  }

  StoreModeState copyWith({
    StoreDevice? device,
    DeviceQueue? queue,
    int? trackIndex,
    bool? playing,
    bool? loading,
    String? Function()? error,
    bool? needsClaim,
  }) =>
      StoreModeState(
        device: device ?? this.device,
        queue: queue ?? this.queue,
        trackIndex: trackIndex ?? this.trackIndex,
        playing: playing ?? this.playing,
        loading: loading ?? this.loading,
        error: error == null ? this.error : error(),
        needsClaim: needsClaim ?? this.needsClaim,
      );
}

final storeModeProvider =
    NotifierProvider<StoreModeController, StoreModeState>(
        StoreModeController.new);

/// Owns the store player lifecycle: claim, queue, HLS playback, heartbeat.
class StoreModeController extends Notifier<StoreModeState> {
  static const heartbeatInterval = Duration(seconds: 10);
  static const queueRefreshInterval = Duration(minutes: 5);

  AudioPlayer? _player;
  Timer? _heartbeatTimer;
  Timer? _queueTimer;
  StreamSubscription<ProcessingState>? _completionSub;
  int _playedSecs = 0;
  int _loadSeq = 0;
  bool _restored = false;

  @override
  StoreModeState build() {
    ref.onDispose(_dispose);
    if (!_restored) {
      _restored = true;
      Future.microtask(restore);
    }
    return const StoreModeState();
  }

  AudioPlayer _ensurePlayer() {
    final existing = _player;
    if (existing != null) return existing;
    final player = AudioPlayer();
    _player = player;
    _completionSub =
        player.processingStateStream.listen((processingState) {
      if (processingState == ProcessingState.completed) {
        unawaited(next());
      }
    });
    return player;
  }

  void _dispose() {
    _heartbeatTimer?.cancel();
    _queueTimer?.cancel();
    _completionSub?.cancel();
    unawaited(_player?.dispose());
    _player = null;
  }

  void _stopHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }

  void _startHeartbeat(String trackId) {
    _stopHeartbeat();
    _playedSecs = 0;
    final api = ref.read(deviceApiProvider);
    _heartbeatTimer = Timer.periodic(heartbeatInterval, (_) {
      final current = state.currentTrack;
      if (current == null ||
          current.trackId != trackId ||
          !state.playing) {
        return;
      }
      _playedSecs += heartbeatInterval.inSeconds;
      unawaited(api
          .heartbeat(trackId: trackId, durationSecs: _playedSecs)
          .then<void>((_) {}, onError: (_) {}));
    });
  }

  void _startQueueRefresh() {
    _queueTimer?.cancel();
    _queueTimer = Timer.periodic(queueRefreshInterval, (_) {
      if (state.device != null && !state.needsClaim) {
        unawaited(refreshQueue());
      }
    });
  }

  double _volumeOf(DeviceQueue q) =>
      (q.volume.clamp(0, 100)) / 100.0;

  /// Restores a previously claimed device (token in secure storage).
  Future<void> restore() async {
    final api = ref.read(deviceApiProvider);
    final token = await api.deviceToken;
    if (token == null || token.isEmpty) {
      state = state.copyWith(needsClaim: true);
      return;
    }
    state = state.copyWith(loading: true);
    try {
      final device = await api.me();
      state = state.copyWith(device: device, needsClaim: false);
      _startQueueRefresh();
      await refreshQueue();
    } on ApiException catch (e) {
      if (e.isAuthError) {
        await api.clearDeviceToken();
        state = state.copyWith(needsClaim: true, error: () => null);
      } else {
        state = state.copyWith(error: () => e.message);
      }
    } finally {
      state = state.copyWith(loading: false);
    }
  }

  /// Pairs this tablet with an 8-char code from the partner console.
  Future<void> claim(String pairingCode) async {
    final api = ref.read(deviceApiProvider);
    state = state.copyWith(loading: true, error: () => null);
    try {
      final (device, _) = await api.claim(pairingCode);
      state = state.copyWith(device: device, needsClaim: false);
      _startQueueRefresh();
      await refreshQueue();
    } on ApiException catch (e) {
      state = state.copyWith(error: () => e.message);
    } finally {
      state = state.copyWith(loading: false);
    }
  }

  /// Unpairs this tablet and stops playback.
  Future<void> unclaim() async {
    final api = ref.read(deviceApiProvider);
    _stopHeartbeat();
    _queueTimer?.cancel();
    await _player?.stop();
    await api.clearDeviceToken();
    state = const StoreModeState(needsClaim: true);
  }

  /// Reloads the schedule-gated queue from the server.
  Future<void> refreshQueue() async {
    final api = ref.read(deviceApiProvider);
    if (state.device == null) {
      state = state.copyWith(needsClaim: true);
      return;
    }
    try {
      final q = await api.queue();
      var index = state.trackIndex;
      if (q.tracks.isEmpty || index >= q.tracks.length) index = 0;
      state = state.copyWith(
        queue: q,
        trackIndex: index,
        needsClaim: false,
        error: () => null,
      );
      await _player?.setVolume(_volumeOf(q));
    } on ApiException catch (e) {
      if (e.isAuthError) {
        state = state.copyWith(needsClaim: true);
      } else {
        state = state.copyWith(error: () => e.message);
      }
    }
  }

  /// Plays the queue entry at [index] via a device stream grant.
  Future<void> playIndex(int index) async {
    final api = ref.read(deviceApiProvider);
    final q = state.queue;
    if (q == null || q.tracks.isEmpty) return;
    final seq = ++_loadSeq;
    final track = q.tracks[index % q.tracks.length];
    final player = _ensurePlayer();
    _stopHeartbeat();
    await player.stop();
    state = state.copyWith(
      trackIndex: index % q.tracks.length,
      loading: true,
      error: () => null,
    );
    try {
      final ticket = await api.stream(track.trackId);
      if (seq != _loadSeq) return;
      await player.setAudioSource(HlsAudioSource(Uri.parse(ticket.url)));
      await player.setVolume(_volumeOf(q));
      await player.play();
      if (seq != _loadSeq) return;
      state = state.copyWith(playing: true);
      _startHeartbeat(track.trackId);
    } on ApiException catch (e) {
      if (seq != _loadSeq) return;
      if (e.isAuthError) {
        state = state.copyWith(needsClaim: true, playing: false);
      } else {
        state = state.copyWith(error: () => e.message, playing: false);
      }
    } finally {
      if (seq == _loadSeq) state = state.copyWith(loading: false);
    }
  }

  /// Toggles playback; starts the queue head when idle.
  Future<void> togglePlay() async {
    final player = _player;
    if (player == null || player.processingState == ProcessingState.idle) {
      await playIndex(state.trackIndex);
      return;
    }
    if (player.playing) {
      await player.pause();
      state = state.copyWith(playing: false);
    } else {
      await player.play();
      final current = state.currentTrack;
      state = state.copyWith(playing: true);
      if (current != null && _heartbeatTimer == null) {
        _startHeartbeat(current.trackId);
      }
    }
  }

  /// Advances to the next queue entry (wraps around).
  Future<void> next() async {
    final q = state.queue;
    if (q == null || q.tracks.isEmpty) return;
    await playIndex((state.trackIndex + 1) % q.tracks.length);
  }

  /// Goes back to the previous queue entry (wraps around).
  Future<void> previous() async {
    final q = state.queue;
    if (q == null || q.tracks.isEmpty) return;
    final prev = (state.trackIndex - 1 + q.tracks.length) % q.tracks.length;
    await playIndex(prev);
  }
}
