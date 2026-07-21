import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProcessingStore } from './processing'

const mockElectronAPI = {
  startProcessing: vi.fn(() => Promise.resolve()),
  cancelProcessing: vi.fn(() => Promise.resolve()),
  selectFolder: vi.fn(() => Promise.resolve(null)),
  onLog: vi.fn(),
  onProgress: vi.fn(),
  onDone: vi.fn(),
  removeAllListeners: vi.fn(),
}

vi.stubGlobal('electronAPI', mockElectronAPI)

describe('useProcessingStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
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

  it('start calls electronAPI.startProcessing with config', async () => {
    const store = useProcessingStore()
    const config = {
      source_dir: '/src',
      output_dir: '/out',
      record_step: 1,
      target_step: 10,
      num_threads: 2,
    }
    await store.start(config)
    expect(mockElectronAPI.startProcessing).toHaveBeenCalledWith(config)
  })

  it('cancel calls electronAPI.cancelProcessing', async () => {
    const store = useProcessingStore()
    await store.cancel()
    expect(mockElectronAPI.cancelProcessing).toHaveBeenCalled()
  })
})
