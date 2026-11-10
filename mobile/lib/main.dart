// App entry: Riverpod scope, go_router shell, dark theme.
library;

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/api_client.dart';
import 'core/providers.dart';
import 'features/auth/age_gate.dart';
import 'core/theme.dart';
import 'features/auth/login_screen.dart';
import 'features/home/home_screen.dart';
import 'features/legal/legal_screens.dart';
import 'features/player/player_screen.dart';
import 'features/playlist/playlist_screens.dart';
import 'features/search/search_screen.dart';
import 'features/store/store_screen.dart';

void main() {
  runApp(const ProviderScope(child: AvibeApp()));
}

final _router = GoRouter(
  initialLocation: '/',
  routes: [
    StatefulShellRoute.indexedStack(
      builder: (context, state, nav) => _Shell(nav: nav),
      branches: [
        StatefulShellBranch(
          routes: [GoRoute(path: '/', builder: (_, __) => const HomeScreen())],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(path: '/search', builder: (_, __) => const SearchScreen())
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/playlists',
              builder: (_, __) => const PlaylistListScreen(),
              routes: [
                GoRoute(
                  path: ':id',
                  builder: (_, s) => PlaylistDetailScreen(
                    playlistId: s.pathParameters['id']!,
                  ),
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(path: '/player', builder: (_, __) => const PlayerScreen())
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(path: '/store', builder: (_, __) => const StoreScreen())
          ],
        ),
      ],
    ),
    GoRoute(path: '/login', builder: (_, __) => const LoginScreen()),
    GoRoute(path: '/legal', builder: (_, __) => const LegalListScreen()),
    GoRoute(
      path: '/legal/terms/:code',
      builder: (_, s) => TermDetailScreen(code: s.pathParameters['code']!),
    ),
    GoRoute(
      path: '/legal/ai-notice',
      builder: (_, __) => const AiNoticeScreen(),
    ),
    GoRoute(path: '/legal/blocks', builder: (_, __) => const BlocksScreen()),
    GoRoute(
      path: '/legal/withdrawal',
      builder: (_, __) => const WithdrawalScreen(),
    ),
  ],
  redirect: (context, state) {
    // Auth gate is enforced lazily per API call (401 -> refresh -> login);
    // deep-linking stays open so public screens work logged out.
    return null;
  },
);

final _scaffoldMessengerKey = GlobalKey<ScaffoldMessengerState>();

class AvibeApp extends ConsumerStatefulWidget {
  const AvibeApp({super.key});

  @override
  ConsumerState<AvibeApp> createState() => _AvibeAppState();
}

class _AvibeAppState extends ConsumerState<AvibeApp> {
  StreamSubscription<void>? _expirySub;
  DateTime? _lastExpiredToast;

  @override
  void initState() {
    super.initState();
    _expirySub = AuthSessionEvents.stream.listen((_) {
      // Concurrent requests fail rotation together; notify once per burst.
      final now = DateTime.now();
      if (_lastExpiredToast != null &&
          now.difference(_lastExpiredToast!) <
              const Duration(seconds: 3)) {
        return;
      }
      _lastExpiredToast = now;
      _scaffoldMessengerKey.currentState?.showSnackBar(
        const SnackBar(content: Text('세션이 만료되었습니다. 다시 로그인하세요.')),
      );
      _router.go('/login');
    });
  }

  @override
  void dispose() {
    _expirySub?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Warm the auth state so 401s refresh eagerly after restart.
    ref.watch(authMeProvider);
    ref.listen(authMeProvider, (prev, next) {
      final me = next.valueOrNull;
      if (me != null && me.needsAgeConfirm) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (context.mounted) maybeShowAgeGate(context, ref, true);
        });
      }
    });
    return MaterialApp.router(
      title: 'AVIBE',
      theme: AppTheme.dark,
      routerConfig: _router,
      scaffoldMessengerKey: _scaffoldMessengerKey,
    );
  }
}

class _Shell extends StatelessWidget {
  const _Shell({required this.nav});

  final StatefulNavigationShell nav;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: nav,
      bottomNavigationBar: NavigationBar(
        selectedIndex: nav.currentIndex,
        onDestinationSelected: nav.goBranch,
        destinations: const [
          NavigationDestination(
              icon: Icon(Icons.home), label: '홈'),
          NavigationDestination(
              icon: Icon(Icons.search), label: '검색'),
          NavigationDestination(
              icon: Icon(Icons.queue_music), label: '플리'),
          NavigationDestination(
              icon: Icon(Icons.play_circle), label: '재생'),
          NavigationDestination(
              icon: Icon(Icons.storefront), label: '매장'),
        ],
      ),
    );
  }
}
