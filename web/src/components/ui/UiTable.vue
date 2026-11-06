<template>
  <div class="w-full overflow-hidden rounded-xl border border-surface-border bg-surface-1">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead class="border-b border-surface-border bg-surface-0/60 font-semibold text-neutral-400 select-none">
          <tr>
            <slot name="header" />
          </tr>
        </thead>
        <tbody class="divide-y divide-surface-border text-neutral-200">
          <slot />
        </tbody>
      </table>
    </div>
    <div v-if="$slots.empty" class="p-8 text-center text-xs text-neutral-500">
      <slot name="empty" />
    </div>
  </div>
</template>

<script setup lang="ts">
// Reusable styled data table wrapper
</script>
