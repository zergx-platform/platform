<script lang="ts">
import { ChevronDown, Pencil, Plus, Save, Trash2, X } from '@lucide/svelte'
import type { PresetInfo, ToolInfo } from '@rucoder/schema'
import { onMount } from 'svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import ConfirmDialog from './ConfirmDialog.svelte'

let presets = $state<PresetInfo[]>([])
let toolsList = $state<ToolInfo[]>([])
let loading = $state(true)

let confirmDelete = $state<PresetInfo | null>(null)
let confirmBusy = $state(false)

// Expanded preset id + its working copy. The working copy is keyed to the
// expanded id so switching presets always re-seeds the form.
let editingId = $state<string | null>(null)
let editData = $state<PresetInfo>({ id: '', system_prompt: '', tools: [], max_turns: 30 })

let showNewPreset = $state(false)
let newPresetId = $state('')

let availableTools = $derived(toolsList.map(t => t.name))
const seedPresetIds = new Set(['default', 'build', 'plan'])

onMount(async () => {
  const [psr, tlr] = await Promise.all([api.presets.list(), api.tools.list()])
  presets = psr.isOk() ? psr.value : []
  toolsList = tlr.isOk() ? tlr.value : []
  loading = false
})

function openPresetEdit(p: PresetInfo) {
  editingId = p.id
  editData = { ...p, tools: [...p.tools] }
}

function togglePresetEdit(p: PresetInfo) {
  if (editingId === p.id) {
    editingId = null
  } else {
    openPresetEdit(p)
  }
}

async function savePresetEdit() {
  const id = editingId
  if (!id) return
  editData = { ...editData, tools: editData.tools.filter(t => availableTools.includes(t)) }
  await api.presets.save(editData)
  const r = await api.presets.list()
  if (r.isOk()) presets = r.value
  editingId = null
}

async function deletePreset(p: PresetInfo) {
  confirmDelete = p
}

async function runConfirmDelete() {
  if (!confirmDelete) return
  confirmBusy = true
  try {
    await api.presets.delete(confirmDelete.id)
    const r = await api.presets.list()
    if (r.isOk()) presets = r.value
  } finally {
    confirmBusy = false
  }
}

async function createPreset() {
  const id = newPresetId.trim()
  if (!id) return
  await api.presets.save({ id, system_prompt: '', tools: [], max_turns: 30 })
  const r = await api.presets.list()
  if (r.isOk()) presets = r.value
  newPresetId = ''
  showNewPreset = false
}

function toggleTool(tool: string) {
  if (editData.tools.includes(tool)) {
    editData.tools = editData.tools.filter(t => t !== tool)
  } else {
    editData.tools = [...editData.tools, tool]
  }
}
</script>

{#if loading}
    <div class="text-xs text-muted-foreground py-2">Loading...</div>
{:else}
    <div class="flex items-center justify-between mb-3">
        <h3 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Presets</h3>
        <Button variant="ghost" size="icon-sm" onclick={() => { newPresetId = ""; showNewPreset = true }}>
            <Plus class="size-3.5" />
        </Button>
    </div>
    {#if showNewPreset}
        <div class="flex items-center gap-2 mb-2 p-2 border border-dashed border-border rounded-md">
            <input type="text" class="flex-1 rounded-md border border-input bg-background px-2 py-1 text-xs" placeholder="preset id..." bind:value={newPresetId} onkeydown={(e) => e.key === "Enter" && createPreset()} />
            <Button size="sm" variant="outline" onclick={createPreset} disabled={!newPresetId.trim()}>Create</Button>
            <Button size="sm" variant="ghost" onclick={() => { showNewPreset = false; newPresetId = "" }}><X class="size-3" /></Button>
        </div>
    {/if}
    <div class="space-y-1">
        {#each presets as p (p.id)}
            <div class="rounded-md border border-border bg-card overflow-hidden">
                <div class="flex items-center justify-between px-3 py-2 text-sm cursor-pointer hover:bg-accent/50" onclick={() => togglePresetEdit(p)} role="button" tabindex="0" onkeydown={(e) => e.key === "Enter" && togglePresetEdit(p)}>
                    <div class="flex items-center gap-2">
                        <span class="font-medium">{p.id}</span>
                        <span class="text-[10px] text-muted-foreground">turns:{p.max_turns}</span>
                        <span class="text-[10px] text-muted-foreground">tools:{p.tools.length}</span>
                    </div>
                    <div class="flex items-center gap-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="presentation">
                        {#if !seedPresetIds.has(p.id)}
                            <Button variant="ghost" size="icon-sm" onclick={() => deletePreset(p)}><Trash2 class="size-3 text-destructive" /></Button>
                        {/if}
                        <ChevronDown class="size-3 text-muted-foreground transition-transform {editingId === p.id ? 'rotate-180' : ''}" />
                    </div>
                </div>

                {#if editingId === p.id}
                    <!-- #key re-mounts the whole editor whenever the expanded preset id changes -->
                    {#key editingId}
                        <div class="px-3 py-3 border-t border-border space-y-3 bg-card/50">
                            <div>
                                <label class="text-xs font-semibold text-muted-foreground block mb-1" for="pe-sys-prompt-{p.id}">System Prompt</label>
                                <textarea id="pe-sys-prompt-{p.id}" class="w-full rounded-md border border-input bg-background px-3 py-2 text-xs font-mono min-h-[80px] resize-y"
                                    bind:value={editData.system_prompt}></textarea>
                            </div>

                            <div>
                                <label class="text-xs font-semibold text-muted-foreground block mb-1" for="pe-max-turns-{p.id}">Max Turns</label>
                                <input id="pe-max-turns-{p.id}" type="number" min="1" max="200" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                                    bind:value={editData.max_turns} />
                            </div>

                            <div>
                                <span class="text-xs font-semibold text-muted-foreground block mb-1" id="pe-tools-label-{p.id}">Tools</span>
                                <div class="flex flex-wrap gap-1" role="group" aria-labelledby="pe-tools-label-{p.id}">
                                    {#each availableTools as t (t)}
                                        <button
                                            type="button"
                                            class="px-2 py-0.5 rounded text-[10px] border {editData.tools.includes(t) ? 'bg-primary text-primary-foreground border-primary' : 'bg-background border-border'}"
                                            onclick={() => toggleTool(t)}
                                        >{t}</button>
                                    {/each}
                                </div>
                            </div>

                            <div class="flex justify-end gap-1">
                                <Button size="sm" variant="outline" onclick={savePresetEdit}><Save class="size-3 mr-1" /> Save</Button>
                                <Button size="sm" variant="ghost" onclick={() => (editingId = null)}><X class="size-3" /></Button>
                            </div>
                        </div>
                    {/key}
                {/if}
            </div>
        {/each}
        {#if presets.length === 0}
            <div class="text-xs text-muted-foreground py-2">No presets.</div>
        {/if}
    </div>
{/if}

<ConfirmDialog
    open={!!confirmDelete}
    title="Delete preset"
    description={confirmDelete ? `Delete preset <strong>${confirmDelete.id}</strong>?` : ''}
    busy={confirmBusy}
    onConfirm={runConfirmDelete}
    onClose={() => { confirmDelete = null; confirmBusy = false }}
/>
