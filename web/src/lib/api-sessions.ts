import type {
	FlatMessage,
	Message,
	PresetInfo,
	ToolConfigMap,
} from "@zergx/schema";
import {
	ChangeEntrySchema,
	MailboxEntrySchema,
	MessageSchema,
	PresetInfoSchema,
	SessionSchema,
	ToolConfigMapSchema,
	ToolInfoSchema,
} from "@zergx/schema";
import { z } from "zod";
import { del, get, OkSchema, patch, post, put } from "./api-core";
import { qs } from "./client";
import { uid } from "./utils";

const TodoSchema = z.object({
	id: z.string(),
	sessionId: z.string(),
	content: z.string(),
	status: z.string(),
	priority: z.string(),
	position: z.number(),
	createdAt: z.string(),
});
export type Todo = z.infer<typeof TodoSchema>;

export const presets = {
	list: () => get(`/api/v1/presets`, z.array(PresetInfoSchema)),
	save: (p: PresetInfo) => post(`/api/v1/presets`, p, OkSchema),
	delete: (id: string) => del(`/api/v1/presets/${id}`, OkSchema),
};

export const tools = {
	list: () =>
		get(`/api/v1/tools`, z.object({ tools: z.array(ToolInfoSchema) })).map(
			(r) => r.tools,
		),
	getConfig: () => get(`/api/v1/tool-config`, ToolConfigMapSchema),
	// Set a single extension config knob by id (e.g. memory/vlm_model).
	// The agent validates against the extension's declared config and delivers
	// the value to the live extension, so tools pick it up immediately.
	setConfigValue: (extId: string, name: string, value: unknown) =>
		put(
			`/api/v1/tool-config/${encodeURIComponent(extId)}/${encodeURIComponent(name)}`,
			{ value },
			z.object({ ok: z.boolean() }),
		),
};

export const sessions = {
	list: () =>
		get(`/api/v1/sessions`, z.object({ sessions: z.array(SessionSchema) })).map(
			(r) => r.sessions,
		),
	create: (params: {
		org?: string;
		repo?: string;
		branch?: string;
		model?: string;
		preset?: string;
	}) =>
		post(`/api/v1/sessions`, params, z.object({ session: SessionSchema })).map(
			(r) => r.session,
		),
	get: (id: string) =>
		get(
			`/api/v1/sessions/${encodeURIComponent(id)}`,
			z.object({ session: SessionSchema }),
		).map((r) => r.session),
	delete: (id: string) => del(`/api/v1/sessions/${encodeURIComponent(id)}`),
	prompt: (id: string, prompt: string, attachments?: string[]) =>
		post(
			`/api/v1/sessions/${encodeURIComponent(id)}/prompt`,
			{
				prompt,
				attachments: (attachments ?? []).map((code) => ({ code })),
			},
			z.object({ ok: z.boolean(), messageId: z.string() }),
		),
	messages: (id: string, opts?: { before?: string; limit?: number }) => {
		const limit = opts?.limit ?? 30;
		const query: Record<string, string | number> = { limit };
		if (opts?.before) query.before = opts.before;
		return get(
			`/api/v1/sessions/${encodeURIComponent(id)}/messages${qs(query)}`,
			z.object({ messages: z.array(MessageSchema) }),
		).map((r) => ({
			messages: r.messages,
			hasMore: r.messages.length >= limit,
		}));
	},
	switchModel: (id: string, model: string) =>
		post(
			`/api/v1/sessions/${encodeURIComponent(id)}/model`,
			{ model },
			z.object({ model: z.string() }),
		),
	interrupt: (id: string) =>
		post(
			`/api/v1/sessions/${encodeURIComponent(id)}/interrupt`,
			undefined,
			z.object({ interrupted: z.boolean() }),
		),
	compact: (id: string) =>
		post(
			`/api/v1/sessions/${encodeURIComponent(id)}/compact`,
			undefined,
			z.object({ ok: z.boolean() }),
		),
	fork: (id: string, branch: string) =>
		post(
			`/api/v1/sessions/${encodeURIComponent(id)}/fork`,
			{ branch },
			z.object({ session: SessionSchema }),
		).map((r) => r.session),
	revert: (id: string, messageId?: string) =>
		post(`/api/v1/sessions/${encodeURIComponent(id)}/undo`, {
			message_id: messageId,
		}),
	markRead: (id: string) =>
		post(
			`/api/v1/sessions/${encodeURIComponent(id)}/read`,
			undefined,
			z.object({ ok: z.boolean() }),
		),
	state: (id: string) =>
		get(
			`/api/v1/sessions/${encodeURIComponent(id)}/state`,
			z.object({ status: z.string(), parts: z.array(z.unknown()).optional() }),
		).map((r) => ({ status: r.status, parts: r.parts ?? [] })),
	mailbox: (id: string) =>
		get(
			`/api/v1/sessions/${encodeURIComponent(id)}/mailbox`,
			z.object({ entries: z.array(MailboxEntrySchema) }),
		).map((r) => r.entries),
	changes: (id: string) =>
		get(
			`/api/v1/sessions/${encodeURIComponent(id)}/changes`,
			z.object({ changes: z.array(ChangeEntrySchema) }),
		).map((r) => r.changes),
	todos: (id: string) =>
		get(
			`/api/v1/sessions/${encodeURIComponent(id)}/todos`,
			z.object({ todos: z.array(TodoSchema) }),
		).map((r) => r.todos),
	settings: (
		id: string,
		settings: {
			model?: string;
			preset?: string;
			max_turns?: number | null;
			system_prompt?: string | null;
			base_image?: string | null;
		},
	) => {
		const body: Record<string, unknown> = { ...settings };
		if (body.max_turns === null) body.max_turns = undefined;
		if (body.system_prompt === null) body.system_prompt = undefined;
		if (body.base_image === null) body.base_image = undefined;
		return patch(
			`/api/v1/sessions/${encodeURIComponent(id)}/settings`,
			body,
			z.object({ session: SessionSchema }),
		).map((r) => r.session);
	},
};

export function flatToMessage(m: FlatMessage): Message {
	const id = m.id || m.created_at || uid();
	if (m.role === "compaction") {
		return {
			id,
			role: "compaction" as const,
			parts: [{ type: "compaction", text: m.content }],
		};
	}
	if (m.role === "tool" || m.tool_name) {
		return {
			id,
			role: "tool" as const,
			parts: [
				{
					type: "tool",
					tool: m.tool_name || "tool",
					state: { status: "complete", output: m.content },
				},
			],
		};
	}
	return {
		id,
		role: (MessageSchema.shape.role.safeParse(m.role).success
			? m.role
			: "assistant") as "assistant" | "user" | "system" | "tool" | "error",
		parts: [{ type: "text", text: m.content }],
	};
}
