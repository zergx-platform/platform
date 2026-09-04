<script lang="ts">
import { getStore } from '$lib/stores.svelte'
import * as api from '$lib/api'

const store = getStore()

import {
  ArrowLeft,
  Building2,
  Code,
  File,
  Folder,
  FolderGit,
  GitBranch,
  GitCommitHorizontal,
  History,
  Tag,
  X,
} from '@lucide/svelte'
import { Button } from '$lib/components/ui/button'
import CodeView from './CodeView.svelte'
import DiffView from './DiffView.svelte'
import TreeNode from './TreeNode.svelte'

let mobileRepoSheet = $state(false)
let view = $state<'tree' | 'commits'>('tree')
let commits = $state<Array<{ change_id: string; commit_id: string; author: string; timestamp: string; message: string }>>([])
let commitsLoading = $state(false)
let tags = $state<Array<{ name: string; target: string }>>([])
let showBlame = $state(false)
let blame = $state<string[]>([])
let blameLoading = $state(false)

function relativeTime(ts: string): string {
  const d = new Date(`${ts}Z`)
  const now = Date.now()
  const diff = now - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return d.toLocaleDateString()
}

function shortId(id: string): string {
  return id.slice(0, 8)
}

async function loadCommits() {
  if (!store.codeOrg || !store.codeRepo) return
  commitsLoading = true
  view = 'commits'
  const [cr, tr] = await Promise.all([
    api.repos.log(store.codeOrg, store.codeRepo, { limit: 100 }),
    api.repos.tags(store.codeOrg, store.codeRepo),
  ])
  commits = cr.isOk() ? cr.value : []
  tags = tr.isOk() ? tr.value : []
  commitsLoading = false
}

async function loadBlame() {
  if (!store.codeOrg || !store.codeRepo || !store.selectedFilePath) return
  blameLoading = true
  showBlame = !showBlame
  if (showBlame && blame.length === 0) {
    const r = await api.repos.blame(
      store.codeOrg,
      store.codeRepo,
      store.codeBookmark || 'main',
      store.selectedFilePath,
    )
    blame = r.isOk() ? r.value : []
  }
  blameLoading = false
}

function showTree() {
  view = 'tree'
}
</script>

