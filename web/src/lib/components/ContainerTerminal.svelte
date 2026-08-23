<script lang="ts">
import { Send } from '@lucide/svelte'
import { onMount } from 'svelte'
import type { ExecResult } from '$lib/api'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'

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

let command = $state('')
let running = $state(false)
let history = $state<string[]>([])
let cmdHistory = $state<string[]>([])
let cmdHistIdx = $state(-1)
let textareaEl: HTMLTextAreaElement | undefined = $state()

function isIncomplete(s: string): boolean {
  let trailing = 0
  for (let i = s.length - 1; i >= 0 && s[i] === '\\'; i--) trailing++
  if (trailing % 2 === 1) return true
  let inSingle = false,
    inDouble = false
  for (let i = 0; i < s.length; i++) {
    const c = s[i]
    if (c === '\\') {
      i++
      continue
    }
    if (c === '"' && !inSingle) inDouble = !inDouble
    if (c === "'" && !inDouble) inSingle = !inSingle
  }
  return inSingle || inDouble
}

let isMulti = $derived(isIncomplete(command))
let promptChar = $derived(isMulti ? '>' : '$')

function resizeTA() {
  requestAnimationFrame(() => {
    if (!textareaEl) return
    textareaEl.style.height = 'auto'
    textareaEl.style.height = `${textareaEl.scrollHeight}px`
  })
}

$effect(() => {
  command
  resizeTA()
})

async function execute() {
  const cmd = command.trim()
  if (!cmd) return
  if (!containerId) {
    history = [...history, `[error] no container bound to this terminal`]
    return
  }
  const lines = command.trim().split('\n')
  const out: string[] = []
  for (let i = 0; i < lines.length; i++) {
    out.push(i === 0 ? `$ ${lines[i]}` : `> ${lines[i]}`)
  }
  history = [...history, ...out]
  cmdHistory = [command, ...cmdHistory].slice(0, 200)
  cmdHistIdx = -1
  running = true

  const r = await api.containers.exec(containerId, cmd)
  if (r.isErr()) {
    history = [...history, `[error] ${r.error}`]
  } else {
    const result = r.value
    if (result.error) {
      history = [...history, `[error] ${result.error}`]
    } else if (result.backgrounded && result.job_id) {
      history = [...history, `[${result.job_id}] backgrounded (see Jobs tab)`]
    } else if (result.exit_code !== undefined) {
      if (result.output) history = [...history, result.output]
      if (result.exit_code !== 0)
        history = [...history, `[exit: ${result.exit_code}]`]
    }
  }
  running = false
  command = ''
  resizeTA()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    if (isIncomplete(command)) return
    e.preventDefault()
    execute()
  } else if (e.key === 'ArrowUp' && !command.includes('\n')) {
    e.preventDefault()
    if (cmdHistIdx < cmdHistory.length - 1) {
      cmdHistIdx++
      command = cmdHistory[cmdHistIdx]
      requestAnimationFrame(resizeTA)
    }
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (cmdHistIdx > 0) {
      cmdHistIdx--
      command = cmdHistory[cmdHistIdx]
      requestAnimationFrame(resizeTA)
    } else if (cmdHistIdx === 0) {
      cmdHistIdx = -1
      command = ''
      requestAnimationFrame(resizeTA)
    }
  }
}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="h-full flex flex-col">
    <!-- Output area -->
    <div class="flex-1 overflow-auto p-3 font-mono text-xs bg-muted/30">
        <div class="whitespace-pre-wrap break-all space-y-0.5">
            {#each history as line, i (i)}
                <div class="leading-relaxed" class:text-muted-foreground={line.startsWith("$ ")} class:text-destructive={line.startsWith("[error]")} class:text-amber-600={line.includes("running (async)")}>
                    {line}
                </div>
            {/each}
        </div>
        {#if history.length === 0}
            <div class="text-muted-foreground text-center py-8">Type a command to start...</div>
        {/if}
    </div>
    <!-- Input -->
    <div class="flex items-start gap-2 px-3 py-2 border-t bg-background shrink-0">
        <span class="text-xs font-mono text-muted-foreground shrink-0 mt-0.5">{promptChar}</span>
        <textarea
            bind:this={textareaEl}
            class="flex-1 bg-transparent text-xs font-mono outline-none resize-none overflow-hidden min-h-5"
            placeholder="command..."
            bind:value={command}
            onkeydown={handleKeydown}
            oninput={resizeTA}
            disabled={running}
            rows={1}
        ></textarea>
        <Button variant="ghost" size="icon" class="shrink-0" onclick={execute} disabled={running || !command.trim()}>
            <Send class="size-3.5" />
        </Button>
    </div>
</div>
