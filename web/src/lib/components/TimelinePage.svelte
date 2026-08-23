<script lang="ts">
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import { Clock, GitBranch } from '@lucide/svelte'
import type { ChangeEntry } from '$lib/api'
import * as api from '$lib/api'

let { onSelectDiff }: { onSelectDiff?: (changeId: string) => void } = $props()

let changes = $state<ChangeEntry[]>([])

$effect(() => {
  const sid = store.activeSessionId
  if (!sid) {
    changes = []
    return
  }
  void api.sessions.changes(sid).then(r => {
    changes = r.isOk() ? r.value : []
  })
})
</script>

<div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto p-3">
        {#if changes.length === 0}
            <p class="text-xs text-muted-foreground">No changes yet</p>
        {:else}
            {#each changes as change, i (change.change_id + ':' + i)}
                <button
                    class="flex items-start gap-2 py-1.5 border-b border-border/50 hover:bg-accent/50 rounded px-1 w-full text-left"
                    onclick={() => onSelectDiff?.(change.change_id)}
                >
                    <GitBranch class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                    <div class="min-w-0 flex-1">
                        <div class="text-xs font-medium truncate">{change.message}</div>
                        <div class="text-[10px] text-muted-foreground truncate">{change.author}</div>
                        <div class="text-[9px] font-mono text-muted-foreground/60 truncate">{change.change_id.slice(0, 16)}</div>
                    </div>
                    <div class="text-[10px] text-muted-foreground shrink-0 flex items-center gap-1">
                        <Clock class="size-2.5" />
                        {new Date(change.timestamp).toLocaleTimeString()}
                    </div>
                </button>
            {/each}
        {/if}
    </div>
</div>
