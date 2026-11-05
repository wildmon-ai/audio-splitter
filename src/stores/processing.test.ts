import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProcessingStore } from './processing'
import type { LogPayload, ProcessingConfig, ProcessingDonePayload, ProgressPayload } from '../types'

// The listeners the store registered, so tests can play backend events.
const listeners = {
  log: [] as Array<(payload: LogPayload) => void>,
  progress: [] as Array<(payload: ProgressPayload) => void>,
  done: [] as Array<(payload: ProcessingDonePayload) => void>,
}

function subscribe<T>(list: Array<(payload: T) => void>) {
  return (callback: (payload: T) => void) => {
    list.push(callback)
    return () => {
      list.splice(list.indexOf(callback), 1)
    }
  }
}

const mockElectronAPI = {
  startProcessing: vi.fn((_config: ProcessingConfig) => Promise.resolve()),
  cancelProcessing: vi.fn(() => Promise.resolve()),
  selectFolder: vi.fn(() => Promise.resolve(null)),
  onLog: vi.fn(subscribe(listeners.log)),
  onProgress: vi.fn(subscribe(listeners.progress)),
  onDone: vi.fn(subscribe(listeners.done)),
}

vi.stubGlobal('electronAPI', mockElectronAPI)

const config: ProcessingConfig = {
  source_dir: '/src',
  output_dir: '/out',
  record_step: 1,
  target_step: 10,
  num_threads: 2,
}

const listenerCount = () => listeners.log.length + listeners.progress.length + listeners.done.length

describe('useProcessingStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    listeners.log.length = 0
    listeners.progress.length = 0
    listeners.done.length = 0
  })

  it('progressPercent is 0 when total is 0', () => {
    const store = useProcessingStore()
    expect(store.progressPercent).toBe(0)
  })

  it('progressPercent computes correctly', () => {
    const store = useProcessingStore()
    store.progress = { current: 12, total: 47 }
    expect(store.progressPercent).toBe(26)
  })

  it('progressPercent is 100 when complete', () => {
    const store = useProcessingStore()
    store.progress = { current: 47, total: 47 }
    expect(store.progressPercent).toBe(100)
  })

  it('start sends the config and follows the backend events', async () => {
    const store = useProcessingStore()
    await store.start(config)
    expect(mockElectronAPI.startProcessing).toHaveBeenCalledWith(config)
    expect(store.isRunning).toBe(true)

    listeners.log[0]({ level: 'warn', message: 'careful' })
    listeners.progress[0]({ phase: 'clips', current: 1, total: 4 })
    expect(store.logs).toEqual([{ id: 0, level: 'warn', message: 'careful' }])
    expect(store.progressPercent).toBe(25)
  })

  it('done ends the job and removes its listeners', async () => {
    const store = useProcessingStore()
    await store.start(config)
    listeners.done[0]({ total_clips: 3, skipped: 1, errors: 0 })

    expect(store.isRunning).toBe(false)
    expect(store.summary).toEqual({ total_clips: 3, skipped: 1, errors: 0 })
    expect(store.elapsedSeconds).not.toBeNull()
    expect(listenerCount()).toBe(0)
  })

  it('a failed start ends the job with the error instead of hanging', async () => {
    mockElectronAPI.startProcessing.mockRejectedValueOnce(
      new Error("Error invoking remote method 'start-processing': Error: The processing engine is missing."),
    )
    const store = useProcessingStore()
    await store.start(config)

    expect(store.isRunning).toBe(false)
    expect(store.summary?.error).toBe('The processing engine is missing.')
    expect(store.logs[store.logs.length - 1]?.message).toBe('The processing engine is missing.')
    expect(listenerCount()).toBe(0)
  })

  it('does not pile up listeners across jobs', async () => {
    const store = useProcessingStore()
    await store.start(config)
    listeners.done[0]({ total_clips: 0, skipped: 0, errors: 0 })
    await store.start(config)
    expect(listenerCount()).toBe(3)
  })

  it('ignores start while a job is running', async () => {
    const store = useProcessingStore()
    await store.start(config)
    await store.start(config)
    expect(mockElectronAPI.startProcessing).toHaveBeenCalledTimes(1)
  })

  it('cancel asks once and shows that it is cancelling', async () => {
    const store = useProcessingStore()
    await store.start(config)
    await store.cancel()
    await store.cancel()
    expect(mockElectronAPI.cancelProcessing).toHaveBeenCalledTimes(1)
    expect(store.isCancelling).toBe(true)

    listeners.done[0]({ total_clips: 1, skipped: 0, errors: 0, cancelled: true })
    expect(store.isCancelling).toBe(false)
    expect(store.summary?.cancelled).toBe(true)
  })

  it('cancel does nothing when no job runs', async () => {
    const store = useProcessingStore()
    await store.cancel()
    expect(mockElectronAPI.cancelProcessing).not.toHaveBeenCalled()
  })

  it('keeps the newest 500 log lines, each with its own id', async () => {
    const store = useProcessingStore()
    await store.start(config)
    for (let i = 0; i < 510; i++) listeners.log[0]({ level: 'info', message: `line ${i}` })

    expect(store.logs).toHaveLength(500)
    expect(store.logs[0]).toEqual({ id: 10, level: 'info', message: 'line 10' })
    expect(new Set(store.logs.map((entry) => entry.id)).size).toBe(500)
  })
})
