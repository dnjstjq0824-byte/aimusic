<template>
  <!-- App-wide track dialogs, opened from any track menu via useTrackActions(). -->
  <AddToPlaylistModal v-if="dialogs.pickerTrack" />
  <DisputeReportModal
    v-if="dialogs.reportTrack"
    :track-id="dialogs.reportTrack.id"
    :track-title="dialogs.reportTrack.title"
    @close="dialogs.closeReport()"
    @reported="onReported"
  />
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue';
import { useTrackActionsStore } from '@/stores/trackActionsStore';
import { useToast } from '@/composables/useToast';

// Dialog code loads on first use (bundle diet).
const AddToPlaylistModal = defineAsyncComponent(() => import('@/components/playlist/AddToPlaylistModal.vue'));
const DisputeReportModal = defineAsyncComponent(() => import('@/components/legal/DisputeReportModal.vue'));

const dialogs = useTrackActionsStore();
const toast = useToast();
const { t } = useI18n();

function onReported(): void {
  dialogs.closeReport();
  toast.success(t('track.toastReported'));
}
</script>
