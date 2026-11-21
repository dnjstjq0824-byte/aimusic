// Social login: system-browser start + app-link return (WI-R12).
// Flow: GET /auth/oauth/providers (enabled buttons) -> open
// /auth/oauth/<p>/start?platform=app in the system browser ->
// provider -> backend callback -> 302 to avibe://oauth/callback ->
// app_links -> POST /auth/oauth/exchange (C-16: tokens never in URLs).
//
// iOS App Store policy: shipping social login on iOS requires Apple
// Sign-In alongside the other providers. The backend already adapts
// Apple (A1); the iOS release MUST configure the Apple key so the
// button appears. This client renders whatever the backend enables
// and never hardcodes the provider set.
library;

import 'dart:async';

import 'package:app_links/app_links.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:url_launcher/url_launcher.dart';

import '../core/api_client.dart';
import '../core/env.dart';
import '../core/error.dart';
import '../core/models.dart';

/// Custom scheme the backend redirects to for platform=app
/// (backend/internal/handler/auth.go MobileOAuthCallbackBase).
const oauthAppScheme = 'avibe';
const oauthAppHost = 'oauth';
const oauthAppPath = '/callback';

/// One row of GET /auth/oauth/providers (WI-A1 contract).
class OAuthProviderInfo {
  OAuthProviderInfo({required this.provider, required this.enabled});

  factory OAuthProviderInfo.fromJson(Map<String, dynamic> j) =>
      OAuthProviderInfo(
        provider: (j['provider'] as String?) ?? '',
        enabled: j['enabled'] == true,
      );

  final String provider;
  final bool enabled;
}

class SocialAuthCancelled implements Exception {
  const SocialAuthCancelled();
}

/// The provider gave no email: the app must show the complete-email
/// screen with this one-time signup token (WI-N9, web parity).
class SocialEmailRequired implements Exception {
  const SocialEmailRequired(this.emailToken);

  final String emailToken;
}

/// Injectable seams for widget tests (real url_launcher/app_links
/// need a device; unit-test the pure URI helpers instead).
typedef LinkStreamFactory = Stream<Uri> Function();
typedef UrlOpener = Future<bool> Function(Uri url);

class SocialAuthService {
  SocialAuthService(
    this._dio,
    this._storage, {
    LinkStreamFactory? linkStream,
    UrlOpener? openUrl,
    Duration returnTimeout = const Duration(minutes: 5),
  })  : _linkStream = linkStream,
        _openUrl = openUrl,
        _returnTimeout = returnTimeout;

  final Dio _dio;
  final FlutterSecureStorage _storage;
  final LinkStreamFactory? _linkStream;
  final UrlOpener? _openUrl;
  final Duration _returnTimeout;

  /// Providers in backend locale display order; only enabled ones
  /// should render as buttons (A1: key missing -> hidden).
  Future<List<OAuthProviderInfo>> providers(String locale) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/auth/oauth/providers',
        queryParameters: {'locale': locale},
      );
      final list = unwrapData<List<dynamic>>(res);
      return list
          .map((e) => OAuthProviderInfo.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// Authorization start URL for the system browser.
  static Uri startUrl(String provider) => Uri.parse(
        '${Env.apiBaseUrl}${Env.apiPrefix}/auth/oauth/${provider.toLowerCase()}'
        '/start?platform=app',
      );

  /// True for the backend's app-link return (code or oauthError).
  static bool isReturnUri(Uri u) =>
      u.scheme == oauthAppScheme &&
      u.host == oauthAppHost &&
      u.path == oauthAppPath;

  /// Runs the browser flow; returns the one-time exchange code.
  /// Throws [SocialAuthCancelled] on timeout/user cancel,
  /// [SocialEmailRequired] when the provider gave no email (WI-N9),
  /// [ApiException] when the backend reports an oauthError.
  Future<String> signIn(String provider) async {
    final stream = _linkStream?.call() ?? AppLinks().uriLinkStream;
    final opener = _openUrl ?? _openExternal;
    final completer = Completer<String>();
    late final StreamSubscription<Uri> sub;
    sub = stream.listen(
      (u) {
        if (!isReturnUri(u)) return;
        final err = u.queryParameters['oauthError'];
        if (err != null && err.isNotEmpty) {
          if (!completer.isCompleted) {
            completer.completeError(ApiException(
                ApiCodes.invalidCredentials, 'oauthError=$err'));
          }
          return;
        }
        // Link-intent returns carry no login code (mobile never sends a
        // link intent); treat as a completed-but-cancelled flow.
        if (u.queryParameters.containsKey('linked')) {
          if (!completer.isCompleted) {
            completer.completeError(const SocialAuthCancelled());
          }
          return;
        }
        // Email-less provider: the app collects the email, then calls
        // POST /auth/oauth/complete-email with this token (WI-N9).
        final emailToken = u.queryParameters['emailToken'] ?? '';
        if (emailToken.isNotEmpty && !completer.isCompleted) {
          completer.completeError(SocialEmailRequired(emailToken));
          return;
        }
        final code = u.queryParameters['code'] ?? '';
        if (code.isNotEmpty && !completer.isCompleted) {
          completer.complete(code);
        }
      },
      onError: (_) {
        if (!completer.isCompleted) {
          completer.completeError(const SocialAuthCancelled());
        }
      },
    );
    try {
      final ok = await opener(startUrl(provider));
      if (!ok) throw const SocialAuthCancelled();
      return await completer.future.timeout(
        _returnTimeout,
        onTimeout: () => throw const SocialAuthCancelled(),
      );
    } finally {
      await sub.cancel();
    }
  }

  /// Files the typed email for a provider identity that gave none
  /// (WI-N9, web parity: POST /auth/oauth/complete-email). The server
  /// mails a verification link; the account is created when clicked.
  /// Returns the filed email on success.
  Future<String> completeEmail({
    required String token,
    required String email,
    required String locale,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/auth/oauth/complete-email',
        data: {'token': token, 'email': email, 'locale': locale},
      );
      final data = unwrapData<Map<String, dynamic>>(res);
      return (data['email'] as String?) ?? email;
    } catch (e) {
      throw toApiException(e);
    }
  }

  /// Swaps the one-time code for a token pair and stores it (C-16).
  Future<TokenPair> exchange(String code) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/auth/oauth/exchange',
        data: {'code': code},
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

  Future<bool> _openExternal(Uri url) {
    // System browser (Custom Tabs / SFSafariViewController-backed):
    // the OAuth start/provider/callback redirects all run in one
    // browser context so the C-12 cookies stay bound there.
    return launchUrl(url, mode: LaunchMode.externalApplication);
  }
}
