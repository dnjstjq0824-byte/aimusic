// AuthRepository behavior (P1): token persistence, register chain,
// error mapping. Dio + in-memory secure storage, no network.
library;

import 'package:avibe/core/api_client.dart';
import 'package:avibe/core/error.dart';
import 'package:avibe/services/auth_repository.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

AuthRepository _repo(Map<String, String> store, AdapterHandler handler) {
  final dio = Dio();
  dio.httpClientAdapter = FakeAdapter(handler);
  dio.interceptors.add(EnvelopeInterceptor());
  return AuthRepository(dio, const FlutterSecureStorage());
}

void main() {
  group('AuthRepository', () {
    test('login stores the token pair', () async {
      final store = mockSecureStorage();
      Map<String, dynamic>? seenBody;
      final repo = _repo(store, (o) async {
        seenBody = Map<String, dynamic>.from(o.data as Map);
        return jsonResponse({
          'data': {
            'tokens': {'accessToken': 'a1', 'refreshToken': 'r1'}
          }
        }, 200);
      });
      final pair = await repo.login(email: 'u@x.com', password: 'pw');
      expect(pair.accessToken, 'a1');
      expect(seenBody?['email'], 'u@x.com');
      expect(seenBody?.containsKey('totpCode'), isFalse);
      expect(store[TokenKeys.access], 'a1');
      expect(store[TokenKeys.refresh], 'r1');
    });

    test('login includes totp only when provided', () async {
      final store = mockSecureStorage();
      final bodies = <Map<String, dynamic>>[];
      final repo = _repo(store, (o) async {
        bodies.add(Map<String, dynamic>.from(o.data as Map));
        return jsonResponse({
          'data': {
            'tokens': {'accessToken': 'a', 'refreshToken': 'r'}
          }
        }, 200);
      });
      await repo.login(email: 'u@x.com', password: 'pw', totpCode: '123456');
      expect(bodies.single['totpCode'], '123456');
    });

    test('login maps envelope errors to ApiException', () async {
      final store = mockSecureStorage();
      final repo = _repo(
          store,
          (o) async => jsonResponse(
              {
                'error': {
                  'code': 'ERR_INVALID_CREDENTIALS',
                  'message': 'nope'
                }
              },
              401));
      final err = await repo
          .login(email: 'u@x.com', password: 'bad')
          .then<Object>((_) => 'no-throw', onError: (e) => e as Object);
      expect((err as ApiException).code, 'ERR_INVALID_CREDENTIALS');
      expect(store, isEmpty);
    });

    test('register chains into login', () async {
      final store = mockSecureStorage();
      final paths = <String>[];
      final repo = _repo(store, (o) async {
        paths.add(o.path);
        return jsonResponse({
          'data': {
            'tokens': {'accessToken': 'a', 'refreshToken': 'r'}
          }
        }, 200);
      });
      await repo.register(
          email: 'n@x.com',
          password: 'pw',
          displayName: 'N',
          ageConfirmed: true);
      expect(paths, ['/auth/register', '/auth/login']);
      expect(store[TokenKeys.access], 'a');
    });

    test('isLoggedIn, logout and me', () async {
      final store = mockSecureStorage();
      final repo = _repo(
          store,
          (o) async => jsonResponse({
                'data': {'userId': 'u1', 'role': 'USER'}
              }, 200));
      expect(await repo.isLoggedIn, isFalse);
      store[TokenKeys.access] = 'a';
      expect(await repo.isLoggedIn, isTrue);
      final me = await repo.me();
      expect(me.userId, 'u1');
      await repo.logout();
      expect(store, isEmpty);
    });

    test('confirmAge sends the flag and optional birth year', () async {
      final store = mockSecureStorage();
      final bodies = <Map<String, dynamic>>[];
      final repo = _repo(store, (o) async {
        bodies.add(Map<String, dynamic>.from(o.data as Map));
        return jsonResponse({'data': {}}, 200);
      });
      await repo.confirmAge(birthYear: 2000);
      expect(bodies.single, {'ageConfirmed': true, 'birthYear': 2000});
      await repo.confirmAge();
      expect(bodies.last.containsKey('birthYear'), isFalse);
    });
  });
}
