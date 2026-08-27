<script lang="ts">
import { createMessages } from '$lib/hooks/useMessages.svelte'
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Circle,
  CircleCheck,
  CircleDot,
  Folder,
  GitBranch,
  GitCommit,
  Inbox,
  Layers,
  ListTodo,
  MoreVertical,
  Send,
  Settings,
  Square,
  SquareTerminal,
  Terminal,
  X,
} from '@lucide/svelte'
import { onMount } from 'svelte'
import type {
  ContainerInfo,
  ModelInfo,
  PresetInfo,
  Session,
  Todo,
} from '$lib/api'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import * as DropdownMenu from '$lib/components/ui/dropdown-menu'
import ChatSidebar from './ChatSidebar.svelte'
import ContainerWorkspace from './ContainerWorkspace.svelte'
import DiffScreen from './DiffScreen.svelte'
import FilesPage from './FilesPage.svelte'
import MailboxPage from './MailboxPage.svelte'
import MessageBubble from './MessageBubble.svelte'
import TimelinePage from './TimelinePage.svelte'

let models = $state<ModelInfo[]>([])
let presets = $state<PresetInfo[]>([])
let showModelPicker = $state(false)
let showPresetPicker = $state(false)
let showImagePicker = $state(false)
let showSettings = $state(false)
let input = $state('')
let scrollEl: HTMLDivElement | undefined
let sessionSettings = $state<{
  max_turns?: number | null
  system_prompt?: string | null
  preset?: string
  base_image?: string
}>({})

let msgHook = $state<ReturnType<typeof createMessages> | null>(null)

let runStatus = $derived(msgHook?.sending ? 'busy' : 'idle')
let initialScrollDone = false

let diffChangeId = $state<string | null>(null)
let todos = $state<Todo[]>([])
let containerRow = $state<ContainerInfo | null>(null)
let containerLoading = $state(false)

// Track an anchor message id across a "load earlier" so we can restore the
// scroll offset after history is prepended (the list grows from the top,
// which would otherwise visually jump).
let loadingMore = false

async function loadEarlier() {
  if (!msgHook || !scrollEl || loadingMore) return
  loadingMore = true
  // Anchor: the topmost message currently rendered; remember its DOM offset
  // so we can realign the viewport to the same content after the prepend.
  const anchorEl = scrollEl.querySelector('[data-msg-id]') as HTMLElement | null
  const anchorTop = anchorEl ? anchorEl.offsetTop : 0
  const prevScrollTop = scrollEl.scrollTop
  await msgHook.loadMore()
  await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(() => r(null))))
  if (anchorEl) {
    // New top content pushed anchorEl down by (newHeight - oldHeight);
    // keep the same visual position by shifting scrollTop accordingly.
    const delta = anchorEl.offsetTop - anchorTop
    scrollEl.scrollTop = prevScrollTop + delta
  }
  loadingMore = false
}

function handlePanelScroll() {
  if (!scrollEl || !msgHook || loadingMore) return
  // Near the top and there is more history → auto-load older messages.
  if (scrollEl.scrollTop < 80 && msgHook.hasMore && !msgHook.loading) {
    void loadEarlier()
  }
}

onMount(() => {
  loadModels()
  loadPresets()
})

// re-init when session changes
$effect(() => {
  const sid = store.activeSessionId
  if (!sid) {
    msgHook = null
    return
  }
  const hook = createMessages(() => store.activeSessionId ?? sid)
  // Todos refresh on demand: the memory extension publishes todos-updated
  // onto the session event stream whenever todowrite lands (no more 5s
  // polling), plus a refresh when a turn settles for good measure.
  const offTodos = hook.onSessionEvent((event, params) => {
    if (event === 'todos-updated' || event === 'turn-complete') {
      void loadTodos(sid)
    }
    if (event === 'tool-result' && typeof params.change_id === 'string') {
      // A repo write just committed mid-turn: refresh the timeline (and
      // the file tree) immediately instead of waiting for turn-complete.
      store.bumpSessionRevision()
    }
    if (event === 'status' && params.type === 'busy') {
      // Turn started = the queued prompt was consumed: mailbox state and
      // the workspace tree may have moved.
      store.bumpSessionRevision()
    }
    if (event === 'turn-complete') {
      store.bumpSessionRevision()
    }
    void params
  })
  hook.init().then(cleanup => {
    msgHook = hook
    // store cleanup for later session switch
    ;(hook as unknown as { _cleanup?: () => void })._cleanup = () => {
      cleanup()
      offTodos()
    }
  })
  return () => {
    const c = (hook as unknown as { _cleanup?: () => void })._cleanup
    offTodos()
    if (c) c()
  }
})