<!-- Desktop: 3-panel layout -->
<div class="hidden lg:flex h-full">
    <!-- Panel 1: Org/Repo selector -->
    <div class="w-52 border-r border-border shrink-0 flex flex-col bg-muted/20">
        <div class="flex items-center gap-2 border-b border-border px-3 py-2 shrink-0">
            <Code class="size-3.5 text-muted-foreground" />
            <span class="text-xs font-semibold">Repositories</span>
        </div>
        <div class="flex-1 overflow-y-auto py-1">
            {#each store.orgs as orgNode (orgNode.org)}
                <div>
                    <div class="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
                        <Building2 class="size-3 shrink-0" />
                        {orgNode.org}
                    </div>
                    {#each orgNode.repos as repoNode (repoNode.repo)}
                        <div class="mb-0.5">
                            <div class="flex items-center gap-1.5 pl-5 pr-3 py-1 text-xs font-medium text-foreground/80 truncate">
                                <FolderGit class="size-3 shrink-0 text-blue-400" />
                                {repoNode.repo}
                            </div>
                            {#each repoNode.bookmarks as bm (bm.bookmark)}
                                <button
                                    class="flex w-full cursor-pointer items-center gap-2 pl-8 pr-3 py-1 text-xs hover:bg-accent/60 transition-colors
                                        {store.codeOrg === orgNode.org && store.codeRepo === repoNode.repo && store.codeBookmark === bm.bookmark ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}"
                                    onclick={() => { store.openRepo(orgNode.org, repoNode.repo, bm.bookmark) }}
                                >
                                    <GitBranch class="size-3 shrink-0" />
                                    <span class="truncate">{bm.bookmark}</span>
                                </button>
                            {/each}
                        </div>
                    {/each}
                </div>
            {/each}
            {#if store.orgs.length === 0}
                <div class="px-3 py-4 text-xs text-muted-foreground text-center">No repositories.<br />Create a session first.</div>
            {/if}
        </div>
    </div>

    <!-- Panel 2: File explorer (tree view) -->
    <div class="w-64 border-r border-border shrink-0 flex flex-col">
        <div class="flex items-center gap-2 border-b border-border px-3 py-2 shrink-0">
            <Folder class="size-3.5 text-blue-400" />
            <span class="text-xs font-semibold truncate">
                {store.codeRepo ? `${store.codeOrg}/${store.codeRepo}` : "Files"}
            </span>
            {#if store.codeBookmark}
                <span class="text-[9px] text-muted-foreground bg-muted px-1 py-0.5 rounded font-mono">{store.codeBookmark}</span>
            {/if}
            {#if store.codeRepo}
                <div class="flex-1"></div>
                <button class="p-1 rounded hover:bg-accent/60 text-muted-foreground" title="Commits" onclick={loadCommits}>
                    <GitCommitHorizontal class="size-3.5" />
                </button>
            {/if}
        </div>
        <div class="flex-1 overflow-y-auto">
            {#if !store.codeRepo}
                <div class="flex flex-col items-center justify-center h-full gap-2 text-muted-foreground">
                    <Folder class="size-8 opacity-30" />
                    <p class="text-xs">Select a repository</p>
                </div>
            {:else if view === 'commits'}
                {#if commitsLoading}
                    <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground animate-pulse">Loading commits...</span></div>
                {:else}
                    {#if tags.length > 0}
                        <div class="px-3 py-2 border-b border-border/40 flex flex-wrap gap-1.5">
                            {#each tags as t (t.name)}
                                <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted text-[10px] font-mono text-muted-foreground">
                                    <Tag class="size-3 text-primary" />{t.name}
                                </span>
                            {/each}
                        </div>
                    {/if}
                    {#if commits.length === 0}
                        <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">No commits</span></div>
                    {:else}
                        <div class="py-1">
                            {#each commits as commit (commit.commit_id)}
                                <button
                                    class="flex w-full items-start gap-2 px-3 py-2 text-left hover:bg-accent/40 transition-colors border-b border-border/40"
                                    onclick={() => { view = 'tree'; store.selectedFilePath = null; store.fileContent = '' }}
                                    title="Back to tree"
                                >
                                    <GitCommitHorizontal class="size-3 mt-0.5 text-primary shrink-0" />
                                    <div class="min-w-0 flex-1">
                                        <div class="text-xs font-medium truncate">{commit.message}</div>
                                        <div class="text-[10px] text-muted-foreground flex items-center gap-2">
                                            <span class="font-mono">{shortId(commit.commit_id)}</span>
                                            <span>{commit.author}</span>
                                            <span>{relativeTime(commit.timestamp)}</span>
                                        </div>
                                    </div>
                                </button>
                            {/each}
                        </div>
                    {/if}
                {/if}
            {:else if store.codeLoading}
                <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground animate-pulse">Loading...</span></div>
            {:else if !store.treeCache[""] || store.treeCache[""].length === 0}
                <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">Empty directory</span></div>
            {:else}
                <TreeNode path="" depth={0} />
            {/if}
        </div>
    </div>

    <!-- Panel 3: File content -->
    <div class="flex-1 flex flex-col min-w-0 bg-background">
        {#if store.selectedFilePath}
            <div class="flex items-center gap-2 border-b border-border px-4 py-2 shrink-0">
                <File class="size-3.5 text-primary" />
                <span class="text-xs font-mono text-muted-foreground truncate">{store.selectedFilePath}</span>
                <div class="flex-1"></div>
                {#if store.activeDiffChangeId}
                    <Button variant="ghost" size="icon" class="size-6" title="Back to file" onclick={() => { store.activeDiffChangeId = null; store.showFileHistory = false }}>
                        <X class="size-3.5" />
                    </Button>
                {:else}
                    <Button variant="ghost" size="icon" class="size-6" title={store.showFileHistory ? "View file" : "History"} onclick={store.showFileHistory ? () => store.showFileHistory = false : () => { store.showFileHistory = true; store.loadFileHistory() }}>
                        <History class="size-3.5" />
                    </Button>
                    <Button variant="ghost" size="icon" class="size-6" title="Blame" onclick={loadBlame}>
                        <GitCommitHorizontal class="size-3.5" />
                    </Button>
                    <Button variant="ghost" size="icon" class="size-6" title="Close" onclick={() => { store.selectedFilePath = null; store.fileContent = ""; store.showFileHistory = false; store.activeDiffChangeId = null; showBlame = false; blame = [] }}>
                        <X class="size-3.5" />
                    </Button>
                {/if}
            </div>
            {#if store.activeDiffChangeId}
                <div class="flex-1 overflow-auto">
                    <DiffView diffText={store.fileDiffs[store.activeDiffChangeId] || ""} />
                </div>
            {:else if showBlame}
                <div class="flex-1 overflow-auto font-mono text-xs">
                    {#if blameLoading}
                        <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground animate-pulse">Loading blame...</span></div>
                    {:else if blame.length === 0}
                        <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">No blame data</span></div>
                    {:else}
                        <pre class="p-4 whitespace-pre-wrap">{blame.join('\n')}</pre>
                    {/if}
                </div>
            {:else if store.showFileHistory}
                <div class="flex-1 overflow-auto">
                    {#if store.fileHistoryLoading}
                        <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground animate-pulse">Loading history...</span></div>
                    {:else if store.fileHistory.length === 0}
                        <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">No changes for this file</span></div>
                    {:else}
                        {#each store.fileHistory as commit (commit.change_id)}
                            <button
                                class="flex w-full items-center gap-2 px-4 py-2 text-xs hover:bg-accent/40 transition-colors text-left border-b border-border/50"
                                onclick={() => store.toggleCommitDiff(commit.change_id)}
                            >
                                <span class="text-primary font-mono">{shortId(commit.change_id)}</span>
                                <span class="text-muted-foreground truncate flex-1">{commit.message}</span>
                                <span class="text-[10px] text-muted-foreground/70 shrink-0">{relativeTime(commit.timestamp)}</span>
                            </button>
                        {/each}
                    {/if}
                </div>
            {:else}
                <div class="flex-1 overflow-auto">
                    <CodeView code={store.fileContent} filepath={store.selectedFilePath} />
                </div>
            {/if}
        {:else}
            <div class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
                {#if store.codeRepo}{#if store.codeLoading}Loading...{:else}Select a file to view{/if}{:else}Select a bookmark to browse files{/if}
            </div>
        {/if}
    </div>
</div>

<!-- Mobile: single panel with back navigation -->
<div class="flex flex-col h-full lg:hidden">
    <!-- Mobile header -->
    <div class="flex items-center gap-2 border-b border-border px-3 py-2 shrink-0">
        {#if store.selectedFilePath}
            {#if store.activeDiffChangeId}
                <Button variant="ghost" size="icon" class="size-7" onclick={() => { store.activeDiffChangeId = null; store.showFileHistory = false }}>
                    <ArrowLeft class="size-4" />
                </Button>
                <span class="text-xs font-mono text-muted-foreground truncate flex-1">{store.selectedFilePath} — {shortId(store.activeDiffChangeId)}</span>
            {:else if store.showFileHistory}
                <Button variant="ghost" size="icon" class="size-7" onclick={() => store.showFileHistory = false}>
                    <ArrowLeft class="size-4" />
                </Button>
                <span class="text-xs font-mono text-muted-foreground truncate flex-1">{store.selectedFilePath} — History</span>
            {:else}
                <Button variant="ghost" size="icon" class="size-7" onclick={() => { store.selectedFilePath = null; store.fileContent = ""; store.showFileHistory = false; store.activeDiffChangeId = null }}>
                    <ArrowLeft class="size-4" />
                </Button>
                <File class="size-3.5 text-primary shrink-0" />
                <span class="text-xs font-mono text-muted-foreground truncate flex-1">{store.selectedFilePath}</span>
                <Button variant="ghost" size="icon" class="size-7" title="History" onclick={() => { store.showFileHistory = true; store.loadFileHistory() }}>
                    <History class="size-4" />
                </Button>
            {/if}
        {:else if store.codeRepo}
            <Button variant="ghost" size="icon" class="size-7" onclick={() => { store.codeRepo = ""; store.codeBookmark = "" }}>
                <ArrowLeft class="size-4" />
            </Button>
            <span class="text-xs font-medium truncate">
                {store.codeOrg}/{store.codeRepo}
            </span>
            {#if store.codeBookmark}
                <span class="text-[9px] text-muted-foreground bg-muted px-1 py-0.5 rounded font-mono">{store.codeBookmark}</span>
            {/if}
        {:else}
            <span class="text-sm font-semibold">Code</span>
        {/if}
    </div>

    <!-- Mobile content area -->
    <div class="flex-1 overflow-y-auto">
        {#if store.selectedFilePath && store.activeDiffChangeId}
            <DiffView diffText={store.fileDiffs[store.activeDiffChangeId] || ""} />
        {:else if store.selectedFilePath && store.showFileHistory}
            {#if store.fileHistoryLoading}
                <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">Loading history...</span></div>
            {:else if store.fileHistory.length === 0}
                <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">No changes for this file</span></div>
            {:else}
                {#each store.fileHistory as commit (commit.change_id)}
                    <button
                        class="flex w-full items-center gap-2 px-4 py-3 text-xs hover:bg-accent transition-colors text-left border-b border-border/50"
                        onclick={() => store.toggleCommitDiff(commit.change_id)}
                    >
                        <span class="text-primary font-mono">{shortId(commit.change_id)}</span>
                        <span class="text-muted-foreground truncate flex-1">{commit.message}</span>
                        <span class="text-[10px] text-muted-foreground/70 shrink-0">{relativeTime(commit.timestamp)}</span>
                    </button>
                {/each}
            {/if}
        {:else if store.selectedFilePath && store.fileContent}
            <CodeView code={store.fileContent} filepath={store.selectedFilePath} />
        {:else if store.codeRepo}
            {#if store.codeLoading}
                <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">Loading...</span></div>
            {:else if !store.treeCache[""] || store.treeCache[""].length === 0}
                <div class="flex justify-center py-8"><span class="text-xs text-muted-foreground">Empty directory</span></div>
            {:else}
                <TreeNode path="" depth={0} />
            {/if}
        {:else}
            <div class="py-2">
                {#each store.orgs as orgNode (orgNode.org)}
                    <div class="flex items-center gap-2 px-4 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
                        <Building2 class="size-3.5 shrink-0" />
                        {orgNode.org}
                    </div>
                    {#each orgNode.repos as repoNode (repoNode.repo)}
                        <div class="flex items-center gap-2 pl-6 pr-4 py-1 text-sm font-medium text-foreground/80">
                            <FolderGit class="size-3.5 shrink-0 text-blue-400" />
                            {repoNode.repo}
                        </div>
                        {#each repoNode.bookmarks as bm (bm.bookmark)}
                            <button
                                class="flex w-full cursor-pointer items-center gap-3 pl-10 pr-4 py-2.5 text-sm hover:bg-accent border-b border-border/30"
                                onclick={() => store.openRepo(orgNode.org, repoNode.repo, bm.bookmark)}
                            >
                                <GitBranch class="size-4 text-muted-foreground shrink-0" />
                                <span>{bm.bookmark}</span>
                            </button>
                        {/each}
                    {/each}
                {/each}
                {#if store.orgs.length === 0}
                    <div class="px-4 py-12 text-sm text-muted-foreground text-center">No repositories.<br />Create a session first.</div>
                {/if}
            </div>
        {/if}
    </div>
</div>
