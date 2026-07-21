<script setup lang="ts">
import { ref, watch } from 'vue'
import type { LogEntry } from '../types'

const props = defineProps<{ logs: LogEntry[] }>()

const container = ref<HTMLElement | null>(null)

let scrollPending = false
watch(
  () => props.logs.length,
  () => {
    if (scrollPending) return
    scrollPending = true
    requestAnimationFrame(() => {
      scrollPending = false
      if (container.value) container.value.scrollTop = container.value.scrollHeight
    })
  }
)

const colorMap: Record<string, string> = {
  info: 'text-gray-600',
  skip: 'text-amber-600',
  error: 'text-red-600',
  pending: 'text-blue-600',
}

const prefixMap: Record<string, string> = {
  info: '·',
  skip: '↷',
  error: '✕',
  pending: '…',
}
</script>

<template>
  <div
    ref="container"
    class="h-52 overflow-y-auto bg-gray-50 px-6 py-4 font-mono text-xs leading-relaxed"
  >
    <div v-if="logs.length === 0" class="text-gray-400 italic">Waiting for output…</div>
    <div
      v-for="(entry, i) in logs"
      :key="i"
      class="flex gap-2"
      :class="colorMap[entry.level] ?? 'text-gray-600'"
    >
      <span class="select-none w-3 text-center flex-shrink-0">{{ prefixMap[entry.level] ?? '·' }}</span>
      <span>{{ entry.message }}</span>
    </div>
  </div>
</template>
