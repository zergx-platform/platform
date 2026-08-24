<script lang="ts">
import { ChevronRight, Eye, RefreshCw, Square, X } from '@lucide/svelte'
import { onDestroy, onMount } from 'svelte'
import type { JobInfo } from '$lib/api'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import ConfirmDialog from './ConfirmDialog.svelte'


let {
  containerId = '',
  containerName = '',
  onclose = () => {},
  crumbs = [],
}: {
  containerId: string
  containerName: string
  onclose: () => void
  crumbs?: { label: string; onclick: () => void }[]
} = $props()

let jobs = $state<JobInfo[]>([])

// Output modal state
let modalJob = $state<JobInfo | null>(null)
let modalStream = $state<'all' | 'stdout' | 'stderr'>('all')
const streamChoices = ['all', 'stdout', 'stderr'] as const
let modalFilter = $state('')

let confirmKill = $state<string | null>(null)
let confirmKillBusy = $state(false)

type StreamState = {
  lines: string[]
  firstLine: number
  lastLine: number
  totalLines: number
  done: boolean
}
let modalStreams = $state<Record<string, Record<string, StreamState>>>({})
let modalPollers = new Map<string, ReturnType<typeof setInterval>>()

let sortedJobs = $derived(
  [...jobs].sort((a, b) => {
    if (a.state === 'running' && b.state !== 'running') return -1
    if (a.state !== 'running' && b.state === 'running') return 1
    return (b.started_at ?? 0) - (a.started_at ?? 0)
  }),
)

let modalOpen = $derived(modalJob !== null)

async function fetchOutput(
  jobId: string,
  stream: string,
  start: number,
  end: number,
) {
  const r = await api.containers.jobOutput(
    containerId,
    jobId,
    stream,
    start,
    end,
  )
  return r.isOk() ? r.value : null
}

async function openModal(j: JobInfo) {
  modalJob = j
  modalStream = 'all'
  modalFilter = ''
  if (!modalStreams[j.id]) modalStreams[j.id] = {}
  for (const s of ['all', 'stdout', 'stderr']) {
    modalStreams[j.id][s] = {
      lines: [],
      firstLine: 0,
      lastLine: -1,
      totalLines: 0,
      done: false,
    }
    const r = await fetchOutput(j.id, s, -200, -1)
    if (r)
      modalStreams[j.id][s] = {
        lines: r.lines,
        firstLine: r.start_line,
        lastLine: r.end_line,
        totalLines: r.total_lines,
        done: r.done,
      }
  }
  startPolling(j.id)
}

function closeModal() {
  modalJob = null
  for (const [jid, pid] of modalPollers) {
    clearInterval(pid)
    modalPollers.delete(jid)
  }
}

function startPolling(jid: string) {
  if (modalPollers.has(jid)) return
  const pid = setInterval(async () => {
    const j = jobs.find(j => j.id === jid)
    if (!j) {
      clearInterval(pid)
      modalPollers.delete(jid)
      return
    }
    for (const s of ['all', 'stdout', 'stderr'] as const) {
      const st = modalStreams[jid]?.[s]
      if (!st || st.done) continue
      const mj = modalStreams[jid]
      if (!mj) continue
      const r = await fetchOutput(jid, s, st.lastLine + 1, -1)
      if (!r) continue
      if (r.lines.length > 0) {
        mj[s] = {
          ...st,
          lines: [...st.lines, ...r.lines],
          lastLine: r.end_line,
          totalLines: r.total_lines,
          done: r.done,
        }
      } else if (r.done) {
        mj[s] = { ...st, done: true }
      }
    }
    const allDone = ['all', 'stdout', 'stderr'].every(
      s => modalStreams[jid]?.[s]?.done,
    )
    if (allDone || j.state !== 'running') {
      clearInterval(pid)
      modalPollers.delete(jid)
    }
  }, 1000)
  modalPollers.set(jid, pid)
}

