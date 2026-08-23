<script lang="ts">
import { Check, FlaskConical, Loader2, Plus, Trash2, X } from '@lucide/svelte'
import { onMount } from 'svelte'
import type { ModelInfo, ProviderInfo } from '$lib/api'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import ConfirmDialog from './ConfirmDialog.svelte'

let { onclose }: { onclose: () => void } = $props()

let providers = $state<Record<string, ProviderInfo>>({})
let loading = $state(true)
let showAdd = $state(false)

// Add form state
let addId = $state('')
let addApiType = $state('openai')
let addUrl = $state('')
let addKey = $state('')
let addModels = $state('')
let addSaving = $state(false)
let addTesting = $state(false)
let testResult = $state<{
  ok: boolean
  models?: string[]
  detail?: string
  error?: string
} | null>(null)
let addError = $state('')

let confirmDelete = $state<string | null>(null)
let confirmBusy = $state(false)

// Model search for existing session
let allModels = $state<ModelInfo[]>([])

const API_TYPES = ['openai', 'anthropic']

const BASE_URL_DEFAULTS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
}

onMount(() => {
  refresh()
})

async function refresh() {
  loading = true
  const pr = await api.providers.list()
  providers = pr.isOk() ? pr.value : {}
  const mr = await api.models.list()
  allModels = mr.isOk() ? mr.value : []
  loading = false
}

async function handleDelete(pid: string) {
  confirmDelete = pid
}

async function runConfirmDelete() {
  if (!confirmDelete) return
  confirmBusy = true
  try {
    const r = await api.providers.delete(confirmDelete)
    if (r.isOk()) await refresh()
  } finally {
    confirmBusy = false
  }
}

async function handleTest() {
  if (!addUrl || !addKey) {
    addError = 'URL and API Key required'
    return
  }
  addTesting = true
  testResult = null
  const r = await api.providers.test({
    api_type: addApiType,
    base_url: addUrl,
    api_key: addKey,
  })
  testResult = r.isOk() ? r.value : { ok: false, error: 'Network error' }
  addTesting = false
}

function useTestModels() {
  if (testResult?.models) addModels = testResult.models.join(', ')
}

async function handleRegister() {
  if (!addId || !addUrl || !addKey) {
    addError = 'Provider ID, URL, and API Key required'
    return
  }
  addSaving = true
  addError = ''
  const modelList = addModels
    .split(',')
    .map(m => m.trim())
    .filter(Boolean)
    .map(id => ({ id, name: id }))
  const r = await api.providers.register({
    provider_id: addId,
    api_type: addApiType,
    base_url: addUrl,
    api_key: addKey,
    models: modelList,
  })
  if (r.isOk()) {
    showAdd = false
    resetForm()
    await refresh()
  } else addError = r.error
  addSaving = false
}

function resetForm() {
  addId = ''
  addApiType = 'openai'
  addUrl = ''
  addKey = ''
  addModels = ''
  testResult = null
  addError = ''
}

