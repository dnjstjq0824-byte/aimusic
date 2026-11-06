// Playlists: list, create/edit, detail with add + reorder + remove.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../home/home_screen.dart';
import '../player/player_controller.dart';
import '../player/player_screen.dart';

class PlaylistListScreen extends ConsumerWidget {
  const PlaylistListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lists = ref.watch(playlistsProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('플레이리스트')),
      floatingActionButton: FloatingActionButton(
        onPressed: () => _showEditDialog(context, ref, null),
        child: const Icon(Icons.add),
      ),
      body: Column(
        children: [
          Expanded(
            child: lists.when(
              data: (items) => items.isEmpty
                  ? const Center(child: Text('플레이리스트가 없습니다. + 버튼으로 만드세요.'))
                  : ListView.builder(
                      itemCount: items.length,
                      itemBuilder: (context, i) {
                        final p = items[i];
                        return ListTile(
                          leading: const Icon(Icons.queue_music),
                          title: Text(p.title),
                          subtitle:
                              Text(p.isPublic ? '공개' : '비공개'),
                          trailing: PopupMenuButton<String>(
                            onSelected: (v) {
                              if (v == 'edit') {
                                _showEditDialog(context, ref, p);
                              } else if (v == 'delete') {
                                _delete(context, ref, p);
                              }
                            },
                            itemBuilder: (_) => const [
                              PopupMenuItem(
                                  value: 'edit', child: Text('이름 변경')),
                              PopupMenuItem(
                                  value: 'delete', child: Text('삭제')),
                            ],
                          ),
                          onTap: () => context.go('/playlists/${p.id}'),
                        );
                      },
                    ),
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(
                child: Text(e is ApiException
                    ? friendlyMessage(e)
                    : '목록을 불러오지 못했습니다.'),
              ),
            ),
          ),
          const MiniPlayer(),
        ],
      ),
    );
  }

  Future<void> _showEditDialog(
    BuildContext context,
    WidgetRef ref,
    Playlist? existing,
  ) async {
    final titleCtrl = TextEditingController(text: existing?.title ?? '');
    bool isPublic = existing?.isPublic ?? false;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: Text(existing == null ? '새 플레이리스트' : '플레이리스트 수정'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleCtrl,
                decoration: const InputDecoration(labelText: '제목'),
                autofocus: true,
              ),
              SwitchListTile(
                title: const Text('공개'),
                value: isPublic,
                onChanged: (v) => setState(() => isPublic = v),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('취소'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('저장'),
            ),
          ],
        ),
      ),
    );
    if (ok != true || !context.mounted) return;
    final title = titleCtrl.text.trim();
    if (title.isEmpty) return;
    try {
      final api = ref.read(apiProvider);
      if (existing == null) {
        await api.createPlaylist(title, isPublic: isPublic);
      } else {
        await api.updatePlaylist(existing.id, title, isPublic: isPublic);
      }
      ref.invalidate(playlistsProvider);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '저장에 실패했습니다.')),
        );
      }
    }
  }

  Future<void> _delete(
    BuildContext context,
    WidgetRef ref,
    Playlist p,
  ) async {
    try {
      await ref.read(apiProvider).deletePlaylist(p.id);
      ref.invalidate(playlistsProvider);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '삭제에 실패했습니다.')),
        );
      }
    }
  }
}

class PlaylistDetailScreen extends ConsumerStatefulWidget {
  const PlaylistDetailScreen({super.key, required this.playlistId});

  final String playlistId;

  @override
  ConsumerState<PlaylistDetailScreen> createState() =>
      _PlaylistDetailScreenState();
}

class _PlaylistDetailScreenState extends ConsumerState<PlaylistDetailScreen> {
  /// Local order buffer; committed to PUT /:id/reorder on save.
  List<Track>? _order;
  bool _saving = false;

