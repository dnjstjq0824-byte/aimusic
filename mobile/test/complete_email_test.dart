import 'package:avibe/core/app_locale.dart';
import 'package:avibe/core/providers.dart';
import 'package:avibe/features/auth/complete_email_screen.dart';
import 'package:avibe/l10n/app_localizations.dart';
import 'package:avibe/services/social_auth_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

/// Dio that answers POST /auth/oauth/complete-email without a network.
Dio stubDio({required String email, int status = 200}) {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        if (options.path == '/auth/oauth/complete-email' &&
            options.method == 'POST') {
          handler.resolve(
            Response<Map<String, dynamic>>(
              requestOptions: options,
              statusCode: status,
              data: {
                'data': {
                  'provider': 'NAVER',
                  'email': email,
                  'emailVerificationRequired': true,
                },
              },
            ),
          );
          return;
        }
        handler.next(options);
      },
    ),
  );
  return dio;
}

Widget harness(SocialAuthService social) {
  return ProviderScope(
    overrides: [socialAuthProvider.overrideWithValue(social)],
    child: const MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: Locale('en'),
      home: CompleteEmailScreen(token: 'tok-123'),
    ),
  );
}

SocialAuthService stubSocial({required String email}) => SocialAuthService(
      stubDio(email: email),
      const FlutterSecureStorage(),
    );

void main() {
  group('CompleteEmailScreen (N9)', () {
    testWidgets('email submit shows the pending notice', (tester) async {
      await tester.pumpWidget(harness(stubSocial(email: 'typed@example.com')));
      await tester.pumpAndSettle();
      expect(find.text('Enter your email'), findsOneWidget);

      await tester.enterText(
          find.byType(TextField), 'Typed@Example.com');
      await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
      await tester.pumpAndSettle();

      expect(find.text('Verification email sent'), findsOneWidget);
      expect(
        find.textContaining('typed@example.com'),
        findsOneWidget,
      );
    });

    testWidgets('blank email shows a validation error', (tester) async {
      await tester.pumpWidget(harness(stubSocial(email: 'x@y.z')));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
      await tester.pump();
      expect(find.text('Enter a valid email address.'), findsOneWidget);
    });
  });

  group('SocialEmailRequired (N9)', () {
    test('signIn surfaces the email token from the app link', () async {
      final social = SocialAuthService(
        Dio(),
        const FlutterSecureStorage(),
        linkStream: () => Stream.value(Uri.parse(
            'avibe://oauth/callback?emailToken=tok-abc')),
        openUrl: (_) async => true,
        returnTimeout: const Duration(seconds: 5),
      );
      try {
        await social.signIn('naver');
        fail('must throw SocialEmailRequired');
      } on SocialEmailRequired catch (e) {
        expect(e.emailToken, 'tok-abc');
      }
    });
  });
}
