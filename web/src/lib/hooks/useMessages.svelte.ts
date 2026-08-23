import * as api from '$lib/api'
import { uid } from '$lib/utils'
import {
  type ChatMessage,
  compareMessages,
  mapMessagesToChat,
  type StreamEvent,
} from './message-utils'

export type { ChatMessage, ChatPart, MsgStatus } from './message-utils'

export function createMessages(sessionId: () => string) {
  let messages = $state<ChatMessage[]>([])
  let sending = $state(false)
  let loading = $state(false)
  let hasMore = $state(false)

  let mountId = 0
  let eventSource: EventSource | null = null
  let streamingId = $state<string | null>(null)
  let nextSeq = 1000000

  function allocSeq(): number {
    return nextSeq++
  }

  function bumpSeqAfter(history: ChatMessage[]): void {
    const maxSeq = history.reduce(
      (max, m) => (typeof m.seq === 'number' && m.seq < nextSeq ? m.seq : max),
      -1,
    )
    if (maxSeq >= 0) nextSeq = Math.max(nextSeq, maxSeq + 1)
  }

  // delta batching (rAF flush) for performance
  let deltaBatch: Record<string, string> = {}
  let deltaRafId: number | undefined

  const sortedMsgs = $derived(
    [...messages].sort((a, b) => compareMessages(a, b)),
  )

  function flushDeltaBatch() {
    deltaRafId = undefined
    const batch = deltaBatch
    deltaBatch = {}
    if (!streamingId || Object.keys(batch).length === 0) return
    const idx = messages.findIndex(m => m.id === streamingId)
    if (idx < 0) return
    const msg = messages[idx]
    const parts = [...msg.parts]
    for (const partId of Object.keys(batch)) {
      const delta = batch[partId]
      const pidx = parts.findIndex(p => p.id === partId)
      if (pidx >= 0) {
        const existing = parts[pidx]
        parts[pidx] = {
          ...existing,
          text: (existing.text ?? '') + delta,
        }
      } else {
        const isReasoning = partId.startsWith('r')
        parts.push({
          id: partId,
          type: isReasoning ? 'reasoning' : 'text',
          text: delta,
        })
      }
    }
    const next = [...messages]
    next[idx] = { ...msg, parts }
    messages = next
  }

  function scheduleDeltaFlush() {
    if (deltaRafId != null) return
    deltaRafId = requestAnimationFrame(flushDeltaBatch)
  }

  function ensureStreamingMsg(forceNew = false): string {
    if (!forceNew && streamingId) {
      const exists = messages.find(m => m.id === streamingId)
      if (exists) return streamingId
    }
    const id = uid()
    streamingId = id
    messages = [
      ...messages,
      {
        id,
        role: 'assistant',
        status: 'streaming',
        parts: [],
        createdAt: new Date().toISOString(),
        seq: allocSeq(),
      },
    ]
    return id
  }

  function ensurePart(msgId: string, partId: string, type: string): void {
    const idx = messages.findIndex(m => m.id === msgId)
    if (idx < 0) return
    const msg = messages[idx]
    if (msg.parts.some(p => p.id === partId)) return
    const next = [...messages]
    next[idx] = {
      ...msg,
      parts: [...msg.parts, { id: partId, type, text: '' }],
    }
    messages = next
  }

  function addToolPart(
    msgId: string,
    partId: string,
    name: string,
    input: unknown,
  ): void {
    const idx = messages.findIndex(m => m.id === msgId)
    if (idx < 0) return
    const msg = messages[idx]
    if (msg.parts.some(p => p.id === partId)) {
      // update existing
      const parts = msg.parts.map(p =>
        p.id === partId
          ? {
              ...p,
              type: 'tool' as const,
              tool: name,
              state: {
                status: 'running' as const,
                title: name,
                input: input as Record<string, unknown>,
              },
            }
          : p,
      )
      const next = [...messages]
      next[idx] = { ...msg, parts }
      messages = next
      return
    }
    const next = [...messages]
    next[idx] = {
      ...msg,
      parts: [
        ...msg.parts,
        {
          id: partId,
          type: 'tool',
          tool: name,
          state: {
            status: 'running',
            title: name,
            input: input as Record<string, unknown>,
          },
        },
      ],
    }
    messages = next
  }

  function updateToolResult(
    partId: string,
    result: unknown,
    errorMsg?: string,
    changeId?: string,
    diff?: string,
    additions?: number,
    deletions?: number,
  ): void {
    if (!streamingId) return
    const idx = messages.findIndex(m => m.id === streamingId)
    if (idx < 0) return
    const msg = messages[idx]
    const parts = msg.parts.map(p => {
      if (p.id !== partId) return p
      return {
        ...p,
        type: 'tool' as const,
        state: {
          ...(p.state ?? {}),
          status: errorMsg ? ('error' as const) : ('complete' as const),
          output:
            typeof result === 'string'
              ? result
              : JSON.stringify(result, null, 2),
          error: errorMsg,
          change_id: changeId ?? p.state?.change_id,
          diff: diff ?? p.state?.diff,
          additions: additions ?? p.state?.additions,
          deletions: deletions ?? p.state?.deletions,
        },
      }
    })
    const next = [...messages]
    next[idx] = { ...msg, parts }
    messages = next
  }

  function finishStreaming(): void {
    if (deltaRafId != null) {
      cancelAnimationFrame(deltaRafId)
      flushDeltaBatch()
    }
    if (!streamingId) {
      sending = false
      return
    }
    const idx = messages.findIndex(m => m.id === streamingId)
    if (idx >= 0) {
      const next = [...messages]
      next[idx] = { ...messages[idx], status: 'complete' }
      messages = next
    }
    streamingId = null
    sending = false
  }

  function addErrorMessage(text: string): void {
    messages = [
      ...messages.filter(m => m.status !== 'streaming'),
      {
        id: uid(),
        role: 'error',
        status: 'error',
        parts: [{ id: uid(), type: 'text', text }],
        createdAt: new Date().toISOString(),
        seq: allocSeq(),
      },
    ]
    streamingId = null
  }

  function handleEvent(ev: StreamEvent): void {
    const { event, params } = ev
    switch (event) {
      case 'step-start':
      case 'text-start':
      case 'reasoning-start':
      case 'tool-input-start': {
        const current = streamingId
          ? messages.find(m => m.id === streamingId)
          : undefined
        const hasToolPart = current?.parts.some(p => p.type === 'tool') ?? false
        const sid = ensureStreamingMsg(
          event === 'step-start' || (event === 'text-start' && hasToolPart),
        )
        if (event === 'text-start' && params.id)
          ensurePart(sid, params.id, 'text')
        else if (event === 'reasoning-start' && params.id)
          ensurePart(sid, `r${params.id}`, 'reasoning')
        break
      }
      case 'text-delta': {
        if (params.id && params.text) {
          ensureStreamingMsg()
          deltaBatch[params.id] = (deltaBatch[params.id] ?? '') + params.text
          scheduleDeltaFlush()
        }
        break
      }
      case 'reasoning-delta': {
        if (params.id && params.text) {
          ensureStreamingMsg()
          const pid = `r${params.id}`
          deltaBatch[pid] = (deltaBatch[pid] ?? '') + params.text
          scheduleDeltaFlush()
        }
        break
      }
      case 'tool-call': {
        const sid = ensureStreamingMsg()
        const tcId = (params.toolCallId as string) ?? params.id
        if (tcId)
          addToolPart(
            sid,
            tcId,
            (params.toolName as string) ?? params.name ?? 'tool',
            params.input,
          )
        break
      }
      case 'tool-result': {
        const tcId = (params.toolCallId as string) ?? params.id
        if (!tcId) break
        updateToolResult(
          tcId,
          (params.formatted as unknown) ??
            (params.output as unknown) ??
            params.result,
          undefined,
          typeof params.change_id === 'string' ? params.change_id : undefined,
          typeof params.diff === 'string' ? params.diff : undefined,
          typeof params.additions === 'number' ? params.additions : undefined,
          typeof params.deletions === 'number' ? params.deletions : undefined,
        )
        break
      }
      case 'tool-error': {
        const tcId = (params.toolCallId as string) ?? params.id
        const errObj = params.error as { message?: string } | string | undefined
        const errMsg =
          typeof errObj === 'string'
            ? errObj
            : (errObj?.message ?? params.message ?? 'tool error')
        if (tcId) updateToolResult(tcId, undefined, errMsg)
        break
      }
      case 'turn-complete': {
        finishStreaming()
        break
      }
      case 'status': {
        const stype = params.type
        if (stype === 'busy' || stype === 'running') {
          sending = true
        } else {
          finishStreaming()
        }
        break
      }
      case 'error':
      case 'provider-error': {
        const errObj = params.error as { message?: string } | string | undefined
        const msg =
          typeof errObj === 'string'
            ? errObj
            : (errObj?.message ?? params.message ?? 'Unknown error')
        addErrorMessage(msg)
        sending = false
        break
      }
      case 'finish':
      case 'step-finish':
      case 'text-end':
      case 'reasoning-end':
      case 'tool-input-end':
        // no-op boundaries
        break
      default:
        break
    }
  }

  function connectSSE(sid: string): void {
    if (eventSource) eventSource.close()
    eventSource = new EventSource(`/api/v1/sessions/${sid}/stream`)
    eventSource.onmessage = e => {
      try {
        const ev = JSON.parse(e.data) as StreamEvent
        handleEvent(ev)
      } catch {
        // ignore parse errors
      }
    }
    // EventSource auto-reconnects on error; no manual handling needed
  }

  async function fetchMessages(before?: string): Promise<void> {
    loading = true
    const r = await api.sessions.messages(sessionId(), { limit: 50, before })
    r.match(
      data => {
        const chat = mapMessagesToChat(data.messages)
        bumpSeqAfter(chat)
        if (before) {
          const existing = new Set(messages.map(m => m.id))
          const merged = [...chat.filter(m => !existing.has(m.id)), ...messages]
          messages = merged
        } else {
          const pending = messages.filter(m => m.status === 'pending')
          messages = [...pending, ...chat]
        }
        hasMore = data.hasMore
        loading = false
      },
      () => {
        loading = false
      },
    )
  }

  async function recover(curMount: number): Promise<void> {
    const r = await api.sessions.state(sessionId())
    if (curMount !== mountId) return
    r.match(
      state => {
        if (state.status === 'busy' || state.status === 'running') {
          sending = true
          streamingId = uid()
          messages = [
            ...messages,
            {
              id: streamingId,
              role: 'assistant',
              status: 'streaming',
              parts: [],
              createdAt: new Date().toISOString(),
              seq: allocSeq(),
            },
          ]
        }
      },
      () => {},
    )
  }

  async function init(): Promise<() => void> {
    const cur = ++mountId
    await fetchMessages()
    if (cur !== mountId) return () => {}
    await recover(cur)
    if (cur !== mountId) return () => {}
    connectSSE(sessionId())
    return () => {
      mountId = 0
      if (deltaRafId != null) cancelAnimationFrame(deltaRafId)
      if (eventSource) {
        eventSource.close()
        eventSource = null
      }
    }
  }

  async function send(text: string): Promise<void> {
    const trimmed = text.trim()
    if (!trimmed || sending) return
    sending = true
    canRedo = false
    // optimistic: clear any stale streaming, add pending user msg + streaming placeholder
    const optimisticUserSeq = allocSeq()
    messages = [
      ...messages.filter(m => m.status !== 'streaming'),
      {
        id: uid(),
        role: 'user',
        status: 'pending',
        parts: [{ id: uid(), type: 'text', text: trimmed }],
        createdAt: new Date().toISOString(),
        seq: optimisticUserSeq,
      },
    ]
    streamingId = uid()
    messages = [
      ...messages,
      {
        id: streamingId,
        role: 'assistant',
        status: 'streaming',
        parts: [],
        createdAt: new Date().toISOString(),
        seq: allocSeq(),
      },
    ]
    const r = await api.sessions.prompt(sessionId(), trimmed)
    r.match(
      data => {
        // swap the optimistic pending user message id for the real one
        messages = messages.map(m =>
          m.status === 'pending' && m.role === 'user'
            ? { ...m, id: data.messageId, status: 'complete' as const }
            : m,
        )
      },
      err => {
        addErrorMessage(typeof err === 'string' ? err : 'Send failed')
        sending = false
      },
    )
  }

  async function loadMore(): Promise<void> {
    if (!hasMore || loading) return
    const first = sortedMsgs.find(m => m.status === 'complete')
    if (!first) return
    await fetchMessages(first.id)
  }

  async function abort(): Promise<void> {
    await api.sessions.interrupt(sessionId())
    finishStreaming()
  }

  let canRedo = $state(false)

  async function revert(messageId: string): Promise<void> {
    const current = sortedMsgs
    await api.sessions.revert(sessionId(), messageId)
    canRedo = true
    const idx = current.findIndex(m => m.id === messageId)
    const keep = idx >= 0 ? current.slice(0, idx) : current
    messages = keep.map(m =>
      m.status === 'streaming' ? { ...m, status: 'complete' as const } : m,
    )
    streamingId = null
    sending = false
    void fetchMessages()
  }

  async function redo(): Promise<void> {
    if (!canRedo) return
    const r = await api.sessions.redo(sessionId())
    if (r.isOk() && r.value.ok) {
      canRedo = false
      void fetchMessages()
    }
  }

  return {
    get messages() {
      return sortedMsgs
    },
    get canRedo() {
      return canRedo
    },
    get sending() {
      return sending
    },
    get loading() {
      return loading
    },
    get hasMore() {
      return hasMore
    },
    send,
    loadMore,
    abort,
    redo,
    revert,
    init,
  }
}
