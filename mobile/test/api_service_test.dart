// ApiService URL/body contracts (P1): URIs must match the backend router
// exactly; playback helpers shape their payloads. No network.
library;

import 'package:avibe/core/api_client.dart';
import 'package:avibe/services/api_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

(FakeAdapter, ApiService) _service(AdapterHandler handler) {
  final adapter = FakeAdapter(handler);
  final dio = Dio();
  dio.httpClientAdapter = adapter;
  dio.interceptors.add(EnvelopeInterceptor());
  return (adapter, ApiService(dio));
}

Map<String, dynamic> _trackJson(String id) => {
      'id': id,
      'partnerId': 'p1',
      'title': 'T-$id',
      'aiProvider': 'SUNO',
      'aiGenerated': true,
      'status': 'READY',
      'playCount': 0,
    };

void main() {
  group('ApiService', () {
    test('listTracks trims params and skips empties', () async {
      RequestOptions? seen;
      final (_, api) = _service((o) async {
        seen = o;
        return jsonResponse(
            {
              'data': {
                'items': [_trackJson('t1')]
              }
            },
            200);
      });
      final tracks = await api.listTracks(
          limit: 5, offset: 10, q: '  lofi  ', sort: '');
      expect(tracks.single.id, 't1');
      expect(seen?.path, '/tracks');
      expect(seen?.queryParameters['limit'], 5);
      expect(seen?.queryParameters['offset'], 10);
      expect(seen?.queryParameters['q'], 'lofi');
      expect(seen?.queryParameters.containsKey('sort'), isFalse);
    });

    test('getStream hits the per-track ticket path', () async {
      RequestOptions? seen;
      final (_, api) = _service((o) async {
        seen = o;
        return jsonResponse({
          'data': {
            'url': 'https://cdn/x.m3u8',
            'expiresAt': '2026-01-01T00:00:00Z',
            'jti': 'j1',
            'quality': 'HQ',
          }
        }, 200);
      });
      final ticket = await api.getStream('t9');
      expect(seen?.path, '/tracks/t9/stream');
      expect(ticket.quality, 'HQ');
    });

    test('heartbeat posts position with optional jti', () async {
      final bodies = <Map<String, dynamic>>[];
      final (_, api) = _service((o) async {
        bodies.add(Map<String, dynamic>.from(o.data as Map));
        return jsonResponse(
            {
              'data': {'countedAsPlay': true, 'fraudSuspected': false}
            },
            200);
      });
      final res = await api.heartbeat(
          trackId: 't1', positionMs: 100, playMs: 5000, jti: 'j1');
      expect(res.countedAsPlay, isTrue);
      expect(bodies.single['jti'], 'j1');
      await api.heartbeat(trackId: 't1', positionMs: 0, playMs: 0);
      expect(bodies.last.containsKey('jti'), isFalse);
    });

    test('getPlaybackState returns null on null data', () async {
      var calls = 0;
      final (_, api) = _service((o) async {
        calls++;
        if (calls == 1) return jsonResponse({'data': null}, 200);
        return jsonResponse({
          'data': {'trackId': 't1', 'positionMs': 9000}
        }, 200);
      });
      expect(await api.getPlaybackState('t1'), isNull);
      expect((await api.getPlaybackState('t1'))?.positionMs, 9000);
    });

    test('setPlaybackState puts the position', () async {
      RequestOptions? seen;
      final (_, api) = _service((o) async {
        seen = o;
        return jsonResponse({
          'data': {'trackId': 't1', 'positionMs': 3000}
        }, 200);
      });
      final state = await api.setPlaybackState('t1', 3000);
      expect(seen?.path, '/playback/state/t1');
      expect(seen?.method, 'PUT');
      expect(state.positionMs, 3000);
    });
  });
}
