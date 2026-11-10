// Store-device API: claim/queue/heartbeat/stream under the device JWT.
// URIs match backend/internal/handler/router.go exactly.
library;

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../core/api_client.dart';
import '../core/error.dart';
import '../core/models.dart';

/// Device-authenticated REST calls (mirrors the web deviceApiClient).
class DeviceApiService {
  DeviceApiService(this._dio, this._storage);

  final Dio _dio;
  final FlutterSecureStorage _storage;

  Future<String?> get deviceToken =>
      _storage.read(key: TokenKeys.device);

  Future<void> saveDeviceToken(String token) =>
      _storage.write(key: TokenKeys.device, value: token);

  Future<void> clearDeviceToken() =>
      _storage.delete(key: TokenKeys.device);

  Future<Options> _auth() async {
    final token = await deviceToken;
    if (token == null || token.isEmpty) {
      throw ApiException('ERR_UNAUTHORIZED', 'device not claimed');
    }
    return Options(headers: {'Authorization': 'Bearer $token'});
  }

  /// POST /store/device/claim (no auth): pairs this tablet by code.
  Future<(StoreDevice, String)> claim(String pairingCode) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/store/device/claim',
        data: {'pairingCode': pairingCode.trim().toUpperCase()},
      );
      final data = unwrapData<Map<String, dynamic>>(res);
      final device =
          StoreDevice.fromJson(data['device'] as Map<String, dynamic>);
      final token = data['token'] as String;
      await saveDeviceToken(token);
      return (device, token);
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// GET /store/device/me: restores the claimed device after restart.
  Future<StoreDevice> me() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/store/device/me',
        options: await _auth(),
      );
      return StoreDevice.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// GET /store/device/queue: schedule-gated playlist + volume.
  Future<DeviceQueue> queue() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/store/device/queue',
        options: await _auth(),
      );
      return DeviceQueue.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// POST /store/device/heartbeat: play attribution for royalty logs.
  Future<bool> heartbeat({
    required String trackId,
    required int durationSecs,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/store/device/heartbeat',
        data: {'trackId': trackId, 'durationSecs': durationSecs},
        options: await _auth(),
      );
      final data = unwrapData<Map<String, dynamic>>(res);
      return (data['counted'] as bool?) ?? false;
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// GET /store/device/stream/:id: signed HLS grant for one queue track.
  Future<StreamTicket> stream(String trackId) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/store/device/stream/$trackId',
        options: await _auth(),
      );
      return StreamTicket.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }
}
