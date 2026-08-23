<script lang="ts">
import { GitCommitHorizontal } from '@lucide/svelte'
import type { ChatPart } from '$lib/hooks/useMessages.svelte'
import DiffView from './DiffView.svelte'
import ReadOutputView from './ReadOutputView.svelte'
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
const title = $derived(part.state?.title)
const hasError = $derived(status === 'error')
const changeId = $derived(
  typeof part.state?.change_id === 'string' ? part.state.change_id : null,
)
const showChange = $derived(!!changeId && !hasError && !!onOpenChange)

const isBash = $derived(tool === 'sandbox-run')
const isRead = $derived(tool === 'read')
const isWrite = $derived(tool === 'write')
const isEdit = $derived(tool === 'edit')
const isDelete = $derived(tool === 'delete')
const isSandboxRead = $derived(tool === 'sandbox-read')
const isGrep = $derived(tool === 'grep')
const isGlob = $derived(tool === 'glob')
const isWebfetch = $derived(tool === 'webfetch')
const isTodowrite = $derived(tool === 'todowrite')

const s = (v: unknown): string => (typeof v === 'string' ? v : '')

function inputSummary(t: string, inp: Record<string, unknown>): string {
  const jobId = s(inp.job_id)
  switch (t) {
    case 'sandbox-read':
    case 'sandbox-write':
    case 'sandbox-edit':
      return s(inp.path)
    case 'sandbox-delete':
      return `delete ${s(inp.path)}`
    case 'git-restore':
      return `restore ${s(inp.path)} @ ${s(inp.rev)}`
    case 'sandbox-job-list':
      return 'list jobs'
    case 'sandbox-job-output':
      return `output ${jobId}${s(inp.grep) ? ` · grep ${s(inp.grep)}` : ''}`
    case 'sandbox-job-wait':
      return `wait ${jobId}`
    case 'sandbox-job-kill':
      return `kill ${jobId}`
    case 'sandbox-job-stdin':
      return `stdin ${jobId}`
    case 'sandbox-port':
      return `port ${s(inp.sandbox_path)} → ${s(inp.repo_path)}`
    case 'explore':
      return s(inp.org) || 'explore orgs/repos'
    case 'list-containerfile-templates':
      return 'list build templates'
    case 'container-build':
      return `build ${s(inp.tag)} ← ${s(inp.dockerfile_path)}${s(inp.context) ? ` (ctx ${s(inp.context)})` : ''}`
    case 'package-publish':
      return `publish ${s(inp.protocol)}${s(inp.name) ? ` ${s(inp.name)}` : ''}${s(inp.version) ? `@${s(inp.version)}` : ''}`
    case 'container-deploy':
      return `deploy ${s(inp.image)}${s(inp.name) ? ` as ${s(inp.name)}` : ''}${inp.port != null ? ` :${inp.port}` : ''}`
    case 'pull-oci-image':
      return `pull image ${s(inp.image)}`
    case 'pull-git-repo':
      return `clone ${s(inp.git_url)}${s(inp.branch) ? ` (${s(inp.branch)})` : ''}`
    case 'list-registry-packages':
      return `list ${s(inp.protocol) || 'oci'} packages${s(inp.name) ? ` · ${s(inp.name)}` : ''}`
    default:
      return browserSummary(t, inp)
  }
}

function browserSummary(t: string, inp: Record<string, unknown>): string {
  const el = s(inp.element)
  const url = s(inp.url)
  switch (t) {
    case 'browser-navigate':
      return `navigate ${url}`
    case 'browser-navigate-back':
      return 'navigate back'
    case 'browser-navigate-forward':
      return 'navigate forward'
    case 'browser-resize':
      return `resize ${inp.width ?? '?'}×${inp.height ?? '?'}`
    case 'browser-click':
      return `click ${el}${inp.doubleClick ? ' (double)' : ''}`
    case 'browser-hover':
      return `hover ${el}`
    case 'browser-type':
      return `type ${el}`
    case 'browser-select-option':
      return `select ${el}`
    case 'browser-drag':
      return `drag ${s(inp.startTarget)} → ${s(inp.endTarget)}`
    case 'browser-drop':
      return `drop onto ${el}`
    case 'browser-file-upload':
      return `upload to ${el}`
    case 'browser-fill-form':
      return `fill form (${Array.isArray(inp.fields) ? inp.fields.length : 0} fields)`
    case 'browser-press-key':
      return `press ${s(inp.key)}`
    case 'browser-find':
      return `find ${s(inp.text) || s(inp.regex)}`
    case 'browser-wait-for':
      return `wait ${s(inp.text) || s(inp.regex) || (inp.time != null ? `${inp.time}ms` : '')}`
    case 'browser-expect-text':
      return `expect ${s(inp.text)}`
    case 'browser-handle-dialog':
      return `dialog ${inp.accept === false ? 'dismiss' : 'accept'}`
    case 'browser-evaluate':
    case 'browser-run-code-unsafe':
      return 'evaluate JS'
    case 'browser-snapshot':
      return 'snapshot'
    case 'browser-take-screenshot':
      return 'screenshot'
    case 'browser-tabs':
      return `tabs ${s(inp.tabAction) || 'list'}${inp.url ? ` ${s(inp.url)}` : ''}`
    default:
      return ''
  }
}