function startAdd() {
  showAdd = true
  addUrl = BASE_URL_DEFAULTS[addApiType]
}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center">
    <div class="absolute inset-0 bg-black/60" onclick={onclose} role="presentation"></div>
    <div class="relative z-10 w-full max-w-xl max-h-[85vh] rounded-lg border bg-card shadow-lg flex flex-col">
        <div class="flex items-center justify-between border-b px-4 py-3 shrink-0">
            <h2 class="text-sm font-semibold">Settings</h2>
            <Button variant="ghost" size="icon" onclick={onclose}><X class="size-4" /></Button>
        </div>
        <div class="flex-1 overflow-y-auto p-4 space-y-6">
            {#if loading}
                <div class="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 class="size-3.5 animate-spin" /> Loading...
                </div>
            {:else}
                <!-- Providers -->
                <div>
                    <div class="flex items-center justify-between mb-3">
                        <h3 class="text-sm font-medium">Providers</h3>
                        <Button variant="outline" size="sm" onclick={startAdd}>
                            <Plus class="size-3.5 mr-1" /> Add
                        </Button>
                    </div>

                    {#if showAdd}
                        <div class="border rounded-md p-3 mb-3 space-y-2">
                            <div class="flex gap-2">
                                <div class="flex-1">
                                    <label class="text-xs font-medium" for="addId">Provider ID</label>
                                    <input id="addId" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={addId} placeholder="e.g. my-openai" />
                                </div>
                                <div>
                                    <label class="text-xs font-medium" for="addType">API Type</label>
                                    <select id="addType" class="mt-0.5 rounded border border-input bg-background px-2 py-1 text-xs" bind:value={addApiType}>
                                        {#each API_TYPES as t}<option value={t}>{t}</option>{/each}
                                    </select>
                                </div>
                            </div>
                            <div>
                                <label class="text-xs font-medium" for="addUrl">Base URL</label>
                                <input id="addUrl" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs font-mono" bind:value={addUrl} placeholder="https://api.openai.com/v1" />
                            </div>
                            <div>
                                <label class="text-xs font-medium" for="addKey">API Key</label>
                                <input id="addKey" type="password" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={addKey} placeholder="sk-..." />
                            </div>
                            <div>
                                <label class="text-xs font-medium" for="addModels">Models (comma-separated IDs)</label>
                                <input id="addModels" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={addModels} placeholder="gpt-4o, gpt-4o-mini" />
                            </div>
                            {#if testResult}
                                <div class="rounded p-2 text-xs {testResult.ok ? 'bg-green-500/10 text-green-600' : 'bg-red-500/10 text-red-600'}">
                                    {testResult.ok ? testResult.detail || "OK" : testResult.error || "Failed"}
                                    {#if testResult.models && testResult.models.length > 0}
                                        <button class="ml-2 underline" onclick={useTestModels}>Use {testResult.models.length} models</button>
                                    {/if}
                                </div>
                            {/if}
                            {#if addError}<div class="text-xs text-red-500">{addError}</div>{/if}
                            <div class="flex gap-2">
                                <Button variant="outline" size="sm" onclick={handleTest} disabled={addTesting}>
                                    {#if addTesting}<Loader2 class="size-3 animate-spin mr-1" />{:else}<FlaskConical class="size-3 mr-1" />{/if}
                                    Test
                                </Button>
                                <Button size="sm" onclick={handleRegister} disabled={addSaving}>
                                    {#if addSaving}<Loader2 class="size-3 animate-spin mr-1" />{:else}<Check class="size-3 mr-1" />{/if}
                                    Register
                                </Button>
                                <Button variant="ghost" size="sm" onclick={() => { showAdd = false; resetForm() }}>Cancel</Button>
                            </div>
                        </div>
                    {/if}

                    {#each Object.entries(providers) as [pid, p] (pid)}
                        <div class="flex items-center gap-2 border border-border rounded-md px-3 py-2 mb-1.5 text-xs">
                            <div class="flex-1 min-w-0">
                                <div class="font-medium">{pid} <span class="text-[10px] text-muted-foreground">({p.api_type})</span></div>
                                <div class="text-[10px] text-muted-foreground truncate">{p.base_url}</div>
                                <div class="text-[10px] text-muted-foreground">{p.models.length} models</div>
                            </div>
                            <Button variant="ghost" size="icon" class="text-red-500 hover:text-red-600" onclick={() => handleDelete(pid)}>
                                <Trash2 class="size-3.5" />
                            </Button>
                        </div>
                    {/each}
                    {#if Object.keys(providers).length === 0 && !showAdd}
                        <p class="text-xs text-muted-foreground">No providers registered</p>
                    {/if}
                </div>

                <!-- Models -->
                <div>
                    <h3 class="text-sm font-medium mb-2">Available Models</h3>
                    {#if allModels.length > 0}
                        <div class="space-y-1">
                            {#each allModels as m (m.provider_id + "/" + m.id)}
                                <div class="text-xs flex items-center gap-2 px-2 py-1 rounded hover:bg-accent">
                                    <span class="font-mono">{m.id}</span>
                                    <span class="text-[10px] text-muted-foreground ml-auto">{m.provider_id}</span>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <p class="text-xs text-muted-foreground">No models available. Register a provider first.</p>
                    {/if}
                </div>
            {/if}
        </div>
    </div>
</div>

<ConfirmDialog
    open={!!confirmDelete}
    title="Delete provider"
    description={confirmDelete ? `Delete provider <strong>${confirmDelete}</strong>?` : ''}
    busy={confirmBusy}
    onConfirm={runConfirmDelete}
    onClose={() => { confirmDelete = null; confirmBusy = false }}
/>
