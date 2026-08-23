import { mount } from 'svelte'
import App from './App.svelte'
import './app.css'
import { preloadHighlighter } from '$lib/highlight'
import {
  detectDisplayMode,
  onAppInstalled,
  onBeforeInstallPrompt,
  setBeforeInstallPrompt,
} from '$lib/stores/pwa.svelte'

preloadHighlighter()

const target = document.getElementById('app')
if (!target) throw new Error('#app element not found')
const app = mount(App, { target })
export default app

if (typeof window !== 'undefined') {
  window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt)
  window.addEventListener('appinstalled', onAppInstalled)

  if (
    'serviceWorker' in navigator &&
    window.location.protocol.startsWith('http')
  ) {
    navigator.serviceWorker.register('/sw.js').catch(() => {})
  }

  try {
    const mode = detectDisplayMode()
    if (mode === 'standalone' || mode === 'fullscreen')
      setBeforeInstallPrompt(null)
  } catch {
    setBeforeInstallPrompt(null)
  }
}
