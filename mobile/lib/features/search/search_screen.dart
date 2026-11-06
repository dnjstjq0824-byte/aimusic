// Search: client-side filter over the catalog (no backend search API).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/error.dart';
import '../../core/providers.dart';
import '../home/home_screen.dart';
import '../player/player_screen.dart';

class SearchScreen extends ConsumerWidget {
  const SearchScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final results = ref.watch(searchResultsProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('검색')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              decoration: const InputDecoration(
                hintText: '곡 제목 또는 AI 제공자로 검색',
                prefixIcon: Icon(Icons.search),
                border: OutlineInputBorder(),
              ),
              onChanged: (v) =>
                  ref.read(searchQueryProvider.notifier).state = v,
            ),
          ),
          Expanded(
            child: results.when(
              data: (tracks) => tracks.isEmpty
                  ? const Center(child: Text('검색 결과가 없습니다.'))
                  : ListView.builder(
                      itemCount: tracks.length,
                      itemBuilder: (context, i) =>
                          TrackTile(track: tracks[i], queue: tracks),
                    ),
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(
                child: Text(e is ApiException
                    ? friendlyMessage(e)
                    : '검색에 실패했습니다.'),
              ),
            ),
          ),
          const MiniPlayer(),
        ],
      ),
    );
  }
}
