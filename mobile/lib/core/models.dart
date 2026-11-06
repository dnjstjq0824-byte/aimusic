// Domain models mirroring backend/internal/domain/entities.go JSON shapes.
library;

/// Track mirrors domain.Track (JSON tags in entities.go).
class Track {
  Track({
    required this.id,
    required this.partnerId,
    required this.title,
    required this.aiProvider,
    required this.aiGenerated,
    required this.status,
    required this.playCount,
    this.albumId,
    this.durationMs,
    this.hlsPrefix,
    this.aiDisclosureText,
    this.bitrateKbps,
  });

  factory Track.fromJson(Map<String, dynamic> j) => Track(
        id: j['id'] as String,
        partnerId: j['partnerId'] as String,
        title: j['title'] as String,
        aiProvider: (j['aiProvider'] as String?) ?? 'UNKNOWN',
        aiGenerated: (j['aiGenerated'] as bool?) ?? true,
        status: (j['status'] as String?) ?? 'UNKNOWN',
        playCount: (j['playCount'] as num?)?.toInt() ?? 0,
        albumId: j['albumId'] as String?,
        durationMs: (j['durationMs'] as num?)?.toInt(),
        hlsPrefix: j['hlsPrefix'] as String?,
        aiDisclosureText: j['aiDisclosureText'] as String?,
        bitrateKbps: (j['bitrateKbps'] as num?)?.toInt(),
      );

  final String id;
  final String partnerId;
  final String title;
  final String aiProvider;
  final bool aiGenerated;
  final String status;
  final int playCount;
  final String? albumId;
  final int? durationMs;
  final String? hlsPrefix;
  final String? aiDisclosureText;
  final int? bitrateKbps;

  bool get isReady => status == 'READY';
  bool get isTakenDown => status == 'TAKEN_DOWN';

  /// Lock-screen / notification subtitle with the AI-generated tag.
  String get subtitle =>
      aiGenerated ? 'AI-generated · $aiProvider' : aiProvider;
}

/// Signed stream URL from GET /tracks/:id/stream.
class StreamTicket {
  StreamTicket({required this.url, required this.expiresAt, required this.jti});

  factory StreamTicket.fromJson(Map<String, dynamic> j) => StreamTicket(
        url: j['url'] as String,
        expiresAt: DateTime.parse(j['expiresAt'] as String),
        jti: j['jti'] as String,
      );

  final String url;
  final DateTime expiresAt;
  final String jti;
}

/// Heartbeat result from POST /playback/heartbeat.
class HeartbeatResult {
  HeartbeatResult({required this.countedAsPlay, required this.fraudSuspected});

  factory HeartbeatResult.fromJson(Map<String, dynamic> j) => HeartbeatResult(
        countedAsPlay: (j['countedAsPlay'] as bool?) ?? false,
        fraudSuspected: (j['fraudSuspected'] as bool?) ?? false,
      );

  final bool countedAsPlay;
  final bool fraudSuspected;
}

/// Resume state from GET/PUT /playback/state/:trackId.
class PlaybackStateModel {
  PlaybackStateModel({required this.trackId, required this.positionMs});

  factory PlaybackStateModel.fromJson(Map<String, dynamic> j) =>
      PlaybackStateModel(
        trackId: j['trackId'] as String,
        positionMs: (j['positionMs'] as num?)?.toInt() ?? 0,
      );

  final String trackId;
  final int positionMs;
}

/// Playlist mirrors domain.Playlist.
class Playlist {
  Playlist({
    required this.id,
    required this.ownerUserId,
    required this.title,
    required this.isPublic,
  });

  factory Playlist.fromJson(Map<String, dynamic> j) => Playlist(
        id: j['id'] as String,
        ownerUserId: j['ownerUserId'] as String,
        title: j['title'] as String,
        isPublic: (j['isPublic'] as bool?) ?? false,
      );

  final String id;
  final String ownerUserId;
  final String title;
  final bool isPublic;
}

