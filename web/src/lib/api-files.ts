import { ResultAsync } from "neverthrow";

/**
 * File storage API (agent-backed, proxied by the platform at /api/v1/files).
 * The client uploads a file to obtain its `code`, then sends only the code(s)
 * to the prompt endpoint — the platform splices them into `[附件 …file:<code>…]`
 * references.
 */

const origin =
	typeof window !== "undefined" ? window.location.origin : "http://localhost";

export type UploadedFile = {
	code: string;
	name: string;
	mime: string;
	size: number;
};

/** Upload a single file (multipart). Resolves to the stored file record. */
export function uploadFile(file: File): ResultAsync<UploadedFile, string> {
	return ResultAsync.fromPromise(
		(async () => {
			const fd = new FormData();
			fd.append("file", file);
			const r = await fetch(`${origin}/api/v1/files`, {
				method: "POST",
				body: fd,
			});
			if (!r.ok) throw new Error(`upload failed (HTTP ${r.status})`);
			return (await r.json()) as UploadedFile;
		})(),
		(e) => (e instanceof Error ? e.message : String(e)),
	);
}

/** Absolute URL to stream/download a stored file's bytes. */
export function fileUrl(code: string): string {
	return `${origin}/api/v1/files/${encodeURIComponent(code)}`;
}
