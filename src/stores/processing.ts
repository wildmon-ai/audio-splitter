import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  ProcessingConfig,
  LogEntry,
  ProgressPayload,
  ProcessingDonePayload,
} from '../types'

export const useProcessingStore = defineStore('processing', () => {
  const isRunning = ref(false)
  const progress = ref<ProgressPayload>({ current: 0, total: 0 })
  const logs = ref<LogEntry[]>([])
  const summary = ref<ProcessingDonePayload | null>(null)
  const elapsedSeconds = ref<number | null>(null)
  let startedAt = 0

  const progressPercent = computed(() => {
    if (progress.value.total === 0) return 0
    return Math.round((progress.value.current / progress.value.total) * 100)
  })

  function cleanup() {
    window.electronAPI.removeAllListeners('log')
    window.electronAPI.removeAllListeners('progress')
    window.electronAPI.removeAllListeners('processing-done')
  }

  async function start(config: ProcessingConfig) {
    isRunning.value = true
    progress.value = { current: 0, total: 0 }
    logs.value = []
    summary.value = null
    elapsedSeconds.value = null
    startedAt = Date.now()

    window.electronAPI.onLog((payload) => {
      logs.value.push({ level: payload.level as LogEntry['level'], message: payload.message })
      if (logs.value.length > 500) logs.value.splice(0, logs.value.length - 500)
    })

    window.electronAPI.onProgress((payload) => {
      progress.value = payload
    })

    window.electronAPI.onDone((payload) => {
      summary.value = payload
      elapsedSeconds.value = Math.round((Date.now() - startedAt) / 1000)
      isRunning.value = false
      cleanup()
    })

    try {
      await window.electronAPI.startProcessing(config)
    } catch (err) {
      logs.value.push({ level: 'error', message: String(err) })
      isRunning.value = false
      cleanup()
    }
  }

  async function cancel() {
    await window.electronAPI.cancelProcessing()
  }

  return { isRunning, progress, progressPercent, logs, summary, elapsedSeconds, start, cancel }
})
