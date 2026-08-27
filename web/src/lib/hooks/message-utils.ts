import type { MessagePart } from '@zergx/schema'
import type { Message } from '$lib/api'
import { uid } from '$lib/utils'

export type MsgStatus = 'pending' | 'streaming' | 'complete' | 'error'

export interface ChatPart extends MessagePart {
  id: string
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'tool' | 'system' | 'error' | 'compaction' | 'event'
  status: MsgStatus
  parts: ChatPart[]
  createdAt?: string
  seq?: number
}

export interface StreamEvent {
  event: string
  params: Record<string, unknown> & {
    id?: string
    text?: string
    name?: string
    input?: unknown
    result?: unknown
    output?: unknown
    formatted?: unknown
    change_id?: string
    diff?: string
    additions?: number
    deletions?: number
    message?: string
    reason?: string
    type?: string
  }
}

export function compareMessages(a: ChatMessage, b: ChatMessage): number {
  const at = a.seq ?? Number.MAX_SAFE_INTEGER
  const bt = b.seq ?? Number.MAX_SAFE_INTEGER
  if (at !== bt) return at - bt
  const apt = a.createdAt ? Date.parse(a.createdAt) || 0 : 0
  const bpt = b.createdAt ? Date.parse(b.createdAt) || 0 : 0
  if (apt && bpt && apt !== bpt) return apt - bpt
  return a.id.localeCompare(b.id)
}

export function mapMessagesToChat(msgs: Message[]): ChatMessage[] {
  return msgs.map((m, i) => ({
    id: m.id,
    role: m.role as ChatMessage['role'],
    status: 'complete' as const,
    createdAt: m.created_at,
    seq: (m as { seq?: number }).seq ?? i,
    parts: (m.parts ?? []).map(p => ({
      ...p,
      id: (p as ChatPart).id || uid(),
    })) as ChatPart[],
  }))
}
