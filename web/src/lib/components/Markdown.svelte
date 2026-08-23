<script lang="ts">
import { marked } from 'marked'
import { cn } from '$lib/utils'

let { content, class: cls }: { content: string; class?: string } = $props()

let html = $derived.by(() => {
  try {
    const result: string = marked.parse(content, { async: false })
    return result
  } catch {
    return content
  }
})
</script>

<div class={cn("prose dark:prose-invert prose-sm max-w-none [&_pre]:bg-card [&_pre]:border [&_pre]:border-border [&_code]:bg-muted [&_code]:rounded [&_code]:px-1 [&_code]:py-0.5", cls)}>
    {@html html}
</div>
