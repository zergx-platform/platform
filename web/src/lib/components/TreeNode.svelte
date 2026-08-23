<script lang="ts">
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import { ChevronDown, ChevronRight, File, Folder } from '@lucide/svelte'
import TreeNode from './TreeNode.svelte'

let {
  path = '',
  depth = 0,
  ancestorsLast = [] as boolean[],
}: {
  path?: string
  depth?: number
  ancestorsLast?: boolean[]
} = $props()

let entries = $derived(store.treeCache[path] ?? [])
let expanded = $derived(store.expandedDirs)

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

function treePrefix(i: number, total: number): string {
  let pre = ''
  for (let d = 0; d < depth; d++) {
    pre += ancestorsLast[d] ? '\u00A0\u00A0\u00A0 ' : '\u2502\u00A0\u00A0 '
  }
  const isLast = i === total - 1
  pre += isLast ? '\u2514\u2500\u2500 ' : '\u251C\u2500\u2500 '
  return pre
}
</script>

{#each entries as entry, i (entry.path)}
    {@const prefix = treePrefix(i, entries.length)}
    {#if entry.is_dir}
        <button
            class="flex w-full cursor-pointer items-center gap-1 text-xs hover:bg-accent/60 transition-colors py-0.5 select-none font-mono text-left text-muted-foreground"
            style="padding-left: 8px; padding-right: 8px;"
            onclick={() => store.toggleDir(entry.path)}
        >
            <span class="whitespace-pre shrink-0">{prefix}</span>
            {#if expanded.has(entry.path)}
                <ChevronDown class="size-3 shrink-0" />
            {:else}
                <ChevronRight class="size-3 shrink-0" />
            {/if}
            <Folder class="size-3.5 shrink-0 text-blue-400" />
            <span class="truncate text-foreground">{entry.name}</span>
        </button>
        {#if expanded.has(entry.path)}
            <TreeNode path={entry.path} depth={depth + 1} ancestorsLast={[...ancestorsLast, i === entries.length - 1]} />
        {/if}
    {:else}
        <button
            class="flex w-full cursor-pointer items-center gap-1 text-xs hover:bg-accent/60 transition-colors py-0.5 select-none font-mono text-left
                {store.selectedFilePath === entry.path ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}"
            style="padding-left: 8px; padding-right: 8px;"
            onclick={() => store.openFileOverlay(entry.path)}
        >            <span class="whitespace-pre shrink-0">{prefix}</span>
            <span class="w-3 shrink-0"></span>
            <File class="size-3.5 shrink-0 text-muted-foreground" />
            <span class="truncate text-foreground flex-1">{entry.name}</span>
            {#if entry.size > 0}
                <span class="text-[9px] text-muted-foreground/50 shrink-0 font-sans">{formatSize(entry.size)}</span>
            {/if}
        </button>
    {/if}
{/each}
