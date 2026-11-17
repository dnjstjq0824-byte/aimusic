// WI-R12: locale resolution + social return-URI unit tests (pure Dart).
import 'package:avibe/core/app_locale.dart';
import 'package:avibe/services/social_auth_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('resolveDeviceLanguage', () {
    test('direct mappings', () {
      expect(resolveDeviceLanguage(const Locale('ko')), 'ko');
      expect(resolveDeviceLanguage(const Locale('ja')), 'ja');
      expect(resolveDeviceLanguage(const Locale('th')), 'th');
      expect(resolveDeviceLanguage(const Locale('en')), 'en');
    });

    test('simplified chinese maps to zh', () {
      expect(resolveDeviceLanguage(const Locale('zh')), 'zh');
      expect(resolveDeviceLanguage(const Locale.fromSubtags(
          languageCode: 'zh', scriptCode: 'Hans', countryCode: 'CN')), 'zh');
    });

    test('traditional chinese falls back to en', () {
      expect(resolveDeviceLanguage(const Locale.fromSubtags(
          languageCode: 'zh', scriptCode: 'Hant', countryCode: 'HK')), 'en');
      expect(resolveDeviceLanguage(const Locale.fromSubtags(
          languageCode: 'zh', scriptCode: 'Hant', countryCode: 'TW')), 'en');
      expect(resolveDeviceLanguage(const Locale.fromSubtags(
          languageCode: 'zh', countryCode: 'TW')), 'en');
    });

    test('unknown and null fall back to en', () {
      expect(resolveDeviceLanguage(const Locale('de')), 'en');
      expect(resolveDeviceLanguage(null), 'en');
    });
  });

  group('server locale vocabulary', () {
    test('zh maps to zh-cn, others pass through', () {
      expect(toServerLocale('zh'), 'zh-cn');
      expect(toServerLocale('ko'), 'ko');
      expect(toServerLocale('en'), 'en');
      expect(toServerLocale('ja'), 'ja');
      expect(toServerLocale('th'), 'th');
    });

    test('normalizeAppLanguage accepts server codes', () {
      expect(normalizeAppLanguage('zh-cn'), 'zh');
      expect(normalizeAppLanguage('zh-Hans'), 'zh');
      expect(normalizeAppLanguage(' KO '), 'ko');
      expect(normalizeAppLanguage('xx'), 'en');
      expect(normalizeAppLanguage(null), 'en');
    });
  });

  group('social return URIs', () {
    test('start URL carries platform=app', () {
      final u = SocialAuthService.startUrl('GOOGLE');
      expect(u.queryParameters['platform'], 'app');
      expect(u.path, endsWith('/auth/oauth/google/start'));
    });

    test('isReturnUri matches only the app callback', () {
      expect(
          SocialAuthService.isReturnUri(
              Uri.parse('avibe://oauth/callback?code=abc')),
          isTrue);
      expect(
          SocialAuthService.isReturnUri(
              Uri.parse('avibe://oauth/callback?oauthError=x')),
          isTrue);
      expect(SocialAuthService.isReturnUri(Uri.parse('avibe://other/x')),
          isFalse);
      expect(SocialAuthService.isReturnUri(
          Uri.parse('https://x/oauth/callback?code=abc')), isFalse);
    });
  });
}