  @override
  Widget build(BuildContext context) {
    final detail = ref.watch(playlistDetailProvider(widget.playlistId));
    return Scaffold(
      appBar: AppBar(
        title: const Text('플레이리스트'),
        actions: [
          if (_order != null)
            TextButton(
              onPressed: _saving ? null : () => _saveOrder(context, ref),
              child: _saving
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('순서 저장'),
            ),
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: '곡 추가',
            onPressed: () => _showAddDialog(context, ref),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: detail.when(
              data: (data) {
                final (playlist, tracks) = data;
                final order = _order ?? tracks;
                if (order.isEmpty) {
                  return Center(
                    child: Text(
                      '‘${playlist.title}’에 곡이 없습니다.\n+ 버튼으로 추가하세요.',
                      textAlign: TextAlign.center,
                    ),
                  );
                }
                return ReorderableListView.builder(
                  header: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Row(
                      children: [
                        Expanded(
                          child: Text(
                            playlist.title,
                            style: Theme.of(context)
                                .textTheme
                                .headlineSmall,
                          ),
                        ),
                        FilledButton.icon(
                          icon: const Icon(Icons.play_arrow),
                          label: const Text('재생'),
                          onPressed: () => playTracks(ref, tracks: order),
                        ),
                      ],
                    ),
                  ),
                  itemCount: order.length,
                  onReorderItem: (oldIndex, newIndex) {
                    setState(() {
                      _order ??= List.of(tracks);
                      final t = _order!.removeAt(oldIndex);
                      _order!.insert(newIndex, t);
                    });
                  },
                  itemBuilder: (context, i) {
                    final t = order[i];
                    return Dismissible(
                      key: ValueKey(t.id),
                      direction: DismissDirection.endToStart,
                      background: Container(
                        color: Colors.red,
                        alignment: Alignment.centerRight,
                        padding: const EdgeInsets.only(right: 16),
                        child: const Icon(Icons.delete, color: Colors.white),
                      ),
                      confirmDismiss: (_) async {
                        await _removeTrack(context, ref, t);
                        return false; // Keep the row; provider refresh updates.
                      },
                      child: TrackTile(track: t, queue: order),
                    );
                  },
                );
              },
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(
                child: Text(e is ApiException
                    ? friendlyMessage(e)
                    : '상세를 불러오지 못했습니다.'),
              ),
            ),
          ),
          const MiniPlayer(),
        ],
      ),
    );
  }

  Future<void> _saveOrder(BuildContext context, WidgetRef ref) async {
    final order = _order;
    if (order == null) return;
    setState(() => _saving = true);
    try {
      await ref.read(apiProvider).reorderPlaylist(
            widget.playlistId,
            order.map((t) => t.id).toList(),
          );
      setState(() => _order = null);
      ref.invalidate(playlistDetailProvider(widget.playlistId));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '순서 저장에 실패했습니다.')),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _removeTrack(
    BuildContext context,
    WidgetRef ref,
    Track t,
  ) async {
    try {
      await ref
          .read(apiProvider)
          .removePlaylistTrack(widget.playlistId, t.id);
      setState(() => _order?.removeWhere((e) => e.id == t.id));
      ref.invalidate(playlistDetailProvider(widget.playlistId));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '삭제에 실패했습니다.')),
        );
      }
    }
  }

  Future<void> _showAddDialog(BuildContext context, WidgetRef ref) async {
    final catalog = await ref.read(homeTracksProvider.future);
    if (!context.mounted) return;
    Track? picked;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('곡 추가'),
        content: SizedBox(
          width: double.maxFinite,
          height: 320,
          child: ListView.builder(
            itemCount: catalog.length,
            itemBuilder: (context, i) {
              final t = catalog[i];
              return ListTile(
                leading: const Icon(Icons.music_note),
                title: Text(t.title),
                subtitle: Text(t.subtitle),
                onTap: () {
                  picked = t;
                  Navigator.pop(context, true);
                },
              );
            },
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('닫기'),
          ),
        ],
      ),
    );
    if (ok != true || picked == null || !context.mounted) return;
    try {
      await ref
          .read(apiProvider)
          .addPlaylistTrack(widget.playlistId, picked!.id);
      ref.invalidate(playlistDetailProvider(widget.playlistId));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e)
                  : '추가에 실패했습니다.')),
        );
      }
    }
  }
}
