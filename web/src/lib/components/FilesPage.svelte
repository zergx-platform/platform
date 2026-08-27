<script lang="ts">
import { ArrowLeft, File, History } from '@lucide/svelte'
import { getStore } from '$lib/stores.svelte'

const store = getStore()

// After a turn settles the agent may have written files: drop the stale
// tree cache so the next expansion fetches fresh entries.
$effect(() => {
  const rev = store.sessionRevision
  if (rev > 0 && !store.selectedFilePath && !store.activeDiffChangeId) {
    void store.refreshFileTree()
  }
})

import { Button } from '$lib/components/ui/button'
import CodeView from './CodeView.svelte'
import DiffView from './DiffView.svelte'
import TreeNode from './TreeNode.svelte'
</script>

<div class="flex flex-col h-full">
    {#if store.activeDiffChangeId}
        <!-- Diff: deepest level, replaces the file view -->
        <div class="flex items-center gap-2 border-b border-border px-2 py-1.5 shrink-0">
            <Button variant="ghost" size="icon-sm" class="size-6" title="Back to file" onclick={() => store.backFileOverlay()}>
                <ArrowLeft class="size-3.5" />
            </Button>
            <span class="text-xs font-mono text-muted-foreground truncate flex-1">
                {store.selectedFilePath} · {store.activeDiffChangeId.slice(0, 10)}
            </span>
        </div>
        <div class="flex-1 min-h-0 overflow-y-auto">
            <DiffView diffText={store.fileDiffs[store.activeDiffChangeId] || ''} />
        </div>
    {:else if store.selectedFilePath}
        <div class="flex-1 min-h-0 flex flex-col">
            <div class="flex items-center gap-2 border-b border-border px-2 py-1.5 shrink-0">
                <Button variant="ghost" size="icon-sm" class="size-6" title="Back to files" onclick={() => store.backFileOverlay()}>
                    <ArrowLeft class="size-3.5" />
                </Button>
                <File class="size-3.5 text-primary shrink-0" />
                <span class="text-xs font-mono text-muted-foreground truncate flex-1" title={store.selectedFilePath}>
                    {store.selectedFilePath}
                </span>
                <Button
                    variant="ghost"
                    size="icon-sm"
                    class="size-6"
                    title={store.showFileHistory ? 'View file' : 'History'}
                    onclick={() => store.showFileHistory ? (store.showFileHistory = false) : store.loadFileHistory()}
                >
                    <History class="size-3.5" />
                </Button>
            </div>
            <div class="flex-1 min-h-0 overflow-y-auto">
                {#if store.showFileHistory}
                    {#if store.fileHistoryLoading}
                        <p class="text-xs text-muted-foreground p-3">Loading history...</p>
                    {:else if store.fileHistory.length === 0}
                        <p class="text-xs text-muted-foreground p-3">No history for this file.</p>
                    {:else}
                        <div class="p-2">
                            {#each store.fileHistory as commit (commit.change_id)}
                                <button
                                    class="w-full text-left px-2 py-1.5 hover:bg-accent/50 rounded border-b border-border/40"
                                    onclick={() => store.openFileChangeOverlay(commit.change_id)}
                                >
                                    <div class="flex items-center gap-2">
                                        <span class="text-[10px] font-mono text-muted-foreground shrink-0">{commit.change_id.slice(0, 10)}</span>
                                        <span class="text-[11px] truncate flex-1">{commit.message || '(no description)'}</span>
                                        <span class="text-[9px] text-muted-foreground shrink-0">{commit.author}</span>
                                    </div>
                                </button>
                            {/each}
                        </div>
                    {/if}
                {:else}
                    <CodeView code={store.fileContent} filepath={store.selectedFilePath} />
                {/if}
            </div>
        </div>
    {:else}
        <div class="flex-1 overflow-y-auto p-2">
            {#if store.codeLoading}
                <p class="text-xs text-muted-foreground p-2">Loading...</p>
            {:else}
                <TreeNode path="" depth={0} />
            {/if}
        </div>
    {/if}
</div>
