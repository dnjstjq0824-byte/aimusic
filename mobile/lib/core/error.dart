// Backend error envelope mapping.
// Success: {"data": ...}  Failure: {"error": {"code": ..., "message": ...}}
// Codes from backend/internal/handler/middleware.go + domain/errors.go.
library;

import '../l10n/app_localizations.dart';

/// Stable backend error codes the app handles explicitly.
abstract final class ApiCodes {
  static const unauthorized = 'ERR_UNAUTHORIZED';
  static const forbidden = 'ERR_FORBIDDEN';
  static const notFound = 'ERR_NOT_FOUND';
  static const notStreamable = 'ERR_NOT_STREAMABLE';
  static const invalidInput = 'ERR_INVALID_INPUT';
  static const rateLimited = 'ERR_RATE_LIMITED';
  static const emailTaken = 'ERR_EMAIL_TAKEN';
  static const invalidCredentials = 'ERR_INVALID_CREDENTIALS';
  static const userInactive = 'ERR_USER_INACTIVE';
  static const totpRequired = 'ERR_TOTP_REQUIRED';
  static const ageConfirmRequired = 'ERR_AGE_CONFIRM_REQUIRED';
  static const consentRequired = 'ERR_CONSENT_REQUIRED';
  static const totpInvalid = 'ERR_TOTP_INVALID';
  static const loginLocked = 'LOGIN_LOCKED';
  static const providerNotAllowed = 'ERR_PROVIDER_NOT_ALLOWED';
  static const internal = 'ERR_INTERNAL';

  /// Client-side transport failures (never sent by the backend).
  static const networkError = 'ERR_NETWORK';
  static const requestFailed = 'ERR_REQUEST_FAILED';
}

/// Typed API failure carrying the backend machine code.
class ApiException implements Exception {
  ApiException(this.code, this.message, {this.statusCode});

  final String code;
  final String message;
  final int? statusCode;

  /// True when the track cannot be played (takedown / not READY).
  /// The audio handler auto-skips on these.
  bool get isNotStreamable =>
      code == ApiCodes.notStreamable || code == ApiCodes.notFound;

  bool get isAuthError =>
      code == ApiCodes.unauthorized ||
      code == ApiCodes.invalidCredentials ||
      code == ApiCodes.userInactive;

  @override
  String toString() => 'ApiException($code): $message';
}

/// Localized human-readable message per code (WI-R12: ARB-backed).
String friendlyMessage(ApiException e, AppLocalizations s) {
  switch (e.code) {
    case ApiCodes.notStreamable:
      return s.errNotStreamable;
    case ApiCodes.notFound:
      return s.errNotFound;
    case ApiCodes.unauthorized:
    case ApiCodes.invalidCredentials:
      return s.errAuth;
    case ApiCodes.userInactive:
      return s.errInactive;
    case ApiCodes.forbidden:
      return s.errForbidden;
    case ApiCodes.rateLimited:
      return s.errRateLimited;
    case ApiCodes.emailTaken:
      return s.errEmailTaken;
    case ApiCodes.totpRequired:
      return s.errTotpRequired;
    case ApiCodes.ageConfirmRequired:
      return s.errAgeConfirm;
    case ApiCodes.consentRequired:
      return s.errConsent;
    case ApiCodes.totpInvalid:
      return s.errTotpInvalid;
    case ApiCodes.loginLocked:
      return s.errLoginLocked;
    case ApiCodes.providerNotAllowed:
      return s.errProviderNotAllowed;
    case ApiCodes.networkError:
      return s.errNetwork;
    case ApiCodes.requestFailed:
      return s.errRequest;
    case ApiCodes.invalidInput:
      return s.errInvalidInput(e.message);
    default:
      return e.message.isEmpty ? s.errGeneric : e.message;
  }
}