async function modalLoadMore() {
  const j = modalJob
  if (!j) return
  const jid = j.id
  const s = modalStreams[jid]?.[modalStream]
  if (!s || s.firstLine <= 0) return
  const r = await fetchOutput(
    jid,
    modalStream,
    Math.max(0, s.firstLine - 200),
    s.firstLine - 1,
  )
  if (!r || r.lines.length === 0) return
  const mj2 = modalStreams[jid]
  if (!mj2) return
  mj2[modalStream] = {
    ...s,
    lines: [...r.lines, ...s.lines],
    firstLine: r.start_line,
  }
}

function filteredLines(jid: string, stream: string): string[] {
  const s = modalStreams[jid]?.[stream]
  if (!s) return []
  const f = modalFilter.trim().toLowerCase()
  if (!f) return s.lines
  return s.lines.filter(l => l.toLowerCase().includes(f))
}

onMount(() => {
  pollJobs()
})

onDestroy(() => {
  for (const pid of modalPollers.values()) clearInterval(pid)
})

async function pollJobs() {
  const r = await api.containers.jobs(containerId)
  if (r.isOk()) jobs = r.value
}

async function killJob(jobId: string) {
  const r = await api.containers.kill(containerId, jobId)
  if (r.isErr()) return
  if (!r.value.ok || !r.value.result?.ok) return
  jobs = jobs.map(j =>
    j.id === jobId ? { ...j, state: 'killed', exit_code: -9 } : j,
  )
  await pollJobs()
}

function requestKillJob(jobId: string) {
  confirmKill = jobId
}

