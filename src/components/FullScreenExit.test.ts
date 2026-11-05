import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createApp, nextTick, type App } from 'vue'
import FullScreenExit from './FullScreenExit.vue'

// The callback the component registered, so tests can play full-screen changes.
let notify: ((isFullScreen: boolean) => void) | null = null
const unsubscribe = vi.fn(() => {
  notify = null
})
const mockElectronAPI = {
  exitFullScreen: vi.fn(() => Promise.resolve()),
  onFullScreenChange: vi.fn((callback: (isFullScreen: boolean) => void) => {
    notify = callback
    return unsubscribe
  }),
}
vi.stubGlobal('electronAPI', mockElectronAPI)

let app: App | null = null
let root: HTMLElement

const button = () => root.querySelector('button')
// Like a real key press: from the page, bubbling up through document to window.
const pressEscape = () =>
  document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))

async function setFullScreen(value: boolean) {
  notify?.(value)
  await nextTick()
}

function unmount() {
  app?.unmount()
  app = null
}

beforeEach(() => {
  vi.clearAllMocks()
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(FullScreenExit)
  app.mount(root)
})

afterEach(() => {
  unmount()
  root.remove()
})

describe('FullScreenExit', () => {
  it('shows nothing, and ignores Escape, outside full screen', () => {
    expect(button()).toBeNull()
    pressEscape()
    expect(mockElectronAPI.exitFullScreen).not.toHaveBeenCalled()
  })

  it('shows a button that leaves full screen', async () => {
    await setFullScreen(true)
    expect(button()?.textContent).toContain('Exit full screen')
    button()!.click()
    expect(mockElectronAPI.exitFullScreen).toHaveBeenCalledTimes(1)
  })

  it('leaves full screen with Escape', async () => {
    await setFullScreen(true)
    pressEscape()
    expect(mockElectronAPI.exitFullScreen).toHaveBeenCalledTimes(1)
  })

  it('leaves Escape alone when the page already used it', async () => {
    await setFullScreen(true)
    const handled = (event: Event) => event.preventDefault()
    document.addEventListener('keydown', handled)
    pressEscape()
    document.removeEventListener('keydown', handled)
    expect(mockElectronAPI.exitFullScreen).not.toHaveBeenCalled()
  })

  it('hides the button after leaving full screen', async () => {
    await setFullScreen(true)
    await setFullScreen(false)
    expect(button()).toBeNull()
  })

  it('stops listening when it is removed', async () => {
    await setFullScreen(true)
    unmount()
    expect(unsubscribe).toHaveBeenCalledTimes(1)
    pressEscape()
    expect(mockElectronAPI.exitFullScreen).not.toHaveBeenCalled()
  })
})
