<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Studio Console"
      title="매장 관리"
      description="사업자등록번호 단위로 매장을 등록하고 플레이어 기기·영업시간·선곡을 관리합니다. 승인된 매장의 인증 플레이어에서만 재생이 집계됩니다."
    >
      <template #actions>
        <UiButton variant="brand" size="sm" @click="registerOpen = true">
          매장 등록
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="매장 목록을 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="stores.length === 0">
      <UiEmptyState
        title="등록된 매장이 없습니다"
        description="사업자등록번호와 매장명을 등록하면 관리자 승인 후 플레이어를 연결할 수 있습니다."
      >
        <template #action>
          <UiButton variant="brand" size="sm" @click="registerOpen = true">
            매장 등록하기
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
              {{ storeStatusKo(store.status) }}
            </UiBadge>
          </div>
          <p class="mt-1 font-mono text-caption text-text-muted">
            사업자 {{ formatBizNo(store.businessNumber) }} · 라이선스 {{ store.licenseCount }}대
          </p>
        </div>
        <div class="flex shrink-0 gap-2">
          <UiButton variant="ghost" size="sm" @click="toggle(store.id)">
            {{ expanded === store.id ? '접기' : '기기·편성 관리' }}
          </UiButton>
        </div>
      </div>

      <div v-if="expanded === store.id" class="mt-4 space-y-5 border-t border-surface-border pt-4">
        <UiErrorBanner :message="detailError(store.id)" title="상세를 불러오지 못했습니다" />
        <section>
          <div class="mb-2 flex items-center justify-between">
            <h3 class="text-xs font-bold uppercase tracking-wider text-text-secondary">플레이어 기기</h3>
            <UiButton variant="secondary" size="sm" @click="openDeviceModal(store)">기기 추가</UiButton>
          </div>
          <ul v-if="devicesOf(store.id).length > 0" class="divide-y divide-surface-border rounded-lg border border-surface-border">
            <li v-for="d in devicesOf(store.id)" :key="d.id" class="flex flex-col gap-2 px-4 py-3 text-xs sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-bold text-white">{{ d.deviceName }}</span>
                  <UiBadge :variant="d.status === 'ACTIVE' ? 'success' : d.status === 'REVOKED' ? 'danger' : 'warning'" size="xs">
                    {{ deviceStatusKo(d.status) }}
                  </UiBadge>
                  <span class="text-text-muted">볼륨 {{ d.volume }}</span>
                </div>
                <p v-if="d.pairingCode" class="mt-1 font-mono text-caption text-brand-300">
                  페어링 코드: {{ d.pairingCode }} ({{ pairingLeft(d) }} 남음)
                </p>
                <p v-else-if="d.lastSeenAt" class="mt-1 text-caption text-text-muted">
                  마지막 접속: {{ new Date(d.lastSeenAt).toLocaleString() }}
                </p>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <input
                  type="range" min="0" max="100" :value="d.volume" class="h-1 w-24 accent-brand-500"
                  :aria-label="`${d.deviceName} 볼륨`"
                  @change="setVolume(store, d, ($event.target as HTMLInputElement).valueAsNumber)"
                />
                <UiButton variant="ghost" size="sm" @click="openLogs(store, d)">재생 기록</UiButton>
                <UiButton v-if="d.status !== 'REVOKED'" variant="ghost" size="sm" @click="revoke(store, d)">해지</UiButton>
              </div>
            </li>
          </ul>
          <p v-else class="rounded-lg border border-dashed border-surface-border px-4 py-3 text-xs text-text-muted">
            등록된 기기가 없습니다. 기기 추가 후 플레이어 화면에서 페어링 코드를 입력하세요.
          </p>
        </section>

        <section>
          <h3 class="mb-2 text-xs font-bold uppercase tracking-wider text-text-secondary">요일별 영업시간·선곡</h3>
          <div class="overflow-x-auto rounded-lg border border-surface-border">
            <table class="w-full min-w-[560px] text-xs">
              <thead>
                <tr class="bg-surface-2/60 text-left text-text-secondary">
                  <th class="px-3 py-2 font-semibold">요일</th>
                  <th class="px-3 py-2 font-semibold">영업시간</th>
                  <th class="px-3 py-2 font-semibold">선곡 플레이리스트</th>
                  <th class="px-3 py-2 font-semibold">상태</th>
                  <th class="px-3 py-2 text-right font-semibold">편집</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-surface-border">
                <tr v-for="w in [1, 2, 3, 4, 5, 6, 0]" :key="w" class="hover:bg-surface-2/40">
                  <td class="px-3 py-2 font-bold text-text-primary">{{ weekdayKo(w) }}</td>
                  <td class="px-3 py-2 font-mono text-text-primary">{{ scheduleText(store.id, w) }}</td>
                  <td class="px-3 py-2 text-text-primary">{{ schedulePlaylistName(store.id, w) }}</td>
                  <td class="px-3 py-2">
                    <UiBadge :variant="scheduleClosed(store.id, w) ? 'danger' : 'success'" size="xs">
                      {{ scheduleClosed(store.id, w) ? '휴무' : '영업' }}
                    </UiBadge>
                  </td>
                  <td class="px-3 py-2 text-right">
                    <UiButton variant="ghost" size="sm" @click="openScheduleModal(store, w)">설정</UiButton>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="mt-1 text-caption text-text-muted">선곡을 비워 두면 공식 큐레이션으로 자동 편성됩니다.</p>
        </section>
      </div>
    </UiCard>

    <UiModal v-model="registerOpen" title="매장 등록" description="사업자등록번호는 체크섬으로 검증됩니다." @close="registerOpen = false">
      <form class="space-y-4" @submit.prevent="register">
        <UiInput v-model="form.bizNo" label="사업자등록번호" placeholder="예: 123-45-67891" required />
        <UiInput v-model="form.storeName" label="매장명" placeholder="예: 강남 1호점" required />
        <UiInput v-model.number="form.licenses" label="플레이어 라이선스 수" type="number" :min="1" :max="100" required />
        <UiErrorBanner :message="formError" title="등록 실패" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="registerOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :disabled="formBusy">등록 신청</UiButton>
        </div>
      </form>
    </UiModal>

    <UiModal v-model="deviceOpen" title="플레이어 기기 추가" description="발급된 페어링 코드를 매장 플레이어 화면에 입력하세요. (15분 유효)" @close="deviceOpen = false">
      <form class="space-y-4" @submit.prevent="createDevice">
        <UiInput v-model="deviceName" label="기기명" placeholder="예: 카운터 태블릿" required />
        <div v-if="issuedCode" class="rounded-lg bg-surface-2 px-4 py-3 text-center">
          <p class="text-caption text-text-secondary">페어링 코드</p>
          <p class="font-mono text-2xl font-bold tracking-[0.3em] text-brand-300">{{ issuedCode }}</p>
        </div>
        <UiErrorBanner :message="formError" title="기기 추가 실패" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="deviceOpen = false">닫기</UiButton>
          <UiButton v-if="!issuedCode" variant="brand" size="sm" type="submit" :disabled="formBusy">발급</UiButton>
        </div>
      </form>
    </UiModal>

    <UiModal v-model="scheduleOpen" title="영업시간·선곡 설정" :description="`${scheduleWeekdayKo} 편성`" @close="scheduleOpen = false">
      <form class="space-y-4" @submit.prevent="saveSchedule">
        <div class="grid grid-cols-2 gap-3">
          <UiInput v-model="scheduleForm.open" label="개점 (HH:MM)" placeholder="09:00" required />
          <UiInput v-model="scheduleForm.close" label="폐점 (HH:MM)" placeholder="22:00" required />
        </div>
        <UiSelect v-model="scheduleForm.playlistId" label="선곡 플레이리스트 (비우면 공식 큐레이션)">
          <option value="">공식 큐레이션 자동 편성</option>
          <option v-for="p in myPlaylists" :key="p.id" :value="p.id">{{ p.title }}</option>
        </UiSelect>
        <UiCheckbox v-model="scheduleForm.closed"><span class="text-xs text-text-primary">휴무일 (이날은 재생하지 않음)</span></UiCheckbox>
        <UiErrorBanner :message="formError" title="저장 실패" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="scheduleOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :disabled="formBusy">저장</UiButton>
        </div>
      </form>
    </UiModal>

    <UiModal v-model="logsOpen" title="기기 재생 기록" :description="logsDeviceName" @close="logsOpen = false">
      <div v-if="logs.length === 0" class="py-6 text-center text-xs text-text-muted">기록이 없습니다.</div>
      <ul v-else class="max-h-80 space-y-2 overflow-y-auto text-xs">
        <li v-for="l in logs" :key="l.id" class="flex items-center justify-between gap-2 rounded-lg bg-surface-2 px-3 py-2">
          <span class="min-w-0 truncate font-mono text-text-primary">{{ l.trackId }}</span>
          <span class="shrink-0 text-text-muted">{{ l.durationPlayedSecs }}초</span>
          <UiBadge :variant="l.countedAsPlay ? 'success' : 'warning'" size="xs">
            {{ l.countedAsPlay ? '집계' : '미집계' }}
          </UiBadge>
        </li>
      </ul>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { partnerApi, playlistApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
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

const WEEKDAYS = ['일', '월', '화', '수', '목', '금', '토'];

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

const scheduleWeekdayKo = computed(() => `${WEEKDAYS[scheduleWeekday.value] ?? ''}요일`);

function weekdayKo(w: number): string {
  return `${WEEKDAYS[w] ?? '?'}요일`;
}

function storeStatusKo(s: string): string {
  return s === 'APPROVED' ? '승인됨' : s === 'SUSPENDED' ? '정지됨' : '승인 대기';
}

function storeStatusVariant(s: string): 'success' | 'danger' | 'warning' {
  return s === 'APPROVED' ? 'success' : s === 'SUSPENDED' ? 'danger' : 'warning';
}

function deviceStatusKo(s: string): string {
  return s === 'ACTIVE' ? '연결됨' : s === 'REVOKED' ? '해지됨' : '페어링 대기';
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
  if (!s) return '미설정 (공식 큐레이션)';
  if (s.isClosed) return '휴무';
  return `${s.openTime} – ${s.closeTime}`;
}

function schedulePlaylistName(storeId: string, weekday: number): string {
  const s = scheduleOf(storeId, weekday);
  if (!s || !s.playlistId) return '공식 큐레이션';
  return myPlaylists.value.find((p) => p.id === s.playlistId)?.title ?? s.playlistId;
}

function scheduleClosed(storeId: string, weekday: number): boolean {
  return scheduleOf(storeId, weekday)?.isClosed ?? false;
}

function pairingLeft(d: StoreDevice): string {
  if (!d.pairingExpiresAt) return '';
  const ms = new Date(d.pairingExpiresAt).getTime() - Date.now();
  if (Number.isNaN(ms)) return '';
  if (ms <= 0) return '만료됨';
  return `${Math.ceil(ms / 60000)}분`;
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    stores.value = (await partnerApi.listStores()) ?? [];
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
    const [devs, schs] = await Promise.all([
      partnerApi.listStoreDevices(storeId),
      partnerApi.listStoreSchedules(storeId),
    ]);
    devices[storeId] = devs;
    schedules[storeId] = schs;
  } catch (err: unknown) {
    detailErrors[storeId] = errMessageOf(err);
  }
}

