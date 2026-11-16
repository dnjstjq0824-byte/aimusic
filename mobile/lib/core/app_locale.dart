// App locale: device mapping, user override, server sync (WI-R12).
// ARB codes: ko/en/ja/zh/th. Server codes: ko/en/ja/zh-cn/th.
// zh covers Simplified only; Traditional devices fall back to en.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../l10n/app_localizations.dart';
import 'providers.dart';

export '../l10n/app_localizations.dart';

/// Non-null l10n accessor for widget code.
extension L10nContext on BuildContext {
  AppLocalizations get l10n => AppLocalizations.of(this)!;
}

/// Supported in-app language codes (ARB set).
const appLanguageCodes = ['ko', 'en', 'ja', 'zh', 'th'];

/// Beta languages ship machine-draft ARB (G-L: native review pending).
const appBetaLanguages = ['ja', 'zh', 'th'];

/// Maps a device locale to an app language code.
/// zh-Hant (HK/TW) falls back to en; zh-Hans/unspecified zh maps to zh.
String resolveDeviceLanguage(Locale? device) {
  if (device == null) return 'en';
  switch (device.languageCode.toLowerCase()) {
    case 'ko':
      return 'ko';
    case 'ja':
      return 'ja';
    case 'th':
      return 'th';
    case 'zh':
      final script = device.scriptCode?.toLowerCase() ?? '';
      if (script == 'hant') return 'en';
      final country = device.countryCode?.toUpperCase() ?? '';
      if (country == 'HK' || country == 'TW' || country == 'MO') return 'en';
      return 'zh';
    default:
      return 'en';
  }
}

/// Normalizes a stored/server locale code to an app language code.
String normalizeAppLanguage(String? code) {
  switch ((code ?? '').trim().toLowerCase()) {
    case 'ko':
      return 'ko';
    case 'ja':
      return 'ja';
    case 'th':
      return 'th';
    case 'zh':
    case 'zh-cn':
    case 'zh-hans':
    case 'zh-hans-cn':
      return 'zh';
    case 'en':
    default:
      return 'en';
  }
}

/// Converts an app language code to the server locale vocabulary (A2).
String toServerLocale(String appCode) =>
    normalizeAppLanguage(appCode) == 'zh' ? 'zh-cn' : normalizeAppLanguage(appCode);

/// Self display name for a language code (locale-invariant by design).
String languageName(AppLocalizations s, String code) {
  switch (normalizeAppLanguage(code)) {
    case 'ko':
      return s.langKo;
    case 'ja':
      return s.langJa;
    case 'zh':
      return s.langZh;
    case 'th':
      return s.langTh;
    case 'en':
    default:
      return s.langEn;
  }
}

/// Display name for a social provider code (GOOGLE/KAKAO/APPLE/NAVER/LINE).
String providerDisplayName(AppLocalizations s, String code) {
  switch (code.trim().toUpperCase()) {
    case 'GOOGLE':
      return s.providerGoogle;
    case 'KAKAO':
      return s.providerKakao;
    case 'APPLE':
      return s.providerApple;
    case 'NAVER':
      return s.providerNaver;
    case 'LINE':
      return s.providerLine;
    default:
      return code;
  }
}

const _overrideKey = 'avibe.locale.override';

/// User language override; null means "follow the device".
final localeOverrideProvider =
    StateNotifierProvider<LocaleOverrideController, String?>(
  (ref) => LocaleOverrideController(ref.watch(secureStorageProvider)),
);

class LocaleOverrideController extends StateNotifier<String?> {
  LocaleOverrideController(this._storage) : super(null) {
    _load();
  }

  final FlutterSecureStorage _storage;

  Future<void> _load() async {
    try {
      final v = await _storage.read(key: _overrideKey);
      if (v != null && v.isNotEmpty) state = normalizeAppLanguage(v);
    } catch (_) {
      // Unreadable override: stay on device language.
    }
  }

  /// Applies a language locally (persisted). Callers sync to the server
  /// via PATCH /auth/me/locale when signed in.
  Future<bool> apply(String code) async {
    final norm = normalizeAppLanguage(code);
    state = norm;
    try {
      await _storage.write(key: _overrideKey, value: norm);
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<void> clear() async {
    state = null;
    try {
      await _storage.delete(key: _overrideKey);
    } catch (_) {}
  }
}
