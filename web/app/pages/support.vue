<!-- Alias -> /help (help center). -->
<script setup lang="ts">
const route = useRoute();
const localePath = useLocalePath();
await navigateTo(localePath({ path: '/help', query: route.query }), { redirectCode: 301, replace: true });
</script>

<template>
  <div />
</template>
