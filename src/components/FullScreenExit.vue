<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

// Full screen hides the title bar and the menu bar, so people need a visible
// way out: this button, or Escape.
const isFullScreen = ref(false)
let unsubscribe: (() => void) | undefined

function exitFullScreen() {
  void window.electronAPI.exitFullScreen()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && isFullScreen.value && !event.defaultPrevented) exitFullScreen()
}

onMounted(() => {
  unsubscribe = window.electronAPI.onFullScreenChange((value) => {
    isFullScreen.value = value
  })
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  unsubscribe?.()
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <button
    v-if="isFullScreen"
    type="button"
    class="fixed right-4 top-4 z-50 inline-flex items-center gap-2 rounded-lg bg-white px-3 py-2 text-sm font-semibold text-gray-800 shadow-lg ring-1 ring-gray-300 hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
    @click="exitFullScreen"
  >
    <svg class="h-4 w-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path stroke-linecap="round" stroke-linejoin="round" d="M9 4v5H4M15 4v5h5M9 20v-5H4M15 20v-5h5" />
    </svg>
    Exit full screen
    <kbd class="rounded border border-gray-300 bg-gray-50 px-1.5 font-mono text-xs font-medium text-gray-600">Esc</kbd>
  </button>
</template>
