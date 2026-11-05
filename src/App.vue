<script setup lang="ts">
import { computed } from 'vue'
import { useProcessingStore } from './stores/processing'
import { usePersisted } from './composables/usePersisted'
import { isString, isValidRecordStep, isValidTargetStep, isValidThreads } from './utils/validation'
import { formatDuration } from './utils/format'
import FolderPicker from './components/FolderPicker.vue'
import ParameterForm from './components/ParameterForm.vue'
import ProgressBar from './components/ProgressBar.vue'
import LogPanel from './components/LogPanel.vue'

const store = useProcessingStore()

const sourceDir = usePersisted('sourceDir', '', isString)
const outputDir = usePersisted('outputDir', '', isString)
const recordStep = usePersisted('recordStep', 1, isValidRecordStep)
const targetStep = usePersisted('targetStep', 10, isValidTargetStep)
const numThreads = usePersisted('numThreads', 0, isValidThreads)

const canStart = computed(
  () =>
    !store.isRunning &&
    sourceDir.value.trim() !== '' &&
    outputDir.value.trim() !== '' &&
    isValidTargetStep(targetStep.value) &&
    isValidThreads(numThreads.value),
)

const result = computed(() => {
  const s = store.summary
  if (!s) return null
  const elapsed = store.elapsedSeconds === null ? '' : ` · ${formatDuration(store.elapsedSeconds)}`
  const counts = `${s.total_clips} clips extracted, ${s.skipped} skipped, ${s.errors} errors${elapsed}`
  if (s.error) {
    return { title: 'Failed.', detail: s.error, tone: 'border-red-200 bg-red-50 text-red-800', icon: 'text-red-500', ok: false }
  }
  if (s.cancelled) {
    return { title: 'Cancelled.', detail: counts, tone: 'border-amber-200 bg-amber-50 text-amber-800', icon: 'text-amber-500', ok: false }
  }
  if (s.errors > 0) {
    return { title: 'Done, with errors.', detail: counts, tone: 'border-amber-200 bg-amber-50 text-amber-800', icon: 'text-amber-500', ok: false }
  }
  return { title: 'Done.', detail: counts, tone: 'border-emerald-200 bg-emerald-50 text-emerald-800', icon: 'text-emerald-500', ok: true }
})

function start() {
  if (!canStart.value) return
  store.start({
    source_dir: sourceDir.value,
    output_dir: outputDir.value,
    record_step: recordStep.value,
    target_step: targetStep.value,
    num_threads: numThreads.value,
  })
}
</script>

<template>
  <div class="min-h-screen bg-gray-100 flex flex-col">

    <!-- Header -->
    <header class="bg-[#0f1117] shadow-md">
      <div class="mx-auto max-w-3xl px-6 py-4 flex items-center gap-4">
        <img src="./assets/logo-nav.webp" alt="Logo" class="h-8 w-auto" />
        <div class="flex-1 text-center">
          <h1 class="text-base font-semibold text-white leading-tight">Audio Splitter</h1>
          <p class="text-xs text-slate-400">Wildmon · Batch audio processing</p>
        </div>
        <div class="h-8 w-auto opacity-0 pointer-events-none" aria-hidden="true">
          <img src="./assets/logo-nav.webp" alt="" class="h-8 w-auto" />
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="mx-auto w-full max-w-3xl px-6 py-8 flex flex-col gap-6">

      <!-- Config card -->
      <section class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div class="border-b border-gray-100 px-6 py-4">
          <h2 class="text-sm font-semibold text-gray-800 uppercase tracking-wide">Configuration</h2>
        </div>
        <div class="px-6 py-5 flex flex-col gap-5">
          <FolderPicker label="Source Folder" v-model="sourceDir" />
          <div class="border-t border-gray-100" />
          <FolderPicker label="Output Folder" v-model="outputDir" />
          <div class="border-t border-gray-100" />
          <ParameterForm v-model:targetStep="targetStep" v-model:numThreads="numThreads" />
        </div>
      </section>

      <!-- Actions -->
      <div class="flex items-center gap-3">
        <button
          @click="start"
          :disabled="!canStart"
          class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          <svg v-if="store.isRunning" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
          </svg>
          <svg v-else class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z"/>
          </svg>
          {{ store.isRunning ? 'Processing…' : 'Start Processing' }}
        </button>
        <button
          @click="store.cancel()"
          :disabled="!store.isRunning || store.isCancelling"
          class="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-5 py-2.5 text-sm font-semibold text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/>
          </svg>
          {{ store.isCancelling ? 'Cancelling…' : 'Cancel' }}
        </button>
      </div>

      <!-- Progress card -->
      <section class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div class="border-b border-gray-100 px-6 py-4">
          <h2 class="text-sm font-semibold text-gray-800 uppercase tracking-wide">Progress</h2>
        </div>
        <div class="px-6 py-5">
          <ProgressBar
            :progress="store.progress"
            :percent="store.progressPercent"
            :is-running="store.isRunning"
            :current-message="store.logs[store.logs.length - 1]?.message"
            :elapsed-seconds="store.elapsedSeconds"
          />
        </div>
      </section>

      <!-- Log card -->
      <section class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div class="border-b border-gray-100 px-6 py-4">
          <h2 class="text-sm font-semibold text-gray-800 uppercase tracking-wide">Log</h2>
        </div>
        <LogPanel :logs="store.logs" />
      </section>

      <!-- Summary -->
      <div
        v-if="result"
        role="status"
        class="flex items-start gap-3 rounded-xl border px-5 py-4 text-sm"
        :class="result.tone"
      >
        <svg class="mt-0.5 h-4 w-4 flex-shrink-0" :class="result.icon" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path v-if="result.ok" stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/>
          <path v-else stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <span><strong>{{ result.title }}</strong> {{ result.detail }}</span>
      </div>

    </main>
  </div>
</template>
