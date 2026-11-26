<!-- /search -> /explore (search lives on the explore page; query is kept). -->
<script setup lang="ts">
const route = useRoute();
const localePath = useLocalePath();
await navigateTo(localePath({ path: '/explore', query: route.query }), { redirectCode: 301, replace: true });
</script>

<template>
  <div />
</template>
