import { z } from 'zod'

export const SessionSchema = z.object({
  id: z.string(),
  org: z.string(),
  repo: z.string(),
  bookmark: z.string(),
  model: z.string(),
  preset: z.string(),
  tip_id: z.string().nullable(),
  max_turns: z.number().nullable(),
  system_prompt: z.string().nullable(),
  base_image: z.string().nullable(),
  unread: z.number().optional(),
  input_tokens: z.number().nullable(),
  output_tokens: z.number().nullable(),
  total_tokens: z.number().nullable(),
  last_input_tokens: z.number().optional(),
  last_output_tokens: z.number().optional(),
  created_at: z.string(),
  updated_at: z.string(),
  // Chat-list extras merged by the platform aggregate from the bus
  // (agent message-fact projection + platform read watermark). Absent when
  // the bus is unavailable or the session has no messages yet.
  last_message_at: z.string().optional(),
  last_message_preview: z.string().optional(),
  unread_count: z.number().optional(),
})
export type Session = z.infer<typeof SessionSchema>

export const SessionInfoSchema = z.object({
  session_id: z.string(),
  bookmark: z.string(),
  message_count: z.number(),
  unread: z.number().optional(),
  model: z.string(),
  preset: z.string(),
})
export type SessionInfo = z.infer<typeof SessionInfoSchema>

export const BookmarkNodeSchema = z.object({
  bookmark: z.string(),
  session: SessionInfoSchema.nullable(),
})
export type BookmarkNode = z.infer<typeof BookmarkNodeSchema>

export const RepoNodeSchema = z.object({
  repo: z.string(),
  bookmarks: z.array(BookmarkNodeSchema),
})
export type RepoNode = z.infer<typeof RepoNodeSchema>

export const OrgNodeSchema = z.object({
  org: z.string(),
  repos: z.array(RepoNodeSchema),
})
export type OrgNode = z.infer<typeof OrgNodeSchema>
