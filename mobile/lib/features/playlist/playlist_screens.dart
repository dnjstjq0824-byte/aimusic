// Playlists: list, create/edit, detail with add + reorder + remove.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/app_locale.dart';
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
    final s = context.l10n;
    final lists = ref.watch(playlistsProvider);
    return Scaffold(
      appBar: AppBar(title: Text(s.playlistTitle)),
      floatingActionButton: FloatingActionButton(
        onPressed: () => _showEditDialog(context, ref, null),
        child: const Icon(Icons.add),
      ),
      body: Column(
        children: [
          Expanded(
            child: lists.when(
              data: (items) => items.isEmpty
                  ? Center(child: Text(s.playlistEmpty))
                  : ListView.builder(
                      itemCount: items.length,
                      itemBuilder: (context, i) {
                        final p = items[i];
                        return ListTile(
                          leading: const Icon(Icons.queue_music),
                          title: Text(p.title),
                          subtitle: Text(p.isPublic
                              ? s.playlistPublic
                              : s.playlistPrivate),
                          trailing: PopupMenuButton<String>(
                            onSelected: (v) {
                              if (v == 'edit') {
                                _showEditDialog(context, ref, p);
                              } else if (v == 'delete') {
                                _delete(context, ref, p);
                              }
                            },
                            itemBuilder: (_) => [
                              PopupMenuItem(
                                  value: 'edit',
                                  child: Text(s.playlistRename)),
                              PopupMenuItem(
                                  value: 'delete',
                                  child: Text(s.playlistDelete)),
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
                    ? friendlyMessage(e, s)
                    : s.playlistListFailed),
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
    final s = context.l10n;
    final titleCtrl = TextEditingController(text: existing?.title ?? '');
    bool isPublic = existing?.isPublic ?? false;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: Text(
              existing == null ? s.playlistNew : s.playlistEdit),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleCtrl,
                decoration:
                    InputDecoration(labelText: s.playlistNameLabel),
                autofocus: true,
              ),
              SwitchListTile(
                title: Text(s.playlistPublic),
                value: isPublic,
                onChanged: (v) => setState(() => isPublic = v),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(s.playlistCancel),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(s.playlistSave),
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
                  ? friendlyMessage(e, s)
                  : s.playlistSaveFailed)),
        );
      }
    }
  }

  Future<void> _delete(
    BuildContext context,
    WidgetRef ref,
    Playlist p,
  ) async {
    final s = context.l10n;
    try {
      await ref.read(apiProvider).deletePlaylist(p.id);
      ref.invalidate(playlistsProvider);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text(e is ApiException
                  ? friendlyMessage(e, s)
                  : s.playlistDeleteFailed)),
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
    final s = context.l10n;
    final detail = ref.watch(playlistDetailProvider(widget.playlistId));
    return Scaffold(
      appBar: AppBar(
        title: Text(s.playlistTitle),
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
                  : Text(s.playlistSaveOrder),
            ),
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: s.playlistAddTrack,
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
                      s.playlistDetailEmpty(playlist.title),
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
                          label: Text(s.playlistPlay),
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
                    ? friendlyMessage(e, s)
                    : s.playlistDetailFailed),
              ),
            ),
          ),
          const MiniPlayer(),
        ],
      ),
    );
  }

  Future<void> _saveOrder(BuildContext context, WidgetRef ref) async {
    final s = context.l10n;
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
                  ? friendlyMessage(e, s)
                  : s.playlistOrderFailed)),
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
    final s = context.l10n;
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
                  ? friendlyMessage(e, s)
                  : s.playlistDeleteFailed)),
        );
      }
    }
  }

  Future<void> _showAddDialog(BuildContext context, WidgetRef ref) async {
    final catalog = await ref.read(homeTracksProvider.future);
    if (!context.mounted) return;
    final s = context.l10n;
    Track? picked;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(s.playlistAddTrack),
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
            child: Text(s.playlistClose),
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
                  ? friendlyMessage(e, s)
                  : s.playlistAddFailed)),
        );
      }
    }
  }
}
