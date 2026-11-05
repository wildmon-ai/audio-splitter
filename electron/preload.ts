import { contextBridge, ipcRenderer, type IpcRendererEvent } from 'electron'

// Each on* function returns an unsubscribe function, so the page can remove
// exactly the listeners it added, and nothing else.
function subscribe(channel: string, callback: (payload: unknown) => void): () => void {
  const listener = (_event: IpcRendererEvent, payload: unknown) => callback(payload)
  ipcRenderer.on(channel, listener)
  return () => {
    ipcRenderer.removeListener(channel, listener)
  }
}

contextBridge.exposeInMainWorld('electronAPI', {
  startProcessing: (config: unknown) => ipcRenderer.invoke('start-processing', config),
  cancelProcessing: () => ipcRenderer.invoke('cancel-processing'),
  selectFolder: () => ipcRenderer.invoke('select-folder'),
  onLog: (callback: (payload: unknown) => void) => subscribe('log', callback),
  onProgress: (callback: (payload: unknown) => void) => subscribe('progress', callback),
  onDone: (callback: (payload: unknown) => void) => subscribe('processing-done', callback),
  exitFullScreen: () => ipcRenderer.invoke('exit-full-screen'),
  onFullScreenChange: (callback: (payload: unknown) => void) => subscribe('full-screen-changed', callback),
})
