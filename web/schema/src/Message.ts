import { z } from 'zod'

export const FlatMessageSchema = z.object({
  id: z.string().optional(),
  session_id: z.string().optional(),
  role: z.string(),
  content: z.string(),
  tool_name: z.string().optional(),
  tool_call_id: z.string().optional(),
  created_at: z.string().optional(),
})
export type FlatMessage = z.infer<typeof FlatMessageSchema>

export const ToolStateSchema = z.object({
  status: z.enum(['pending', 'running', 'complete', 'error']).optional(),
  title: z.string().optional(),
  error: z.string().optional(),
  input: z.record(z.string(), z.unknown()).optional(),
  output: z.string().optional(),
  change_id: z.string().optional(),
  diff: z.string().optional(),
  additions: z.number().optional(),
  deletions: z.number().optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
})
export type ToolState = z.infer<typeof ToolStateSchema>

export const MessagePartSchema = z.object({
  id: z.string().optional(),
  type: z.string(),
  text: z.string().optional(),
  tool: z.string().optional(),
  tool_call_id: z.string().optional(),
  state: ToolStateSchema.optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
})
export type MessagePart = z.infer<typeof MessagePartSchema>

export const MessageSchema = z.object({
  id: z.string(),
  role: z.enum(['user', 'assistant', 'tool', 'system', 'error']),
  parts: z.array(MessagePartSchema),
  created_at: z.string().optional(),
})
export type Message = z.infer<typeof MessageSchema>

export const MailboxEntrySchema = z.object({
  id: z.string(),
  msg_type: z.string(),
  payload: z.string(),
  effective_at: z.string().optional(),
  status: z.string(),
  created_at: z.string(),
  consumed_at: z.string().nullable(),
  seq: z.number().optional(),
})
export type MailboxEntry = z.infer<typeof MailboxEntrySchema>

export const ChangeEntrySchema = z.object({
  change_id: z.string(),
  commit_id: z.string(),
  author: z.string(),
  timestamp: z.string(),
  message: z.string(),
})
export type ChangeEntry = z.infer<typeof ChangeEntrySchema>
