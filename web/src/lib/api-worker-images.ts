import { z } from 'zod'
import { get, post } from './api-core'
import { origin } from './client'

const WorkerImageSchema = z.object({ tag: z.string(), image: z.string() })

export const workerImages = {
  list: () =>
    get(
      `${origin}/api/v1/worker-image/list`,
      z.object({ ok: z.boolean(), images: z.array(WorkerImageSchema) }),
    ).map(r => r.images),
  build: (base_image: string) =>
    post(`${origin}/api/v1/worker-image/build`, { base_image }),
}
