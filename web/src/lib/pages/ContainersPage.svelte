<script lang="ts">
import { onMount } from 'svelte'
import type { Deployment, Sandbox } from '@recoder-neo/schema'
import * as api from '$lib/api'
import {
  closeContainerUrl,
  onHashChange,
  openContainerUrl,
  router,
} from '$lib/router.svelte'
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import {
  ArrowLeft,
  Box,
  Globe,
  Loader2,
  Plus,
  RefreshCw,
  Server,
  Terminal,
  Trash2,
  X,
} from '@lucide/svelte'
import ConfirmDialog from '$lib/components/ConfirmDialog.svelte'
import ContainerWorkspace from '$lib/components/ContainerWorkspace.svelte'
import { Button } from '$lib/components/ui/button'
import * as Card from '$lib/components/ui/card'

let containers = $state<Sandbox[]>([])
let deployments = $state<Deployment[]>([])
let loading = $state(false)
let creating = $state(false)
let deleting = $state<string | null>(null)
let error = $state('')
let selectedTerminal = $derived(router.containers.containerId)

let confirmDelete = $state<Sandbox | null>(null)
let confirmBusy = $state(false)

// Deployments
let showDeploy = $state(false)
let deployName = $state('')
let deployImage = $state('')
let deployReplicas = $state(1)
let deployPort = $state(8080)
let deploySession = $state('')
let deploying = $state(false)
let deployError = $state('')
let confirmDeployDelete = $state<string | null>(null)
let deployDeleteBusy = $state(false)

// Create container dialog
let showCreate = $state(false)
let createImage = $state('')
let createError = $state('')

// Worker images (build)
let showBuildWorker = $state(false)
let buildBaseImage = $state('')
let buildError = $state('')
let building = $state(false)

onMount(() => {
  loadAll()
  return onHashChange(() => {})
})

async function loadAll() {
  loading = true
  error = ''
  const [cr, dr] = await Promise.all([
    api.containers.list(),
    api.containers.deployments(),
  ])
  containers = cr.isOk() ? cr.value : []
  deployments = dr.isOk() ? dr.value : []
  if (cr.isErr() && dr.isErr()) error = cr.error
  loading = false
}

function openCreate() {
  createImage = ''
  createError = ''
  showCreate = true
}

function openBuildWorker() {
  buildBaseImage = ''
  buildError = ''
  showBuildWorker = true
}

async function onBuildWorker() {
  if (!buildBaseImage.trim()) return
  building = true
  buildError = ''
  const r = await api.containers.buildImage({
    raw: true,
    dockerfile: `FROM ${buildBaseImage}\n`,
    tag: 'worker',
  })
  if (r.isOk()) {
    showBuildWorker = false
    await loadAll()
  } else {
    buildError = r.error
  }
  building = false
}

async function onCreateContainer() {
  createError = 'Sandboxes are created by the agent on first use. Use the ops-extension UI to deploy services.'
  showCreate = false
}

function openDeploy() {
  deployName = ''
  deployImage = ''
  deployReplicas = 1
  deployPort = 8080
  deploySession = store.activeSessionId ?? ''
  deployError = ''
  showDeploy = true
}

async function onDeploy() {
  if (!deployName.trim() || !deployImage.trim()) {
    deployError = 'Name and image required'
    return
  }
  deploying = true
  deployError = ''
  const r = await api.containers.deploy({
    name: deployName,
    image: deployImage,
    replicas: deployReplicas,
    port: deployPort,
    session: deploySession || undefined,
  })
  if (r.isOk()) {
    showDeploy = false
    await loadAll()
  } else {
    deployError = r.error
  }
  deploying = false
}

async function destroyDeployment(name: string) {
  deployDeleteBusy = true
  const r = await api.containers.destroyDeployment(name)
  if (r.isOk()) await loadAll()
  else error = r.error
  deployDeleteBusy = false
}

async function destroyContainer(session: string) {
  deleting = session
  error = ''
  const r = await api.containers.destroySandbox(session)
  if (r.isOk()) {
    if (selectedTerminal === session) closeContainerUrl()
    await loadAll()
  } else {
    error = r.error
  }
  deleting = null
}

async function runConfirmDelete() {
  if (!confirmDelete) return
  confirmBusy = true
  try {
    await destroyContainer(confirmDelete.session)
  } finally {
    confirmBusy = false
  }
}

