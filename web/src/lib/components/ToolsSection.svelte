<script lang="ts">
import { ChevronDown, Save } from '@lucide/svelte'
import type { ProviderInfo, ToolConfigMap, ToolInfo } from '@zergx/schema'
import { onMount } from 'svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import ToolIcon from './ToolIcon.svelte'

let { providers }: { providers: Record<string, ProviderInfo> } = $props()

let toolsList = $state<ToolInfo[]>([])
let toolConfig = $state<ToolConfigMap>({})
let expandedTool = $state<string | null>(null)
let loading = $state(true)

onMount(async () => {
  const [tlr, tcr] = await Promise.all([
    api.tools.list(),
    api.tools.getConfig(),
  ])
  toolsList = tlr.isOk() ? tlr.value : []
  toolConfig = tcr.isOk() ? tcr.value : {}
  loading = false
})

function toggleToolExpanded(name: string) {
  expandedTool = expandedTool === name ? null : name
}

async function saveToolConfig(name: string) {
  const cfg = toolConfig[name] ?? {}
  const r = await api.tools.setConfig({ [name]: cfg })
  if (r.isOk()) toolConfig = r.value.config
}

function setToolConfigValue(name: string, key: string, value: unknown) {
  toolConfig = {
    ...toolConfig,
    [name]: { ...(toolConfig[name] ?? {}), [key]: value },
  }
}

function categorizeTools(tools: ToolInfo[]): Map<string, ToolInfo[]> {
  const cats = new Map<string, ToolInfo[]>()
  for (const t of tools) {
    const cat = t.category || 'other'
    if (!cats.has(cat)) cats.set(cat, [])
    cats.get(cat)?.push(t)
  }
  return cats
}
</script>

{#if loading}
    <div class="text-xs text-muted-foreground py-2">Loading...</div>
{:else if toolsList.length === 0}
    <div class="text-xs text-muted-foreground py-2">No tools registered.</div>
{:else}
    {#each [...categorizeTools(toolsList)] as [cat, catTools] (cat)}
        <div class="mb-3">
            <span class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider block mb-1">{cat}</span>
            <div class="space-y-1">
                {#each catTools as t (t.name)}
                    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                    <div class="rounded-md border border-border bg-card/30 overflow-hidden">
                        <div class="flex items-center justify-between px-3 py-1.5 cursor-pointer hover:bg-accent/30" onclick={() => toggleToolExpanded(t.name)} role="button" tabindex="0" onkeydown={(e) => e.key === "Enter" && toggleToolExpanded(t.name)}>
                            <div class="flex items-center gap-2 min-w-0">
                                <ToolIcon name={t.name} />
                                <span class="text-xs font-mono font-medium shrink-0">{t.name}</span>
                            </div>
                            <div class="flex items-center gap-2 shrink-0 ml-2">
                                {#if t.configFields && t.configFields.length > 0}
                                    {#if Object.keys(toolConfig[t.name] ?? {}).length > 0}
                                        <span class="text-[9px] bg-green-500/10 text-green-500 px-1.5 py-0.5 rounded">configured</span>
                                    {:else}
                                        <span class="text-[9px] bg-yellow-500/10 text-yellow-500 px-1.5 py-0.5 rounded">needs config</span>
                                    {/if}
                                {:else}
                                    <span class="text-[9px] text-muted-foreground">no config</span>
                                {/if}
                                <ChevronDown class="size-3 text-muted-foreground transition-transform {expandedTool === t.name ? 'rotate-180' : ''}" />
                            </div>
                        </div>
                        {#if expandedTool === t.name}
                            <div class="px-3 py-2 border-t border-border space-y-2 bg-card/50">
                                {#if t.description}
                                    <p class="text-[10px] text-muted-foreground break-all whitespace-pre-wrap">{t.description}</p>
                                {/if}
                                {#if t.parameters && Object.keys(t.parameters).length > 0}
                                    <div>
                                        <span class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider block mb-1">Parameters</span>
                                        <pre class="rounded-md bg-muted p-2 text-[10px] font-mono whitespace-pre-wrap break-all overflow-x-auto">{JSON.stringify(t.parameters, null, 2)}</pre>
                                    </div>
                                {/if}
                                {#if t.configFields && t.configFields.length > 0}
                                    {#each t.configFields as f (f.key)}
                                    <div>
                                        <label class="text-[10px] font-medium text-muted-foreground block mb-0.5" for="tf-{t.name}-{f.key}">{f.label}</label>
                                        {#if f.type === "select-provider"}
                                            <select id="tf-{t.name}-{f.key}" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                                                value={toolConfig[t.name]?.[f.key] ?? ""}
                                                onchange={(e) => setToolConfigValue(t.name, f.key, e.currentTarget.value || null)}>
                                                <option value="">None</option>
                                                {#each Object.entries(providers) as [pid, pv] (pid)}
                                                    <option value={pid}>{pid}{pv.api_type ? ` (${pv.api_type})` : ""}</option>
                                                {/each}
                                            </select>
                                        {:else if f.type === "select-model"}
                                            {@const providerId = toolConfig[t.name]?.[f.dependsOnProvider ?? "provider_id"] ?? ""}
                                            {@const providerIdStr = typeof providerId === "string" ? providerId : ""}
                                            {@const providerModels = providerIdStr ? (providers[providerIdStr]?.models ?? []) : []}
                                            <select id="tf-{t.name}-{f.key}" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                                                value={toolConfig[t.name]?.[f.key] ?? ""}
                                                onchange={(e) => setToolConfigValue(t.name, f.key, e.currentTarget.value || null)}>
                                                <option value="">None</option>
                                                {#each providerModels as m (m.id)}
                                                    <option value={m.id}>{m.name || m.id}</option>
                                                {/each}
                                            </select>
                                            {#if !providerId}
                                                <p class="text-[9px] text-muted-foreground mt-0.5">Select a provider first</p>
                                            {:else if providerModels.length === 0}
                                                <p class="text-[9px] text-muted-foreground mt-0.5">No models registered for this provider</p>
                                            {/if}
                                        {:else if f.type === "text"}
                                            <input type="text" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs" placeholder={f.placeholder ?? ""}
                                                value={toolConfig[t.name]?.[f.key] ?? ""}
                                                oninput={(e) => setToolConfigValue(t.name, f.key, e.currentTarget.value || null)} />
                                        {:else if f.type === "number"}
                                            <input type="number" class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs" placeholder={f.placeholder ?? ""}
                                                value={toolConfig[t.name]?.[f.key] ?? ""}
                                                oninput={(e) => setToolConfigValue(t.name, f.key, e.currentTarget.value ? Number(e.currentTarget.value) : null)} />
                                        {/if}
                                    </div>
                                    {/each}
                                    <div class="flex justify-end pt-1">
                                        <Button size="sm" variant="outline" onclick={() => saveToolConfig(t.name)}>
                                            <Save class="size-3 mr-1" /> Save
                                        </Button>
                                    </div>
                                {/if}
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        </div>
    {/each}
{/if}
