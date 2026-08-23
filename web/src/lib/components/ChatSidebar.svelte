<script lang="ts">
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import {
  Building2,
  ChevronDown,
  ChevronRight,
  Clock,
  Download,
  FolderGit,
  GitFork,
  Plus,
  Trash2,
  X,
} from '@lucide/svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import * as Collapsible from '$lib/components/ui/collapsible'
import { cn } from '$lib/utils'
import ConfirmDialog from './ConfirmDialog.svelte'
import NewItemDialog from './NewItemDialog.svelte'

const recentSessions = $derived.by(() => {
  return [...store.sessions].sort((a, b) => {
    const at = a.updated_at ? Date.parse(a.updated_at) || 0 : 0
    const bt = b.updated_at ? Date.parse(b.updated_at) || 0 : 0
    return bt - at
  })
})

let orgDialogOpen = $state(false)
let repoDialogOpen = $state(false)
let repoTargetOrg = $state('')
let cloneDialogOpen = $state(false)
let cloneTargetOrg = $state('')
let cloneUrl = $state('')
let cloneBusy = $state(false)
let cloneError = $state('')
let cloneName = $state('')
let cloneToken = $state('')
let cloneRev = $state('')

interface ConfirmState {
  title: string
  description: string
  action: () => Promise<void> | void
}
let confirm = $state<ConfirmState | null>(null)
let confirmBusy = $state(false)
let adopting = $state<string | null>(null)
let adoptError = $state('')

async function openBookmark(org: string, repo: string, branch: string, sessionId: string | null) {
  if (sessionId) {
    store.pickSession(sessionId)
    return
  }
  // Unbound (orphan) bookmark: adopt it — repo-extension creates the
  // workspace session and binds it — then open the chat.
  const key = `${org}/${repo}/${branch}`
  if (adopting) return
  adopting = key
  adoptError = ''
  const r = await api.repos.adoptSession(org, repo, branch)
  adopting = null
  if (r.isErr()) {
    adoptError = `${key}: ${r.error}`
    return
  }
  await Promise.all([store.refreshRepos(), store.refreshSessions()])
  store.pickSession(r.value.session_name)
}

async function runConfirm() {
  if (!confirm) return
  confirmBusy = true
  try {
    await confirm.action()
  } finally {
    confirmBusy = false
  }
}

function submitClone() {
  const name = cloneName.trim()
  if (!name || !cloneUrl.trim() || cloneBusy) return
  void cloneRepo(name)
    .then(() => {
      cloneName = ''
      cloneDialogOpen = false
    })
    .catch(() => {})
}

async function createOrg(name: string) {
  const r = await api.repos.ensureOrg(name)
  if (r.isOk()) store.refreshRepos()
}

async function createRepo(name: string) {
  if (!repoTargetOrg) return
  const r = await api.repos.ensureRepo(repoTargetOrg, name)
  if (r.isOk()) {
    store.refreshRepos()
    store.refreshSessions()
  }
}

async function cloneRepo(name: string) {
  if (!cloneTargetOrg) return
  const url = cloneUrl.trim()
  if (!url) return
  cloneBusy = true
  cloneError = ''
  const token = cloneToken.trim() || undefined
  const rev = cloneRev.trim() || undefined
  const r = await api.repos.cloneRepo(cloneTargetOrg, name, url, token, rev)
  cloneBusy = false
  if (r.isErr()) {
    cloneError = r.error
    throw new Error(r.error)
  }
  cloneUrl = ''
  cloneToken = ''
  cloneRev = ''
  store.refreshRepos()
  store.refreshSessions()
}

function deleteOrg(org: string, e: Event) {
  e.stopPropagation()
  confirm = {
    title: 'Delete organization',
    description: `Delete organization <strong>${org}</strong>? This removes all its repos and sessions.`,
    action: () => store.deleteOrg(org),
  }
}
</script>

