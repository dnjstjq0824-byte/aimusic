// Runtime environment: base URL resolution for the API client.
library;

class Env {
  const Env._();

  /// Base URL override via `flutter run --dart-define=API_BASE_URL=...`.
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );

  /// API prefix matching backend/internal/handler/router.go (`/api/v1`).
  static const String apiPrefix = '/api/v1';

  /// Heartbeat interval (ms of wall clock between POST /playback/heartbeat).
  static const int heartbeatIntervalMs = 15000;

  /// Stream URL refresh margin before expiry (seconds).
  static const int streamRefreshMarginSecs = 30;
}
