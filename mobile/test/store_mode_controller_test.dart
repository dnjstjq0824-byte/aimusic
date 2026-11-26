// StoreModeController state machine (P1): claim/restore/unclaim,
// queue refresh, auth-loss handling. Fake device API, no audio.
library;

import 'package:avibe/core/error.dart';
import 'package:avibe/core/models.dart';
import 'package:avibe/core/providers.dart';
import 'package:avibe/features/store/store_mode_controller.dart';
import 'package:avibe/services/device_api_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

StoreDevice _device() => StoreDevice(
      id: 'd1',
      businessId: 'b1',
      deviceName: 'Counter',
      status: 'ACTIVE',
      volume: 70,
    );

DeviceQueue _queue() => DeviceQueue(
      businessId: 'b1',
      businessName: 'S',
      open: true,
      reason: 'scheduled',
      volume: 60,
      tracks: [
        DeviceQueueTrack(trackId: 't1', title: 'A'),
        DeviceQueueTrack(trackId: 't2', title: 'B'),
      ],
    );

class FakeDeviceApi extends DeviceApiService {
  FakeDeviceApi() : super(Dio(), const FlutterSecureStorage());

  String? token;
  StoreDevice? device;
  DeviceQueue? queueToReturn;
  Object? failure;

  @override
  Future<String?> get deviceToken async => token;

  @override
  Future<void> saveDeviceToken(String token) async => this.token = token;

  @override
  Future<void> clearDeviceToken() async => token = null;

  @override
  Future<(StoreDevice, String)> claim(String code) async {
    if (failure != null) throw failure!;
    await saveDeviceToken('tok-new');
    return (device!, 'tok-new');
  }

  @override
  Future<StoreDevice> me() async {
    if (failure != null) throw failure!;
    return device!;
  }

  @override
  Future<DeviceQueue> queue() async {
    if (failure != null) throw failure!;
    return queueToReturn!;
  }
}

Future<ProviderContainer> _container(FakeDeviceApi api) async {
  final container = ProviderContainer(
    overrides: [deviceApiProvider.overrideWithValue(api)],
  );
  addTearDown(container.dispose);
  // build() schedules restore() in a microtask; let it settle.
  container.read(storeModeProvider);
  await Future<void>.delayed(Duration.zero);
  await Future<void>.delayed(Duration.zero);
  return container;
}

void main() {
  group('StoreModeController', () {
    test('restore without a token needs claim', () async {
      final container = await _container(FakeDeviceApi());
      final s = container.read(storeModeProvider);
      expect(s.loading, isFalse);
      expect(s.needsClaim, isTrue);
      expect(s.queue, isNull);
    });

    test('restore with a token loads device and queue', () async {
      final api = FakeDeviceApi()
        ..token = 'tok-0'
        ..device = _device()
        ..queueToReturn = _queue();
      final container = await _container(api);
      final s = container.read(storeModeProvider);
      expect(s.needsClaim, isFalse);
      expect(s.device?.id, 'd1');
      expect(s.queue?.tracks, hasLength(2));
      expect(s.currentTrack?.trackId, 't1');
    });

    test('restore 401 clears the token and needs claim', () async {
      final api = FakeDeviceApi()
        ..token = 'dead'
        ..failure = ApiException(ApiCodes.unauthorized, 'expired');
      final container = await _container(api);
      final s = container.read(storeModeProvider);
      expect(api.token, isNull);
      expect(s.needsClaim, isTrue);
    });

    test('claim loads and unclaim resets', () async {
      final api = FakeDeviceApi()
        ..device = _device()
        ..queueToReturn = _queue();
      final container = await _container(api);
      final notifier = container.read(storeModeProvider.notifier);
      await notifier.claim('AB12');
      var s = container.read(storeModeProvider);
      expect(s.needsClaim, isFalse);
      expect(api.token, 'tok-new');
      expect(s.queue?.tracks, hasLength(2));
      await notifier.unclaim();
      s = container.read(storeModeProvider);
      expect(api.token, isNull);
      expect(s.needsClaim, isTrue);
      expect(s.device, isNull);
      expect(s.queue, isNull);
    });

    test('claim failure surfaces the message', () async {
      final api = FakeDeviceApi()
        ..failure = ApiException(ApiCodes.invalidInput, 'bad code');
      final container = await _container(api);
      final notifier = container.read(storeModeProvider.notifier);
      await notifier.claim('NOPE');
      final s = container.read(storeModeProvider);
      expect(s.error, isNotNull);
      expect(s.needsClaim, isTrue);
    });

    test('player controls are safe with no queue', () async {
      final container = await _container(FakeDeviceApi());
      final notifier = container.read(storeModeProvider.notifier);
      await notifier.togglePlay();
      await notifier.next();
      await notifier.previous();
      expect(container.read(storeModeProvider).queue, isNull);
    });
  });
}