function termContainer(): Sandbox | undefined {
  if (!selectedTerminal) return undefined
  return containers.find(x => x.session === selectedTerminal)
}
</script>

<div class="flex flex-col h-full">
    {#if selectedTerminal}
        <div class="shrink-0 border-b border-border bg-card px-4 py-2.5 flex items-center gap-2">
            <Button variant="ghost" size="sm" class="gap-1.5" onclick={closeContainerUrl}>
                <ArrowLeft class="size-4" /> Back
            </Button>
            <span class="text-muted-foreground/40">/</span>
            <span class="text-sm font-mono">{termContainer()?.session || selectedTerminal}</span>
        </div>
        <div class="flex-1 min-h-0">
            <ContainerWorkspace
                containerId={selectedTerminal}
                containerName={termContainer()?.session || selectedTerminal}
                onclose={closeContainerUrl}
            />
        </div>
    {:else}
        <!-- Header -->
        <div class="shrink-0 border-b border-border bg-card px-4 sm:px-6 py-4">
            <div class="flex items-center justify-between gap-4 flex-wrap">
                <div class="flex items-center gap-2.5">
                    <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                        <Box class="size-5" />
                    </div>
                    <div>
                        <h1 class="text-lg font-semibold leading-tight">Containers</h1>
                        <p class="text-xs text-muted-foreground">Standalone worker pods — pure command sandbox</p>
                    </div>
                </div>
                <div class="flex items-center gap-2">
                    <Button variant="outline" size="sm" onclick={openDeploy}>
                        <Globe class="size-3.5" />
                        <span class="hidden sm:inline">Deploy Service</span>
                    </Button>
                    <Button variant="outline" size="sm" onclick={openBuildWorker}>
                        <Server class="size-3.5" />
                        <span class="hidden sm:inline">Build Image</span>
                    </Button>
                    <Button variant="outline" size="sm" onclick={loadAll} disabled={loading}>
                        <RefreshCw class="size-3.5 {loading ? 'animate-spin' : ''}" />
                        <span class="hidden sm:inline">Refresh</span>
                    </Button>
                    <Button size="sm" onclick={openCreate}>
                        <Plus class="size-3.5" />
                        <span class="hidden sm:inline">New Container</span>
                        <span class="sm:hidden">New</span>
                    </Button>
                </div>
            </div>
        </div>

        <div class="flex-1 min-h-0 overflow-auto px-4 sm:px-6 py-5" style="padding-bottom: max(1.25rem, env(safe-area-inset-bottom));">
            <div class="mx-auto max-w-6xl space-y-4">
                {#if error}
                    <div class="text-sm text-destructive bg-destructive/10 rounded-md px-3 py-2">{error}</div>
                {/if}

                <!-- Deployments -->
                <div class="rounded-lg border border-border bg-card p-4">
                    <div class="flex items-center justify-between mb-3">
                        <div>
                            <h2 class="text-sm font-semibold">Deployments</h2>
                            <p class="text-xs text-muted-foreground">Services deployed from sandboxes or images.</p>
                        </div>
                        <Button variant="outline" size="sm" onclick={openDeploy}>
                            <Globe class="size-3.5" />
                            <span>Deploy</span>
                        </Button>
                    </div>
                    {#if deployments.length === 0}
                        <p class="text-xs text-muted-foreground py-1">No deployments yet.</p>
                    {:else}
                        <div class="space-y-2">
                            {#each deployments as d (d.name)}
                                <div class="flex items-center gap-3 rounded-md border border-border/60 px-3 py-2 text-xs">
                                    <Globe class="size-3.5 text-primary shrink-0" />
                                    <div class="flex-1 min-w-0">
                                        <div class="font-mono font-medium truncate">{d.name}</div>
                                        <div class="text-muted-foreground truncate">{d.image}</div>
                                    </div>
                                    <span class="shrink-0 px-2 py-0.5 rounded-full text-[10px] bg-muted text-muted-foreground">{d.ready}/{d.replicas} ready</span>
                                    {#if d.ports.length > 0}
                                        <span class="shrink-0 text-muted-foreground font-mono">:{d.ports.join(',:')}</span>
                                    {/if}
                                    <span class="shrink-0 text-muted-foreground text-[10px]">{d.age}</span>
                                    <Button variant="ghost" size="icon" class="size-6 shrink-0 text-destructive hover:bg-destructive/10" disabled={deployDeleteBusy}
                                        onclick={() => destroyDeployment(d.name)}>
                                        <Trash2 class="size-3.5" />
                                    </Button>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>

                <!-- Container list -->
                {#if loading && containers.length === 0}
                    <div class="flex items-center justify-center py-12 text-sm text-muted-foreground">
                        <Loader2 class="size-4 animate-spin mr-2" /> Loading...
                    </div>
                {:else if containers.length === 0}
                    <div class="flex flex-col items-center justify-center py-8 text-center border border-dashed rounded-lg">
                        <Box class="size-6 text-muted-foreground/40 mb-2" />
                        <p class="text-sm text-muted-foreground">No containers running.</p>
                    </div>
                {:else}
                    <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                        {#each containers as c (c.session)}
                            <Card.Root>
                                <Card.Header class="pb-2">
                                    <div class="flex items-center justify-between gap-2">
                                        <Card.Title class="text-sm font-mono truncate">{c.pod_name}</Card.Title>
                                        <div class="flex items-center gap-1.5 shrink-0">
<span class="px-2 py-0.5 rounded-full text-[10px] font-medium bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400">sandbox</span>
                                            <span class="flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-medium
                                                {c.status === 'running' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                                                 c.status === 'starting' ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400' :
                                                 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'}">
                                                <span class="size-1.5 rounded-full {c.status === 'running' ? 'bg-green-500' : c.status === 'starting' ? 'bg-amber-500' : 'bg-red-500'}"></span>
                                                {c.status}
                                            </span>
                                        </div>
                                    </div>
                                    <Card.Description class="text-xs">
<span class="font-mono break-all">{c.session}</span>
                                    </Card.Description>
                                </Card.Header>
                                <Card.Content class="text-xs text-muted-foreground space-y-1.5 pt-1">
                                    {#if c.worker_url}
                                        <div class="flex items-start gap-1.5">
                                            <Server class="size-3 shrink-0 mt-0.5" />
                                            <span class="break-all font-mono">{c.worker_url}</span>
                                        </div>
                                    {/if}
                                </Card.Content>
                                <Card.Footer class="pt-2 gap-2">
                                    <Button size="sm" variant="outline" class="flex-1" disabled={c.status !== 'running'} onclick={() => openContainerUrl(c.session)}>
                                        <Terminal class="size-3.5" /> Terminal
                                    </Button>
                                    <Button size="sm" variant="ghost" class="text-destructive hover:bg-destructive/10"
                                        disabled={deleting === c.session}
                                        onclick={() => { confirmDelete = c }}>
                                        {#if deleting === c.session}<Loader2 class="size-3.5 animate-spin" />{:else}<Trash2 class="size-3.5" />{/if}
                                    </Button>
                                </Card.Footer>
                            </Card.Root>
                        {/each}
                    </div>
                {/if}
            </div>
        </div>
    {/if}
</div>

<ConfirmDialog
    open={!!confirmDelete}
    title="Destroy container"
    description={confirmDelete ? `Destroy <strong>${confirmDelete.pod_name || confirmDelete.session}</strong>? This stops and removes the worker pod.` : ''}
    confirmText="Destroy"
    busy={confirmBusy}
    onConfirm={runConfirmDelete}
    onClose={() => { confirmDelete = null; confirmBusy = false }}
/>

<!-- Build Worker Image Modal -->
{#if showBuildWorker}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/40" role="presentation" onclick={() => showBuildWorker = false}>
        <div class="bg-card border border-border rounded-lg shadow-xl w-full max-w-md mx-4 space-y-3 p-4" role="dialog" tabindex="-1" aria-label="Build image" onclick={(e) => e.stopPropagation()}>
            <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold">Build Image</h3>
                <Button variant="ghost" size="icon" class="size-6" onclick={() => showBuildWorker = false}><X class="size-3.5" /></Button>
            </div>

            <div>
                <label class="text-xs font-medium text-muted-foreground" for="bwi-base">Base Image</label>
                <input id="bwi-base" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs font-mono"
                    placeholder="debian:trixie-slim" bind:value={buildBaseImage} />
                <p class="text-[10px] text-muted-foreground mt-1">Builds a minimal image from this base and pushes it to the registry.</p>
            </div>

            {#if buildError}
                <p class="text-xs text-destructive">{buildError}</p>
            {/if}

            <div class="flex items-center gap-2 pt-1">
                <Button size="sm" onclick={onBuildWorker} disabled={building || !buildBaseImage.trim()}>
                    {#if building}<Loader2 class="size-3.5 animate-spin mr-1" />{/if}Build
                </Button>
                <Button variant="ghost" size="sm" onclick={() => showBuildWorker = false}>Cancel</Button>
            </div>
        </div>
    </div>
{/if}

<!-- Deploy Service Modal -->
{#if showDeploy}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/40" role="presentation" onclick={() => showDeploy = false}>
        <div class="bg-card border border-border rounded-lg shadow-xl w-full max-w-md mx-4 space-y-3 p-4" role="dialog" tabindex="-1" aria-label="Deploy service" onclick={(e) => e.stopPropagation()}>
            <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold">Deploy Service</h3>
                <Button variant="ghost" size="icon" class="size-6" onclick={() => showDeploy = false}><X class="size-3.5" /></Button>
            </div>

            <div>
                <label class="text-xs font-medium text-muted-foreground" for="dep-name">Name</label>
                <input id="dep-name" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs font-mono"
                    placeholder="my-service" bind:value={deployName} />
            </div>
            <div>
                <label class="text-xs font-medium text-muted-foreground" for="dep-image">Image</label>
                <input id="dep-image" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs font-mono"
                    placeholder="nginx:latest" bind:value={deployImage} />
            </div>
            <div class="grid grid-cols-2 gap-3">
                <div>
                    <label class="text-xs font-medium text-muted-foreground" for="dep-replicas">Replicas</label>
                    <input id="dep-replicas" type="number" min="1" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs" bind:value={deployReplicas} />
                </div>
                <div>
                    <label class="text-xs font-medium text-muted-foreground" for="dep-port">Port</label>
                    <input id="dep-port" type="number" min="1" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs" bind:value={deployPort} />
                </div>
            </div>
            <div>
                <label class="text-xs font-medium text-muted-foreground" for="dep-session">Session (optional)</label>
                <input id="dep-session" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs font-mono"
                    placeholder="org:repo:branch" bind:value={deploySession} />
            </div>

            {#if deployError}
                <p class="text-xs text-destructive">{deployError}</p>
            {/if}

            <div class="flex items-center gap-2 pt-1">
                <Button size="sm" onclick={onDeploy} disabled={deploying || !deployName.trim() || !deployImage.trim()}>
                    {#if deploying}<Loader2 class="size-3.5 animate-spin mr-1" />{/if}Deploy
                </Button>
                <Button variant="ghost" size="sm" onclick={() => showDeploy = false}>Cancel</Button>
            </div>
        </div>
    </div>
{/if}

<!-- Create Container Modal -->
{#if showCreate}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/40" role="presentation" onclick={() => showCreate = false}>
        <div class="bg-card border border-border rounded-lg shadow-xl w-full max-w-md mx-4 space-y-3 p-4" role="dialog" tabindex="-1" aria-label="Create container" onclick={(e) => e.stopPropagation()}>
            <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold">Create Container</h3>
                <Button variant="ghost" size="icon" class="size-6" onclick={() => showCreate = false}><X class="size-3.5" /></Button>
            </div>

            <div>
                <label class="text-xs font-medium text-muted-foreground" for="cc-image">Image</label>
                <input id="cc-image" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs font-mono"
                    placeholder="docker.io/library/debian:trixie" bind:value={createImage} />
                <p class="text-[10px] text-muted-foreground mt-1">Image is pulled by recore from the configured registry.</p>
            </div>

            {#if createError}
                <p class="text-xs text-destructive">{createError}</p>
            {/if}

            <div class="flex items-center gap-2 pt-1">
                <Button size="sm" onclick={onCreateContainer} disabled={creating || !createImage.trim()}>
                    {#if creating}<Loader2 class="size-3.5 animate-spin mr-1" />{/if}Create
                </Button>
                <Button variant="ghost" size="sm" onclick={() => showCreate = false}>Cancel</Button>
            </div>
        </div>
    </div>
{/if}