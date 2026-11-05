import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  LogEntry,
  LogLevel,
  ProcessingConfig,
  ProcessingDonePayload,
  ProgressPayload,
} from '../types'
import { errorMessage } from '../utils/format'

const MAX_LOGS = 500

export const useProcessingStore = defineStore('processing', () => {
  const isRunning = ref(false)
  const isCancelling = ref(false)
  const progress = ref<ProgressPayload>({ phase: 'clips', current: 0, total: 0 })
  const logs = ref<LogEntry[]>([])
  const summary = ref<ProcessingDonePayload | null>(null)
  const elapsedSeconds = ref<number | null>(null)
  let startedAt = 0
  let nextLogId = 0
  let unsubscribers: Array<() => void> = []

  const progressPercent = computed(() => {
    if (progress.value.total === 0) return 0
    return Math.round((progress.value.current / progress.value.total) * 100)
  })

  function addLog(level: LogLevel, message: string) {
    logs.value.push({ id: nextLogId++, level, message })
    if (logs.value.length > MAX_LOGS) logs.value.splice(0, logs.value.length - MAX_LOGS)
  }

  function stopListening() {
    for (const unsubscribe of unsubscribers) unsubscribe()
    unsubscribers = []
  }

  function finish(result: ProcessingDonePayload) {
    summary.value = result
    elapsedSeconds.value = Math.round((Date.now() - startedAt) / 1000)
    isRunning.value = false
    isCancelling.value = false
    stopListening()
  }

  async function start(config: ProcessingConfig) {
    if (isRunning.value) return
    isRunning.value = true
    isCancelling.value = false
    progress.value = { phase: 'scan', current: 0, total: 0 }
    logs.value = []
    summary.value = null
    elapsedSeconds.value = null
    startedAt = Date.now()

    stopListening()
    unsubscribers = [
      window.electronAPI.onLog((payload) => addLog(payload.level, payload.message)),
      window.electronAPI.onProgress((payload) => {
        progress.value = payload
      }),
      window.electronAPI.onDone(finish),
    ]

    try {
      await window.electronAPI.startProcessing(config)
    } catch (err) {
      // Without this the page would wait forever for a 'done' that never comes.
      const message = errorMessage(err)
      addLog('error', message)
      finish({ total_clips: 0, skipped: 0, errors: 0, error: message })
    }
  }

  async function cancel() {
    if (!isRunning.value || isCancelling.value) return
    isCancelling.value = true
    try {
      await window.electronAPI.cancelProcessing()
    } catch (err) {
      isCancelling.value = false
      addLog('error', errorMessage(err))
    }
  }

  return {
    isRunning,
    isCancelling,
    progress,
    progressPercent,
    logs,
    summary,
    elapsedSeconds,
    start,
    cancel,
  }
})