async function register(): Promise<void> {
  formBusy.value = true;
  formError.value = null;
  try {
    const created = await partnerApi.registerStore({
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
    formError.value = errMessageOf(err);
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
    const created = await partnerApi.createStoreDevice(targetStore.value.id, deviceName.value);
    issuedCode.value = created.pairingCode ?? null;
    devices[targetStore.value.id] = await partnerApi.listStoreDevices(targetStore.value.id);
  } catch (err: unknown) {
    formError.value = errMessageOf(err);
  } finally {
    formBusy.value = false;
  }
}

async function setVolume(store: StoreBusiness, d: StoreDevice, volume: number): Promise<void> {
  try {
    const updated = await partnerApi.updateStoreDevice(store.id, d.id, { deviceName: d.deviceName, volume });
    const list = devices[store.id] ?? [];
    const idx = list.findIndex((x) => x.id === d.id);
    if (idx >= 0) list[idx] = updated;
  } catch (err: unknown) {
    detailErrors[store.id] = errMessageOf(err);
  }
}

async function revoke(store: StoreBusiness, d: StoreDevice): Promise<void> {
  if (!window.confirm(`'${d.deviceName}' 기기를 해지할까요? 재생이 즉시 중단됩니다.`)) return;
  try {
    await partnerApi.revokeStoreDevice(store.id, d.id);
    devices[store.id] = await partnerApi.listStoreDevices(store.id);
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
    await partnerApi.upsertStoreSchedule(targetStore.value.id, {
      weekday: scheduleWeekday.value,
      openTime: scheduleForm.open,
      closeTime: scheduleForm.close,
      playlistId: scheduleForm.playlistId || null,
      isClosed: scheduleForm.closed,
    });
    schedules[targetStore.value.id] = await partnerApi.listStoreSchedules(targetStore.value.id);
    scheduleOpen.value = false;
  } catch (err: unknown) {
    formError.value = errMessageOf(err);
  } finally {
    formBusy.value = false;
  }
}

async function openLogs(store: StoreBusiness, d: StoreDevice): Promise<void> {
  logsDeviceName.value = d.deviceName;
  logs.value = [];
  logsOpen.value = true;
  try {
    logs.value = (await partnerApi.storeDeviceLogs(store.id, d.id, 50)) ?? [];
  } catch (err: unknown) {
    detailErrors[store.id] = errMessageOf(err);
    logsOpen.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
