<script lang="ts">
import { marked } from "marked";
import { fileUrl } from "$lib/api-files";
import { cn } from "$lib/utils";

let { content, class: cls }: { content: string; class?: string } = $props();

// The platform splices `[附件 <name> | file:<code> | <mime> | <size>]` into the
// prompt text. Render it as a clickable download link before markdown so the
// user can preview/download the referenced file.
const FILE_REF =
	/\[附件\s+([^|\]]+)\s*\|\s*file:([0-9a-zA-Z]+)\s*\|\s*([^|\]]*)\s*\|\s*([^\]]*)\]/g;

let html = $derived.by(() => {
	let text = content;
	text = text.replace(
		FILE_REF,
		(_m, name, code, mime, size) =>
			`[📎 ${name || code} (${mime ?? ""})](${fileUrl(code)})`,
	);
	try {
		const result: string = marked.parse(text, { async: false });
		return result;
	} catch {
		return text;
	}
});
</script>

<div class={cn("prose dark:prose-invert prose-sm max-w-none [&_pre]:bg-card [&_pre]:border [&_pre]:border-border [&_code]:bg-muted [&_code]:rounded [&_code]:px-1 [&_code]:py-0.5", cls)}>
    {@html html}
</div>
