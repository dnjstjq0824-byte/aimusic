// Auth repository: POST /auth/register, /login, /refresh, GET /auth/me.
library;

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../core/api_client.dart';
import '../core/models.dart';

/// Owns the token pair in secure storage plus auth REST calls.
class AuthRepository {
  AuthRepository(this._dio, this._storage);

  final Dio _dio;
  final FlutterSecureStorage _storage;

  Future<bool> get isLoggedIn async {
    final token = await _storage.read(key: TokenKeys.access);
    return token != null && token.isNotEmpty;
  }

  Future<(TokenPair, Map<String, dynamic>)> register({
    required String email,
    required String password,
    required String displayName,
    int? birthYear,
    required bool ageConfirmed,
    String? inviteCode,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/auth/register',
        data: {
          'email': email,
          'password': password,
          'displayName': displayName,
          if (birthYear != null) 'birthYear': birthYear,
          'ageConfirmed': ageConfirmed,
          if (inviteCode != null && inviteCode.isNotEmpty) 'inviteCode': inviteCode,
        },
      );
      // Register returns the user; log in immediately for a token pair.
      final tokens = await login(email: email, password: password);
      return (tokens, unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<TokenPair> login({
    required String email,
    required String password,
    String? totpCode,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/auth/login',
        data: {
          'email': email,
          'password': password,
          if (totpCode != null && totpCode.isNotEmpty) 'totpCode': totpCode,
        },
      );
      final data = unwrapData<Map<String, dynamic>>(res);
      final pair =
          TokenPair.fromJson(data['tokens'] as Map<String, dynamic>);
      await _storage.write(key: TokenKeys.access, value: pair.accessToken);
      await _storage.write(key: TokenKeys.refresh, value: pair.refreshToken);
      return pair;
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<AuthMe> me() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/auth/me');
      return AuthMe.fromJson(unwrapData<Map<String, dynamic>>(res));
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> confirmAge({int? birthYear}) async {
    try {
      await _dio.post<Map<String, dynamic>>(
        '/auth/age-confirm',
        data: {
          'ageConfirmed': true,
          if (birthYear != null) 'birthYear': birthYear,
        },
      );
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> logout() async {
    await _storage.delete(key: TokenKeys.access);
    await _storage.delete(key: TokenKeys.refresh);
  }
}
