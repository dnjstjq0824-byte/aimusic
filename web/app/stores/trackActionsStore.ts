import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { Track } from '@/api/types';

/**
 * Targets for the app-wide track dialogs (add-to-playlist picker, report).
 * One host component in the app layout renders them, so every track menu
 * (cards, rows, player bar, fullscreen player, queue) shares one flow.
 */
export const useTrackActionsStore = defineStore('trackActions', () => {
  const pickerTrack = ref<Track | null>(null);
  const reportTrack = ref<Track | null>(null);

  function openPicker(track: Track): void {
    pickerTrack.value = track;
  }
  function closePicker(): void {
    pickerTrack.value = null;
  }
  function openReport(track: Track): void {
    reportTrack.value = track;
  }
  function closeReport(): void {
    reportTrack.value = null;
  }

  return { pickerTrack, reportTrack, openPicker, closePicker, openReport, closeReport };
});