/// Playlist entry mirrors domain.PlaylistTrack.
class PlaylistEntry {
  PlaylistEntry({
    required this.id,
    required this.playlistId,
    required this.trackId,
    required this.position,
  });

  factory PlaylistEntry.fromJson(Map<String, dynamic> j) => PlaylistEntry(
        id: j['id'] as String,
        playlistId: j['playlistId'] as String,
        trackId: j['trackId'] as String,
        position: (j['position'] as num?)?.toInt() ?? 0,
      );

  final String id;
  final String playlistId;
  final String trackId;
  final int position;
}

/// Legal term mirrors domain.LegalTerm.
class LegalTerm {
  LegalTerm({
    required this.id,
    required this.code,
    required this.version,
    required this.titleKo,
    required this.bodyMd,
    required this.isRequired,
  });

  factory LegalTerm.fromJson(Map<String, dynamic> j) => LegalTerm(
        id: j['id'] as String,
        code: j['code'] as String,
        version: j['version'] as String,
        titleKo: j['titleKo'] as String,
        bodyMd: j['bodyMd'] as String,
        isRequired: (j['isRequired'] as bool?) ?? false,
      );

  final String id;
  final String code;
  final String version;
  final String titleKo;
  final String bodyMd;
  final bool isRequired;
}

/// User block mirrors domain.UserBlock (GET /disputes/blocks).
class UserBlock {
  UserBlock({
    required this.id,
    required this.reason,
    required this.strikeNumber,
    this.expiresAt,
  });

  factory UserBlock.fromJson(Map<String, dynamic> j) => UserBlock(
        id: j['id'] as String,
        reason: (j['reason'] as String?) ?? '',
        strikeNumber: (j['strikeNumber'] as num?)?.toInt() ?? 0,
        expiresAt: j['expiresAt'] == null
            ? null
            : DateTime.parse(j['expiresAt'] as String),
      );

  final String id;
  final String reason;
  final int strikeNumber;
  final DateTime? expiresAt;
}

/// Dispute mirrors domain.CopyrightDispute (POST /disputes/report).
class Dispute {
  Dispute({
    required this.id,
    required this.trackId,
    required this.reason,
    required this.status,
  });

  factory Dispute.fromJson(Map<String, dynamic> j) => Dispute(
        id: j['id'] as String,
        trackId: j['trackId'] as String,
        reason: (j['reason'] as String?) ?? '',
        status: (j['status'] as String?) ?? '',
      );

  final String id;
  final String trackId;
  final String reason;
  final String status;
}

/// Dispute reasons from domain/entities.go.
abstract final class DisputeReasons {
  static const aiPolicy = 'AI_POLICY_VIOLATION';
  static const voice = 'VOICE_CLONE';
  static const sample = 'UNAUTHORIZED_SAMPLE';
  static const other = 'OTHER';

  static const all = [aiPolicy, voice, sample, other];

  static String label(String reason) {
    switch (reason) {
      case aiPolicy:
        return 'AI 정책 위반';
      case voice:
        return '무단 보이스 클론';
      case sample:
        return '무단 샘플 사용';
      default:
        return '기타';
    }
  }
}

/// Legal term codes from domain/entities.go.
abstract final class TermCodes {
  static const tos = 'TOS';
  static const partnerContract = 'PARTNER_CONTRACT';
  static const privacy = 'PRIVACY';
  static const copyright = 'COPYRIGHT_POLICY';
}

/// Auth token pair from POST /auth/login and POST /auth/refresh.
class TokenPair {
  TokenPair({required this.accessToken, required this.refreshToken});

  factory TokenPair.fromJson(Map<String, dynamic> j) => TokenPair(
        accessToken: j['accessToken'] as String,
        refreshToken: j['refreshToken'] as String,
      );

  final String accessToken;
  final String refreshToken;
}

/// Minimal auth principal from GET /auth/me.
class AuthMe {
  AuthMe({required this.userId, required this.role});

  factory AuthMe.fromJson(Map<String, dynamic> j) => AuthMe(
        userId: j['userId'] as String,
        role: (j['role'] as String?) ?? 'USER',
      );

  final String userId;
  final String role;
}
