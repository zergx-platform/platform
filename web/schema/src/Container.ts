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

export const SandboxSchema = z.object({
  container_id: z.string(),
  session: z.string(),
  pod_name: z.string(),
  status: z.string(),
  worker_url: z.string(),
  pod_ip: z.string(),
  synced_rev: z.string(),
})
export type Sandbox = z.infer<typeof SandboxSchema>

export const DeploymentSchema = z.object({
  name: z.string(),
  image: z.string(),
  replicas: z.number(),
  ready: z.number(),
  namespace: z.string(),
  age: z.string(),
  ports: z.array(z.number()),
  session: z.string().optional(),
})
export type Deployment = z.infer<typeof DeploymentSchema>

export const DeploymentPodSchema = z.object({
  name: z.string(),
  ip: z.string(),
  phase: z.string(),
  ready: z.boolean(),
  image: z.string(),
  age: z.string(),
  restarts: z.number(),
})
export type DeploymentPod = z.infer<typeof DeploymentPodSchema>

export const ContainerfileTemplateSchema = z.object({
  name: z.string(),
  content: z.string(),
})
export type ContainerfileTemplate = z.infer<typeof ContainerfileTemplateSchema>

export const ImageBuildResultSchema = z.object({
  ok: z.boolean(),
  image_id: z.string().optional(),
  image: z.string().optional(),
  pushed: z.boolean().optional(),
})
export type ImageBuildResult = z.infer<typeof ImageBuildResultSchema>

export const OpsDependencySchema = z.object({
  name: z.string(),
  ok: z.boolean(),
  status: z.number().optional(),
  error: z.string().optional(),
})
export type OpsDependency = z.infer<typeof OpsDependencySchema>

export const OpsStatusSchema = z.object({
  ok: z.boolean(),
  version: z.string(),
  deps: z.array(OpsDependencySchema),
  sandboxes: z.number(),
})
export type OpsStatus = z.infer<typeof OpsStatusSchema>

export const PublishSpecSchema = z.object({
  protocol: z.string(),
  args: z.array(z.string()).nullable(),
  required: z.array(z.string()).nullable(),
})
export type PublishSpec = z.infer<typeof PublishSpecSchema>

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
