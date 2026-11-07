// Typed API surface. URIs match backend/internal/handler/router.go exactly.
library;

import 'package:dio/dio.dart';

import '../core/api_client.dart';
import '../core/error.dart';
import '../core/models.dart';

/// All REST calls used by the mobile app.
class ApiService {
  ApiService(this._dio);

  final Dio _dio;

  // ---- Tracks: GET /tracks, GET /tracks/:id, GET /tracks/:id/stream ----

  Future<List<Track>> listTracks(
      {int limit = 20, int offset = 0, String? q, String? sort}) async {
    try {
      final params = <String, dynamic>{'limit': limit, 'offset': offset};
      if (q != null && q.trim().isNotEmpty) params['q'] = q.trim();
      if (sort != null && sort.trim().isNotEmpty) params['sort'] = sort.trim();
      final res = await _dio.get<Map<String, dynamic>>(
        '/tracks',
        queryParameters: params,
      );
      final page = unwrapData<Map<String, dynamic>>(res);
      final list = page['items'] as List<dynamic>;
      return list.map((e) => Track.fromJson(e as Map<String, dynamic>)).toList();
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<Track> getTrack(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/tracks/$id');
      return Track.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// Signed HLS ticket. Throws ERR_NOT_STREAMABLE for takedown/non-READY.
  Future<StreamTicket> getStream(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/tracks/$id/stream');
      return StreamTicket.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  // ---- Playback: POST /playback/heartbeat, GET+PUT /playback/state/:id ----

  Future<HeartbeatResult> heartbeat({
    required String trackId,
    required int positionMs,
    required int playMs,
    String? jti,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/playback/heartbeat',
        data: {
          'trackId': trackId,
          'positionMs': positionMs,
          'playMs': playMs,
          if (jti != null) 'jti': jti,
        },
      );
      return HeartbeatResult.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<PlaybackStateModel?> getPlaybackState(String trackId) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/playback/state/$trackId');
      final data = res.data!['data'];
      if (data == null) return null;
      return PlaybackStateModel.fromJson(data as Map<String, dynamic>);
    } on ApiException {
      rethrow;
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<PlaybackStateModel> setPlaybackState(
    String trackId,
    int positionMs,
  ) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/playback/state/$trackId',
        data: {'positionMs': positionMs},
      );
      return PlaybackStateModel.fromJson(
          unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  // ---- Playlists (all JWT): CRUD + entries + reorder ----

  Future<List<Playlist>> listPlaylists() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/playlists');
      final list = unwrapData<List<dynamic>>(res);
      return list
          .map((e) => Playlist.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<Playlist> createPlaylist(String title, {bool isPublic = false}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/playlists',
        data: {'title': title, 'isPublic': isPublic},
      );
      return Playlist.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<Playlist> getPlaylist(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/playlists/$id');
      return Playlist.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<Playlist> updatePlaylist(
    String id,
    String title, {
    bool isPublic = false,
  }) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/playlists/$id',
        data: {'title': title, 'isPublic': isPublic},
      );
      return Playlist.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> deletePlaylist(String id) async {
    try {
      await _dio.delete<Map<String, dynamic>>('/playlists/$id');
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<PlaylistEntry>> listPlaylistTracks(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/playlists/$id/tracks');
      final list = unwrapData<List<dynamic>>(res);
      return list
          .map((e) => PlaylistEntry.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> addPlaylistTrack(String id, String trackId) async {
    try {
      await _dio.post<Map<String, dynamic>>(
        '/playlists/$id/tracks',
        data: {'trackId': trackId},
      );
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> removePlaylistTrack(String id, String trackId) async {
    try {
      await _dio.delete<Map<String, dynamic>>('/playlists/$id/tracks/$trackId');
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// Reorder must include every current entry ID (backend validates).
  Future<void> reorderPlaylist(String id, List<String> trackIds) async {
    try {
      await _dio.put<Map<String, dynamic>>(
        '/playlists/$id/reorder',
        data: {'trackIds': trackIds},
      );
    } catch (e) {
      throw toApiException(e);
    }
  }

  // ---- Disputes: POST /disputes/report, POST /:id/counter, GET /blocks ----

  Future<Dispute> reportTrack({
    required String trackId,
    required String reason,
    String? description,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/disputes/report',
        data: {
          'trackId': trackId,
          'reason': reason,
          if (description != null && description.isNotEmpty)
            'description': description,
        },
      );
      return Dispute.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<UserBlock>> myBlocks() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/disputes/blocks');
      final list = unwrapData<List<dynamic>>(res);
      return list
          .map((e) => UserBlock.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// Blocks a partner/artist (POST /users/blocks) for UGC store compliance.
  Future<void> blockArtist({
    required String partnerId,
    String? reason,
  }) async {
    try {
      await _dio.post<Map<String, dynamic>>(
        '/users/blocks',
        data: {
          'partnerId': partnerId,
          if (reason != null && reason.isNotEmpty) 'reason': reason,
        },
      );
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// Immediate self-withdrawal (POST /auth/withdraw).
  Future<void> withdrawAccount() async {
    try {
      await _dio.post<Map<String, dynamic>>('/auth/withdraw');
    } catch (e) {
      throw toApiException(e);
    }
  }

  // ---- Legal: GET /legal/terms, GET /legal/terms/:code, consents ----

  Future<List<LegalTerm>> listTerms() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/legal/terms');
      final list = unwrapData<List<dynamic>>(res);
      return list
          .map((e) => LegalTerm.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<LegalTerm> getTerm(String code) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/legal/terms/$code');
      return LegalTerm.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> createConsent(String termsId) async {
    try {
      await _dio.post<Map<String, dynamic>>(
        '/legal/consents',
        data: {'termsId': termsId},
      );
    } catch (e) {
      throw toApiException(e);
    }
  }
}
