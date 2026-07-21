<script setup lang="ts">
import type { ProgressPayload } from '../types'

const props = defineProps<{
  progress: ProgressPayload
  currentMessage?: string
  isRunning: boolean
  elapsedSeconds?: number | null
}>()

const elapsedLabel = () => {
  if (!props.elapsedSeconds) return null
  const m = Math.floor(props.elapsedSeconds / 60)
  const s = props.elapsedSeconds % 60
  return m > 0 ? `${m}m ${s}s` : `${s}s`
}

const percent = () => {
  if (props.progress.total === 0) return 0
  return Math.round((props.progress.current / props.progress.total) * 100)
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center justify-between text-xs">
      <span v-if="progress.total > 0" class="flex items-center gap-2">
        <span class="font-semibold text-gray-800">{{ progress.current }}</span>
        <span class="text-gray-300">/</span>
        <span class="font-semibold text-gray-800">{{ progress.total }}</span>
        <span class="text-gray-400">clips</span>
      </span>
      <span v-else-if="isRunning" class="text-blue-500 font-medium">Starting…</span>
      <span v-else class="text-gray-400">Idle</span>
      <span class="flex items-center gap-2">
        <span v-if="elapsedLabel()" class="text-gray-400">{{ elapsedLabel() }}</span>
        <span v-if="progress.total > 0" class="font-semibold text-gray-700">{{ percent() }}%</span>
      </span>
    </div>

    <div class="h-2 w-full overflow-hidden rounded-full bg-gray-100">
      <div
        class="h-full rounded-full bg-blue-500 transition-all duration-300"
        :style="{ width: `${percent()}%` }"
      />
    </div>

    <p v-if="currentMessage" class="truncate text-xs text-gray-500 font-mono">
      {{ currentMessage }}
    </p>
  </div>
</template>