async function runConfirmKill() {
  if (!confirmKill) return
  confirmKillBusy = true
  try {
    await killJob(confirmKill)
  } finally {
    confirmKillBusy = false
  }
}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b shrink-0">
        <div class="flex items-center gap-1.5 min-w-0">
            {#each crumbs as crumb, i (crumb.label + i)}
                <span class="text-sm text-muted-foreground/70 truncate max-w-[120px]">{crumb.label}</span>
                <ChevronRight class="size-3 text-muted-foreground/50 shrink-0" />
            {/each}
            <span class="text-sm font-mono font-semibold truncate">{containerName}</span>
            <ChevronRight class="size-3 text-muted-foreground shrink-0" />
            <span class="text-sm text-muted-foreground shrink-0">jobs</span>
        </div>
        <div class="flex items-center gap-1 shrink-0">
            <Button variant="ghost" size="icon" onclick={pollJobs} title="Refresh"><RefreshCw class="size-3.5" /></Button>
        </div>
    </div>

    <div class="flex-1 overflow-auto">
        {#each sortedJobs as j}
            <div class="px-4 py-2.5 border-b border-border/50 text-xs">
                <div class="flex items-center justify-between mb-0.5">
                    <span class="font-mono text-[10px]">{j.id}</span>
                    <span class="inline-block px-1.5 py-0.5 rounded text-[10px] font-medium
                        {j.state === 'running' ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30' :
                         j.state === 'done' ? 'bg-green-100 text-green-700 dark:bg-green-900/30' :
                         j.state === 'failed' ? 'bg-red-100 text-red-700 dark:bg-red-900/30' :
                         j.state === 'killed' ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30' :
                         'bg-gray-100 text-gray-700 dark:bg-gray-900/30'}">
                        {j.state}{j.exit_code !== 0 && j.exit_code !== -1 ? ` (${j.exit_code})` : ''}
                    </span>
                </div>
                <div class="text-[11px] text-muted-foreground break-all leading-relaxed mb-1.5">{j.command}</div>
                <div class="flex items-center gap-1.5">
                    <Button variant="ghost" size="icon" class="h-5 w-5" title="View output"
                        onclick={() => openModal(j)}>
                        <Eye class="size-2.5" />
                    </Button>
                    <Button variant="ghost" size="icon" class="h-5 w-5" title="Kill"
                        onclick={() => requestKillJob(j.id)}>
                        <Square class={j.state === "running" ? "size-2.5 text-destructive" : "size-2.5 text-muted-foreground"} />
                    </Button>
                </div>
            </div>
        {/each}
        {#if jobs.length === 0}
            <div class="text-xs text-muted-foreground text-center py-8">No jobs</div>
        {/if}
    </div>

    <!-- Output Modal -->
    {#if modalOpen && modalJob}
        <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
            <!-- Backdrop -->
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
            <div class="absolute inset-0 bg-black/40" role="presentation" onclick={closeModal}></div>
            <!-- Modal -->
            <div class="relative z-10 bg-card border rounded-lg shadow-xl w-full max-w-3xl max-h-[85vh] flex flex-col min-h-0 max-h-[calc(100dvh-2rem)]" role="dialog" aria-label="Job output">
                <!-- Modal header -->
                <div class="flex items-center justify-between px-4 py-3 border-b shrink-0">
                    <div class="min-w-0">
                        <div class="text-xs font-semibold">{modalJob.id} — {modalJob.state}</div>
                        <div class="text-[11px] text-muted-foreground truncate mt-0.5">{modalJob.command}</div>
                    </div>
                    <Button variant="ghost" size="icon" class="shrink-0" onclick={closeModal}><X class="size-4" /></Button>
                </div>

                <!-- Tabs + filter -->
                <div class="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-2 border-b shrink-0">
                    {#each streamChoices as s (s)}
                        <button class="px-2 py-0.5 text-[11px] rounded {modalStream === s ? 'bg-accent font-medium' : 'text-muted-foreground'}"
                            onclick={() => modalStream = s}>{s}</button>
                    {/each}
                    <div class="flex-1"></div>
                    <input class="w-20 sm:w-32 bg-transparent border-b text-[10px] outline-none placeholder:text-muted-foreground"
                        placeholder="filter..."
                        bind:value={modalFilter}
                    />
                </div>

                <!-- Output area -->
                <div class="flex-1 min-h-0 overflow-auto p-3 font-mono text-[11px] whitespace-pre-wrap">
                    {#if modalStreams[modalJob.id]?.[modalStream]?.firstLine > 0}
                        <button onclick={modalLoadMore} class="text-[10px] text-blue-500 hover:underline mb-2 block">
                            ↑ Load earlier lines...
                        </button>
                    {/if}
                    {#if filteredLines(modalJob.id, modalStream).length === 0}
                        <div class="text-muted-foreground italic">{modalStreams[modalJob.id]?.[modalStream]?.done ? "No output" : "Waiting for output..."}</div>
                    {:else}
                        {filteredLines(modalJob.id, modalStream).join("")}
                        {#if !modalStreams[modalJob.id]?.[modalStream]?.done}
                            <span class="text-muted-foreground">...</span>
                        {/if}
                    {/if}
                </div>

                <!-- Modal footer -->
                <div class="px-4 py-2 border-t shrink-0 flex items-center justify-between text-[10px] text-muted-foreground">
                    <div>
                        {#if modalStreams[modalJob.id]?.[modalStream]?.totalLines > 0}
                            lines {modalStreams[modalJob.id][modalStream].firstLine + 1}–{modalStreams[modalJob.id][modalStream].lastLine + 1} / {modalStreams[modalJob.id][modalStream].totalLines}
                        {/if}
                    </div>
                    <Button variant="outline" size="sm" class="text-[10px] h-6 px-2"
                        onclick={() => modalJob && requestKillJob(modalJob.id)}>
                        <Square class="size-2.5 mr-1" /> Kill
                    </Button>
                </div>
            </div>
        </div>
    {/if}
</div>

<ConfirmDialog
    open={!!confirmKill}
    title="Kill job"
    description={confirmKill ? `Kill job <strong>${confirmKill}</strong>? This sends SIGKILL to the process group.` : ''}
    confirmText="Kill"
    busy={confirmKillBusy}
    onConfirm={runConfirmKill}
    onClose={() => { confirmKill = null; confirmKillBusy = false }}
/>