$effect(() => {
  const s = store.activeSession
  if (s) {
    sessionSettings = {
      max_turns: s.max_turns,
      system_prompt: s.system_prompt,
      preset: s.preset,
      base_image: s.base_image ?? undefined,
    }
  }
})

$effect(() => {
  const sid = store.activeSessionId
  if (sid) {
    diffChangeId = store.diffChangeId
  } else {
    diffChangeId = null
  }
  if (!sid) {
    todos = []
    containerRow = null
    return
  }
  void loadTodos(sid)
})

$effect(() => {
  const s = store.activeSession
  if (s && store.sessionOverlay === 'files') {
    if (
      store.codeOrg !== s.org ||
      store.codeRepo !== s.repo ||
      store.codeBranch !== s.branch
    ) {
      void store.openRepo(s.org, s.repo, s.branch)
    }
  }
})

$effect(() => {
  const sid = store.activeSessionId
  if (!sid || store.sessionOverlay !== 'container') {
    containerRow = null
    return
  }
  void loadContainer(sid)
})

const sessionWorkerId = $derived.by(() => {
  const s = store.activeSession
  if (!s) return null
  return `${s.org}:${s.repo}:${s.branch}`
})

// auto-scroll on new content
$effect(() => {
  if (msgHook && msgHook.messages.length > 0) {
    if (!initialScrollDone && scrollEl) {
      scrollEl.scrollTop = scrollEl.scrollHeight
      initialScrollDone = true
    } else if (scrollEl && msgHook.sending) {
      // follow streaming tail
      const nearBottom =
        scrollEl.scrollHeight - scrollEl.scrollTop - scrollEl.clientHeight < 120
      if (nearBottom) scrollEl.scrollTop = scrollEl.scrollHeight
    }
  }
})

const currentModelName = $derived.by(() => {
  const m = models.find(m => m.id === store.activeSession?.model)
  return m?.name || store.activeSession?.model || 'Select model'
})

const currentBaseImageLabel = $derived.by(() => {
  const img = store.activeSession?.base_image
  if (!img) return 'debian-trixie-slim'
  const m = img.match(/(?:zergx)-worker:([^/]+)$/)
  return m ? m[1] : img
})

const todoDone = $derived(
  todos.filter(t => t.status === 'completed' || t.status === 'cancelled')
    .length,
)

const totalTokens = $derived(store.activeSession?.total_tokens ?? 0)

const tokensLabel = $derived.by(() => {
  if (totalTokens <= 0) return ''
  if (totalTokens < 1000) return `${totalTokens}`
  return `${(totalTokens / 1000).toFixed(1)}k`
})

// ---- resizable side panel ----
const PANEL_WIDTH_KEY = 'zergx-panel-width'
const DEFAULT_PANEL_WIDTH = 480

function clampPanelWidth(w: number): number {
  const vw = typeof window !== 'undefined' ? window.innerWidth : 1280
  return Math.min(Math.max(Math.round(w), 360), Math.min(960, Math.floor(vw * 0.7)))
}

function loadPanelWidth(): number {
  const raw = Number(localStorage.getItem(PANEL_WIDTH_KEY))
  return clampPanelWidth(Number.isFinite(raw) && raw > 0 ? raw : DEFAULT_PANEL_WIDTH)
}

let panelWidth = $state(loadPanelWidth())
let panelResizing = $state(false)

