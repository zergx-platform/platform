import { z } from 'zod'
import { get, put } from './api-core'
import { origin } from './client'

const K8sConfigSchema = z.object({
  url: z.string(),
  ca_file: z.string().nullable(),
  namespace: z.string(),
  worker_image: z.string(),
  token_set: z.boolean(),
})

const PodmanStatusSchema = z.object({
  connected: z.boolean(),
  version: z.unknown(),
})

export const infra = {
  podmanStatus: () =>
    get(
      `${origin}/api/v1/podman/status`,
      z.object({ ok: z.boolean(), data: PodmanStatusSchema }),
    ).map(r => r.data),
  k8sConfig: () =>
    get(
      `${origin}/api/v1/k8s/config`,
      z.object({ ok: z.boolean(), data: K8sConfigSchema }),
    ).map(r => r.data),
  updateK8sConfig: (cfg: {
    url: string
    token?: string
    ca_file?: string
    namespace: string
    worker_image: string
  }) => put(`${origin}/api/v1/k8s/config`, cfg, z.object({ ok: z.boolean() })),
  updatePodmanConfig: (url: string) =>
    put(
      `${origin}/api/v1/podman/config`,
      { url },
      z.object({ ok: z.boolean() }),
    ),
}
