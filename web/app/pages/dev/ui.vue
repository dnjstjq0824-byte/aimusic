<template>
  <div class="mx-auto max-w-6xl px-4 py-10 sm:px-6">
    <UiPageHeader
      title="Design System v3 · Flow"
      subtitle="Warm graphite canvas, cyan signal accents, quiet surfaces. The single source of truth — every production component below renders live."
    />

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Typography</h2>
      <UiCard class="mt-4 space-y-3 p-6">
        <p class="type-display">Display — Beyond Your Music 123</p>
        <p class="text-h1 font-semibold text-white">H1 — Chart-topping AI tracks</p>
        <p class="text-h2 font-semibold text-white">H2 — Fresh drops this week</p>
        <p class="text-h3 font-semibold text-white">H3 — Playlist details</p>
        <p class="text-body text-text-primary">Body — The quick brown fox jumps over the lazy dog. 빠른 갈색 여우가 게으른 개를 뛰어넘습니다.</p>
        <p class="text-body-sm text-text-secondary">Body small — Secondary explanations and metadata lines.</p>
        <p class="text-caption text-text-muted">Caption — Timestamps, counts, and legal footnotes.</p>
      </UiCard>
    </section>

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Buttons</h2>
      <UiCard class="mt-4 flex flex-wrap items-center gap-3 p-6">
        <UiButton variant="brand" size="md">Play now</UiButton>
        <UiButton variant="secondary" size="md">Add to queue</UiButton>
        <UiButton variant="ghost" size="md">Dismiss</UiButton>
        <UiButton variant="danger" size="md">Remove</UiButton>
        <UiButton variant="brand" size="sm">Small</UiButton>
        <UiButton variant="brand" size="md" loading>Loading</UiButton>
        <UiButton variant="brand" size="md" disabled>Disabled</UiButton>
      </UiCard>
    </section>

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Cards &amp; badges</h2>
      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <UiCard class="space-y-3 p-6">
          <p class="font-display text-lg text-white">Midnight Drive</p>
          <p class="text-sm text-text-secondary">Neon Coast · 3:24</p>
          <div class="flex flex-wrap items-center gap-2">
            <UiBadge variant="brand" size="xs">NEW</UiBadge>
            <UiBadge variant="neutral" size="xs">AI</UiBadge>
            <UiBadge variant="success" size="xs">READY</UiBadge>
            <UiBadge variant="warning" size="xs">PENDING</UiBadge>
            <UiBadge variant="danger" size="xs">FAILED</UiBadge>
            <VerifiedSeal
              :size="16"
              label="Verified"
              full-label="Verified certificate"
              :show-label="false"
              cert-number="AVB-DEMO0001"
            />
          </div>
        </UiCard>
        <UiCard hoverable class="p-6">
          <p class="font-display text-lg text-white">Hoverable card</p>
          <p class="mt-1 text-sm text-text-secondary">Border lifts on hover; click opens the detail.</p>
        </UiCard>
      </div>
    </section>

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Inputs</h2>
      <UiCard class="mt-4 grid gap-4 p-6 md:grid-cols-2">
        <UiInput v-model="text" label="Track title" placeholder="Name your track" />
        <UiInput v-model="text" label="With error" error="Title is required" placeholder="Name your track" />
        <UiTextarea v-model="area" label="Description" placeholder="Tell listeners about this drop" />
        <UiSelect v-model="choice" label="License" :options="licenseOptions" />
      </UiCard>
    </section>

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Prompt box &amp; hero glow</h2>
      <div class="hero-glow mt-4 rounded-2xl p-6 sm:p-10">
        <p class="type-display">Describe the track.</p>
        <PromptBox v-if="i18nReady" v-model="prompt" class="mt-6" />
      </div>
    </section>

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Studio KPI cards</h2>
      <div class="mt-4 grid gap-4 sm:grid-cols-3">
        <KpiCard title="Streams (30d)" value="1,284,092" sub="+12.4% vs prior" tone="brand" />
        <KpiCard title="Revenue (30d)" value="₩8,412,000" sub="Settles on the 15th" tone="success" />
        <KpiCard title="Pending reviews" value="3" sub="Oldest waiting 2 days" />
      </div>
    </section>

    <section class="mt-10">
      <h2 class="text-h3 font-semibold text-white">Console shell preview</h2>
      <ConsoleShell
        id="showcase-shell"
        class="mt-4"
        home-to="/dev/ui"
        home-label="AVIBE"
        console-name="Showcase"
        :menu="shellMenu"
        menu-label="Menu"
        open-label="Open navigation"
        close-label="Close navigation"
        listener-home-label="Back to AVIBE"
        logout-label="Log out"
      >
        <UiCard class="p-6">
          <p class="text-body text-text-primary">Console content renders inside the shared shell: sidebar, top bar, and footer stay identical across studio, business, and admin.</p>
        </UiCard>
      </ConsoleShell>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import ConsoleShell from '@/components/console/ConsoleShell.vue';
import KpiCard from '@/components/studio/KpiCard.vue';
import PromptBox from '@/components/studio/PromptBox.vue';
import { ensureConsoleMessages, type ConsoleI18n } from '@/utils/consoleMessages';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';

const { t } = useI18n();

const text = ref('');
const area = ref('');
const choice = ref('non-commercial');
const prompt = ref('Lo-fi hip hop, warm piano, 84 bpm');

const licenseOptions = [
  { value: 'non-commercial', label: 'Non-commercial' },
  { value: 'commercial', label: 'Commercial' },
];

const shellMenu = [{ to: '/dev/ui', name: 'dev-ui', label: 'Showcase' }];

// /dev/ui is not a console path, so console namespaces don't autoload —
// fetch them for the PromptBox/KpiCard preview (same loader consoles use).
// PromptBox mounts after the merge: vue-i18n does not re-render keys that
// were missing at first render (see plugins/console-i18n.client.ts).
const i18nReady = ref(false);
onMounted(() => {
  const { $i18n } = useNuxtApp();
  void ensureConsoleMessages($i18n as unknown as ConsoleI18n, $i18n.locale.value)
    .catch(() => {})
    .finally(() => {
      i18nReady.value = true;
    });
});

definePageMeta({ layout: 'app' });
</script>
