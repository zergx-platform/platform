import type {
  BookmarkNode,
  ChangeEntry,
  ContainerInfo,
  Deployment,
  DiffFile,
  ExecResult,
  FileEntry,
  FlatMessage,
  JobInfo,
  MailboxEntry,
  Message,
  MessagePart,
  ModelInfo,
  OrgNode,
  PresetInfo,
  ProviderInfo,
  Session,
  SessionInfo,
  SessionTab,
  SiderTab,
  ToolConfigMap,
  ToolInfo,
  ToolState,
} from '@recoder-neo/schema'
import { ResultAsync } from 'neverthrow'
import { z } from 'zod'

export type FileCommit = {
  change_id: string
  commit_id: string
  author: string
  timestamp: string
  message: string
}

export type {
  BookmarkNode,
  ChangeEntry,
  ContainerInfo,
  Deployment,
  DiffFile,
  ExecResult,
  FileEntry,
  FlatMessage,
  JobInfo,
  MailboxEntry,
  Message,
  MessagePart,
  ModelInfo,
  OrgNode,
  PresetInfo,
  ProviderInfo,
  Session,
  SessionInfo,
  SessionTab,
  SiderTab,
  ToolConfigMap,
  ToolInfo,
  ToolState,
}

async function httpErr(r: Response): Promise<string> {
  const jsonResult = await ResultAsync.fromPromise(
    r.json(),
    () => `HTTP ${r.status}`,
  )
  if (jsonResult.isErr()) return jsonResult.error
  const j = jsonResult.value as Record<string, unknown>
  return j?.error ? String(j.error) : `HTTP ${r.status}`
}

export function doFetch<T>(
  url: string,
  init?: RequestInit,
  schema?: z.ZodSchema<T>,
): ResultAsync<T, string> {
  return ResultAsync.fromPromise(
    (async () => {
      const r = await fetch(url, init)
      if (!r.ok) throw new Error(await httpErr(r))
      const raw = await r.json()
      if (schema) {
        const parsed = schema.safeParse(raw)
        if (!parsed.success)
          throw new Error(`Schema validation failed: ${parsed.error.message}`)
        return parsed.data
      }
      return raw as T
    })(),
    e => (e instanceof Error ? e.message : String(e)),
  )
}

export function get<T>(
  url: string,
  schema?: z.ZodSchema<T>,
): ResultAsync<T, string> {
  return doFetch(url, undefined, schema)
}

export function post<T>(
  url: string,
  body?: unknown,
  schema?: z.ZodSchema<T>,
): ResultAsync<T, string> {
  return doFetch(
    url,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    },
    schema,
  )
}

export function del<T>(
  url: string,
  schema?: z.ZodSchema<T>,
): ResultAsync<T, string> {
  return doFetch(url, { method: 'DELETE' }, schema)
}

export function patch<T>(
  url: string,
  body?: unknown,
  schema?: z.ZodSchema<T>,
): ResultAsync<T, string> {
  return doFetch(
    url,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    },
    schema,
  )
}

export function put<T>(
  url: string,
  body?: unknown,
  schema?: z.ZodSchema<T>,
): ResultAsync<T, string> {
  return doFetch(
    url,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    },
    schema,
  )
}

export const OkSchema = z.object({ ok: z.boolean() })
