import type { ProviderInfo } from '@zergx/schema'
import {
  DiffFileSchema,
  FileEntrySchema,
  ModelInfoSchema,
  OrgNodeSchema,
  ProvidersMapSchema,
  SessionSchema,
} from '@zergx/schema'
import { z } from 'zod'
import { del, get, OkSchema, post, put } from './api-core'
import { qs } from './client'

export const repos = {
  list: () =>
    get(
      `/api/v1/repos`,
      z.object({ orgs: z.array(OrgNodeSchema) }),
    ).map(r => r.orgs),
  listFiles: (org: string, repo: string, dir = '', bookmark?: string) => {
    const query: Record<string, string> = { org, repo, path: dir, depth: '1' }
    if (bookmark) query.bookmark = bookmark
    return get(
      `/api/v1/fs/list${qs({ org, repo, path: dir, bookmark })}`,
      z.object({ entries: z.array(FileEntrySchema) }),
    ).map(r => r.entries)
  },
  readFile: (org: string, repo: string, filePath: string, bookmark?: string) => {
    const query: Record<string, string> = { org, repo, path: filePath }
    if (bookmark) query.bookmark = bookmark
    return get(
      `/api/v1/fs/read${qs({ org, repo, path: filePath, bookmark })}`,
      z.object({ content: z.string() }),
    ).map(r => r.content)
  },
  deleteBookmark: (org: string, repo: string, bookmark: string) =>
    del(`/api/v1/repos/${org}/${repo}/${encodeURIComponent(bookmark)}`),
  deleteRepo: (org: string, repo: string) =>
    del(`/api/v1/repos/${org}/${repo}`),
  deleteOrg: (org: string) => del(`/api/v1/repos/${org}`),
  forkRepo: (params: {
    source_org: string
    source_repo: string
    source_bookmark: string
    target_org: string
    target_repo: string
    target_bookmark?: string
  }) =>
    post(
      `/api/v1/repos/fork`,
      params,
      z.object({ session: SessionSchema }),
    ).map(r => r.session),
  adoptSession: (org: string, repo: string, bookmark: string) =>
    post(
      `/api/v1/repos/${org}/${repo}/bookmarks/${encodeURIComponent(bookmark)}/session`,
      undefined,
      z.object({ ok: z.boolean(), session_name: z.string(), adopted: z.boolean().optional() }),
    ),
  ensureOrg: (org: string) =>
    post(
      `/api/v1/repos/ensure-org`,
      { org },
      z.object({ ok: z.boolean(), org: z.string() }),
    ),
  ensureRepo: (org: string, repo: string) =>
    post(
      `/api/v1/repos/ensure`,
      { org, repo },
      z.object({ ok: z.boolean(), session_id: z.string() }),
    ),
  cloneRepo: (
    org: string,
    repo: string,
    git_url: string,
    token?: string,
    rev?: string,
  ) =>
    post(
      `/api/v1/repos/clone`,
      { org, repo, git_url, token, rev },
      z.object({ ok: z.boolean(), session_id: z.string() }),
    ),
  diffChange: (org: string, repo: string, changeId: string) =>
    get(
      `/api/v1/repos/${org}/${repo}/diff/${changeId}`,
      z.object({ files: z.array(DiffFileSchema).optional() }),
    ).map(r => r.files || []),
  fileAtChange: (
    org: string,
    repo: string,
    changeId: string,
    filePath: string,
  ) =>
    get(
      `/api/v1/repos/${org}/${repo}/file/${changeId}${qs({ path: filePath })}`,
      z.object({ content: z.string() }),
    ).map(r => r.content),
  fileLog: (
    org: string,
    repo: string,
    filePath: string,
    bookmark?: string,
    limit?: number,
  ) => {
    const query: Record<string, string | number> = { path: filePath }
    if (bookmark) query.bookmark = bookmark
    if (limit) query.limit = limit
    return get(
      `/api/v1/repos/${org}/${repo}/file-log${qs(query)}`,
      z.object({
        commits: z.array(
          z.object({
            change_id: z.string(),
            commit_id: z.string(),
            author: z.string(),
            timestamp: z.string(),
            message: z.string(),
          }),
        ),
      }),
    ).map(r => r.commits)
  },
  fileDiff: (org: string, repo: string, changeId: string, filePath: string) =>
    get(
      `/api/v1/repos/${org}/${repo}/file-diff/${changeId}${qs({ path: filePath })}`,
      z.object({ diff: z.string() }),
    ).map(r => r.diff),
  log: (
    org: string,
    repo: string,
    opts?: { rev?: string; limit?: number },
  ) => {
    const query: Record<string, string | number> = {}
    if (opts?.rev) query.rev = opts.rev
    if (opts?.limit) query.limit = opts.limit
    const q = Object.keys(query).length ? `?${new URLSearchParams(query as Record<string, string>)}` : ''
    return get(
      `/api/v1/repos/${org}/${repo}/log${q}`,
      z.object({
        commits: z.array(
          z.object({
            change_id: z.string(),
            commit_id: z.string(),
            author: z.string(),
            timestamp: z.string(),
            message: z.string(),
          }),
        ),
      }),
    ).map(r => r.commits)
  },
  tags: (org: string, repo: string) =>
    get(
      `/api/v1/repos/${org}/${repo}/tags`,
      z.object({
        tags: z.array(z.object({ name: z.string(), target: z.string() })),
      }),
    ).map(r => r.tags),
  blame: (org: string, repo: string, rev: string, filePath: string) =>
    get(
      `/api/v1/git-blame/${org}/${repo}${qs({ rev, path: filePath })}`,
      z.object({ blame: z.array(z.string()) }),
    ).map(r => r.blame),
  graph: (org: string, repo: string, limit?: number) => {
    const q = limit ? `?limit=${limit}` : ''
    return get(
      `/api/v1/repos/${org}/${repo}/graph${q}`,
      z.object({
        graph: z.array(
          z.object({
            commit_id: z.string(),
            change_id: z.string(),
            message: z.string(),
            author: z.string(),
            parents: z.array(z.string()),
            edge_types: z.array(z.string()),
            is_head: z.boolean(),
          }),
        ),
      }),
    ).map(r => r.graph)
  },
  mirrors: () =>
    get(
      `/api/v1/repos/mirrors`,
      z.object({
        mirrors: z.array(
          z.object({
            org: z.string(),
            repo: z.string(),
            mirror_url: z.string(),
            has_secret: z.boolean(),
          }),
        ),
        bookmark_mirrors: z.array(
          z.object({
            org: z.string(),
            repo: z.string(),
            bookmark: z.string(),
            mirror_url: z.string(),
            remote_bookmark: z.string(),
            has_secret: z.boolean(),
          }),
        ),
      }),
    ),
}

export const config = {
  get: () =>
    get(`/api/v1/config`, z.record(z.string(), z.string())),
  set: (entries: Record<string, string>) =>
    put(`/api/v1/config`, entries),
}

export const providers = {
  list: () =>
    get(
      `/api/v1/providers`,
      z.object({ providers: ProvidersMapSchema }),
    ).map(r => r.providers),
  register: (p: Omit<ProviderInfo, 'api_key'> & { api_key: string }) =>
    post(`/api/v1/providers`, p, OkSchema),
  delete: (pid: string) => del(`/api/v1/providers/${pid}`),
  test: (params: { api_type: string; base_url: string; api_key: string }) =>
    post(
      `/api/v1/providers/test`,
      params,
      z.object({
        ok: z.boolean(),
        models: z.array(z.string()).optional(),
        detail: z.string().optional(),
        error: z.string().optional(),
      }),
    ),
}

export const models = {
  list: () =>
    get(
      `/api/v1/models`,
      z.object({
        models: z.array(
          ModelInfoSchema.extend({ provider_id: z.string().optional() }),
        ),
      }),
    ).map(r =>
      r.models.map(m => ({ ...m, provider_id: m.provider_id ?? '' })),
    ),
}
