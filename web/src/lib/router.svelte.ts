import { z } from 'zod'

const RouteSchema = z.enum(['chat', 'code', 'config', 'containers', 'packages'])
export type Route = z.infer<typeof RouteSchema>

export const OVERLAYS = [
  'timeline',
  'files',
  'mailbox',
  'container',
  'todos',
] as const
export type Overlay = (typeof OVERLAYS)[number]

export interface ChatLocation {
  sessionId: string | null
  overlay: Overlay | null
  // Selected change id inside the timeline overlay (master→detail drill-in).
  changeId: string | null
  // Files overlay drill-in: encoded file path and optional change id.
  filePath: string | null
  fileChangeId: string | null
}

export interface ContainersLocation {
  containerId: string | null
}

const hasWindow = typeof window !== 'undefined'

function parseHash(): {
  route: Route
  chat: ChatLocation
  containers: ContainersLocation
} {
  const empty = {
    sessionId: null as string | null,
    overlay: null as Overlay | null,
    changeId: null as string | null,
    filePath: null as string | null,
    fileChangeId: null as string | null,
  }
  if (!hasWindow)
    return {
      route: 'chat',
      chat: { ...empty },
      containers: { containerId: null },
    }
  const h = window.location.hash.replace(/^#\/?/, '')
  const parts = h.split('/').filter(Boolean)
  if (parts[0] === 's' && parts[1]) {
    const sessionId = parts[1]
    const overlayRaw = parts[2] ?? ''
    const overlay = (OVERLAYS as readonly string[]).includes(overlayRaw)
      ? (overlayRaw as Overlay)
      : null
    const changeId = overlay === 'timeline' ? (parts[3] ?? null) : null
    let filePath: string | null = null
    let fileChangeId: string | null = null
    if (overlay === 'files') {
      // /s/{id}/files/{encPath} or /s/{id}/files/{encPath}/{changeId}
      // path may contain encoded '/' so we must decode the raw segment.
      const rawPath = parts[3] ?? null
      if (rawPath) {
        filePath = decodeURIComponent(rawPath)
        fileChangeId = parts[4] ?? null
      }
    }
    return {
      route: 'chat',
      chat: { sessionId, overlay, changeId, filePath, fileChangeId },
      containers: { containerId: null },
    }
  }
  if (parts[0] === 'containers') {
    return {
      route: 'containers',
      chat: { ...empty },
      containers: { containerId: parts[1] ?? null },
    }
  }
  if (parts.length === 0 || parts[0] === 'home')
    return {
      route: 'chat',
      chat: { ...empty },
      containers: { containerId: null },
    }
  const parsed = RouteSchema.safeParse(parts[0])
  return {
    route: parsed.success ? parsed.data : 'chat',
    chat: { ...empty },
    containers: { containerId: null },
  }
}

let current = $state<Route>(parseHash().route)
let chatLoc = $state<ChatLocation>(parseHash().chat)
let containersLoc = $state<ContainersLocation>(parseHash().containers)

const hashListeners = new Set<() => void>()

if (hasWindow) {
  window.addEventListener('hashchange', () => {
    const parsed = parseHash()
    current = parsed.route
    chatLoc = parsed.chat
    containersLoc = parsed.containers
    for (const fn of hashListeners) fn()
  })
}

export function navigate(route: Route) {
  if (!hasWindow) {
    current = route
    return
  }
  window.location.hash = `/${route}`
}

export function openSessionUrl(
  sessionId: string,
  overlay: Overlay | null = null,
  changeId: string | null = null,
  filePath: string | null = null,
  fileChangeId: string | null = null,
) {
  if (!hasWindow) return
  if (!overlay) {
    window.location.hash = `/s/${sessionId}`
    return
  }
  if (overlay === 'timeline' && changeId) {
    window.location.hash = `/s/${sessionId}/timeline/${changeId}`
    return
  }
  if (overlay === 'files' && filePath) {
    const seg = encodeURIComponent(filePath)
    window.location.hash = fileChangeId
      ? `/s/${sessionId}/files/${seg}/${fileChangeId}`
      : `/s/${sessionId}/files/${seg}`
    return
  }
  window.location.hash = `/s/${sessionId}/${overlay}`
}

export function closeSessionUrl() {
  if (!hasWindow) return
  window.location.hash = '/chat'
}

export function openContainerUrl(containerId: string) {
  if (!hasWindow) return
  window.location.hash = `/containers/${containerId}`
}

export function closeContainerUrl() {
  if (!hasWindow) return
  window.location.hash = '/containers'
}

export function onHashChange(fn: () => void): () => void {
  hashListeners.add(fn)
  return () => hashListeners.delete(fn)
}

export const router = {
  get current(): Route {
    return current
  },
  get chat(): ChatLocation {
    return chatLoc
  },
  get containers(): ContainersLocation {
    return containersLoc
  },
}

const TAB_ROUTES: Route[] = ['chat', 'code', 'containers', 'packages', 'config']
let tabStack = $state<Route[]>([])

function tabFor(route: Route): Route {
  return TAB_ROUTES.includes(route) ? route : 'chat'
}

function ensureStack() {
  if (tabStack.length === 0) tabStack = [tabFor(current)]
}

export function switchTab(route: Route) {
  const target = tabFor(route)
  ensureStack()
  if (target === tabStack[tabStack.length - 1]) {
    navigate(target)
    return
  }
  navigate(target)
  tabStack = [...tabStack, target]
}

export function goBackTab(): boolean {
  ensureStack()
  if (tabStack.length <= 1) return false
  tabStack = tabStack.slice(0, -1)
  const prev = tabStack[tabStack.length - 1]
  navigate(prev)
  return true
}

export const tabRouter = {
  get canGoBack() {
    ensureStack()
    return tabStack.length > 1
  },
  switchTab,
  goBackTab,
}
