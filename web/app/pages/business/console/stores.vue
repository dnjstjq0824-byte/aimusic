<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('business.console.stEyebrow')"
      :title="t('business.console.stTitle')"
      :description="t('business.console.stDesc')"
    >
      <template #actions>
        <UiButton variant="brand" size="sm" @click="registerOpen = true">
          {{ t('business.console.stRegister') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('business.console.stLoadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="stores.length === 0">
      <UiEmptyState
        :title="t('business.console.stEmpty')"
        :description="t('business.console.stEmptyDesc')"
      >
        <template #action>
          <UiButton variant="brand" size="sm" @click="registerOpen = true">
            {{ t('business.console.stRegister') }}
          </UiButton>
        </template>
      </UiEmptyState>
    </div>

    <UiCard v-for="store in stores" :key="store.id" padding="md">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-base font-bold text-white">{{ store.storeName }}</h2>
            <UiBadge :variant="storeStatusVariant(store.status)" size="xs">
              {{ storeStatus(store.status) }}
            </UiBadge>
          </div>
          <p class="mt-1 font-mono text-caption text-text-muted">
            {{ t('business.console.stMeta', { bizNo: formatBizNo(store.businessNumber), n: store.licenseCount }) }}
          </p>
        </div>
        <div class="flex shrink-0 gap-2">
          <UiButton variant="ghost" size="sm" @click="toggle(store.id)">
            {{ expanded === store.id ? t('business.console.stCollapse') : t('business.console.stManage') }}
          </UiButton>
        </div>
      </div>

      <div v-if="expanded === store.id" class="mt-4 space-y-5 border-t border-surface-border pt-4">
        <UiErrorBanner :message="detailError(store.id)" :title="t('business.console.stDetailFail')" />
        <section>
          <div class="mb-2 flex items-center justify-between">
            <h3 class="text-xs font-bold uppercase tracking-wider text-text-secondary">{{ t('business.console.stDevices') }}</h3>
            <UiButton variant="secondary" size="sm" @click="openDeviceModal(store)">{{ t('business.console.stAddDevice') }}</UiButton>
          </div>
          <ul v-if="devicesOf(store.id).length > 0" class="divide-y divide-surface-border rounded-lg border border-surface-border">
            <li v-for="d in devicesOf(store.id)" :key="d.id" class="flex flex-col gap-2 px-4 py-3 text-xs sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-bold text-white">{{ d.deviceName }}</span>
                  <UiBadge :variant="d.status === 'ACTIVE' ? 'success' : d.status === 'REVOKED' ? 'danger' : 'warning'" size="xs">
                    {{ deviceStatus(d.status) }}
                  </UiBadge>
                  <span class="text-text-muted">{{ t('business.console.stVolume', { n: d.volume }) }}</span>
                </div>
                <p v-if="d.pairingCode" class="mt-1 font-mono text-caption text-brand-300">
                  {{ t('business.console.stPairing', { code: d.pairingCode, left: pairingLeft(d) }) }}
                </p>
                <p v-else-if="d.lastSeenAt" class="mt-1 text-caption text-text-muted">
                  {{ t('business.console.stLastSeen', { at: fmtDateTime(d.lastSeenAt) }) }}
                </p>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <input
                  type="range" min="0" max="100" :value="d.volume" class="h-1 w-24 accent-brand-500"
                  :aria-label="t('business.console.stVolumeAria', { name: d.deviceName })"
                  @change="setVolume(store, d, ($event.target as HTMLInputElement).valueAsNumber)"
                />
                <UiButton variant="ghost" size="sm" @click="openLogs(store, d)">{{ t('business.console.stLogs') }}</UiButton>
                <UiButton v-if="d.status !== 'REVOKED'" variant="ghost" size="sm" @click="revoke(store, d)">{{ t('business.console.stRevoke') }}</UiButton>
              </div>
            </li>
          </ul>
          <p v-else class="rounded-lg border border-dashed border-surface-border px-4 py-3 text-xs text-text-muted">
            {{ t('business.console.stNoDevices') }}
          </p>
        </section>

        <section>
          <h3 class="mb-2 text-xs font-bold uppercase tracking-wider text-text-secondary">{{ t('business.console.stSchedule') }}</h3>
          <div class="overflow-x-auto rounded-lg border border-surface-border">
            <table class="w-full min-w-[560px] text-xs">
              <thead>
                <tr class="bg-surface-2/60 text-left text-text-secondary">
                  <th class="px-3 py-2 font-semibold">{{ t('business.console.stColDay') }}</th>
                  <th class="px-3 py-2 font-semibold">{{ t('business.console.stColHours') }}</th>
                  <th class="px-3 py-2 font-semibold">{{ t('business.console.stColPlaylist') }}</th>
                  <th class="px-3 py-2 font-semibold">{{ t('business.console.stColStatus') }}</th>
                  <th class="px-3 py-2 text-right font-semibold">{{ t('business.console.stColEdit') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-surface-border">
                <tr v-for="w in [1, 2, 3, 4, 5, 6, 0]" :key="w" class="hover:bg-surface-2/40">
                  <td class="px-3 py-2 font-bold text-text-primary">{{ weekdayName(w) }}</td>
                  <td class="px-3 py-2 font-mono text-text-primary">{{ scheduleText(store.id, w) }}</td>
                  <td class="px-3 py-2 text-text-primary">{{ schedulePlaylistName(store.id, w) }}</td>
                  <td class="px-3 py-2">
                    <UiBadge :variant="scheduleClosed(store.id, w) ? 'danger' : 'success'" size="xs">
                      {{ scheduleClosed(store.id, w) ? t('business.console.stClosed') : t('business.console.stOpen') }}
                    </UiBadge>
                  </td>
                  <td class="px-3 py-2 text-right">
                    <UiButton variant="ghost" size="sm" @click="openScheduleModal(store, w)">{{ t('business.console.stSet') }}</UiButton>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="mt-1 text-caption text-text-muted">{{ t('business.console.stScheduleHint') }}</p>
        </section>
      </div>
    </UiCard>

    <UiModal v-model="registerOpen" :title="t('business.console.regTitle')" :description="t('business.console.regDesc')" @close="registerOpen = false">
      <form class="space-y-4" @submit.prevent="register">
        <UiInput v-model="form.bizNo" :label="t('business.console.regBizNo')" :placeholder="t('business.console.regBizNoPh')" required />
        <UiInput v-model="form.storeName" :label="t('business.console.regName')" :placeholder="t('business.console.regNamePh')" required />
        <UiInput v-model.number="form.licenses" :label="t('business.console.regLicenses')" type="number" :min="1" :max="100" required />
        <UiErrorBanner :message="formError" :title="t('business.console.regFail')" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="registerOpen = false">{{ t('common.actions.cancel') }}</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :disabled="formBusy">{{ t('business.console.regSubmit') }}</UiButton>
        </div>
      </form>
    </UiModal>

    <UiModal v-model="deviceOpen" :title="t('business.console.devTitle')" :description="t('business.console.devDesc')" @close="deviceOpen = false">
      <form class="space-y-4" @submit.prevent="createDevice">
        <UiInput v-model="deviceName" :label="t('business.console.devName')" :placeholder="t('business.console.devNamePh')" required />
        <div v-if="issuedCode" class="rounded-lg bg-surface-2 px-4 py-3 text-center">
          <p class="text-caption text-text-secondary">{{ t('business.console.devCode') }}</p>
          <p class="font-mono text-2xl font-bold tracking-[0.3em] text-brand-300">{{ issuedCode }}</p>
        </div>
        <UiErrorBanner :message="formError" :title="t('business.console.devFail')" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="deviceOpen = false">{{ t('common.actions.close') }}</UiButton>
          <UiButton v-if="!issuedCode" variant="brand" size="sm" type="submit" :disabled="formBusy">{{ t('business.console.devIssue') }}</UiButton>
        </div>
      </form>
    </UiModal>

    <UiModal v-model="scheduleOpen" :title="t('business.console.schTitle')" :description="scheduleWeekdayName" @close="scheduleOpen = false">
      <form class="space-y-4" @submit.prevent="saveSchedule">
        <div class="grid grid-cols-2 gap-3">
          <UiInput v-model="scheduleForm.open" :label="t('business.console.schOpen')" placeholder="09:00" required />
          <UiInput v-model="scheduleForm.close" :label="t('business.console.schClose')" placeholder="22:00" required />
        </div>
        <UiSelect v-model="scheduleForm.playlistId" :label="t('business.console.schPlaylist')">
          <option value="">{{ t('business.console.officialCuration') }}</option>
          <option v-for="p in myPlaylists" :key="p.id" :value="p.id">{{ p.title }}</option>
        </UiSelect>
        <UiCheckbox v-model="scheduleForm.closed"><span class="text-xs text-text-primary">{{ t('business.console.schClosed') }}</span></UiCheckbox>
        <UiErrorBanner :message="formError" :title="t('business.console.schFail')" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="scheduleOpen = false">{{ t('common.actions.cancel') }}</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :disabled="formBusy">{{ t('common.actions.save') }}</UiButton>
        </div>
      </form>
    </UiModal>

    <UiModal v-model="logsOpen" :title="t('business.console.logsTitle')" :description="logsDeviceName" @close="logsOpen = false">
      <div v-if="logs.length === 0" class="py-6 text-center text-xs text-text-muted">{{ t('business.console.logsEmpty') }}</div>
      <ul v-else class="max-h-80 space-y-2 overflow-y-auto text-xs">
        <li v-for="l in logs" :key="l.id" class="flex items-center justify-between gap-2 rounded-lg bg-surface-2 px-3 py-2">
          <span class="min-w-0 truncate font-mono text-text-primary">{{ l.trackId }}</span>
          <span class="shrink-0 text-text-muted">{{ t('business.console.logsSecs', { n: l.durationPlayedSecs }) }}</span>
          <UiBadge :variant="l.countedAsPlay ? 'success' : 'warning'" size="xs">
            {{ l.countedAsPlay ? t('business.console.logsCounted') : t('business.console.logsUncounted') }}
          </UiBadge>
        </li>
      </ul>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, reactive, ref } from 'vue';
import { consoleApi, playlistApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useApiError } from '@/composables/useApiError';
import type { Playlist, StoreBusiness, StoreDevice, StorePlayLog, StoreSchedule } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, te, locale } = useI18n();
const apiError = useApiError();

function storeStatus(s: string): string {
  const key = `business.console.storeStatus.${s}`;
  return te(key) ? String(t(key)) : s;
}

function deviceStatus(s: string): string {
  const key = `business.console.deviceStatus.${s}`;
  return te(key) ? String(t(key)) : s;
}

function weekdayName(w: number): string {
  const key = `business.console.weekday.${w}`;
  return te(key) ? String(t(key)) : String(w);
}

function fmtDateTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString(locale.value === 'zh-cn' ? 'zh-CN' : locale.value);
  } catch {
    return iso;
  }
}

const stores = ref<StoreBusiness[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const expanded = ref<string | null>(null);
const devices = reactive<Record<string, StoreDevice[]>>({});
const schedules = reactive<Record<string, StoreSchedule[]>>({});
const detailErrors = reactive<Record<string, string | null>>({});
const myPlaylists = ref<Playlist[]>([]);

const registerOpen = ref(false);
const deviceOpen = ref(false);
const scheduleOpen = ref(false);
const logsOpen = ref(false);
const formBusy = ref(false);
const formError = ref<string | null>(null);
const form = reactive({ bizNo: '', storeName: '', licenses: 1 });
const deviceName = ref('');
const issuedCode = ref<string | null>(null);
const targetStore = ref<StoreBusiness | null>(null);
const scheduleWeekday = ref(1);
const scheduleForm = reactive({ open: '09:00', close: '22:00', playlistId: '', closed: false });
const logs = ref<StorePlayLog[]>([]);
const logsDeviceName = ref('');

const scheduleWeekdayName = computed(() => t('business.console.schFor', { day: weekdayName(scheduleWeekday.value) }));

function storeStatusVariant(s: string): 'success' | 'danger' | 'warning' {
  return s === 'APPROVED' ? 'success' : s === 'SUSPENDED' ? 'danger' : 'warning';
}

function formatBizNo(raw: string): string {
  return raw.length === 10 ? `${raw.slice(0, 3)}-${raw.slice(3, 5)}-${raw.slice(5)}` : raw;
}

function devicesOf(storeId: string): StoreDevice[] {
  return devices[storeId] ?? [];
}

function detailError(storeId: string): string | null {
  return detailErrors[storeId] ?? null;
}

function scheduleOf(storeId: string, weekday: number): StoreSchedule | undefined {
  return (schedules[storeId] ?? []).find((s) => s.weekday === weekday);
}

function scheduleText(storeId: string, weekday: number): string {
  const s = scheduleOf(storeId, weekday);
  if (!s) return t('business.console.stUnset');
  if (s.isClosed) return t('business.console.stClosed');
  return `${s.openTime} – ${s.closeTime}`;
}

function schedulePlaylistName(storeId: string, weekday: number): string {
  const s = scheduleOf(storeId, weekday);
  if (!s || !s.playlistId) return t('business.console.officialCuration');
  return myPlaylists.value.find((p) => p.id === s.playlistId)?.title ?? s.playlistId;
}

function scheduleClosed(storeId: string, weekday: number): boolean {
  return scheduleOf(storeId, weekday)?.isClosed ?? false;
}

function pairingLeft(d: StoreDevice): string {
  if (!d.pairingExpiresAt) return '';
  const ms = new Date(d.pairingExpiresAt).getTime() - Date.now();
  if (Number.isNaN(ms)) return '';
  if (ms <= 0) return t('business.console.stExpired');
  return t('business.console.stMinutes', { n: Math.ceil(ms / 60000) });
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    stores.value = (await consoleApi.listStores()) ?? [];
    myPlaylists.value = await playlistApi.listMine().catch(() => []);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function toggle(storeId: string): Promise<void> {
  if (expanded.value === storeId) {
    expanded.value = null;
    return;
  }
  expanded.value = storeId;
  detailErrors[storeId] = null;
  try {
    const [devs, weekRows] = await Promise.all([
      consoleApi.listStoreDevices(storeId),
      consoleApi.listStoreSchedules(storeId),
    ]);
    devices[storeId] = devs;
    schedules[storeId] = weekRows;
  } catch (err: unknown) {
    detailErrors[storeId] = errMessageOf(err);
  }
}

async function register(): Promise<void> {
  formBusy.value = true;
  formError.value = null;
  try {
    const created = await consoleApi.registerStore({
      businessNumber: form.bizNo,
      storeName: form.storeName,
      licenseCount: form.licenses,
    });
    stores.value.unshift(created);
    registerOpen.value = false;
    form.bizNo = '';
    form.storeName = '';
    form.licenses = 1;
  } catch (err: unknown) {
    formError.value = apiError(err);
  } finally {
    formBusy.value = false;
  }
}

function openDeviceModal(store: StoreBusiness): void {
  targetStore.value = store;
  deviceName.value = '';
  issuedCode.value = null;
  formError.value = null;
  deviceOpen.value = true;
}

async function createDevice(): Promise<void> {
  if (!targetStore.value) return;
  formBusy.value = true;
  formError.value = null;
  try {
    const created = await consoleApi.createStoreDevice(targetStore.value.id, deviceName.value);
    issuedCode.value = created.pairingCode ?? null;
    devices[targetStore.value.id] = await consoleApi.listStoreDevices(targetStore.value.id);
  } catch (err: unknown) {
    formError.value = apiError(err);
  } finally {
    formBusy.value = false;
  }
}

async function setVolume(store: StoreBusiness, d: StoreDevice, volume: number): Promise<void> {
  try {
    const updated = await consoleApi.updateStoreDevice(store.id, d.id, { deviceName: d.deviceName, volume });
    const list = devices[store.id] ?? [];
    const idx = list.findIndex((x) => x.id === d.id);
    if (idx >= 0) list[idx] = updated;
  } catch (err: unknown) {
    detailErrors[store.id] = errMessageOf(err);
  }
}

async function revoke(store: StoreBusiness, d: StoreDevice): Promise<void> {
  if (!window.confirm(t('business.console.stRevokeConfirm', { name: d.deviceName }))) return;
  try {
    await consoleApi.revokeStoreDevice(store.id, d.id);
    devices[store.id] = await consoleApi.listStoreDevices(store.id);
  } catch (err: unknown) {
    detailErrors[store.id] = errMessageOf(err);
  }
}

function openScheduleModal(store: StoreBusiness, weekday: number): void {
  targetStore.value = store;
  scheduleWeekday.value = weekday;
  const s = scheduleOf(store.id, weekday);
  scheduleForm.open = s?.openTime ?? '09:00';
  scheduleForm.close = s?.closeTime ?? '22:00';
  scheduleForm.playlistId = s?.playlistId ?? '';
  scheduleForm.closed = s?.isClosed ?? false;
  formError.value = null;
  scheduleOpen.value = true;
}

async function saveSchedule(): Promise<void> {
  if (!targetStore.value) return;
  formBusy.value = true;
  formError.value = null;
  try {
    await consoleApi.upsertStoreSchedule(targetStore.value.id, {
      weekday: scheduleWeekday.value,
      openTime: scheduleForm.open,
      closeTime: scheduleForm.close,
      playlistId: scheduleForm.playlistId || null,
      isClosed: scheduleForm.closed,
    });
    schedules[targetStore.value.id] = await consoleApi.listStoreSchedules(targetStore.value.id);
    scheduleOpen.value = false;
  } catch (err: unknown) {
    formError.value = apiError(err);
  } finally {
    formBusy.value = false;
  }
}

async function openLogs(store: StoreBusiness, d: StoreDevice): Promise<void> {
  logsDeviceName.value = d.deviceName;
  logs.value = [];
  logsOpen.value = true;
  try {
    logs.value = (await consoleApi.storeDeviceLogs(store.id, d.id, 50)) ?? [];
  } catch (err: unknown) {
    detailErrors[store.id] = errMessageOf(err);
    logsOpen.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
