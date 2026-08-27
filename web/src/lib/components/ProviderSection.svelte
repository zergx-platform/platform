<script lang="ts">
import { Plus, X } from '@lucide/svelte'
import type {
  MdProvider,
  ModelInfo,
  ProviderInfo,
  ProviderModel,
} from '@zergx/schema'
import { MdProvidersSchema } from '@zergx/schema'
import { onMount } from 'svelte'
import { z } from 'zod'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import AddProviderForm from './AddProviderForm.svelte'
import ConfirmDialog from './ConfirmDialog.svelte'
import ModelEditModal from './ModelEditModal.svelte'
import ProviderCard from './ProviderCard.svelte'

let { compact = false }: { compact?: boolean } = $props()

let providers = $state<Record<string, ProviderInfo>>({})
let models = $state<ModelInfo[]>([])
let loading = $state(true)

let confirmDelete = $state<string | null>(null)
let confirmBusy = $state(false)

let mdProviders = $state<Record<string, MdProvider>>({})
let mdLoading = $state(false)
let mdError = $state('')
let mdProviderList = $derived(
  Object.values(mdProviders).sort((a, b) =>
    (a.name ?? a.id ?? '').localeCompare(b.name ?? b.id ?? ''),
  ),
)

// Edit model modal
let editProviderId = $state('')
let editModelIdx = $state(-1)
let editModel = $state<ProviderModel>({ id: '', name: '' })

onMount(async () => {
  await refreshProviders()
  loadModelsDev()
})

async function refreshProviders() {
  const r = await api.providers.list()
  if (r.isOk()) providers = r.value
  const mr = await api.models.list()
  if (mr.isOk()) models = mr.value
  loading = false
}

async function loadModelsDev() {
  const cached = localStorage.getItem('zergx-models-dev')
  const ModelCacheSchema = z.preprocess(
    v => (typeof v === 'string' ? JSON.parse(v) : v),
    z.object({ data: MdProvidersSchema, ts: z.number() }),
  )
  if (cached) {
    const parsed = ModelCacheSchema.safeParse(cached)
    if (parsed.success) {
      const { data, ts } = parsed.data
      if (Date.now() - ts < 3600_000) {
        mdProviders = data
        return
      }
    }
  }
  mdLoading = true
  mdError = ''
  try {
    const r = await fetch('https://models.opencode.ai/api.json')
    if (!r.ok) throw new Error(`${r.status}`)
    const raw = await r.json()
    const parsed = MdProvidersSchema.safeParse(raw)
    if (!parsed.success) throw new Error('Invalid response format')
    mdProviders = parsed.data
    localStorage.setItem(
      'zergx-models-dev',
      JSON.stringify({ data: mdProviders, ts: Date.now() }),
    )
  } catch (e: unknown) {
    mdError = e instanceof Error ? e.message : 'Failed'
  }
  mdLoading = false
}

async function deleteProvider(id: string) {
  confirmDelete = id
}

async function runConfirmDelete() {
  if (!confirmDelete) return
  confirmBusy = true
  try {
    await api.providers.delete(confirmDelete)
    await refreshProviders()
  } finally {
    confirmBusy = false
  }
}
function openModelEdit(pid: string, idx: number, m: ProviderModel) {
  editProviderId = pid
  editModelIdx = idx
  editModel = { ...m }
}

function saveModelEdit() {
  if (!editProviderId || editModelIdx < 0) return
  const p = providers[editProviderId]
  if (!p) return
  const updated = [...p.models]
  updated[editModelIdx] = { ...editModel }
  api.providers
    .register({ ...p, models: updated })
    .then(() => refreshProviders())
  editProviderId = ''
  editModelIdx = -1
  editModel = { id: '', name: '' }
}

function cancelModelEdit() {
  editProviderId = ''
  editModelIdx = -1
  editModel = { id: '', name: '' }
}
</script>

<div>
    {#if loading}
        <div class="text-xs text-muted-foreground py-2">Loading...</div>
    {:else}
        <div class="space-y-1">
            {#each Object.entries(providers) as [id, p] (id)}
                <ProviderCard providerId={id} provider={p} onEditModel={openModelEdit} onDelete={deleteProvider} />
            {/each}
            {#if Object.keys(providers).length === 0}
                <div class="text-xs text-muted-foreground py-2">No providers. Add one to get started.</div>
            {/if}
        </div>
        <AddProviderForm {mdProviderList} {mdProviders} onRegistered={refreshProviders} />
        {#if compact && mdError}
            <span class="text-[10px] text-yellow-500">models.dev: {mdError}</span>
        {/if}
    {/if}
</div>

{#if editProviderId && editModelIdx >= 0}
    <ModelEditModal {editModel} onSave={saveModelEdit} onCancel={cancelModelEdit} />
{/if}

<ConfirmDialog
    open={!!confirmDelete}
    title="Delete provider"
    description={confirmDelete ? `Delete provider <strong>${confirmDelete}</strong>?` : ''}
    busy={confirmBusy}
    onConfirm={runConfirmDelete}
    onClose={() => { confirmDelete = null; confirmBusy = false }}
/>
