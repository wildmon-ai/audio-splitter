import type { LogPayload, ProcessingConfig, ProcessingDonePayload, ProgressPayload } from './index'

type Unsubscribe = () => void

interface ElectronAPI {
  startProcessing: (config: ProcessingConfig) => Promise<void>
  cancelProcessing: () => Promise<void>
  selectFolder: () => Promise<string | null>
  onLog: (cb: (payload: LogPayload) => void) => Unsubscribe
  onProgress: (cb: (payload: ProgressPayload) => void) => Unsubscribe
  onDone: (cb: (payload: ProcessingDonePayload) => void) => Unsubscribe
}

declare global {
  interface Window {
    electronAPI: ElectronAPI
  }
}
