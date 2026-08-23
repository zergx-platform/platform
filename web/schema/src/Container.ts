import { z } from 'zod'

export const ContainerInfoSchema = z.object({
  id: z.string(),
  name: z.string(),
  image: z.string().nullable(),
  worker_url: z.string().nullable(),
  container_id: z.string().nullable(),
  session_id: z.string().nullable(),
  org: z.string().nullable(),
  repo: z.string().nullable(),
  branch: z.string().nullable(),
  status: z.string(),
  created_at: z.string().nullable(),
  kind: z.enum(['worker', 'deploy']).default('worker'),
  service_url: z.string().nullable().default(null),
})
export type ContainerInfo = z.infer<typeof ContainerInfoSchema>

export const JobInfoSchema = z.object({
  id: z.string(),
  command: z.string(),
  state: z.string(),
  exit_code: z.number(),
  started_at: z.number().nullable().optional(),
  finished_at: z.number().nullable().optional(),
  stdout: z.string().optional(),
})
export type JobInfo = z.infer<typeof JobInfoSchema>

export const ExecResultSchema = z.object({
  exit_code: z.number().optional(),
  output: z.string().optional(),
  job_id: z.string().optional(),
  backgrounded: z.boolean().optional(),
  note: z.string().optional(),
  error: z.string().optional(),
})
export type ExecResult = z.infer<typeof ExecResultSchema>

export const WorkerEventSchema = z.object({
  event: z.string(),
  params: z.record(z.string(), z.unknown()),
})
export type WorkerEvent = z.infer<typeof WorkerEventSchema>
