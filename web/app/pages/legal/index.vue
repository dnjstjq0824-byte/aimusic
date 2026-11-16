<template>
  <div />
</template>

<script setup lang="ts">
// /legal renders the default document (TOS). R8 replaces the [code] alias.
const localePath = useLocalePath();
await navigateTo(localePath('/legal/TOS'), { redirectCode: 301 });
</script>
