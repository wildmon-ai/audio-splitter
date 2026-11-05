import { app, BrowserWindow, ipcMain, dialog, Menu, shell, type MenuItemConstructorOptions } from 'electron'
import { spawn, type ChildProcess } from 'child_process'
import * as fs from 'fs'
import * as path from 'path'
import * as readline from 'readline'

let mainWindow: BrowserWindow | null = null
let helpWindow: BrowserWindow | null = null
let backend: ChildProcess | null = null
let jobRunning = false

const isDev = !app.isPackaged
const isMac = process.platform === 'darwin'
const DEV_SERVER_URL = 'http://localhost:1420'
const REPO_URL = 'https://github.com/wildmon-ai/audio-splitter'

function backendPath(): string {
  const exe = `audio-splitter-backend${process.platform === 'win32' ? '.exe' : ''}`
  return isDev ? path.join(__dirname, '..', 'dist-backend', exe) : path.join(process.resourcesPath, exe)
}

function sendToRenderer(channel: string, payload: unknown) {
  if (mainWindow && !mainWindow.isDestroyed()) mainWindow.webContents.send(channel, payload)
}

// The page waits for 'processing-done' after every start. If the backend dies
// mid-job, send one, so the page does not stay on "Processing…" forever.
function failRunningJob(message: string) {
  if (!jobRunning) return
  jobRunning = false
  sendToRenderer('log', { level: 'error', message })
  sendToRenderer('processing-done', { total_clips: 0, skipped: 0, errors: 0, error: message })
}

function forwardBackendEvent(line: string) {
  let event: Record<string, unknown>
  try {
    event = JSON.parse(line)
  } catch {
    return // not an event
  }
  switch (event.type) {
    case 'log':
      sendToRenderer('log', { level: event.level, message: event.message })
      break
    case 'progress':
      sendToRenderer('progress', { phase: event.phase, current: event.current, total: event.total })
      break
    case 'done':
      jobRunning = false
      sendToRenderer('processing-done', {
        total_clips: event.total_clips,
        skipped: event.skipped,
        errors: event.errors,
        cancelled: event.cancelled === true,
        error: event.error,
      })
      break
  }
}

function startBackend(): ChildProcess {
  const cmd = backendPath()
  if (!fs.existsSync(cmd)) {
    throw new Error(
      isDev
        ? `Processing engine not found at ${cmd}. Run "pnpm build:backend".`
        : 'The processing engine is missing. Please reinstall Audio Splitter.',
    )
  }
  const proc = spawn(cmd, [], { stdio: ['pipe', 'pipe', 'inherit'], windowsHide: true })
  // Writing to a backend that just died fails with EPIPE; 'exit' reports it.
  proc.stdin?.on('error', () => {})
  proc.on('error', (err) => {
    if (backend === proc) backend = null
    failRunningJob(`The processing engine could not run: ${err.message}`)
  })
  proc.on('exit', (code, signal) => {
    if (backend === proc) backend = null
    failRunningJob(`The processing engine stopped unexpectedly (${signal ?? `exit code ${code}`}).`)
  })
  readline.createInterface({ input: proc.stdout! }).on('line', forwardBackendEvent)
  return proc
}

function sendToBackend(command: object) {
  // Started on demand, so it comes back after a crash, and after macOS closed
  // the last window (which stops it) and the app is reopened from the dock.
  backend ??= startBackend()
  backend.stdin!.write(JSON.stringify(command) + '\n')
}

function cancelJob() {
  if (backend && jobRunning) sendToBackend({ cmd: 'cancel' })
}

// Closing stdin makes the backend cancel its job, stop ffmpeg and exit.
function stopBackend() {
  const proc = backend
  if (!proc) return
  backend = null
  jobRunning = false
  proc.stdin?.end()
  const timer = setTimeout(() => proc.kill(), 5000)
  timer.unref()
  proc.once('exit', () => clearTimeout(timer))
}

function openExternally(url: string) {
  if (url.startsWith('https://')) void shell.openExternal(url)
}

// The app's pages never navigate away; links open in the system browser.
function keepNavigationInside(win: BrowserWindow) {
  win.webContents.setWindowOpenHandler(({ url }) => {
    openExternally(url)
    return { action: 'deny' }
  })
  win.webContents.on('will-navigate', (event) => {
    if (event.url === win.webContents.getURL()) return // a reload
    event.preventDefault()
    openExternally(event.url)
  })
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
      sandbox: true,
    },
  })
  keepNavigationInside(mainWindow)
  // A reload (View → Reload in development) or a crashed page loses track of
  // the job: stop it.
  mainWindow.webContents.on('did-start-loading', cancelJob)
  mainWindow.webContents.on('render-process-gone', cancelJob)

  // Full screen hides the title bar and the menu bar, so the page shows a
  // button to leave it.
  const reportFullScreen = () => sendToRenderer('full-screen-changed', mainWindow?.isFullScreen() ?? false)
  mainWindow.on('enter-full-screen', reportFullScreen)
  mainWindow.on('leave-full-screen', reportFullScreen)
  mainWindow.webContents.on('did-finish-load', reportFullScreen)

  if (isDev) {
    mainWindow.loadURL(DEV_SERVER_URL)
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'))
  }

  mainWindow.on('closed', () => { mainWindow = null })
}

