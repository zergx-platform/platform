<script lang="ts">
import { Check, Copy, Paperclip, Undo2 } from '@lucide/svelte'
import { fileUrl } from '$lib/api'
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

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

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
            <div class="w-1.5 h-1.5 rounded-full bg-muted-foreground/40 animate-pulse"></div>
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
                {:else if part.type === "file"}
                    {#if (part.mime ?? "").startsWith("image/")}
                        <a
                            href={fileUrl(part.code ?? "")}
                            target="_blank"
                            rel="noreferrer"
                            class="block max-w-[240px] rounded-lg overflow-hidden border border-border bg-muted/40 hover:bg-accent/40 transition-colors"
                        >
                            <img
                                src={fileUrl(part.code ?? "")}
                                alt={part.name || part.code}
                                loading="lazy"
                                class="block w-full h-36 object-cover"
                            />
                        </a>
                        {#if part.size}
                            <div class="mt-0.5 text-[10px] text-muted-foreground px-0.5">{formatSize(part.size)}</div>
                        {/if}
                    {:else}
                        <a
                            href={fileUrl(part.code ?? "")}
                            class="inline-flex items-center gap-2 rounded border border-border bg-muted/40 px-2.5 py-1.5 text-xs hover:bg-accent/40 transition-colors"
                            target="_blank"
                            rel="noreferrer"
                        >
                            <Paperclip class="size-3.5 shrink-0 text-muted-foreground" />
                            <span class="truncate font-medium">{part.name || part.code}</span>
                            {#if part.size}<span class="shrink-0 text-[10px] text-muted-foreground">{formatSize(part.size)}</span>{/if}
                        </a>
                    {/if}
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
