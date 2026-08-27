import { z } from 'zod'
import { ResultAsync } from 'neverthrow'
import { del, get, post } from './api-core'
import { qs } from './client'
import {
  SandboxSchema,
  DeploymentSchema,
  DeploymentPodSchema,
  ContainerfileTemplateSchema,
  ImageBuildResultSchema,
  OpsStatusSchema,
  PublishSpecSchema,
  ExecResultSchema,
  JobInfoSchema,
  OciCatalogSchema,
  PackageTypesResponseSchema,
  PackageVersionsResponseSchema,
  ZergxConfigSchema,
} from '@zergx/schema'

export const containers = {
  /** Session sandboxes from ops-extension. */
  list: () =>
    get(
      `/api/v1/sandboxes`,
      z.object({ sandboxes: z.array(SandboxSchema) }),
    ).map(r => r.sandboxes),

  /** Deployments owned by ops-extension. */
  deployments: () =>
    get(
      `/api/v1/deployments`,
      z.object({ deployments: z.array(DeploymentSchema) }),
    ).map(r => r.deployments),

  /** Pods of one deployment. */
  deploymentPods: (name: string) =>
    get(
      `/api/v1/deployments/${encodeURIComponent(name)}/pods`,
      z.object({ pods: z.array(DeploymentPodSchema) }),
    ).map(r => r.pods),

  /** Recent k8s events of one deployment (warnings surfaced in the UI). */
  deploymentEvents: (name: string) =>
    get(
      `/api/v1/deployments/${encodeURIComponent(name)}/events`,
      z.object({
        events: z.array(
          z.object({
            type: z.string().optional(),
            reason: z.string().optional(),
            message: z.string().optional(),
            age: z.string().optional(),
          }),
        ),
      }),
    ).map(r => r.events),

  /** Rollout restart (kicks a re-pull of the current image tag). */
  restartDeployment: (name: string) =>
    post(
      `/api/v1/deployments/${encodeURIComponent(name)}/restart`,
      undefined,
      z.object({ ok: z.boolean() }),
    ),

  /** Rollout status of one deployment. */
  deploymentStatus: (name: string) =>
    get(
      `/api/v1/deployments/${encodeURIComponent(name)}/status`,
      z.object({
        observed_generation: z.number(),
        updated_replicas: z.number(),
        ready_replicas: z.number(),
        available_replicas: z.number(),
        unavailable_replicas: z.number(),
        conditions: z.array(z.unknown()),
      }),
    ),

  deploy: (body: {
    name: string
    image: string
    replicas?: number
    port?: number
    env?: Record<string, string>
    session?: string
  }) =>
    post(
      `/api/v1/deployments`,
      body,
      z.object({ ok: z.boolean(), name: z.string(), image: z.string() }),
    ),

  destroySandbox: (session: string) => del(`/api/v1/sandboxes/${session}`),
  destroyDeployment: (name: string) => del(`/api/v1/deployments/${encodeURIComponent(name)}`),

  /** ops-extension aggregated health/status. */
  status: () =>
    get(`/api/v1/status`, OpsStatusSchema),

  /** Built-in Containerfile templates. */
  containerfileTemplates: () =>
    get(
      `/api/v1/containerfile-templates`,
      z.object({ templates: z.array(ContainerfileTemplateSchema) }),
    ).map(r => r.templates),

  /** Build + push an image from a raw Containerfile or a repo archive. */
  buildImage: (body: {
    dockerfile?: string
    tag?: string
    raw?: boolean
    org?: string
    repo?: string
    bookmark?: string
    push?: boolean
  }) => post(`/api/v1/images/build`, body, ImageBuildResultSchema),

  /** Per-protocol publish specs (dynamic publish form). */
  publishSpecs: () =>
    get(
      `/api/v1/publish-specs`,
      z.object({ specs: z.array(PublishSpecSchema) }),
    ).map(r => r.specs),

  /** Publish a package from a repo. */
  publishPackage: (body: {
    protocol: string
    org: string
    repo: string
    bookmark: string
    session: string
    name: string
    version: string
    file: string
    dockerfile_path: string
  }) => post(`/api/v1/packages/publish`, body, ImageBuildResultSchema),

  jobs: (session: string) =>
    get(
      `/api/v1/sandboxes/${session}/jobs`,
      z.object({ jobs: z.object({ jobs: z.array(JobInfoSchema) }) }),
    ).map(r => r.jobs?.jobs ?? []),

  exec: (session: string, command: string, timeoutMs?: number) =>
    post(
      `/api/v1/sandboxes/${session}/exec`,
      { command },
      ExecResultSchema,
    ),

  kill: (session: string, jobId: string) =>
    post(
      `/api/v1/sandboxes/${session}/jobs/${encodeURI(jobId)}/kill`,
      undefined,
      z.object({
        ok: z.boolean(),
        result: z.object({ ok: z.boolean().optional() }).optional(),
        error: z.string().optional(),
      }),
    ),

  jobOutput: (
    session: string,
    jobId: string,
    stream: string,
    start: number,
    end: number,
  ) =>
    get(
      `/api/v1/sandboxes/${session}/jobs/${encodeURI(jobId)}/output${qs({ stream, start, end })}`,
      z.object({
        lines: z.array(z.string()),
        total_lines: z.number(),
        start_line: z.number(),
        end_line: z.number(),
        done: z.boolean(),
      }),
    ).map(r => ({ ...r, lines: r.lines ?? [] })),
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

  zergxConfig: () =>
    ResultAsync.fromPromise(
      (async () => {
        const r = await fetch('/api/v1/zergx-config')
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        const parsed = ZergxConfigSchema.safeParse(await r.json())
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
