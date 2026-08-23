import { z } from 'zod'

export const ConfigEntrySchema = z.object({
  key: z.string(),
  value: z.string(),
})
export type ConfigEntry = z.infer<typeof ConfigEntrySchema>

export const ProviderModelSchema = z.object({
  id: z.string(),
  name: z.string(),
  context_limit: z.number().optional(),
  output_limit: z.number().optional(),
  max_tokens: z.number().optional(),
  temperature: z.number().optional(),
  reasoning: z.boolean().optional(),
  tool_call: z.boolean().optional(),
})
export type ProviderModel = z.infer<typeof ProviderModelSchema>

export const ProviderInfoSchema = z.object({
  provider_id: z.string(),
  api_type: z.string(),
  base_url: z.string(),
  api_key: z.string(),
  headers: z.record(z.string(), z.string()).optional(),
  models: z.array(ProviderModelSchema),
})
export type ProviderInfo = z.infer<typeof ProviderInfoSchema>

export const ProvidersMapSchema = z.record(z.string(), ProviderInfoSchema)
export type ProvidersMap = z.infer<typeof ProvidersMapSchema>

export const ToolConfigMapSchema = z.record(
  z.string(),
  z.record(z.string(), z.unknown()),
)
export type ToolConfigMap = z.infer<typeof ToolConfigMapSchema>

export const ModelInfoSchema = z.object({
  id: z.string(),
  name: z.string(),
  provider_id: z.string(),
  context_limit: z.number().optional(),
  output_limit: z.number().optional(),
  max_tokens: z.number().optional(),
  temperature: z.number().optional(),
  reasoning: z.boolean().optional(),
  tool_call: z.boolean().optional(),
})
export type ModelInfo = z.infer<typeof ModelInfoSchema>

export const DiffFileSchema = z.object({
  path: z.string(),
  diff_text: z.string().optional(),
})
export type DiffFile = z.infer<typeof DiffFileSchema>

export const FileEntrySchema = z.object({
  name: z.string(),
  path: z.string(),
  is_dir: z.boolean(),
  size: z.number(),
})
export type FileEntry = z.infer<typeof FileEntrySchema>

export const PresetInfoSchema = z.object({
  id: z.string(),
  system_prompt: z.string(),
  tools: z.array(z.string()),
  max_turns: z.number(),
})
export type PresetInfo = z.infer<typeof PresetInfoSchema>

export const ToolConfigFieldSchema = z.object({
  key: z.string(),
  label: z.string(),
  type: z.enum(['select-provider', 'select-model', 'text', 'number']),
  placeholder: z.string().optional(),
  dependsOnProvider: z.string().optional(),
})
export type ToolConfigField = z.infer<typeof ToolConfigFieldSchema>

export const ToolInfoSchema = z.object({
  name: z.string(),
  description: z.string(),
  category: z.string(),
  parameters: z.record(z.string(), z.unknown()).nullable(),
  configFields: z.array(ToolConfigFieldSchema).nullable(),
})
export type ToolInfo = z.infer<typeof ToolInfoSchema>

