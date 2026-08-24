import { z } from 'zod'
import { get, put } from './api-core'
import { origin } from './client'

const K8sConfigSchema = z.object({
  ok: z.boolean(),
  namespace: z.string(),
  worker_image: z.string(),
})

export const infra = {
  k8sConfig: () =>
    get(
      `${origin}/api/v1/infra/k8s/config`,
      K8sConfigSchema,
    ),
}
