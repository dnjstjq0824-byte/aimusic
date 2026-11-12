// Runtime environment: base URL resolution for the API client.
library;

import 'package:flutter/foundation.dart';

class Env {
  const Env._();

  /// Base URL override via `flutter run --dart-define=API_BASE_URL=...`.
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );

  /// True for loopback hosts (desktop + emulator dev servers).
  static bool isLoopbackUrl(String url) {
    final host = Uri.tryParse(url)?.host.toLowerCase() ?? '';
    return host == 'localhost' ||
        host == '127.0.0.1' ||
        host == '::1' ||
        host == '[::1]' ||
        host == '10.0.2.2';
  }

  /// Fail fast when a release build points at a dev server (F-07).
  /// Called from main() before runApp; debug/profile builds always pass.
  static void assertReleaseConfig() {
    if (kReleaseMode && isLoopbackUrl(apiBaseUrl)) {
      throw StateError(
        'Release build points at a loopback API ($apiBaseUrl). '
        'Rebuild with --dart-define=API_BASE_URL=https://....',
      );
    }
  }

  /// API prefix matching backend/internal/handler/router.go (`/api/v1`).
  static const String apiPrefix = '/api/v1';

  /// Heartbeat interval (ms of wall clock between POST /playback/heartbeat).
  static const int heartbeatIntervalMs = 15000;

  /// Stream URL refresh margin before expiry (seconds).
  static const int streamRefreshMarginSecs = 30;
}
