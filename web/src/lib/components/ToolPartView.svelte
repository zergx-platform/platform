<script lang="ts">
import { GitCommitHorizontal } from '@lucide/svelte'
import type { ChatPart } from '$lib/hooks/useMessages.svelte'
import DiffView from './DiffView.svelte'
import ToolIcon from './ToolIcon.svelte'

let {
  part,
  isStreaming = false,
  onOpenChange,
}: {
  part: ChatPart
  isStreaming?: boolean
  onOpenChange?: (changeId: string) => void
} = $props()

const status = $derived(part.state?.status || 'complete')
const tool = $derived(part.tool || 'tool')
const input = $derived((part.state?.input ?? {}) as Record<string, unknown>)
const output = $derived(part.state?.output)
const hasError = $derived(status === 'error')
const changeId = $derived(
  typeof part.state?.change_id === 'string' ? part.state.change_id : null,
)
const showChange = $derived(!!changeId && !hasError && !!onOpenChange)

const statusDot = $derived(
  status === 'running'
    ? 'bg-yellow-500 animate-pulse'
    : status === 'pending'
      ? 'bg-orange-400 animate-pulse'
      : hasError
        ? 'bg-red-500'
        : 'bg-green-500',
)

function toolDisplayName(t: string): string {
  const map: Record<string, string> = { todowrite: 'todo' }
  return map[t] || t
}

function fmtOutput(raw: unknown): string {
  if (raw === null || raw === undefined) return ''
  return typeof raw === 'string' ? raw : JSON.stringify(raw, null, 2)
}

let open = $state(true)
function toggle() {
  open = !open
}
</script>

<div class="border rounded text-xs {hasError ? 'border-destructive/40 bg-destructive/5' : 'border-border bg-background/50'}">
        <div class="flex items-stretch">
        <button
            class="flex items-center gap-1.5 px-2 py-1 flex-1 min-w-0 text-left hover:bg-accent/30 cursor-pointer {showChange ? 'rounded-tl' : 'rounded-t'}"
            onclick={toggle}
        >
            <span class="w-1.5 h-1.5 rounded-full shrink-0 {statusDot}"></span>
            <ToolIcon name={tool} />
            <span class="font-mono font-medium shrink-0">{toolDisplayName(tool)}</span>
            {#if part.state?.title && part.state.title !== tool}
                <span class="text-muted-foreground truncate italic">{part.state.title}</span>
            {/if}
            <span class="text-muted-foreground ml-auto shrink-0 text-[10px]">{open ? '▲' : '▼'}</span>
        </button>
        {#if showChange && changeId}
            <button
                class="flex items-center gap-1 px-2 rounded-tr hover:bg-accent/40 text-muted-foreground hover:text-foreground transition-colors shrink-0"
                title="View change diff"
                onclick={() => onOpenChange?.(changeId)}
            >
                <GitCommitHorizontal class="size-3.5 text-primary" />
                <span class="font-mono text-[10px]">{changeId.slice(0, 8)}</span>
            </button>
        {/if}
    </div>

    {#if open}
        <div class="px-2 pb-2 space-y-1.5">
            {#if hasError}
                {#if Object.keys(input).length > 0}
                    <div>
                        <div class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-0.5">Input</div>
                        <pre class="rounded p-1.5 bg-muted font-mono text-[11px] max-h-40 overflow-y-auto overflow-x-auto whitespace-pre-wrap break-all">{JSON.stringify(input, null, 2)}</pre>
                    </div>
                {/if}
                {#if part.state?.error}
                    <div class="bg-red-100 dark:bg-red-950/30 rounded p-1.5 text-red-700 dark:text-red-400 whitespace-pre-wrap break-all font-mono text-[11px]">
                        {part.state.error}
                    </div>
                {/if}
            {:else}
                {#if Object.keys(input).length > 0}
                    <div>
                        <div class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-0.5">Input</div>
                        <pre class="rounded p-1.5 bg-muted font-mono text-[11px] max-h-40 overflow-y-auto overflow-x-auto whitespace-pre-wrap break-all">{JSON.stringify(input, null, 2)}</pre>
                    </div>
                {/if}

                <!-- Result metadata -->
                {#if changeId}
                    <div class="flex items-center gap-1">
                        <GitCommitHorizontal class="size-3.5 text-primary" />
                        <span class="font-mono text-[10px] text-primary">{changeId.slice(0, 8)}</span>
                    </div>
                {/if}
                {#if part.state?.diff}
                    <DiffView diffText={part.state.diff} />
                {/if}
                {#if (part.state?.additions ?? 0) > 0 || (part.state?.deletions ?? 0) > 0}
                    <div class="text-[11px] font-mono px-0.5">
                        <span class="text-green-500">+{part.state?.additions ?? 0}</span>
                        <span class="text-red-500"> −{part.state?.deletions ?? 0}</span>
                    </div>
                {/if}

                <!-- Result content -->
                {#if output != null && output !== ''}
                    <pre class="rounded p-1.5 bg-muted text-[11px] max-h-56 overflow-y-auto overflow-x-auto whitespace-pre-wrap break-all font-mono">{fmtOutput(output)}</pre>
                {:else if isStreaming && status === 'running'}
                    <div class="text-muted-foreground italic px-1">running...</div>
                {/if}
            {/if}
        </div>
    {/if}
</div>
