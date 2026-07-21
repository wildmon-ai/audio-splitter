<script setup lang="ts">
defineProps<{ targetStep: number; numThreads: number }>()

const emit = defineEmits<{
  'update:targetStep': [value: number]
  'update:numThreads': [value: number]
}>()
</script>

<template>
  <div class="flex flex-col gap-5">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-xs font-semibold uppercase tracking-wide text-gray-500">Recording Interval</p>
        <p class="mt-0.5 text-sm text-gray-700">Fixed at <span class="font-semibold">1 minute</span> per clip</p>
      </div>
      <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold text-gray-500">Fixed</span>
    </div>

    <div class="border-t border-gray-100" />

    <div class="flex items-center justify-between gap-6">
      <div class="flex-1">
        <p class="text-xs font-semibold uppercase tracking-wide text-gray-500">Extract Every N Minutes</p>
        <p class="mt-0.5 text-xs text-gray-400">Skip N recorded minutes between each extracted clip</p>
      </div>
      <input
        :value="targetStep"
        @input="emit('update:targetStep', Number(($event.target as HTMLInputElement).value))"
        type="number"
        min="1"
        class="w-24 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-center text-sm font-semibold text-gray-800 focus:border-blue-500 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors"
      />
    </div>

    <div class="border-t border-gray-100" />

    <div class="flex items-center justify-between gap-6">
      <div class="flex-1">
        <p class="text-xs font-semibold uppercase tracking-wide text-gray-500">Parallel Processes</p>
        <p class="mt-0.5 text-xs text-gray-400">Concurrent ffmpeg processes · 0 = auto (all CPU cores)</p>
      </div>
      <input
        :value="numThreads"
        @input="emit('update:numThreads', Number(($event.target as HTMLInputElement).value))"
        type="number"
        min="0"
        class="w-24 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-center text-sm font-semibold text-gray-800 focus:border-blue-500 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors"
      />
    </div>
  </div>
</template>
