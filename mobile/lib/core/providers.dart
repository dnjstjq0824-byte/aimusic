// Riverpod providers: Dio, API, auth, audio handler, feature state.
library;

import 'package:audio_service/audio_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:just_audio/just_audio.dart';

import '../core/api_client.dart';
import '../core/models.dart';
import '../services/api_service.dart';
import '../services/audio_handler.dart';
import '../services/auth_repository.dart';
import '../services/device_api_service.dart';

final secureStorageProvider = Provider<FlutterSecureStorage>(
  (ref) => const FlutterSecureStorage(),
);

final dioProvider = Provider<Dio>(
  (ref) => ApiClientFactory.create(storage: ref.watch(secureStorageProvider)),
);

final apiProvider = Provider<ApiService>(
  (ref) => ApiService(ref.watch(dioProvider)),
);

/// Device Dio without the user auth interceptor (store-device JWT instead).
final deviceDioProvider = Provider<Dio>(
  (ref) => ApiClientFactory.createDevice(),
);

final deviceApiProvider = Provider<DeviceApiService>(
  (ref) => DeviceApiService(
    ref.watch(deviceDioProvider),
    ref.watch(secureStorageProvider),
  ),
);

final authRepositoryProvider = Provider<AuthRepository>(
  (ref) => AuthRepository(
    ref.watch(dioProvider),
    ref.watch(secureStorageProvider),
  ),
);

/// Current principal; null when logged out.
final authMeProvider = FutureProvider<AuthMe?>((ref) async {
  final repo = ref.watch(authRepositoryProvider);
  if (!await repo.isLoggedIn) return null;
  try {
    return await repo.me();
  } catch (_) {
    return null;
  }
});

/// Singleton background audio handler (lazy; created on first playback).
final audioHandlerProvider = FutureProvider<AvibeAudioHandler>((ref) async {
  final api = ref.watch(apiProvider);
  return AudioService.init(
    builder: () => AvibeAudioHandler(
      player: AudioPlayer(),
      resolveStream: api.getStream,
      onHeartbeat: heartbeatHookFor(api),
      onPosition: positionHookFor(api),
    ),
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'ai.music.channel.audio',
      androidNotificationChannelName: 'AVIBE playback',
      androidNotificationOngoing: true,
      androidStopForegroundOnPause: true,
    ),
  );
});

/// Home track feed (READY tracks, first page).
final homeTracksProvider = FutureProvider<List<Track>>((ref) async {
  final api = ref.watch(apiProvider);
  return api.listTracks(limit: 30);
});

/// Search query state; results come from the server-side track search
/// (GET /tracks?q=...), newest first.
final searchQueryProvider = StateProvider<String>((ref) => '');

final searchResultsProvider = FutureProvider<List<Track>>((ref) async {
  final api = ref.watch(apiProvider);
  final query = ref.watch(searchQueryProvider).trim();
  return api.listTracks(limit: 50, q: query.isEmpty ? null : query);
});

/// Current user's playlists.
final playlistsProvider = FutureProvider<List<Playlist>>((ref) async {
  final api = ref.watch(apiProvider);
  return api.listPlaylists();
});

/// Entries + resolved tracks for one playlist.
final playlistDetailProvider =
    FutureProvider.family<(Playlist, List<Track>), String>((ref, id) async {
  final api = ref.watch(apiProvider);
  final playlist = await api.getPlaylist(id);
  final entries = await api.listPlaylistTracks(id);
  final tracks = <Track>[];
  for (final e in entries) {
    try {
      tracks.add(await api.getTrack(e.trackId));
    } catch (_) {
      // Entry for a removed/taken-down track: skip in the UI list.
    }
  }
  return (playlist, tracks);
});

/// Legal terms list.
final legalTermsProvider = FutureProvider<List<LegalTerm>>((ref) async {
  final api = ref.watch(apiProvider);
  return api.listTerms();
});

/// Strike blocks for the report/block menu.
final myBlocksProvider = FutureProvider<List<UserBlock>>((ref) async {
  final api = ref.watch(apiProvider);
  return api.myBlocks();
});
