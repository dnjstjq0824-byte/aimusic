// Backend error envelope mapping.
// Success: {"data": ...}  Failure: {"error": {"code": ..., "message": ...}}
// Codes from backend/internal/handler/middleware.go + domain/errors.go.
library;

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

/// Human-readable Korean fallback messages per code.
String friendlyMessage(ApiException e) {
  switch (e.code) {
    case ApiCodes.notStreamable:
      return '재생할 수 없는 곡입니다 (차단 또는 심사 중). 다음 곡으로 이동합니다.';
    case ApiCodes.notFound:
      return '요청한 항목을 찾을 수 없습니다.';
    case ApiCodes.unauthorized:
    case ApiCodes.invalidCredentials:
      return '로그인이 필요하거나 인증이 만료되었습니다.';
    case ApiCodes.userInactive:
      return '비활성화된 계정입니다. 고객센터에 문의하세요.';
    case ApiCodes.forbidden:
      return '권한이 없습니다.';
    case ApiCodes.rateLimited:
      return '요청이 너무 많습니다. 잠시 후 다시 시도하세요.';
    case ApiCodes.emailTaken:
      return '이미 등록된 이메일입니다.';
    case ApiCodes.totpRequired:
      return '2단계 인증 코드가 필요합니다.';
    case ApiCodes.ageConfirmRequired:
      return '만 14세 이상 확인이 필요합니다. 프로필에서 연령을 확인하세요.';
    case ApiCodes.consentRequired:
      return '필수 약관 동의가 필요합니다. 약관 화면에서 동의하세요.';
    case ApiCodes.totpInvalid:
      return '인증 코드가 올바르지 않습니다. 다시 입력하세요.';
    case ApiCodes.loginLocked:
      return '로그인 시도가 너무 많습니다. 15분 후 다시 시도하세요.';
    case ApiCodes.providerNotAllowed:
      return '허용 목록에 없는 제공자입니다. 허용 제휴 제공자를 선택하세요.';
    case ApiCodes.invalidInput:
      return '입력값이 올바르지 않습니다: ${e.message}';
    default:
      return e.message.isEmpty ? '일시적인 오류가 발생했습니다.' : e.message;
  }
}
