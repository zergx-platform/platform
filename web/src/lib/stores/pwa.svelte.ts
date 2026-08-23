import { getContext, setContext } from 'svelte'

const KEY = Symbol('pwa')

export type PwaStore = {
  readonly canInstall: boolean
  readonly installed: boolean
  readonly isStandalone: boolean
  readonly isAndroid: boolean
  readonly displayMode: string
  promptInstall(): void
}

type BeforeInstallPromptEvent = Event & {
  prompt(): Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

type InstallState = {
  canInstall: boolean
  installed: boolean
  promptEvent: BeforeInstallPromptEvent | null
}

export function detectDisplayMode(): string {
  if (typeof window === 'undefined') return 'browser'
  if (window.matchMedia('(display-mode: standalone)').matches)
    return 'standalone'
  if (window.matchMedia('(display-mode: fullscreen)').matches)
    return 'fullscreen'
  if (window.matchMedia('(display-mode: minimal-ui)').matches)
    return 'minimal-ui'
  if (window.matchMedia('(display-mode: window-controls-overlay)').matches)
    return 'window-controls-overlay'
  return 'browser'
}

export function detectAndroid(): boolean {
  if (typeof navigator === 'undefined') return false
  return /android/i.test(navigator.userAgent)
}

let installState = $state<InstallState>({
  canInstall: false,
  installed: false,
  promptEvent: null,
})

let displayMode = $state<string>(detectDisplayMode())

export function setBeforeInstallPrompt(ev: Event | null) {
  const typed = (ev ?? null) as BeforeInstallPromptEvent | null
  installState.promptEvent = typed
  installState.canInstall = typed !== null
}

export function setInstalled() {
  installState.installed = true
  displayMode = detectDisplayMode()
}

function buildStore(): PwaStore {
  return {
    get canInstall() {
      return installState.canInstall
    },
    get installed() {
      return installState.installed || displayMode === 'standalone'
    },
    get isStandalone() {
      return displayMode === 'standalone' || displayMode === 'fullscreen'
    },
    get isAndroid() {
      return detectAndroid()
    },
    get displayMode() {
      return displayMode
    },
    promptInstall() {
      const ev = installState.promptEvent
      if (!ev) return
      installState.promptEvent = null
      installState.canInstall = false
      void ev.prompt()
      void ev.userChoice.then(choice => {
        if (choice.outcome === 'accepted') {
          installState.installed = true
          displayMode = detectDisplayMode()
        }
      })
    },
  }
}

let _store: PwaStore | null = null

export function createPwaStore(): PwaStore {
  _store = buildStore()
  setContext(KEY, _store)
  return _store
}

export function getPwaStore(): PwaStore {
  const ctx = getContext<PwaStore>(KEY)
  if (ctx) return ctx
  if (_store) return _store
  _store = buildStore()
  return _store
}

export function onBeforeInstallPrompt(ev: Event) {
  ev.preventDefault()
  setBeforeInstallPrompt(ev)
}

export function onAppInstalled() {
  setInstalled()
}
