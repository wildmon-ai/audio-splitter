<script setup lang="ts">
import { computed } from 'vue'
import type { ProgressPayload } from '../types'
import { formatDuration } from '../utils/format'

const props = defineProps<{
  progress: ProgressPayload
  percent: number
  currentMessage?: string
  isRunning: boolean
  elapsedSeconds?: number | null
}>()

const isScanning = computed(() => props.progress.phase === 'scan')
const elapsedLabel = computed(() => (props.elapsedSeconds == null ? null : formatDuration(props.elapsedSeconds)))
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center justify-between text-xs">
      <span v-if="progress.total > 0" class="flex items-center gap-2">
        <span v-if="isScanning" class="text-gray-500">Reading</span>
        <span class="font-semibold text-gray-800">{{ progress.current }}</span>
        <span class="text-gray-300">/</span>
        <span class="font-semibold text-gray-800">{{ progress.total }}</span>
        <span class="text-gray-400">{{ isScanning ? 'files' : 'clips' }}</span>
      </span>
      <span v-else-if="isRunning" class="text-blue-500 font-medium">Starting…</span>
      <span v-else class="text-gray-400">Idle</span>
      <span class="flex items-center gap-2">
        <span v-if="elapsedLabel" class="text-gray-400">{{ elapsedLabel }}</span>
        <span v-if="progress.total > 0" class="font-semibold text-gray-700">{{ percent }}%</span>
      </span>
    </div>

    <div
      class="h-2 w-full overflow-hidden rounded-full bg-gray-100"
      role="progressbar"
      :aria-valuenow="percent"
      aria-valuemin="0"
      aria-valuemax="100"
    >
      <div
        class="h-full rounded-full transition-all duration-300"
        :class="isScanning ? 'bg-blue-300' : 'bg-blue-500'"
        :style="{ width: `${percent}%` }"
      />
    </div>

    <p v-if="currentMessage" class="truncate text-xs text-gray-500 font-mono">
      {{ currentMessage }}
    </p>
  </div>
</template>
