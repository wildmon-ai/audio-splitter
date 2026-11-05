/** 75 → "1m 15s", 9 → "9s" */
export function formatDuration(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`
}

/**
 * The message of an error thrown by an IPC call, without the wrapper that
 * ipcRenderer.invoke adds: "Error invoking remote method 'x': Error: message".
 */
export function errorMessage(err: unknown): string {
  const message = err instanceof Error ? err.message : String(err)
  return message.replace(/^Error invoking remote method '[^']*': (?:Error: )?/, '')
}
