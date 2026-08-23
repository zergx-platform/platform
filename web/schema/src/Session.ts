import { z } from 'zod'

export const SessionSchema = z.object({
  id: z.string(),
  org: z.string(),
  repo: z.string(),
  branch: z.string(),
  model: z.string(),
  preset: z.string(),
  parent_id: z.string().nullable(),
  tip_id: z.string().nullable(),
  fork_at_msg_id: z.string().nullable(),
  worker_url: z.string().nullable(),
  container_id: z.string().nullable(),
  max_turns: z.number().nullable(),
  system_prompt: z.string().nullable(),
  base_image: z.string().nullable(),
  unread: z.number().optional(),
  input_tokens: z.number().nullable(),
  output_tokens: z.number().nullable(),
  total_tokens: z.number().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
})
export type Session = z.infer<typeof SessionSchema>

export const SessionInfoSchema = z.object({
  session_id: z.string(),
  branch: z.string(),
  message_count: z.number(),
  unread: z.number().optional(),
  model: z.string(),
  preset: z.string(),
  parent_id: z.string().nullable(),
})
export type SessionInfo = z.infer<typeof SessionInfoSchema>

export const BookmarkNodeSchema = z.object({
  branch: z.string(),
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
