// Auth + envelope interceptor behavior (P1): header attach, refresh retry,
// error mapping. No network, no platform channels.
library;

import 'package:avibe/core/api_client.dart';
import 'package:avibe/core/error.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

Dio _testDio(AdapterHandler handler) {
  final dio = Dio();
  dio.httpClientAdapter = FakeAdapter(handler);
  dio.interceptors.add(AuthInterceptor(
    dio: dio,
    storage: const FlutterSecureStorage(),
  ));
  dio.interceptors.add(EnvelopeInterceptor());
  return dio;
}

void main() {
  group('AuthInterceptor.onRequest', () {
    test('attaches the stored access token', () async {
      final seen = <String?>[];
      mockSecureStorage({TokenKeys.access: 'a1'});
      final dio = _testDio((o) async {
        seen.add(o.headers['Authorization'] as String?);
        return jsonResponse({'data': {}}, 200);
      });
      await dio.get('/tracks');
      expect(seen, ['Bearer a1']);
    });

    test('skips auth-free paths', () async {
      final seen = <String?>[];
      mockSecureStorage({TokenKeys.access: 'a1'});
      final dio = _testDio((o) async {
        seen.add(o.headers['Authorization'] as String?);
        return jsonResponse({'data': {}}, 200);
      });
      await dio.post('/auth/refresh', data: {});
      expect(seen, [null]);
    });
  });

  group('AuthInterceptor refresh flow', () {
    test('401 rotates the pair and retries once with the new token', () async {
      final store = mockSecureStorage({
        TokenKeys.access: 'old-a',
        TokenKeys.refresh: 'old-r',
      });
      final authHeaders = <String?>[];
      final dio = _testDio((o) async {
        if (o.path == '/auth/refresh') {
          return jsonResponse({
            'data': {'accessToken': 'new-a', 'refreshToken': 'new-r'}
          }, 200);
        }
        authHeaders.add(o.headers['Authorization'] as String?);
        if (o.headers['Authorization'] == 'Bearer new-a') {
          return jsonResponse({'data': {'ok': true}}, 200);
        }
        return jsonResponse(
            {'error': {'code': 'ERR_UNAUTHORIZED', 'message': 'expired'}},
            401);
      });
      final res = await dio.get('/tracks');
      expect(res.data, {'data': {'ok': true}});
      expect(authHeaders, ['Bearer old-a', 'Bearer new-a']);
      expect(store[TokenKeys.access], 'new-a');
      expect(store[TokenKeys.refresh], 'new-r');
    });

    test('failed rotation wipes tokens and notifies expiry', () async {
      final store = mockSecureStorage({
        TokenKeys.access: 'old-a',
        TokenKeys.refresh: 'dead-r',
      });
      final dio = _testDio((o) async => jsonResponse(
          {'error': {'code': 'ERR_UNAUTHORIZED', 'message': 'expired'}},
          401));
      final expired = AuthSessionEvents.stream.first;
      await expectLater(dio.get('/tracks'), throwsA(isA<DioException>()));
      await expired;
      expect(store.containsKey(TokenKeys.access), isFalse);
      expect(store.containsKey(TokenKeys.refresh), isFalse);
    });

    test('does not refresh non-401 failures or retried requests', () async {
      mockSecureStorage({TokenKeys.access: 'a', TokenKeys.refresh: 'r'});
      var refreshes = 0;
      final dio = _testDio((o) async {
        if (o.path == '/auth/refresh') refreshes++;
        return jsonResponse(
            {'error': {'code': 'ERR_FORBIDDEN', 'message': 'no'}}, 403);
      });
      await expectLater(dio.get('/x'), throwsA(isA<DioException>()));
      expect(refreshes, 0);
    });
  });

  group('EnvelopeInterceptor', () {
    test('maps error envelopes to ApiException', () async {
      mockSecureStorage();
      final dio = _testDio((o) async => jsonResponse(
          {
            'error': {'code': 'ERR_INVALID_INPUT', 'message': 'bad'}
          },
          400));
      final err = await dio
          .get('/x')
          .then<Object>((_) => 'no-throw', onError: (e) => e as Object);
      expect(err, isA<DioException>());
      final apiErr = (err as DioException).error;
      expect(apiErr, isA<ApiException>());
      expect((apiErr as ApiException).code, 'ERR_INVALID_INPUT');
      expect((apiErr).statusCode, 400);
    });
  });

  group('toApiException', () {
    test('maps timeouts and connection errors to network', () {
      for (final t in [
        DioExceptionType.connectionTimeout,
        DioExceptionType.receiveTimeout,
        DioExceptionType.connectionError,
      ]) {
        final e = toApiException(DioException(
          requestOptions: RequestOptions(path: '/x'),
          type: t,
        ));
        expect(e.code, ApiCodes.networkError);
      }
    });

    test('passes ApiException through and wraps the rest', () {
      final inner = ApiException('ERR_X', 'm');
      expect(
          toApiException(DioException(
            requestOptions: RequestOptions(path: '/x'),
            error: inner,
          )),
          same(inner));
      expect(toApiException(inner), same(inner));
      expect(toApiException(StateError('x')).code, ApiCodes.internal);
    });
  });

  group('unwrapData', () {
    test('extracts the data field', () {
      final res = Response<Map<String, dynamic>>(
        requestOptions: RequestOptions(path: '/x'),
        data: {
          'data': {'a': 1}
        },
      );
      expect(unwrapData<Map<String, dynamic>>(res), {'a': 1});
    });
  });
}
