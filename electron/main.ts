import { app, BrowserWindow, ipcMain, dialog } from 'electron'
import { spawn, ChildProcess } from 'child_process'
import * as path from 'path'
import * as readline from 'readline'

let mainWindow: BrowserWindow | null = null
let backendProcess: ChildProcess | null = null

const isDev = !app.isPackaged

function getBackendCommand(): { cmd: string; args: string[] } {
  const ext = process.platform === 'win32' ? '.exe' : ''
  if (isDev) {
    return {
      cmd: path.join(__dirname, '..', 'dist-backend', `audio-splitter-backend${ext}`),
      args: [],
    }
  }
  return {
    cmd: path.join(process.resourcesPath, `audio-splitter-backend${ext}`),
    args: [],
  }
}

function spawnBackend(): ChildProcess {
  const { cmd, args } = getBackendCommand()
  const proc = spawn(cmd, args, { stdio: ['pipe', 'pipe', 'inherit'], windowsHide: true })

  const rl = readline.createInterface({ input: proc.stdout! })
  rl.on('line', (line) => {
    if (!mainWindow || !line.trim()) return
    try {
      const payload = JSON.parse(line)
      if (payload.type === 'log') {
        mainWindow.webContents.send('log', { level: payload.level, message: payload.message })
      } else if (payload.type === 'progress') {
        mainWindow.webContents.send('progress', { current: payload.current, total: payload.total })
      } else if (payload.type === 'done') {
        mainWindow.webContents.send('processing-done', {
          total_clips: payload.total_clips,
          skipped: payload.skipped,
          errors: payload.errors,
        })
      }
    } catch {
      // ignore malformed lines
    }
  })

  proc.on('exit', (code) => {
    if (code !== 0 && code !== null && mainWindow) {
      mainWindow.webContents.send('log', {
        level: 'error',
        message: `Backend process exited with code ${code}`,
      })
    }
  })

  return proc
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 800,
    height: 700,
    resizable: true,
    icon: path.join(__dirname, '..', 'build', 'icon.png'),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  })

  if (isDev) {
    mainWindow.loadURL('http://localhost:1420')
    // mainWindow.webContents.openDevTools()
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'))
  }

  mainWindow.on('closed', () => { mainWindow = null })
}

app.whenReady().then(() => {
  backendProcess = spawnBackend()
  createWindow()
})

ipcMain.handle('start-processing', (_event, config: unknown) => {
  if (!backendProcess?.stdin?.writable) return
  backendProcess.stdin.write(JSON.stringify({ cmd: 'start', config }) + '\n')
})

ipcMain.handle('cancel-processing', () => {
  if (!backendProcess?.stdin?.writable) return
  backendProcess.stdin.write(JSON.stringify({ cmd: 'cancel' }) + '\n')
})

ipcMain.handle('select-folder', async () => {
  if (!mainWindow) return null
  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory'],
  })
  return result.canceled ? null : result.filePaths[0]
})

app.on('window-all-closed', () => {
  backendProcess?.kill()
  if (process.platform !== 'darwin') app.quit()
})

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow()
})