<div class="flex flex-col h-full overflow-hidden">
    <div class="shrink-0 px-3 py-2 border-b border-border">
        <Button
            variant="outline"
            class="w-full justify-start text-xs text-muted-foreground"
            onclick={() => (orgDialogOpen = true)}
        >
            <Plus class="size-3 mr-1" />New organization
        </Button>
    </div>

    {#if adoptError}
        <div class="mx-2 my-1 rounded border border-destructive/40 bg-destructive/10 px-2 py-1 text-[10px] text-destructive flex items-start gap-1">
            <span class="flex-1 break-all">{adoptError}</span>
            <button class="shrink-0 font-medium" onclick={() => (adoptError = '')}>✕</button>
        </div>
    {/if}
    <div class="flex-1 overflow-y-auto px-2 py-1">
        <!-- Recent sessions: flat list (org/repo/branch), expanded by default -->
        {#if recentSessions.length > 0}
            <div class="mb-2">
                <div class="px-1 py-1 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Recent</div>
                {#each recentSessions as s (s.id)}
                    {@const isActive = store.activeSessionId === s.id}
                    <div
                        class={cn("flex items-center gap-1.5 rounded px-2 py-1 cursor-pointer hover:bg-accent group", isActive && "bg-accent")}
                        onclick={() => store.pickSession(s.id)}
                        onkeydown={(e) => { if (e.key === "Enter") store.pickSession(s.id) }}
                        role="button"
                        tabindex="0"
                    >
                        <Clock class="size-3 text-muted-foreground shrink-0" />
                        <span class={cn("text-[11px] flex-1 truncate", isActive && "font-medium")}>
                            {#if s.org}
                                <span class="font-mono text-muted-foreground">{s.org}/</span>{s.repo}<span class="text-muted-foreground">/{s.branch}</span>
                            {:else}
                                <span class="font-mono">{s.id}</span>
                            {/if}
                        </span>
                        {#if (s.unread ?? 0) > 0 && !isActive}
                            <span class="min-w-4 h-4 px-1 rounded-full bg-red-500 text-white text-[9px] flex items-center justify-center shrink-0 font-medium">{s.unread}</span>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}

        <!-- Browse all: tree view (org/repo/branch), collapsed by default -->
        <div class="mt-2">
            <div class="px-1 py-1 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">All repositories</div>
        {#each store.orgs as orgNode}
            <div class="mb-0.5">
                <Collapsible.Root>
                    <div class="flex items-center gap-1 rounded px-1 py-0.5 hover:bg-accent group">
                        <Collapsible.Trigger class="flex items-center gap-1 flex-1 min-w-0">
                            <ChevronRight class="size-3 text-muted-foreground shrink-0 group-data-[expanded]:hidden" />
                            <ChevronDown class="size-3 text-muted-foreground shrink-0 hidden group-data-[expanded]:block" />
                            <Building2 class="size-3 text-muted-foreground shrink-0" />
                            <span class="text-xs font-medium truncate">{orgNode.org}</span>
                        </Collapsible.Trigger>
                        <button
                            class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 rounded p-0.5 text-muted-foreground hover:text-foreground shrink-0"
                            title="New repo"
                            onclick={(e) => { e.stopPropagation(); repoTargetOrg = orgNode.org; repoDialogOpen = true }}
                        >
                            <Plus class="size-2.5" />
                        </button>
                        <button
                            class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 rounded p-0.5 text-muted-foreground hover:text-foreground shrink-0"
                            title="Clone repo into {orgNode.org}"
                            onclick={(e) => { e.stopPropagation(); cloneTargetOrg = orgNode.org; cloneDialogOpen = true }}
                        >
                            <Download class="size-2.5" />
                        </button>
                        {#if !orgNode.repos?.length}
                            <button class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 rounded p-0.5 text-muted-foreground hover:text-destructive shrink-0" onclick={(e) => deleteOrg(orgNode.org, e)}>
                                <Trash2 class="size-2.5" />
                            </button>
                        {/if}
                    </div>
                    <Collapsible.Content>
                        <div class="ml-3">
                            {#each orgNode.repos || [] as repoNode}
                                <div>
                                    <Collapsible.Root>
                                        <div class="flex items-center gap-1 rounded px-1 py-0.5 hover:bg-accent group">
                                            <Collapsible.Trigger class="flex items-center gap-1 flex-1 min-w-0">
                                                <ChevronRight class="size-2.5 text-muted-foreground shrink-0 group-data-[expanded]:hidden" />
                                                <ChevronDown class="size-2.5 text-muted-foreground shrink-0 hidden group-data-[expanded]:block" />
                                                <FolderGit class="size-3 text-blue-400 shrink-0" />
                                                <span class="text-[11px] truncate">{repoNode.repo}</span>
                                            </Collapsible.Trigger>
                                            <button class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 rounded p-0.5 text-muted-foreground hover:text-destructive shrink-0"
                                                onclick={(e) => { e.stopPropagation(); confirm = { title: 'Delete repo', description: `Delete repo <strong>${orgNode.org}/${repoNode.repo}</strong>? This removes all its sessions.`, action: () => store.deleteRepo(orgNode.org, repoNode.repo) } }}>
                                                <Trash2 class="size-2.5" />
                                            </button>
                                        </div>
                                        <Collapsible.Content>
                                            <div class="ml-3">
                                                {#each repoNode.bookmarks || [] as bm}
                                                    {@const session = bm.session}
                                                    {@const isActive = session && store.activeSessionId === session.session_id}
                                                    <div
                                                        class={cn("flex items-center gap-1 rounded px-1 py-0.5 cursor-pointer hover:bg-accent group", isActive && "bg-accent")}
                                                        onclick={() => { if (adopting === `${orgNode.org}/${repoNode.repo}/${bm.branch}`) return; void openBookmark(orgNode.org, repoNode.repo, bm.branch, session ? session.session_id : null) }}
                                                        onkeydown={(e) => { if (e.key === "Enter" && adopting !== `${orgNode.org}/${repoNode.repo}/${bm.branch}`) void openBookmark(orgNode.org, repoNode.repo, bm.branch, session ? session.session_id : null) }}
                                                        role="button"
                                                        tabindex="0"
                                                    >
                                                        <span class={cn("inline-block h-2 w-2 rounded-full shrink-0", session ? "bg-green-500" : "bg-muted-foreground/40")}></span>
                                                        <span class={cn("text-[11px] flex-1 truncate", isActive && "font-medium", !session && "text-muted-foreground")}>
                                                            {adopting === `${orgNode.org}/${repoNode.repo}/${bm.branch}` ? `${bm.branch}…` : bm.branch}
                                                        </span>
                                                        {#if session}
                                                            <button class="rounded p-0.5 text-muted-foreground hover:text-foreground opacity-100 sm:opacity-0 sm:group-hover:opacity-100 shrink-0"
                                                                onclick={(e) => { e.stopPropagation(); store.openFork(session.session_id) }}>
                                                                <GitFork class="size-2.5" />
                                                            </button>
                                                            <button class="rounded p-0.5 text-muted-foreground hover:text-destructive opacity-100 sm:opacity-0 sm:group-hover:opacity-100 shrink-0"
                                                                onclick={(e) => { e.stopPropagation(); confirm = { title: 'Delete session', description: `Delete session <strong>${bm.branch}</strong> in ${orgNode.org}/${repoNode.repo}?`, action: () => store.deleteBookmark(orgNode.org, repoNode.repo, bm.branch) } }}>
                                                                <Trash2 class="size-2.5" />
                                                            </button>
                                                        {/if}
                                                    </div>
                                                {/each}
                                            </div>
                                        </Collapsible.Content>
                                    </Collapsible.Root>
                                </div>
                            {/each}
                        </div>
                    </Collapsible.Content>
                </Collapsible.Root>
            </div>
        {/each}
        </div>
    </div>

    <NewItemDialog
        bind:open={orgDialogOpen}
        title="New organization"
        label="Organization name"
        placeholder="my-org"
        onSubmit={createOrg}
    />
    <NewItemDialog
        bind:open={repoDialogOpen}
        title="New repo in {repoTargetOrg}"
        label="Repo name"
        placeholder="my-repo"
        onSubmit={createRepo}
    />

    <ConfirmDialog
        open={!!confirm}
        title={confirm?.title ?? ''}
        description={confirm?.description ?? ''}
        busy={confirmBusy}
        onConfirm={runConfirm}
        onClose={() => { confirm = null; confirmBusy = false }}
    />

    {#if cloneDialogOpen}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div class="fixed inset-0 z-50" onclick={() => { if (!cloneBusy) { cloneDialogOpen = false; cloneError = '' } }} role="presentation"></div>
        <div class="fixed inset-0 z-50 flex items-center justify-center pointer-events-none">
            <div class="pointer-events-auto relative w-full max-w-sm rounded-lg border bg-card p-5 shadow-lg mx-4" role="dialog" aria-label="Clone repo">
                <h2 class="text-base font-semibold mb-3">Clone into {cloneTargetOrg}</h2>
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="clone-url">Git URL</label>
                <input
                    id="clone-url"
                    class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm mb-3"
                    placeholder="https://github.com/user/repo.git"
                    bind:value={cloneUrl}
                    oninput={() => (cloneError = '')}
                    disabled={cloneBusy}
                />
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="clone-name">Repo name</label>
                <!-- svelte-ignore a11y_autofocus -->
                <input
                    id="clone-name"
                    class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    placeholder="my-repo"
                    bind:value={cloneName}
                    onkeydown={(e) => { if (e.key === 'Enter' && cloneUrl.trim() && cloneName.trim() && !cloneBusy) submitClone() }}
                    disabled={cloneBusy}
                    autofocus
                />
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="clone-token">Access token (optional)</label>
                <input
                    id="clone-token"
                    type="password"
                    class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    placeholder="ghp_…"
                    bind:value={cloneToken}
                    disabled={cloneBusy}
                />
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="clone-rev">Branch / tag / commit (optional)</label>
                <input
                    id="clone-rev"
                    class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    placeholder="main"
                    bind:value={cloneRev}
                    disabled={cloneBusy}
                />
                {#if cloneError}
                    <p class="text-xs text-destructive mt-1.5 line-clamp-3">{cloneError}</p>
                {/if}
                <div class="flex gap-2 justify-end mt-4">
                    <Button variant="outline" size="sm" onclick={() => { if (!cloneBusy) { cloneDialogOpen = false; cloneError = '' } }}>Cancel</Button>
                    <Button size="sm" onclick={submitClone} disabled={cloneBusy || !cloneUrl.trim() || !cloneName.trim()}>
                        {cloneBusy ? 'Cloning...' : 'Clone'}
                    </Button>
                </div>
            </div>
        </div>
    {/if}
</div>
