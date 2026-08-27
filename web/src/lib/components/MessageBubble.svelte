<script lang="ts">
import { Check, Copy, Undo2 } from '@lucide/svelte'
import type { ChatMessage } from '$lib/hooks/useMessages.svelte'
import MarkdownRenderer from './Markdown.svelte'
import ToolPartView from './ToolPartView.svelte'

let {
  msg,
  undo,
  onOpenChange,
}: {
  msg: ChatMessage
  undo: (messageId: string) => void
  onOpenChange?: (changeId: string) => void
} = $props()
let copied = $state(false)

let isUser = $derived(msg.role === 'user')
let isError = $derived(msg.role === 'error')
let isStreaming = $derived(msg.status === 'streaming')

function getText() {
  return msg.parts
    .filter(p => p.type === 'text' || p.type === 'reasoning')
    .map(p => p.text ?? '')
    .join('\n')
}

function handleCopy() {
  try {
    navigator.clipboard.writeText(getText())
  } catch {}
  copied = true
  setTimeout(() => (copied = false), 1500)
}
</script>

<div class="flex flex-col mb-3 {isUser ? 'items-end' : 'items-start'}" data-msg-id={msg.id}>
    <div
        class="max-w-[90%] sm:max-w-[80%] rounded-lg px-3 sm:px-4 py-2.5 text-sm space-y-1.5 relative select-text overflow-hidden min-w-0 {isError ? 'border border-destructive/40 bg-destructive/10' : isUser ? 'bg-primary/10 border' : isStreaming ? 'bg-muted border' : 'bg-card border border-border'}"
    >
        {#if isError}
            <div class="text-xs font-medium text-destructive mb-1">Error</div>
        {/if}

        {#if isStreaming && msg.parts.length === 0}
            <div class="text-xs text-muted-foreground italic">thinking...</div>
        {:else}
            {#each msg.parts as part (part.id)}
                {#if part.type === "text"}
                    <MarkdownRenderer content={part.text ?? ""} />
                {:else if part.type === "reasoning"}
                    <details open class="text-xs border-l-2 border-amber-400/50 pl-3 py-1 my-1 -ml-1 bg-amber-500/5 rounded-r">
                        <summary class="text-amber-400 cursor-pointer font-medium">Thinking{#if isStreaming}...{/if}</summary>
                        <MarkdownRenderer content={part.text ?? ""} class="mt-1.5 text-muted-foreground" />
                    </details>
                {:else if part.type === "tool" && part.state}
                    <ToolPartView {part} {isStreaming} {onOpenChange} />
                {:else if part.type === "compaction"}
                    <details class="rounded-lg border bg-muted/40 px-3 py-2 text-xs">
                        <summary class="cursor-pointer select-none text-muted-foreground">
                            <span class="inline-flex items-center gap-1.5">历史已压缩 · 查看摘要</span>
                        </summary>
                        <pre class="mt-2 whitespace-pre-wrap font-sans text-foreground">{part.text ?? ""}</pre>
                    </details>
                {/if}
            {/each}
        {/if}
    </div>

    {#if !isStreaming}
        <div class="flex items-center gap-1 mt-1">
            <button
                onclick={handleCopy}
                class="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent"
                title="Copy message"
            >
                {#if copied}<Check class="size-3.5 text-green-500" />{:else}<Copy class="size-3.5" />{/if}
            </button>
            <button
                onclick={() => undo(msg.id)}
                class="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent"
                title="Undo"
            >
                <Undo2 class="size-3.5" />
            </button>
        </div>
    {/if}
</div>
