<script lang="ts">
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import { CheckCircle, Clock } from '@lucide/svelte'
import type { MailboxEntry } from '$lib/api'
import * as api from '$lib/api'

let entries = $state<MailboxEntry[]>([])

$effect(() => {
  const sid = store.activeSessionId
  if (!sid) {
    entries = []
    return
  }
  void api.sessions.mailbox(sid).then(r => {
    entries = r.isOk() ? r.value : []
  })
})
</script>

<div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto p-3">
        {#each entries as entry (entry.id)}
            <div class="border border-border rounded-md p-2 mb-2 text-xs">
                <div class="flex items-center gap-2 mb-1">
                    <span class="font-medium text-primary">{entry.msg_type}</span>
                    {#if entry.consumed_at}
                        <span class="text-green-500 flex items-center gap-1"><CheckCircle class="size-2.5" /> consumed</span>
                    {:else}
                        <span class="text-muted-foreground flex items-center gap-1"><Clock class="size-2.5" /> pending</span>
                    {/if}
                    <span class="ml-auto text-[10px] text-muted-foreground">
                        {new Date(entry.created_at).toLocaleString()}
                    </span>
                </div>
                <pre class="whitespace-pre-wrap text-muted-foreground max-h-32 overflow-y-auto">{entry.payload.slice(0, 500)}</pre>
            </div>
        {/each}
        {#if entries.length === 0}
            <p class="text-xs text-muted-foreground">No messages</p>
        {/if}
    </div>
</div>