function startPanelResize(e: PointerEvent): void {
  e.preventDefault()
  panelResizing = true
  document.body.classList.add('select-none')
  const onMove = (ev: PointerEvent): void => {
    panelWidth = clampPanelWidth(window.innerWidth - ev.clientX)
  }
  const onUp = (): void => {
    panelResizing = false
    document.body.classList.remove('select-none')
    localStorage.setItem(PANEL_WIDTH_KEY, String(panelWidth))
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

const overlayTabs = [
  { id: 'timeline', label: 'Timeline', icon: GitBranch },
  { id: 'files', label: 'Files', icon: Folder },
  { id: 'container', label: 'Container', icon: SquareTerminal },
  { id: 'todos', label: 'Todos', icon: ListTodo },
  { id: 'mailbox', label: 'Mailbox', icon: Inbox },
] as const

function toggleOverlay(id: (typeof overlayTabs)[number]['id']): void {
  if (store.sessionOverlay === id) store.closeOverlay()
  else store.openOverlay(id)
}

const overlayTitle = $derived.by(() => {
  switch (store.sessionOverlay) {
    case 'timeline':
      return 'Timeline'
    case 'files':
      return store.selectedFilePath
        ? `Files · ${store.selectedFilePath}`
        : 'Files'
    case 'mailbox':
      return 'Mailbox'
    case 'container':
      return 'Container'
    case 'todos':
      return 'Todos'
    default:
      return ''
  }
})

const statusDot = $derived.by(() => {
  switch (runStatus) {
    case 'running':
    case 'busy':
      return 'bg-yellow-500 animate-pulse'
    case 'error':
      return 'bg-red-500'
    default:
      return 'bg-green-600'
  }
})

async function loadTodos(sid: string) {
  const r = await api.sessions.todos(sid)
  if (store.activeSessionId === sid) todos = r.isOk() ? r.value : []
}

async function loadContainer(sid: string) {
  containerLoading = true
  const s = store.activeSession
  if (!s) {
    containerLoading = false
    return
  }
  // Session name is org:repo:bookmark; the sandbox pod is keyed by it.
  const sessionName = `${s.org}:${s.repo}:${s.branch}`
  const r = await api.containers.list()
  if (r.isOk()) {
    const sb = r.value.find(c => c.session === sessionName)
    containerRow = sb
      ? {
          id: sb.session,
          name: sb.pod_name,
          image: null,
          worker_url: sb.worker_url,
          container_id: sb.container_id,
          session_id: sb.session,
          org: s.org,
          repo: s.repo,
          branch: s.branch,
          status: sb.status,
          created_at: null,
          kind: 'worker',
          service_url: null,
        }
      : null
  }
  containerLoading = false
}

async function createBoundContainer() {
  if (!store.activeSessionId) return
  const s = store.activeSession
  if (!s) return
  containerLoading = true
  // Sandboxes are created lazily by the first sandbox tool call; a no-op exec
  // through the gateway warms it up on demand.
  await api.containers.exec(`${s.org}:${s.repo}:${s.branch}`, 'true')
  await loadContainer(store.activeSessionId)
  containerLoading = false
}

function jumpToFile(path: string) {
  const s = store.activeSession
  if (!s) return
  // Open the file inside the chat right panel (files overlay), never leaving
  // the chat view.
  void store.openRepo(s.org, s.repo, s.branch).then(() => {
    void store.openFileOverlay(path)
  })
}

function openChangeDiff(changeId: string) {
  store.openChange(changeId)
}

function handleBack() {
  if (store.sessionOverlay) {
    if (store.sessionOverlay === 'files' && store.selectedFilePath) {
      // step out of the file/diff within the files overlay
      store.backFileOverlay()
      return
    }
    if (store.sessionOverlay === 'timeline' && store.diffChangeId) {
      store.openOverlay('timeline')
      return
    }
    store.closeOverlay()
    return
  }
  store.closeSession()
}

async function loadModels() {
  const r = await api.models.list()
  models = r.isOk() ? r.value : []
}

async function loadPresets() {
  const r = await api.presets.list()
  presets = r.isOk() ? r.value : []
}

function applySession(updated: Session) {
  const idx = store.sessions.findIndex(s => s.id === updated.id)
  if (idx >= 0) store.sessions[idx] = updated
}

async function switchModel(modelId: string) {
  if (!store.activeSessionId) return
  const r = await api.sessions.settings(store.activeSessionId, {
    model: modelId,
  })
  if (r.isOk()) applySession(r.value)
  showModelPicker = false
}

async function switchPreset(presetId: string) {
  if (!store.activeSessionId) return
  const r = await api.sessions.settings(store.activeSessionId, {
    preset: presetId,
  })
  if (r.isOk()) applySession(r.value)
  showPresetPicker = false
}

async function switchBaseImage(image: string) {
  if (!store.activeSessionId) return
  const r = await api.sessions.settings(store.activeSessionId, {
    base_image: image,
  })
  if (r.isOk()) applySession(r.value)
  showImagePicker = false
}

async function saveSettings() {
  if (!store.activeSessionId) return
  const updates: Record<string, unknown> = {}
  if (sessionSettings.preset !== undefined)
    updates.preset = sessionSettings.preset
  if (sessionSettings.max_turns !== undefined)
    updates.max_turns = sessionSettings.max_turns
  if (sessionSettings.system_prompt !== undefined)
    updates.system_prompt = sessionSettings.system_prompt || null
  if (sessionSettings.base_image !== undefined)
    updates.base_image = sessionSettings.base_image
  const r = await api.sessions.settings(store.activeSessionId, updates)
  if (r.isOk()) applySession(r.value)
  showSettings = false
}

async function send() {
  const text = input.trim()
  if (!text || !msgHook || msgHook.sending) return
  input = ''
  await msgHook.send(text)
  if (scrollEl) scrollEl.scrollTop = scrollEl.scrollHeight
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

async function undo(messageId: string) {
  if (!msgHook) return
  await msgHook.revert(messageId)
}

async function compact() {
  const sid = store.activeSessionId
  if (!sid) return
  const r = await api.sessions.compact(sid)
  r.match(
    () => {
      // Recreate the message hook so it re-fetches history with the new
      // compaction checkpoint.
      const hook = createMessages(() => store.activeSessionId ?? sid)
      hook.init().then(cleanup => {
        msgHook = hook
        ;(hook as unknown as { _cleanup?: () => void })._cleanup = cleanup
      })
    },
    () => {},
  )
}
</script>

<div class="flex flex-col h-full">
    <div class="flex items-center gap-1.5 px-2 sm:px-3 py-2 shrink-0 border-b border-border bg-card">
        <Button variant="ghost" size="icon-sm" class="shrink-0" onclick={handleBack} title="Back">
            <ArrowLeft class="size-4" />
        </Button>
        <span class={`size-2 rounded-full shrink-0 ${statusDot}`} title={runStatus}></span>
        <span class="text-sm font-medium text-muted-foreground truncate max-w-[160px]">
            {store.activeSession ? `${store.activeSession.org}/${store.activeSession.repo}` : "Chat"}
        </span>
        {#if store.sessionOverlay}
            <ChevronRight class="size-3 text-muted-foreground/50 shrink-0" />
            <span class="text-sm font-medium text-foreground truncate min-w-0">{overlayTitle}</span>
        {/if}
        <div class="flex-1"></div>

        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <Button variant="ghost" size="icon-sm" class="size-8 shrink-0" {...props} title="Menu">
                        <MoreVertical class="size-4" />
                    </Button>
                {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end" class="w-56">
                <DropdownMenu.Item onclick={() => (showSettings = true)}>
                    <Settings class="size-4" />
                    Session settings
                </DropdownMenu.Item>
                <DropdownMenu.Item onclick={() => compact()}>
                    <Layers class="size-4" />
                    Compact history
                </DropdownMenu.Item>
                <DropdownMenu.Separator />
                <DropdownMenu.Item onclick={() => (store.openOverlay('timeline'))}>
                    <GitCommit class="size-4" />
                    Timeline
                </DropdownMenu.Item>
                <DropdownMenu.Item onclick={() => (store.openOverlay('files'))}>
                    <Folder class="size-4" />
                    Files
                </DropdownMenu.Item>
                <DropdownMenu.Item onclick={() => (store.openOverlay('mailbox'))}>
                    <Inbox class="size-4" />
                    Mailbox
                </DropdownMenu.Item>
                <DropdownMenu.Item onclick={() => (store.openOverlay('container'))}>
                    <Terminal class="size-4" />
                    Container
                </DropdownMenu.Item>
                {#if todos.length > 0}
                    <DropdownMenu.Item onclick={() => (store.openOverlay('todos'))}>
                        <ListTodo class="size-4" />
                        Todos
                        <span class="ml-auto font-mono text-[10px] text-muted-foreground">{todoDone}/{todos.length}</span>
                    </DropdownMenu.Item>
                {/if}
            </DropdownMenu.Content>
        </DropdownMenu.Root>
    </div>

    {#if showSettings}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div class="fixed inset-0 z-30" role="presentation" onclick={() => showSettings = false}></div>
        <div class="absolute top-10 right-3 z-40 w-80 rounded-lg border bg-popover shadow-lg p-4 space-y-3" role="dialog" tabindex="-1" aria-label="Session settings" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
            <div class="flex items-center justify-between">
                <span class="text-sm font-semibold">Session Settings</span>
                <Button variant="ghost" size="icon-sm" onclick={() => showSettings = false}><X class="size-3" /></Button>
            </div>

            <div>
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="ss-preset">Preset</label>
                <select id="ss-preset" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                    bind:value={sessionSettings.preset}>
                    {#each presets as p (p.id)}
                        <option value={p.id}>{p.id}</option>
                    {/each}
                </select>
            </div>

            <div>
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="ss-max-turns">Max Turns</label>
                <input id="ss-max-turns" type="number" min="1" max="200" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                    bind:value={sessionSettings.max_turns} placeholder="inherit" />
            </div>

            <div>
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="ss-sys-prompt">System Prompt (blank = inherit from preset)</label>
                <textarea id="ss-sys-prompt" class="w-full rounded-md border border-input bg-background px-2 py-1 text-[10px] font-mono min-h-[60px] resize-y"
                    bind:value={sessionSettings.system_prompt} placeholder="use preset default"></textarea>
            </div>

            <div>
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="ss-base-image">Worker Base Image</label>
                <select id="ss-base-image" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                    bind:value={sessionSettings.base_image}>
                    <option value="">default (debian:trixie-slim)</option>
                </select>
            </div>

            <div class="flex justify-end gap-2 pt-1">
                <Button size="sm" variant="outline" onclick={() => showSettings = false}>Cancel</Button>
                <Button size="sm" onclick={saveSettings}>Apply</Button>
            </div>
        </div>
    {/if}

    <div class="flex-1 min-h-0 relative flex">
        <!-- Desktop: repo/session sidebar -->
        <div class="hidden lg:flex shrink-0 w-64 border-r border-border bg-muted/20 flex-col">
            <ChatSidebar />
        </div>

        <!-- Main chat column -->
        <div class="flex flex-col h-full flex-1 min-w-0">
            <div class="flex-1 overflow-y-auto px-3 sm:px-4 py-2" bind:this={scrollEl} onscroll={handlePanelScroll}>
                {#if msgHook && msgHook.hasMore}
                    <div class="text-center py-2">
                        <Button variant="ghost" size="sm" onclick={() => loadEarlier()} disabled={msgHook.loading}>
                            {msgHook.loading ? "Loading..." : "Load earlier"}
                        </Button>
                    </div>
                {/if}
                {#if msgHook}
                    {#each msgHook.messages as msg (msg.id)}
                        <MessageBubble {msg} {undo} onOpenChange={openChangeDiff} />
                    {/each}
                {/if}
            </div>
            <div class="border-t border-border shrink-0" style="padding-bottom: env(safe-area-inset-bottom);">
                <div class="flex gap-2 p-3 pb-1">
                    <textarea
                        class="flex-1 min-h-[48px] max-h-[50vh] text-sm placeholder:text-muted-foreground dark:bg-input/30 border-input flex rounded-md border bg-transparent px-3 py-2 outline-none transition-[color,box-shadow] disabled:opacity-50"
                        rows={2} placeholder="Type a message..."
                        bind:value={input}
                        onkeydown={handleKeydown}
                        disabled={msgHook?.sending}
                    ></textarea>
                    {#if msgHook && msgHook.sending}
                        <Button variant="destructive" size="icon" class="size-10 shrink-0" onclick={() => msgHook?.abort()} title="Stop">
                            <Square class="size-4" />
                        </Button>
                    {:else}
                        <Button onclick={send} disabled={!input.trim()} size="icon" variant="outline" class="size-10 shrink-0 border-primary/40 bg-primary/10 text-primary hover:bg-primary/20 hover:border-primary/60" title="Send">
                            <Send class="size-4" />
                        </Button>
                    {/if}
                </div>

                <div class="flex items-center gap-2 px-3 py-1.5 text-xs shrink-0 flex-wrap bg-muted/5 relative">
                    <div class="relative">
                        <button class="flex items-center gap-1 px-2 py-0.5 rounded border border-input hover:bg-accent/40 transition-colors" onclick={() => showModelPicker = !showModelPicker}>
                            <span class="font-medium truncate max-w-[180px]">{currentModelName}</span>
                            <ChevronDown class="size-3" />
                        </button>
                        {#if showModelPicker}
                            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                            <div class="absolute bottom-full left-0 mb-1 w-64 rounded-md border bg-popover shadow-md z-50 max-h-48 overflow-auto" role="listbox" tabindex="-1" aria-label="Model picker" onclick={() => showModelPicker = false}>
                                {#each models as m (m.provider_id + "/" + m.id)}
                                    <button class="w-full text-left px-3 py-1.5 text-xs hover:bg-accent flex items-center justify-between {m.id === store.activeSession?.model ? 'bg-accent/60 font-medium' : ''}" onclick={() => switchModel(m.id)}>
                                        <span class="truncate">{m.name || m.id}</span>
                                        <span class="text-[10px] text-muted-foreground">{m.provider_id}</span>
                                    </button>
                                {/each}
                                {#if models.length === 0}
                                    <div class="px-3 py-2 text-xs text-muted-foreground">No models. Add a provider in Settings.</div>
                                {/if}
                            </div>
                        {/if}
                    </div>

                    <div class="relative">
                        <button class="flex items-center gap-1 px-2 py-0.5 rounded border border-input hover:bg-accent/40 transition-colors" onclick={() => showPresetPicker = !showPresetPicker}>
                            <span class="font-medium">{store.activeSession?.preset || "preset"}</span>
                            <ChevronDown class="size-3" />
                        </button>
                        {#if showPresetPicker}
                            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                            <div class="absolute bottom-full left-0 mb-1 w-48 rounded-md border bg-popover shadow-md z-50 max-h-48 overflow-auto" role="listbox" tabindex="-1" aria-label="Preset picker" onclick={() => showPresetPicker = false}>
                                {#each presets as p (p.id)}
                                    <button class="w-full text-left px-3 py-1.5 text-xs hover:bg-accent {p.id === store.activeSession?.preset ? 'bg-accent/60 font-medium' : ''}" onclick={() => switchPreset(p.id)}>
                                        {p.id}
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </div>

                    <div class="relative">
                        <button class="flex items-center gap-1 px-2 py-0.5 rounded border border-input hover:bg-accent/40 transition-colors" onclick={() => { showImagePicker = !showImagePicker }}>
                            <span class="font-medium">{currentBaseImageLabel}</span>
                            <ChevronDown class="size-3" />
                        </button>
                        {#if showImagePicker}
                            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                            <div class="absolute bottom-full left-0 mb-1 w-56 rounded-md border bg-popover shadow-md z-50 max-h-48 overflow-auto" role="listbox" tabindex="-1" aria-label="Worker image picker" onclick={() => showImagePicker = false}>
                                <button class="w-full text-left px-3 py-1.5 text-xs hover:bg-accent {!store.activeSession?.base_image ? 'bg-accent/60 font-medium' : ''}" onclick={() => switchBaseImage('')}>debian-trixie-slim (default)</button>
                            </div>
                        {/if}
                    </div>

                    {#if msgHook?.sending}
                        <span class="w-1.5 h-1.5 rounded-full bg-yellow-500 animate-pulse ml-auto shrink-0"></span>
                    {/if}
                    {#if tokensLabel}
                        <span class="text-[10px] font-mono text-muted-foreground ml-auto shrink-0" title="session tokens (in+out)">
                            {tokensLabel}
                        </span>
                    {/if}
                </div>
            </div>
        </div>

        {#if store.sessionOverlay}
            <!-- Desktop: side panel next to chat -->
            <div class="hidden lg:flex shrink-0 relative border-l border-border bg-background flex-col {panelResizing ? '' : 'transition-[width] duration-150'}"
                 style={`width: ${panelWidth}px; max-width: 70vw`}>
                <!-- Drag handle: resize the side panel (persisted) -->
                <div class="absolute left-0 top-0 bottom-0 w-1.5 z-20 cursor-col-resize group"
                     style="touch-action: none"
                     role="separator" aria-orientation="vertical" aria-label="Resize panel"
                     title="Drag to resize"
                     onpointerdown={startPanelResize}>
                    <div class="h-full w-full {panelResizing ? 'bg-primary/50' : 'group-hover:bg-primary/30 group-active:bg-primary/50'}"></div>
                </div>
                <div class="flex items-center gap-0.5 border-b border-border px-2 py-1.5 shrink-0" role="tablist" aria-label="Session panels">
                    {#each overlayTabs as tab (tab.id)}
                        <button
                            role="tab"
                            aria-selected={store.sessionOverlay === tab.id}
                            title={store.sessionOverlay === tab.id ? `Close ${tab.label}` : tab.label}
                            class="flex items-center gap-1.5 px-2 py-1 rounded text-[11px] transition-colors {store.sessionOverlay === tab.id ? 'bg-accent text-accent-foreground font-medium' : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'}"
                            onclick={() => toggleOverlay(tab.id)}
                        >
                            <tab.icon class="size-3.5 shrink-0" />
                            <span class="hidden xl:inline">{tab.label}</span>
                        </button>
                    {/each}
                    <Button variant="ghost" size="icon-sm" class="size-6 ml-auto shrink-0" title="Close panel" onclick={() => store.closeOverlay()}>
                        <X class="size-3.5" />
                    </Button>
                </div>
                <div class="flex-1 min-h-0 relative">
                    {#if store.sessionOverlay === 'timeline'}
                        <div class="absolute inset-0">
                            {#if diffChangeId}
                                <DiffScreen
                                    changeId={diffChangeId}
                                    sessionOrg={store.activeSession?.org}
                                    sessionRepo={store.activeSession?.repo}
                                    onclose={() => store.openOverlay('timeline')}
                                    onselectFile={jumpToFile}
                                />
                            {:else}
                                <TimelinePage onSelectDiff={id => store.openChange(id)} />
                            {/if}
                        </div>
                    {:else if store.sessionOverlay === 'files'}
                        <div class="absolute inset-0">
                            <FilesPage />
                        </div>
                    {:else if store.sessionOverlay === 'mailbox'}
                        <div class="absolute inset-0">
                            <MailboxPage />
                        </div>
                    {:else if store.sessionOverlay === 'container'}
                        <div class="absolute inset-0 flex flex-col">
                            {#if containerLoading}
                                <p class="text-xs text-muted-foreground p-3">Loading...</p>
                            {:else if containerRow}
                                <ContainerWorkspace
                                    containerId={containerRow.id}
                                    containerName={containerRow.name}
                                    onclose={() => store.closeOverlay()}
                                />
                            {:else if sessionWorkerId}
                                <ContainerWorkspace
                                    containerId={sessionWorkerId}
                                    containerName={store.activeSession?.container_id ?? 'session-worker'}
                                    onclose={() => store.closeOverlay()}
                                />
                            {:else}
                                <div class="flex-1 flex flex-col items-center justify-center gap-2 text-xs text-muted-foreground px-4 text-center">
                                    <p>No worker container yet — it starts automatically when the agent runs bash or other tools.</p>
                                    <Button size="sm" variant="outline" onclick={createBoundContainer} disabled={containerLoading}>
                                        Create container now
                                    </Button>
                                </div>
                            {/if}
                        </div>
                    {:else if store.sessionOverlay === 'todos'}
                        <div class="overflow-y-auto p-3 space-y-1">
                            {#if todos.length === 0}
                                <p class="text-xs text-muted-foreground text-center py-6">No todos yet — the agent tracks its plan here via <span class="font-mono">todowrite</span>.</p>
                            {/if}
                            {#each todos as t (t.id)}
                                <div class="flex items-start gap-2 text-xs px-1 py-1.5 rounded hover:bg-accent/40">
                                    {#if t.status === 'completed'}
                                        <CircleCheck class="size-3.5 text-green-500 mt-0.5 shrink-0" />
                                    {:else if t.status === 'in_progress'}
                                        <CircleDot class="size-3.5 text-yellow-500 mt-0.5 shrink-0" />
                                    {:else if t.status === 'cancelled'}
                                        <X class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                                    {:else}
                                        <Circle class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                                    {/if}
                                    <span class="flex-1 {t.status === 'completed' || t.status === 'cancelled' ? 'line-through text-muted-foreground' : ''}">{t.content}</span>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            </div>

            <!-- Mobile: full-screen overlay -->
            <div class="absolute inset-0 z-20 bg-background flex flex-col lg:hidden">
                <div class="flex-1 min-h-0 relative">
                    {#if store.sessionOverlay === 'timeline'}
                        <div class="absolute inset-0">
                            {#if diffChangeId}
                                <DiffScreen
                                    changeId={diffChangeId}
                                    sessionOrg={store.activeSession?.org}
                                    sessionRepo={store.activeSession?.repo}
                                    onclose={() => store.openOverlay('timeline')}
                                    onselectFile={jumpToFile}
                                />
                            {:else}
                                <TimelinePage onSelectDiff={id => store.openChange(id)} />
                            {/if}
                        </div>
                    {:else if store.sessionOverlay === 'files'}
                        <div class="absolute inset-0">
                            <FilesPage />
                        </div>
                    {:else if store.sessionOverlay === 'mailbox'}
                        <div class="absolute inset-0">
                            <MailboxPage />
                        </div>
                    {:else if store.sessionOverlay === 'container'}
                        <div class="absolute inset-0 flex flex-col">
                            {#if containerLoading}
                                <p class="text-xs text-muted-foreground p-3">Loading...</p>
                            {:else if containerRow}
                                <ContainerWorkspace
                                    containerId={containerRow.id}
                                    containerName={containerRow.name}
                                    onclose={() => store.closeOverlay()}
                                />
                            {:else if sessionWorkerId}
                                <ContainerWorkspace
                                    containerId={sessionWorkerId}
                                    containerName={store.activeSession?.container_id ?? 'session-worker'}
                                    onclose={() => store.closeOverlay()}
                                />
                            {:else}
                                <div class="flex-1 flex flex-col items-center justify-center gap-2 text-xs text-muted-foreground px-4 text-center">
                                    <p>No worker container yet — it starts automatically when the agent runs bash or other tools.</p>
                                    <Button size="sm" variant="outline" onclick={createBoundContainer} disabled={containerLoading}>
                                        Create container now
                                    </Button>
                                </div>
                            {/if}
                        </div>
                    {:else if store.sessionOverlay === 'todos'}
                        <div class="overflow-y-auto p-3 space-y-1">
                            {#if todos.length === 0}
                                <p class="text-xs text-muted-foreground text-center py-6">No todos yet — the agent tracks its plan here via <span class="font-mono">todowrite</span>.</p>
                            {/if}
                            {#each todos as t (t.id)}
                                <div class="flex items-start gap-2 text-xs px-1 py-1.5 rounded hover:bg-accent/40">
                                    {#if t.status === 'completed'}
                                        <CircleCheck class="size-3.5 text-green-500 mt-0.5 shrink-0" />
                                    {:else if t.status === 'in_progress'}
                                        <CircleDot class="size-3.5 text-yellow-500 mt-0.5 shrink-0" />
                                    {:else if t.status === 'cancelled'}
                                        <X class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                                    {:else}
                                        <Circle class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                                    {/if}
                                    <span class="flex-1 {t.status === 'completed' || t.status === 'cancelled' ? 'line-through text-muted-foreground' : ''}">{t.content}</span>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            </div>
        {/if}
    </div>

</div>
