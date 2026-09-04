<script lang="ts">
import { ChevronDown, Save } from '@lucide/svelte'
import type { ProviderInfo, ToolConfigItem, ToolConfigMap, ToolInfo } from '@zergx/schema'
import { onMount } from 'svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import ToolIcon from './ToolIcon.svelte'

let { providers }: { providers: Record<string, ProviderInfo> } = $props()

let toolsList = $state<ToolInfo[]>([])
let toolConfig = $state<ToolConfigMap>({})
let expandedTool = $state<string | null>(null)
let loading = $state(true)

// Provider/model cascade state for a VLM-style config knob (`vlm_model`
// holds a `provider_id/model_id` ref). Keyed by tool name → provider id.
let modelProvider = $state<Record<string, string>>({})
let modelValue = $state<Record<string, string>>({})

onMount(async () => {
	const [tlr, tcr] = await Promise.all([
		api.tools.list(),
		api.tools.getConfig(),
	])
	toolsList = tlr.isOk() ? tlr.value : []
	toolConfig = tcr.isOk() ? tcr.value : {}
	// Seed the cascade from any stored value that already encodes a ref.
	for (const t of toolsList) {
		const cfg = toolConfig[t.name] ?? {}
		for (const c of t.config ?? []) {
			const val = cfg[c.name]
			if (typeof val === 'string' && val.includes('/')) {
				const [pid, mid] = val.split('/')
				modelProvider[t.name] = pid
				modelValue[t.name] = mid ?? ''
			}
		}
	}
	loading = false
})

function toggleToolExpanded(name: string) {
	expandedTool = expandedTool === name ? null : name
}

function setToolConfigValue(name: string, key: string, value: unknown) {
	toolConfig = {
		...toolConfig,
		[name]: { ...(toolConfig[name] ?? {}), [key]: value },
	}
}

/** Save a single extension config knob by config-name (data-driven path). */
async function saveConfigValue(
	tool: ToolInfo,
	c: ToolConfigItem,
	value: unknown,
) {
	const extId = tool.category
	if (!extId || value == null || value === '') return
	const r = await api.tools.setConfigValue(extId, c.name, value)
	if (r.isOk()) setToolConfigValue(tool.name, c.name, value)
}

/** Compile the on-screen value of a knob, folding in the VLM cascade. */
function resolvedValue(t: ToolInfo, c: ToolConfigItem): unknown {
	const current = toolConfig[t.name]?.[c.name] ?? ''
	if (isModelRef(c)) {
		const pid = modelProvider[t.name]
		const mid = modelValue[t.name]
		if (pid && mid) return `${pid}/${mid}`
		return current
	}
	return current
}

/** Save all declared knobs of a tool via the per-knob path. */
async function saveToolConfig(t: ToolInfo) {
	for (const c of t.config ?? []) {
		const v = resolvedValue(t, c)
		await saveConfigValue(t, c, v)
	}
}

function configValue(t: ToolInfo, c: ToolConfigItem): unknown {
	return toolConfig[t.name]?.[c.name] ?? ''
}

function isModelRef(c: ToolConfigItem): boolean {
	return /model/i.test(c.name)
}

function parseRef(t: ToolInfo, c: ToolConfigItem): [string, string] {
	const ref = String(resolvedValue(t, c) ?? '')
	if (ref.includes('/')) {
		const [pid, mid] = ref.split('/')
		return [pid ?? '', mid ?? '']
	}
	return ['', '']
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

/** True when any declared required_config of this tool is unset. */
function configMissingRequired(t: ToolInfo): boolean {
	for (const name of t.required_config ?? []) {
		const v = toolConfig[t.name]?.[name]
		if (v == null || v === '') return true
	}
	return false
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
								{#if t.config && t.config.length > 0}
									{@const requiredMissing = configMissingRequired(t)}
									{#if requiredMissing}
										<span class="text-[9px] bg-red-500/10 text-red-500 px-1.5 py-0.5 rounded">required</span>
									{:else if Object.keys(toolConfig[t.name] ?? {}).length > 0}
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
								{#if t.config && t.config.length > 0}
									{#each t.config as c (c.name)}
										<div>
											{#if c.description}
												<label class="text-[10px] text-muted-foreground block mb-0.5">{c.description}</label>
											{/if}
											<label class="text-[10px] font-medium text-muted-foreground block mb-0.5">{c.name}</label>

											{#if isModelRef(c)}
												<!-- Provider/model cascade for a VLM-style knob. -->
												{@const [pid, mid] = parseRef(t, c)}
												{@const providerModels = pid ? (providers[pid]?.models ?? []) : []}
												<div class="space-y-1">
													<select class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
														value={pid}
														onchange={(e) => { modelProvider[t.name] = e.currentTarget.value; modelValue[t.name] = ''; }}>
														<option value="">None</option>
														{#each Object.entries(providers) as [pId, pv] (pId)}
															<option value={pId}>{pId}{pv.api_type ? ` (${pv.api_type})` : ''}</option>
														{/each}
													</select>
													<select class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
														value={mid}
														disabled={!pid}
														onchange={(e) => { modelValue[t.name] = e.currentTarget.value; }}>
														<option value="">None</option>
														{#each providerModels as m (m.id)}
															<option value={m.id}>{m.name || m.id}</option>
														{/each}
													</select>
													{#if !pid}
														<p class="text-[9px] text-muted-foreground mt-0.5">Select a provider first</p>
													{:else if providerModels.length === 0}
														<p class="text-[9px] text-muted-foreground mt-0.5">No models registered for this provider</p>
													{/if}
												</div>
											{:else if c.type === 'enum' && (c.enum_values?.length ?? 0) > 0}
												<select class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
													value={String(configValue(t, c) ?? '')}
													onchange={(e) => void saveConfigValue(t, c, e.currentTarget.value || null)}>
													<option value="">None</option>
													{#each c.enum_values ?? [] as v (v)}
														<option value={v}>{v}</option>
													{/each}
												</select>
											{:else}
												<input
													type={c.type === 'number' ? 'number' : (c.type === 'boolean' ? 'checkbox' : 'text')}
													class="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
													placeholder={c.default ? String(c.default) : ''}
													value={String(configValue(t, c) ?? '')}
													onchange={(e) => {
														let v: unknown = e.currentTarget.value
														if (c.type === 'number') v = e.currentTarget.value ? Number(e.currentTarget.value) : null
														if (c.type === 'boolean') v = e.currentTarget.checked
														void saveConfigValue(t, c, v)
													}}
												/>
											{/if}
										</div>
									{/each}
									<div class="flex justify-end pt-1">
										<Button size="sm" variant="outline" onclick={() => saveToolConfig(t)}>
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