export const SessionRowSchema = z.object({
  id: z.string(),
  org: z.string(),
  repo: z.string(),
  branch: z.string(),
  model: z.string(),
  preset: z.string(),
  tipId: z.string().nullable(),
  parentId: z.string().nullable(),
  forkAtMsgId: z.string().nullable(),
  workerUrl: z.string().nullable(),
  containerId: z.string().nullable(),
  maxTurns: z.number().nullable(),
  systemPrompt: z.string().nullable(),
  revert: z.string().nullable(),
  redoTipId: z.string().nullable(),
  inputTokens: z.number().nullable(),
  outputTokens: z.number().nullable(),
  totalTokens: z.number().nullable(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type SessionRow = z.infer<typeof SessionRowSchema>

export const PresetRowSchema = z.object({
  id: z.string(),
  systemPrompt: z.string(),
  tools: z.string(),
  maxTurns: z.number(),
})
export type PresetRow = z.infer<typeof PresetRowSchema>

export const MessageRowSchema = z.object({
  id: z.string(),
  sessionId: z.string(),
  role: z.string(),
  content: z.string(),
  partsJson: z.string(),
  prevId: z.string().nullable(),
  toolName: z.string(),
  toolCallId: z.string(),
  createdAt: z.string(),
})
export type MessageRow = z.infer<typeof MessageRowSchema>

export const MailboxRowSchema = z.object({
  id: z.string(),
  sessionId: z.string(),
  msgType: z.string(),
  payload: z.string(),
  effectiveAt: z.string().nullable(),
  status: z.string(),
  createdAt: z.string(),
  consumedAt: z.string().nullable(),
  seq: z.number().nullable(),
})
export type MailboxRow = z.infer<typeof MailboxRowSchema>

export const ConfigRowSchema = z.object({
  key: z.string(),
  value: z.string(),
})
export type ConfigRow = z.infer<typeof ConfigRowSchema>

export const ContainerRowSchema = z.object({
  id: z.string(),
  name: z.string(),
  image: z.string().nullable(),
  workerUrl: z.string().nullable(),
  containerId: z.string().nullable(),
  sessionId: z.string().nullable(),
  org: z.string().nullable(),
  repo: z.string().nullable(),
  branch: z.string().nullable(),
  status: z.string(),
  createdAt: z.string(),
})
export type ContainerRow = z.infer<typeof ContainerRowSchema>

export const PromptBodySchema = z.object({
  prompt: z
    .string()
    .refine(v => v.trim().length > 0, 'prompt must not be empty'),
})
export type PromptBody = z.infer<typeof PromptBodySchema>

export const ModelBodySchema = z.object({ model: z.string() })
export type ModelBody = z.infer<typeof ModelBodySchema>

export const SessionSettingsBodySchema = z.object({
  model: z.string().optional(),
  preset: z.string().optional(),
  max_turns: z.number().optional(),
  system_prompt: z.string().optional(),
  base_image: z.string().optional(),
})
export type SessionSettingsBody = z.infer<typeof SessionSettingsBodySchema>

export const SessionForkBodySchema = z.object({ branch: z.string().optional() })
export type SessionForkBody = z.infer<typeof SessionForkBodySchema>

export const RepoForkBodySchema = z.object({
  source_org: z.string(),
  source_repo: z.string(),
  source_branch: z.string(),
  target_org: z.string(),
  target_repo: z.string(),
  target_branch: z.string().optional(),
})
export type RepoForkBody = z.infer<typeof RepoForkBodySchema>

export const RepoEnsureBodySchema = z.object({
  org: z.string(),
  repo: z.string().optional(),
  branch: z.string().optional(),
})
export type RepoEnsureBody = z.infer<typeof RepoEnsureBodySchema>

export const CloneBodySchema = z.object({
  org: z.string(),
  repo: z.string(),
  git_url: z.string(),
  token: z.string().optional(),
  rev: z.string().optional(),
})
export type CloneBody = z.infer<typeof CloneBodySchema>

export const ContainerCreateBodySchema = z.object({
  image: z.string().optional(),
  session_id: z.string().optional(),
  org: z.string().optional(),
  repo: z.string().optional(),
  branch: z.string().optional(),
})
export type ContainerCreateBody = z.infer<typeof ContainerCreateBodySchema>

export const ContainerExecBodySchema = z.object({ command: z.string() })
export type ContainerExecBody = z.infer<typeof ContainerExecBodySchema>

export const ProviderTestBodySchema = z.object({
  base_url: z.string(),
  api_key: z.string(),
})
export type ProviderTestBody = z.infer<typeof ProviderTestBodySchema>

export const PresetBodySchema = z.object({
  id: z.string(),
  system_prompt: z.string().optional(),
  tools: z.array(z.string()).optional(),
  max_turns: z.number().optional(),
})
export type PresetBody = z.infer<typeof PresetBodySchema>

export const OpenAIUsageSchema = z.object({
  prompt_tokens: z.number().optional(),
  completion_tokens: z.number().optional(),
  total_tokens: z.number().optional(),
})
export type OpenAIUsage = z.infer<typeof OpenAIUsageSchema>

export const OpenAIDeltaSchema = z.object({
  content: z.string().optional(),
  reasoning_content: z.string().optional(),
  tool_calls: z
    .array(
      z.object({
        index: z.number().optional(),
        id: z.string().optional(),
        function: z
          .object({
            name: z.string().optional(),
            arguments: z.string().optional(),
          })
          .optional(),
      }),
    )
    .optional(),
})
export type OpenAIDelta = z.infer<typeof OpenAIDeltaSchema>

export const OpenAIChoiceSchema = z.object({
  delta: OpenAIDeltaSchema.optional(),
  finish_reason: z.string().optional(),
})
export type OpenAIChoice = z.infer<typeof OpenAIChoiceSchema>

export const OpenAISSEChunkSchema = z.object({
  choices: z.array(OpenAIChoiceSchema).optional(),
  usage: OpenAIUsageSchema.optional(),
})
export type OpenAISSEChunk = z.infer<typeof OpenAISSEChunkSchema>

export const DockerInspectSchema = z.object({
  NetworkSettings: z
    .object({
      Ports: z
        .record(
          z.string(),
          z.array(z.object({ HostPort: z.string().optional() })),
        )
        .optional(),
    })
    .optional(),
})
export type DockerInspectRaw = z.infer<typeof DockerInspectSchema>

export const GitCommitSchema = z.object({
  change_id: z.string(),
  author: z.string(),
  timestamp: z.string(),
  bookmarks: z.string(),
  commit_id: z.string(),
  message: z.string(),
})
export type GitCommit = z.infer<typeof GitCommitSchema>

export const RepoBookmarkNodeSchema = z.object({
  session_id: z.string(),
  branch: z.string(),
  message_count: z.number(),
  unread: z.number().optional(),
  model: z.string(),
  preset: z.string(),
  parent_id: z.string().nullable(),
})
export type RepoBookmarkNode = z.infer<typeof RepoBookmarkNodeSchema>

export const SessionUpdatesSchema = z.object({
  model: z.string().optional(),
  preset: z.string().optional(),
  maxTurns: z.number().optional(),
  systemPrompt: z.string().optional(),
})
export type SessionUpdates = z.infer<typeof SessionUpdatesSchema>

export const MdProviderModelDefSchema = z
  .object({
    name: z.string().optional(),
    limit: z
      .object({
        context: z.number().optional(),
        output: z.number().optional(),
      })
      .optional(),
    reasoning: z.boolean().optional(),
    tool_call: z.boolean().optional(),
  })
  .passthrough()
export const MdProviderSchema = z
  .object({
    id: z.string().optional(),
    name: z.string().optional(),
    npm: z.string().optional(),
    api: z.string().optional(),
    models: z.record(z.string(), MdProviderModelDefSchema).optional(),
  })
  .passthrough()
export const MdProvidersSchema = z.record(z.string(), MdProviderSchema)
export type MdProvider = z.infer<typeof MdProviderSchema>
