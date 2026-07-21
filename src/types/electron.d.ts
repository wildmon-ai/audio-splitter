import type { ProcessingConfig, ProcessingDonePayload } from './index'

interface ElectronAPI {
  startProcessing: (config: ProcessingConfig) => Promise<void>
  cancelProcessing: () => Promise<void>
  selectFolder: () => Promise<string | null>
  onLog: (cb: (payload: { level: string; message: string }) => void) => void
  onProgress: (cb: (payload: { current: number; total: number }) => void) => void
  onDone: (cb: (payload: ProcessingDonePayload) => void) => void
  removeAllListeners: (channel: string) => void
}

declare global {
  interface Window {
    electronAPI: ElectronAPI
  }
}
