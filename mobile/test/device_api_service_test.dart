// DeviceApiService behavior (P1): claim normalization, Bearer attach,
// auth-required paths. Dio + in-memory secure storage, no network.
library;

import 'package:avibe/core/api_client.dart';
import 'package:avibe/core/error.dart';
import 'package:avibe/services/device_api_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

(FakeAdapter, DeviceApiService) _service(
    Map<String, String> store, AdapterHandler handler) {
  final adapter = FakeAdapter(handler);
  final dio = Dio();
  dio.httpClientAdapter = adapter;
  dio.interceptors.add(EnvelopeInterceptor());
  return (adapter, DeviceApiService(dio, const FlutterSecureStorage()));
}

Map<String, dynamic> _deviceJson() => {
      'id': 'd1',
      'businessId': 'b1',
      'deviceName': 'Counter',
      'status': 'ACTIVE',
      'volume': 70,
    };

void main() {
  group('DeviceApiService', () {
    test('claim normalizes the code and saves the token', () async {
      final store = mockSecureStorage();
      Map<String, dynamic>? seenBody;
      final (_, api) = _service(store, (o) async {
        seenBody = Map<String, dynamic>.from(o.data as Map);
        return jsonResponse({
          'data': {
            'device': _deviceJson(),
            'token': 'tok-1',
          }
        }, 200);
      });
      final (device, token) = await api.claim(' ab12 ');
      expect(seenBody, {'pairingCode': 'AB12'});
      expect(device.id, 'd1');
      expect(token, 'tok-1');
      expect(store[TokenKeys.device], 'tok-1');
    });

    test('claim failure saves nothing', () async {
      final store = mockSecureStorage();
      final (_, api) = _service(
          store,
          (o) async => jsonResponse(
              {
                'error': {'code': 'ERR_INVALID_INPUT', 'message': 'bad'}
              },
              400));
      final err = await api
          .claim('NOPE')
          .then<Object>((_) => 'no-throw', onError: (e) => e as Object);
      expect((err as ApiException).code, 'ERR_INVALID_INPUT');
      expect(store, isEmpty);
    });

    test('authed calls attach Bearer and parse', () async {
      final store = mockSecureStorage({TokenKeys.device: 'tok-9'});
      String? seenAuth;
      final (_, api) = _service(store, (o) async {
        seenAuth = o.headers['Authorization'] as String?;
        return jsonResponse({
          'data': {
            'businessId': 'b1',
            'businessName': 'S',
            'open': true,
            'reason': 'scheduled',
            'volume': 60,
            'tracks': [
              {'trackId': 't1', 'title': 'A'}
            ],
          }
        }, 200);
      });
      final q = await api.queue();
      expect(seenAuth, 'Bearer tok-9');
      expect(q.tracks.single.trackId, 't1');
    });

    test('calls without a token fail before hitting the network', () async {
      mockSecureStorage();
      final (adapter, api) = _service(
          {}, (o) async => jsonResponse({'data': {}}, 200));
      final err = await api
          .queue()
          .then<Object>((_) => 'no-throw', onError: (e) => e as Object);
      expect((err as ApiException).code, 'ERR_UNAUTHORIZED');
      expect(adapter.requests, isEmpty);
    });

    test('heartbeat posts track and duration', () async {
      final store = mockSecureStorage({TokenKeys.device: 't'});
      Map<String, dynamic>? seenBody;
      String? seenPath;
      final (_, api) = _service(store, (o) async {
        seenPath = o.path;
        seenBody = Map<String, dynamic>.from(o.data as Map);
        return jsonResponse({'data': {}}, 200);
      });
      await api.heartbeat(trackId: 't1', durationSecs: 30);
      expect(seenPath, '/store/device/heartbeat');
      expect(seenBody, {'trackId': 't1', 'durationSecs': 30});
    });
  });
}
