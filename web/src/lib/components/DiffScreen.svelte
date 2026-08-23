<script lang="ts">
import { ArrowLeft } from '@lucide/svelte'
import { onMount } from 'svelte'
import type { DiffFile } from '$lib/api'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import DiffView from './DiffView.svelte'
import LazyDiff from './LazyDiff.svelte'

let {
  changeId,
  sessionOrg,
  sessionRepo,
  onclose,
  onselectFile,
}: {
  changeId: string
  sessionOrg?: string
  sessionRepo?: string
  onclose: () => void
  onselectFile: (path: string) => void
} = $props()

let files = $state<DiffFile[]>([])
let error = $state('')
let diffs = $state<Record<string, string>>({})

onMount(async () => {
  if (!sessionOrg || !sessionRepo) return
  const r = await api.repos.diffChange(sessionOrg, sessionRepo, changeId)
  files = r.isOk() ? r.value : []
  if (files.length === 0) error = 'No changes found'
})
</script>

<div class="flex flex-col h-full">
    <div class="flex items-center gap-2 border-b border-border px-3 py-2 shrink-0">
        <Button variant="ghost" size="icon" onclick={onclose}><ArrowLeft class="size-4" /></Button>
        <span class="text-sm font-mono text-muted-foreground truncate">Diff: {changeId.slice(0, 12)}</span>
    </div>
    <div class="flex-1 overflow-y-auto">
        {#if error}
            <p class="text-sm text-muted-foreground p-4">{error}</p>
        {:else}
            {#each files as f (f.path)}
                <div class="border-b border-border/50">
                    <button class="w-full text-left px-4 py-2 text-xs font-mono text-primary hover:bg-accent/40 transition-colors" onclick={() => onselectFile(f.path)}>
                        {f.path}
                    </button>
                    <div class="px-2 pb-3">
                        <LazyDiff sessionOrg={sessionOrg ?? ''} sessionRepo={sessionRepo ?? ''} {changeId} filePath={f.path} />
                    </div>
                </div>
            {/each}
        {/if}
    </div>
</div>
