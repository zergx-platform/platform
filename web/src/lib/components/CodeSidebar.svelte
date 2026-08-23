<script lang="ts">
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import { RefreshCw } from '@lucide/svelte'
import { Button } from '$lib/components/ui/button'
import TreeNode from './TreeNode.svelte'
</script>

<div class="flex flex-col h-full">
    <div class="flex items-center gap-1 border-b border-border px-2 py-1.5 shrink-0">
        <span class="text-xs font-medium truncate">{store.activeSession?.org}/{store.activeSession?.repo}</span>
    </div>
    <div class="flex-1 overflow-y-auto">
        {#if store.codeLoading}
            <div class="flex justify-center p-4"><span class="text-xs text-muted-foreground">Loading...</span></div>
        {:else}
            <TreeNode path="" depth={0} />
        {/if}
    </div>
    <div class="border-t border-border p-2 shrink-0">
        <Button variant="ghost" size="sm" class="text-[10px]" onclick={() => store.openRepo(store.codeOrg, store.codeRepo, store.codeBranch)}>
            <RefreshCw class="size-2.5 mr-1" /> Refresh
        </Button>
    </div>
</div>
