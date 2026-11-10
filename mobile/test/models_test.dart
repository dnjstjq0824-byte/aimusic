import 'package:avibe/core/models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Track distribution metadata', () {
    test('parses isrc/bpm/musicalKey', () {
      final t = Track.fromJson({
        'id': 't1',
        'partnerId': 'p1',
        'title': 'Neon',
        'aiProvider': 'SUNO',
        'aiGenerated': true,
        'status': 'READY',
        'playCount': 3,
        'isrc': 'KRA122600001',
        'bpm': 120.0,
        'musicalKey': 'C Major',
      });
      expect(t.isrc, 'KRA122600001');
      expect(t.bpm, 120.0);
      expect(t.musicalKey, 'C Major');
    });

    test('missing metadata defaults to null', () {
      final t = Track.fromJson({
        'id': 't1',
        'partnerId': 'p1',
        'title': 'Neon',
      });
      expect(t.isrc, isNull);
      expect(t.bpm, isNull);
      expect(t.musicalKey, isNull);
    });
  });

  group('StreamTicket', () {
    test('parses HQ grant', () {
      final t = StreamTicket.fromJson({
        'url': 'https://cdn/x/master.m3u8?sig=abc',
        'expiresAt': '2026-11-10T12:00:00Z',
        'jti': 'jti-1',
        'quality': 'HQ',
      });
      expect(t.url, contains('master.m3u8'));
      expect(t.jti, 'jti-1');
      expect(t.quality, 'HQ');
      expect(t.expiresAt, DateTime.parse('2026-11-10T12:00:00Z'));
    });

    test('defaults missing quality to SQ', () {
      final t = StreamTicket.fromJson({
        'url': 'https://cdn/x/stream_0.m3u8?sig=abc',
        'expiresAt': '2026-11-10T12:00:00Z',
        'jti': 'jti-2',
      });
      expect(t.quality, 'SQ');
    });
  });

  group('HeartbeatResult', () {
    test('parses counted + fraud flags', () {
      final r = HeartbeatResult.fromJson({
        'countedAsPlay': true,
        'fraudSuspected': false,
      });
      expect(r.countedAsPlay, isTrue);
      expect(r.fraudSuspected, isFalse);
    });

    test('missing flags default to false', () {
      final r = HeartbeatResult.fromJson({});
      expect(r.countedAsPlay, isFalse);
      expect(r.fraudSuspected, isFalse);
    });
  });

  group('StoreDevice', () {
    test('parses claim response device', () {
      final d = StoreDevice.fromJson({
        'id': 'dev-1',
        'businessId': 'biz-1',
        'deviceName': 'Counter Tablet',
        'status': 'ACTIVE',
        'volume': 80,
        'pairingCode': null,
        'lastSeenAt': '2026-11-10T12:00:00Z',
      });
      expect(d.id, 'dev-1');
      expect(d.deviceName, 'Counter Tablet');
      expect(d.volume, 80);
      expect(d.pairingCode, isNull);
    });

    test('missing volume defaults to 70', () {
      final d = StoreDevice.fromJson({
        'id': 'dev-1',
        'businessId': 'biz-1',
      });
      expect(d.volume, 70);
      expect(d.deviceName, '');
      expect(d.status, '');
    });
  });

  group('DeviceQueue', () {
    test('parses schedule gate + tracks', () {
      final q = DeviceQueue.fromJson({
        'businessId': 'biz-1',
        'businessName': 'Test Cafe',
        'open': true,
        'reason': '',
        'playlistId': 'pl-1',
        'playlistName': 'Morning',
        'volume': 65,
        'tracks': [
          {'trackId': 't1', 'title': 'Song A', 'durationMs': 180000},
          {'trackId': 't2', 'title': 'Song B'},
        ],
      });
      expect(q.open, isTrue);
      expect(q.tracks, hasLength(2));
      expect(q.tracks[0].durationMs, 180000);
      expect(q.tracks[1].durationMs, isNull);
      expect(q.volume, 65);
    });

    test('closed store carries the reason', () {
      final q = DeviceQueue.fromJson({
        'businessId': 'biz-1',
        'businessName': 'Test Cafe',
        'open': false,
        'reason': 'CLOSED_NOW',
        'tracks': [],
      });
      expect(q.open, isFalse);
      expect(q.reason, 'CLOSED_NOW');
      expect(q.tracks, isEmpty);
    });
  });

  group('AuthMe', () {
    test('flags missing age confirmation', () {
      final m = AuthMe.fromJson({'userId': 'u-1', 'role': 'USER'});
      expect(m.needsAgeConfirm, isTrue);
    });

    test('confirmed age skips the gate', () {
      final m = AuthMe.fromJson({
        'userId': 'u-1',
        'role': 'USER',
        'ageConfirmedAt': '2026-01-01T00:00:00Z',
      });
      expect(m.needsAgeConfirm, isFalse);
    });
  });
}
