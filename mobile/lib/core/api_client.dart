// Dio client with Bearer [REDACTED] interceptor, refresh-on-401, and the
// {"data": ...} / {"error": {"code","message"}} envelope handling.
library;

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'env.dart';
import 'error.dart';
import 'models.dart';

/// Secure-storage keys for the token pair.
abstract final class TokenKeys {
  static const access = 'avibe.access_token';
  static const refresh = 'avibe.refresh_token';
}

/// Builds the shared Dio instance rooted at [Env.apiBaseUrl]/api/v1.
class ApiClientFactory {
  static Dio create({FlutterSecureStorage? storage}) {
    final dio = Dio(
      BaseOptions(
        baseUrl: Env.apiBaseUrl + Env.apiPrefix,
        connectTimeout: const Duration(seconds: 10),
        receiveTimeout: const Duration(seconds: 20),
        contentType: 'application/json',
      ),
    );
    dio.interceptors.add(
      AuthInterceptor(
        dio: dio,
        storage: storage ?? const FlutterSecureStorage(),
      ),
    );
    dio.interceptors.add(EnvelopeInterceptor());
    return dio;
  }
}

/// Attaches `Authorization: Bearer <access>` and retries once after refresh.
class AuthInterceptor extends Interceptor {
  AuthInterceptor({required this.dio, required this.storage});

  final Dio dio;
  final FlutterSecureStorage storage;

  bool _refreshing = false;

  static const _authFree = {'/auth/login', '/auth/register', '/auth/refresh'};

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    if (!_authFree.contains(options.path)) {
      final access = await storage.read(key: TokenKeys.access);
      if (access != null && access.isNotEmpty) {
        options.headers['Authorization'] = 'Bearer $access';
      }
    }
    handler.next(options);
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final status = err.response?.statusCode;
    final req = err.requestOptions;
    if (status == 401 &&
        !_authFree.contains(req.path) &&
        req.extra['retried'] != true &&
        !_refreshing) {
      _refreshing = true;
      try {
        final refresh = await storage.read(key: TokenKeys.refresh);
        if (refresh != null && refresh.isNotEmpty) {
          final res = await dio.post<Map<String, dynamic>>(
            '/auth/refresh',
            data: {'refreshToken': refresh},
          );
          final data = res.data?['data'] as Map<String, dynamic>?;
          if (data != null) {
            final pair = TokenPair.fromJson(data);
            await storage.write(key: TokenKeys.access, value: pair.accessToken);
            await storage.write(
                key: TokenKeys.refresh, value: pair.refreshToken);
            req.extra['retried'] = true;
            req.headers['Authorization'] = 'Bearer ${pair.accessToken}';
            final retry = await dio.fetch<Map<String, dynamic>>(req);
            _refreshing = false;
            return handler.resolve(retry);
          }
        }
      } catch (_) {
        // Fall through to the original 401 below.
      }
      _refreshing = false;
    }
    handler.next(err);
  }

  Future<void> clear() async {
    await storage.delete(key: TokenKeys.access);
    await storage.delete(key: TokenKeys.refresh);
  }
}

/// Unwraps {"data"} on success and throws [ApiException] on {"error"}.
class EnvelopeInterceptor extends Interceptor {
  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    final res = err.response;
    if (res?.data is Map<String, dynamic>) {
      final body = res!.data as Map<String, dynamic>;
      final error = body['error'];
      if (error is Map<String, dynamic>) {
        return handler.reject(
          DioException(
            requestOptions: err.requestOptions,
            response: res,
            type: err.type,
            error: ApiException(
              (error['code'] as String?) ?? ApiCodes.internal,
              (error['message'] as String?) ?? 'request failed',
              statusCode: res.statusCode,
            ),
          ),
        );
      }
    }
    if (err.error is ApiException) return handler.next(err);
    handler.next(err);
  }
}

/// Extracts `data` from a decoded envelope body.
T unwrapData<T>(Response<Map<String, dynamic>> res) =>
    res.data!['data'] as T;

/// Converts any Dio failure into an [ApiException].
ApiException toApiException(Object e) {
  if (e is DioException) {
    if (e.error is ApiException) return e.error as ApiException;
    final status = e.response?.statusCode;
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.connectionError) {
      return ApiException(
          ApiCodes.internal, '네트워크 연결에 실패했습니다.', statusCode: status);
    }
    return ApiException(ApiCodes.internal, '요청에 실패했습니다.',
        statusCode: status);
  }
  if (e is ApiException) return e;
  return ApiException(ApiCodes.internal, e.toString());
}