const summary = $derived(inputSummary(tool, input))

let open = $state(true)
function toggle() {
  open = !open
}

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
            {#if title && title !== tool}
                <span class="text-muted-foreground truncate italic">{title}</span>
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
            {#if part.state?.error}
                <div class="bg-red-100 dark:bg-red-950/30 rounded p-1.5 text-red-700 dark:text-red-400 whitespace-pre-wrap break-all font-mono text-[11px]">
                    {part.state.error}
                </div>
            {/if}

            {#if isBash && input.command}
                <div class="bg-muted rounded p-1.5 font-mono text-[11px] whitespace-pre-wrap overflow-x-auto"><span class="text-muted-foreground select-none">$ </span>{input.command}{#if input.workdir}<span class="text-muted-foreground text-[9px] block mt-0.5">cwd: {input.workdir}</span>{/if}</div>
            {:else if (isRead || isWrite || isEdit || isDelete) && input.path}
                <div class="bg-muted rounded px-1.5 py-1 font-mono text-[11px] flex items-center gap-2">
                    <span class="text-sky-500">⤷</span>
                    <span class="break-all">{input.path}</span>
                </div>
            {:else if isGrep && input.pattern}
                <div class="bg-muted rounded px-1.5 py-1 font-mono text-[11px]">grep {input.pattern}{input.include ? ` (${input.include})` : ''}{input.path ? ` · ${input.path}` : ''}</div>
            {:else if isGlob && input.pattern}
                <div class="bg-muted rounded px-1.5 py-1 font-mono text-[11px]">glob {input.pattern}</div>
            {:else if isWebfetch && input.url}
                <div class="bg-muted rounded px-1.5 py-1 font-mono text-[11px] flex items-center gap-2">
                    <span class="text-violet-500">⤷</span>
                    <span class="break-all">{input.url}</span>
                </div>
            {:else if isTodowrite && Array.isArray(input.todos)}
                <div class="space-y-0.5">
                    {#each input.todos as t, i}
                        <div class="flex items-center gap-1 text-[11px] {t.status === 'completed' ? 'text-muted-foreground line-through' : t.status === 'in_progress' ? 'font-medium' : 'text-muted-foreground'}">
                            <span class="w-3 text-center shrink-0 {t.status === 'completed' ? 'text-green-500' : t.status === 'in_progress' ? 'text-yellow-500' : 'text-gray-400'}">
                                {t.status === 'completed' ? '✓' : t.status === 'in_progress' ? '●' : '○'}
                            </span>
                            <span class="text-[9px] shrink-0 w-10 truncate {t.priority === 'high' ? 'text-red-400 font-medium' : ''}">{t.priority}</span>
                            <span class="truncate">{t.content}</span>
                        </div>
                    {/each}
                </div>
            {:else if summary}
                <div class="bg-muted rounded px-1.5 py-1 font-mono text-[11px] flex items-center gap-2">
                    <span class="text-muted-foreground/60">{tool}</span>
                    <span class="break-all">{summary}</span>
                </div>
            {:else if Object.keys(input).length > 0}
                <div class="text-muted-foreground/70 text-[10px] font-mono px-1 flex items-center gap-1"><span class="text-slate-400">{"{"}</span>args</div>
                <pre class="bg-muted rounded p-1.5 font-mono text-[11px] whitespace-pre-wrap break-all overflow-x-auto">{JSON.stringify(input, null, 2)}</pre>
            {/if}

            {#if output != null && output !== ''}
                {#if isRead || isSandboxRead}
                    <ReadOutputView output={fmtOutput(output)} filePath={String(input.path ?? '')} />
                {:else if isWrite && input.content}
                    <div class="text-green-700 dark:text-green-400 text-[11px] px-1">Wrote file.</div>
                    <ReadOutputView output={String(input.content)} filePath={String(input.path ?? '')} />
                {:else if isTodowrite}
                    <!-- no output for todowrite -->
                {:else if (isEdit || tool === 'sandbox-edit') && part.state?.diff}
                    <div class="text-green-700 dark:text-green-400 text-[11px] px-1 flex items-center gap-2">
                        Edit applied successfully.
                        <span class="font-mono text-[10px] text-muted-foreground">
                            <span class="text-green-500">+{part.state.additions ?? 0}</span>
                            <span class="text-red-500"> −{part.state.deletions ?? 0}</span>
                        </span>
                    </div>
                    <DiffView diffText={part.state.diff} />
                {:else}
                    <pre class="rounded p-1.5 bg-muted text-[11px] max-h-56 overflow-y-auto overflow-x-auto whitespace-pre-wrap break-all font-mono">{fmtOutput(output)}</pre>
                {/if}
            {:else if isStreaming && status === 'running'}
                <div class="text-muted-foreground italic px-1">running...</div>
            {/if}
        </div>
    {/if}
</div>
