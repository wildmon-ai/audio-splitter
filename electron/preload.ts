import { contextBridge, ipcRenderer } from 'electron'

contextBridge.exposeInMainWorld('electronAPI', {
  startProcessing: (config: unknown) =>
    ipcRenderer.invoke('start-processing', config),

  cancelProcessing: () =>
    ipcRenderer.invoke('cancel-processing'),

  selectFolder: () =>
    ipcRenderer.invoke('select-folder'),

  onLog: (cb: (payload: { level: string; message: string }) => void) =>
    ipcRenderer.on('log', (_event, payload) => cb(payload)),

  onProgress: (cb: (payload: { current: number; total: number }) => void) =>
    ipcRenderer.on('progress', (_event, payload) => cb(payload)),

  onDone: (cb: (payload: { total_clips: number; skipped: number; errors: number }) => void) =>
    ipcRenderer.on('processing-done', (_event, payload) => cb(payload)),

  removeAllListeners: (channel: string) =>
    ipcRenderer.removeAllListeners(channel),
})