function openHelpWindow() {
  if (helpWindow) {
    helpWindow.focus()
    return
  }
  helpWindow = new BrowserWindow({
    width: 640,
    height: 720,
    title: 'Audio Splitter — Help',
    parent: mainWindow ?? undefined,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  })
  helpWindow.removeMenu()
  keepNavigationInside(helpWindow)

  if (isDev) {
    helpWindow.loadURL(`${DEV_SERVER_URL}/help.html`)
  } else {
    helpWindow.loadFile(path.join(__dirname, '..', 'dist', 'help.html'))
  }
  helpWindow.on('closed', () => { helpWindow = null })
}

// Same menus as Electron's default, but with an app-specific Help menu
// instead of the default links to the Electron website.
function buildMenu(): Menu {
  const helpSubmenu: MenuItemConstructorOptions[] = [
    { label: 'How to Use Audio Splitter', accelerator: 'F1', click: () => openHelpWindow() },
    { label: 'Report an Issue…', click: () => openExternally(`${REPO_URL}/issues`) },
  ]
  if (!isMac) helpSubmenu.push({ type: 'separator' }, { role: 'about' })

  const template: MenuItemConstructorOptions[] = []
  if (isMac) template.push({ role: 'appMenu' })
  template.push({ role: 'fileMenu' })
  // macOS needs the Edit menu for Cmd+C, Cmd+V and Cmd+A to work in text
  // fields and the log. Windows and Linux handle those keys without it.
  if (isMac) template.push({ role: 'editMenu' })
  template.push(
    { label: 'View', submenu: viewSubmenu() },
    isMac ? { role: 'windowMenu' } : { label: 'Window', submenu: windowSubmenu() },
    { role: 'help', submenu: helpSubmenu },
  )
  return Menu.buildFromTemplate(template)
}

// Text size and full screen. Reload would stop a running job, so only the
// development build has it, with the developer tools.
function viewSubmenu(): MenuItemConstructorOptions[] {
  const items: MenuItemConstructorOptions[] = [
    { role: 'resetZoom' },
    // Electron's Zoom In key is Ctrl/Cmd and +, which needs Shift on most
    // keyboards: accept Ctrl/Cmd and = too, as browsers do.
    { role: 'zoomIn', accelerator: 'CommandOrControl+=' },
    { role: 'zoomIn', accelerator: 'CommandOrControl+Plus', visible: false },
    { role: 'zoomOut' },
    { type: 'separator' },
    { role: 'togglefullscreen' },
  ]
  if (isDev) items.push({ type: 'separator' }, { role: 'reload' }, { role: 'forceReload' }, { role: 'toggleDevTools' })
  return items
}

// Electron's Window menu has a Zoom item that only works on macOS, where it
// maximizes the window. Windows and Linux get Maximize instead.
function windowSubmenu(): MenuItemConstructorOptions[] {
  return [
    { role: 'minimize' },
    {
      label: 'Maximize',
      click: (_item, window) => {
        if (window?.isMaximized()) window.unmaximize()
        else window?.maximize()
      },
    },
    { type: 'separator' },
    { role: 'close' },
  ]
}

app.whenReady().then(() => {
  app.setAboutPanelOptions({
    applicationName: 'Audio Splitter',
    applicationVersion: app.getVersion(),
    website: REPO_URL,
  })
  Menu.setApplicationMenu(buildMenu())
  createWindow()
})

ipcMain.handle('start-processing', (_event, config: unknown) => {
  if (jobRunning) throw new Error('A job is already running.')
  sendToBackend({ cmd: 'start', config })
  jobRunning = true
})

ipcMain.handle('cancel-processing', () => {
  cancelJob()
})

ipcMain.handle('exit-full-screen', () => {
  mainWindow?.setFullScreen(false)
})

ipcMain.handle('select-folder', async () => {
  if (!mainWindow) return null
  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory', 'createDirectory'],
  })
  return result.canceled ? null : result.filePaths[0]
})

app.on('window-all-closed', () => {
  stopBackend()
  if (!isMac) app.quit()
})

app.on('before-quit', stopBackend)

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow()
})
