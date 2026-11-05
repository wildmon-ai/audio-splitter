<script setup lang="ts">
import { ref, watch } from 'vue'
import type { LogEntry, LogLevel } from '../types'

const props = defineProps<{ logs: LogEntry[] }>()

const container = ref<HTMLElement | null>(null)

// Scroll to the newest line at most once per frame. Watch the newest id, not
// the length: the length stops changing once the log is capped at 500 lines.
let scrollPending = false
watch(
  () => props.logs[props.logs.length - 1]?.id,
  () => {
    if (scrollPending) return
    scrollPending = true
    requestAnimationFrame(() => {
      scrollPending = false
      if (container.value) container.value.scrollTop = container.value.scrollHeight
    })
  },
)

const styles: Record<LogLevel, { color: string; prefix: string }> = {
  info: { color: 'text-gray-600', prefix: '·' },
  warn: { color: 'text-amber-600', prefix: '!' },
  error: { color: 'text-red-600', prefix: '✕' },
}
const styleOf = (level: LogLevel) => styles[level] ?? styles.info
</script>

<template>
  <div
    ref="container"
    class="h-52 overflow-y-auto bg-gray-50 px-6 py-4 font-mono text-xs leading-relaxed"
  >
    <div v-if="logs.length === 0" class="text-gray-400 italic">Waiting for output…</div>
    <div
      v-for="entry in logs"
      :key="entry.id"
      class="flex gap-2"
      :class="styleOf(entry.level).color"
    >
      <span class="select-none w-3 text-center flex-shrink-0">{{ styleOf(entry.level).prefix }}</span>
      <span>{{ entry.message }}</span>
    </div>
  </div>
</template>
