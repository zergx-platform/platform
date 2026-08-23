import {
  ContainerInfoSchema,
  ExecResultSchema,
  JobInfoSchema,
  OciCatalogSchema,
  PackageTypesResponseSchema,
  PackageVersionsResponseSchema,
  RecoreConfigSchema,
} from '@recoder-neo/schema'
import { ResultAsync } from 'neverthrow'
import { z } from 'zod'
import { del, get, post } from './api-core'
import { qs } from './client'

export const containers = {
  list: () =>
    get(
      `/api/v1/containers`,
      z.object({ containers: z.array(ContainerInfoSchema) }),
    ).map(r => r.containers),
  create: (opts?: {
    image?: string
    session_id?: string
    org?: string
    repo?: string
    branch?: string
  }) =>
    post(
      `/api/v1/containers`,
      opts ?? {},
      z.object({ container: ContainerInfoSchema }),
    ).map(r => r.container),
  destroy: (cid: string) =>
    del(`/api/v1/containers/${cid}`),
  jobs: (cid: string) =>
    get(
      `/api/v1/containers/${cid}/jobs`,
      z.object({ jobs: z.array(JobInfoSchema) }),
    ).map(r => r.jobs || []),
  exec: (cid: string, command: string, timeoutMs?: number) =>
    post(
      `/api/v1/containers/${cid}/exec`,
      {
        command,
        timeout_ms: timeoutMs ?? 120000,
      },
      ExecResultSchema,
    ),
  kill: (cid: string, jobId: string) =>
    post(
      `/api/v1/containers/${cid}/kill/${jobId}`,
      undefined,
      z.object({
        ok: z.boolean(),
        result: z.object({ ok: z.boolean().optional() }).optional(),
        error: z.string().optional(),
      }),
    ),
  jobOutput: (
    cid: string,
    jobId: string,
    stream: string,
    start: number,
    end: number,
  ) =>
    get(
      `/api/v1/containers/${cid}/jobs/${jobId}/output${qs({ stream, start, end })}`,
      z.object({
        lines: z.array(z.string()),
        total_lines: z.number(),
        start_line: z.number(),
        end_line: z.number(),
        done: z.boolean(),
      }),
    ).map(r => ({ ...r, lines: r.lines ?? [] })),
  events: (cid: string) =>
    get(
      `/api/v1/containers/${cid}/events`,
      z.object({ events: z.array(z.unknown()) }),
    ).map(r => r.events),
}

export const packages = {
  listTypes: () =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch('/api/v1/packages')
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        const raw = await r.json()
        const parsed = PackageTypesResponseSchema.safeParse(raw)
        if (!parsed.success)
          throw new Error(`Schema validation failed: ${parsed.error.message}`)
        return parsed.data.types
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    ),

  listAll: (opts?: {
    type?: string
    q?: string
    limit?: number
    offset?: number
  }) => {
    const { type, q, limit, offset } = opts ?? {}
    const params = new URLSearchParams()
    if (type) params.set('type', type)
    if (q) params.set('q', q)
    if (limit) params.set('limit', String(limit))
    if (offset) params.set('offset', String(offset))
    const qs = params.toString() ? `?${params.toString()}` : ''
    return ResultAsync.fromPromise(
      (async () => {
        const r = await fetch(`/api/v1/packages/list${qs}`)
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        const raw = await r.json()
        const schema = z.object({
          ok: z.boolean(),
          data: z.object({
            packages: z.array(
              z.object({
                name: z.string(),
                type: z.string(),
                latest_version: z.string().nullable(),
                versions: z.number(),
              }),
            ),
            total: z.number(),
            offset: z.number(),
            limit: z.number(),
          }),
        })
        const parsed = schema.safeParse(raw)
        if (!parsed.success)
          throw new Error(`Schema validation failed: ${parsed.error.message}`)
        return parsed.data.data
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    )
  },

  versions: (type: string, name: string) =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch(
          `/api/v1/packages/${encodeURIComponent(type)}/${encodeURIComponent(name)}/versions`,
        )
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        const parsed = PackageVersionsResponseSchema.safeParse(await r.json())
        if (!parsed.success)
          throw new Error(`Schema validation failed: ${parsed.error.message}`)
        return parsed.data.data
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    ),

  deletePackage: (type: string, name: string) =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch(
          `/api/v1/packages/${encodeURIComponent(type)}/${encodeURIComponent(name)}`,
          { method: 'DELETE' },
        )
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return (await r.json()) as { ok: boolean }
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    ),

  deleteVersion: (type: string, name: string, version: string) =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch(
          `/api/v1/packages/${encodeURIComponent(type)}/${encodeURIComponent(name)}/${encodeURIComponent(version)}`,
          { method: 'DELETE' },
        )
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return (await r.json()) as { ok: boolean }
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    ),

  recoreConfig: () =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch('/api/v1/recore-config')
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        const parsed = RecoreConfigSchema.safeParse(await r.json())
        if (!parsed.success)
          throw new Error(`Schema validation failed: ${parsed.error.message}`)
        return parsed.data
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    ),

  ociCatalog: () =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch('/v2/_catalog')
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        const parsed = OciCatalogSchema.safeParse(await r.json())
        if (!parsed.success)
          throw new Error(`Schema validation failed: ${parsed.error.message}`)
        return parsed.data
      })(),
      e => (e instanceof Error ? e.message : String(e)),
    ),
}
