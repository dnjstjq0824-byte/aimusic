import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { useTrackActionsStore } from '@/stores/trackActionsStore';
import { useToast } from '@/composables/useToast';
import type { Track } from '@/api/types';

/**
 * Shared behaviour for every track "more" menu: queue, play-next,
 * add-to-playlist (global picker) and report (global dialog). Components
 * call these directly so no page has to re-wire the same handlers.
 */
export function useTrackActions() {
  const auth = useAuthStore();
  const player = usePlayerStore();
  const dialogs = useTrackActionsStore();
  const toast = useToast();
  const router = useRouter();
  const route = useRoute();
  const localePath = useLocalePath();
  const { t } = useI18n();

  function requireLogin(): boolean {
    if (auth.isAuthenticated) return true;
    toast.info(t('playlist.toastNeedLogin'));
    void router.push(localePath({ name: 'login', query: { redirect: route.fullPath } }));
    return false;
  }

  function enqueue(track: Track): void {
    if (player.enqueue(track)) toast.info(t('track.toastQueued'));
    else toast.info(t('player.toastAlreadyQueued'));
  }

  function playNext(track: Track): void {
    player.queueNext(track);
    toast.info(t('player.toastPlayNext'));
  }

  function addToPlaylist(track: Track): void {
    if (!requireLogin()) return;
    dialogs.openPicker(track);
  }

  function report(track: Track): void {
    if (!requireLogin()) return;
    dialogs.openReport(track);
  }

  return { enqueue, playNext, addToPlaylist, report, requireLogin };
}
