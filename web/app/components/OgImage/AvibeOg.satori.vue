<!-- AVIBE OG image template (WI-S2, Satori): cover + title + creator +
Verified seal. CJK/Thai titles render `plain` (textless cover fallback). -->
<template>
  <div
    :style="{
      width: '1200px',
      height: '630px',
      display: 'flex',
      flexDirection: 'row',
      alignItems: 'center',
      backgroundColor: '#17161c',
      color: '#ffffff',
      fontFamily: 'Inter, sans-serif',
      padding: '64px',
    }"
  >
    <div
      :style="{
        width: '360px',
        height: '360px',
        borderRadius: '24px',
        backgroundColor: '#14b8a6',
        backgroundImage: `linear-gradient(135deg, #14b8a6 0%, #0e7490 55%, #312e81 100%)`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        marginRight: '56px',
        flexShrink: 0,
      }"
    >
      <div :style="{ fontSize: '120px', fontWeight: 900, color: '#04121a' }">A</div>
    </div>
    <div :style="{ display: 'flex', flexDirection: 'column', flex: 1, minWidth: 0 }">
      <div v-if="!plain" :style="{ display: 'flex', flexDirection: 'column' }">
        <div
          :style="{
            fontSize: '64px',
            fontWeight: 800,
            lineHeight: 1.15,
            maxHeight: '230px',
            overflow: 'hidden',
            display: '-webkit-box',
          }"
        >
          {{ title }}
        </div>
        <div :style="{ fontSize: '32px', color: '#8A909C', marginTop: '16px' }">{{ creator }}</div>
      </div>
      <div v-else :style="{ display: 'flex', flexDirection: 'column' }">
        <div :style="{ fontSize: '72px', fontWeight: 900, letterSpacing: '-2px' }">AVIBE</div>
        <div :style="{ fontSize: '30px', color: '#8A909C', marginTop: '12px' }">Beyond Your Music</div>
      </div>
      <div v-if="!plain || verified" :style="{ display: 'flex', alignItems: 'center', marginTop: '32px' }">
        <div v-if="!plain" :style="{ fontSize: '28px', fontWeight: 900, letterSpacing: '4px', color: '#14b8a6' }">AVIBE</div>
        <div
          v-if="verified"
          :style="{
            marginLeft: '20px',
            fontSize: '22px',
            fontWeight: 700,
            color: '#04121a',
            backgroundColor: '#22C55E',
            borderRadius: '999px',
            padding: '6px 20px',
          }"
        >
          VERIFIED
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string;
    creator?: string;
    verified?: boolean;
    plain?: boolean;
  }>(),
  { title: 'AVIBE', creator: '', verified: false, plain: false },
);
</script>
