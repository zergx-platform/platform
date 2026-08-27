<script lang="ts">
import {
  Check,
  ChevronDown,
  ChevronRight,
  FlaskConical,
  Loader2,
  Plus,
  Search,
  X,
} from '@lucide/svelte'
import type { MdProvider, ProviderModel } from '@zergx/schema'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'

let {
  mdProviderList,
  mdProviders,
  onRegistered,
}: {
  mdProviderList: MdProvider[]
  mdProviders: Record<string, MdProvider>
  onRegistered: () => void
} = $props()

let showForm = $state(false)
let templateId = $state('')
let providerId = $state('')
let baseUrl = $state('')
let apiKey = $state('')
let apiType = $state('openai-compatible')
let headers = $state<{ k: string; v: string }[]>([])
let modelIds = $state<string[]>([])
let expandedModels = $state(false)
let searchQuery = $state('')
let testStatus = $state<'idle' | 'testing' | 'ok' | 'error'>('idle')
let testMessage = $state('')
let registering = $state(false)

let templateModels = $derived.by(() => {
  const tp = templateId ? mdProviders[templateId] : null
  return tp?.models || {}
})

let filteredModelKeys = $derived.by(() => {
  const all = Object.keys(templateModels)
  const q = searchQuery.toLowerCase().trim()
  if (!q) return all.sort()
  return all
    .filter(
      k =>
        k.toLowerCase().includes(q) ||
        (templateModels[k]?.name || '').toLowerCase().includes(q),
    )
    .sort()
})

function npmToType(npm: string): string {
  if (!npm || npm.includes('openai-compatible')) return 'openai-compatible'
  if (npm.includes('anthropic')) return 'anthropic'
  if (npm.includes('openai')) return 'openai'
  if (npm.includes('google') || npm.includes('gemini')) return 'gemini'
  return npm.replace('@ai-sdk/', '').replace(/-ai-sdk-provider$/, '')
}

function onTemplateChange() {
  const tp = mdProviders[templateId]
  if (!tp) {
    providerId = ''
    baseUrl = ''
    apiType = 'openai-compatible'
    modelIds = []
    return
  }
  providerId = tp.id || templateId
  baseUrl = tp.api || ''
  apiType = npmToType(tp.npm || '')
  modelIds = []
  expandedModels = false
  searchQuery = ''
}

function reset() {
  templateId = ''
  providerId = ''
  baseUrl = ''
  apiKey = ''
  apiType = 'openai-compatible'
  headers = []
  modelIds = []
  expandedModels = false
  searchQuery = ''
  testStatus = 'idle'
  testMessage = ''
  registering = false
}

function toggleModel(id: string) {
  modelIds = modelIds.includes(id)
    ? modelIds.filter(x => x !== id)
    : [...modelIds, id]
}

function buildModels(): ProviderModel[] {
  return modelIds.map(mid => {
    const m = templateModels[mid] || {}
    return {
      id: typeof m.id === 'string' ? m.id : mid,
      name: typeof m.name === 'string' ? m.name : mid,
      context_limit: m.limit?.context,
      output_limit: m.limit?.output,
      reasoning: true,
      tool_call: true,
    }
  })
}

function fmt(n?: number): string {
  if (!n) return ''
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`
  return String(n)
}

async function onTest() {
  testStatus = 'testing'
  testMessage = ''
  const r = await api.providers.test({
    api_type: apiType,
    base_url: baseUrl,
    api_key: apiKey,
  })
  if (r.isOk() && r.value.ok) {
    testStatus = 'ok'
    testMessage = r.value.detail || 'OK'
  } else {
    testStatus = 'error'
    testMessage = r.isErr() ? r.error : r.value.error || 'Unknown'
  }
}

async function onRegister() {
  registering = true
  const models = buildModels()
  const hdrs: Record<string, string> = {}
  for (const h of headers) if (h.k.trim()) hdrs[h.k.trim()] = h.v
  const r = await api.providers.register({
    provider_id: providerId,
    api_type: apiType,
    base_url: baseUrl,
    api_key: apiKey,
    headers: Object.keys(hdrs).length > 0 ? hdrs : undefined,
    models,
  })
  if (r.isOk()) {
    showForm = false
    reset()
    onRegistered()
  }
  registering = false
}
</script>

{#if showForm}
    <div class="mt-4 p-4 rounded-md border border-border bg-card/30 space-y-3">
        <h4 class="text-xs font-semibold text-foreground">Add Provider</h4>

        <div>
            <label class="text-[10px] font-medium text-muted-foreground" for="ap-tmpl">Template (api.json)</label>
            <select id="ap-tmpl" class="mt-0.5 flex w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                bind:value={templateId} onchange={onTemplateChange}>
                <option value="">Choose a provider template...</option>
                {#each mdProviderList as tp (tp.id)}
                    <option value={tp.id}>{tp.name || tp.id} ({tp.id})</option>
                {/each}
            </select>
            <p class="text-[9px] text-muted-foreground mt-0.5">Pre-fills form below. All fields can be edited.</p>
        </div>

        <div class="grid grid-cols-2 gap-3">
            <div>
                <label class="text-[10px] font-medium text-muted-foreground" for="ap-id">Provider ID *</label>
                <input id="ap-id" type="text" class="mt-0.5 flex w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    placeholder="e.g. tal-ai" bind:value={providerId} />
            </div>
            <div>
                <label class="text-[10px] font-medium text-muted-foreground" for="ap-type">API Type</label>
                <input id="ap-type" type="text" class="mt-0.5 flex w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    placeholder="openai-compatible" bind:value={apiType} />
            </div>
        </div>
        <div>
            <label class="text-[10px] font-medium text-muted-foreground" for="ap-url">Base URL *</label>
            <input id="ap-url" type="text" class="mt-0.5 flex w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                placeholder="https://..." bind:value={baseUrl} />
        </div>
        <div>
            <label class="text-[10px] font-medium text-muted-foreground" for="ap-key">API Key *</label>
            <input id="ap-key" type="password" class="mt-0.5 flex w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                placeholder="sk-..." bind:value={apiKey} />
        </div>
        <div>
            <button class="flex items-center gap-1 text-[10px] font-medium text-muted-foreground hover:text-foreground mb-1" onclick={() => headers = [...headers, { k: "", v: "" }]}>
                <Plus class="size-3" /> Headers {headers.length > 0 ? `(${headers.length})` : ""}
            </button>
            {#each headers as h, i}
                <div class="flex items-center gap-1 mb-1">
                    <input type="text" placeholder="Key" class="w-[120px] rounded-md border border-input bg-background px-2 py-1 text-[10px] placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" bind:value={h.k} />
                    <input type="text" placeholder="Value" class="flex-1 rounded-md border border-input bg-background px-2 py-1 text-[10px] placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" bind:value={h.v} />
                    <Button variant="ghost" size="icon" class="size-5 shrink-0" onclick={() => { headers = headers.filter((_, j) => j !== i) }}>
                        <X class="size-3" />
                    </Button>
                </div>
            {/each}
        </div>

        {#if templateId && Object.keys(templateModels).length > 0}
            <div>
                <button class="flex items-center gap-1 text-[10px] font-medium text-muted-foreground hover:text-foreground"
                    onclick={() => expandedModels = !expandedModels}>
                    {#if expandedModels}<ChevronDown class="size-3" />{:else}<ChevronRight class="size-3" />{/if}
                    Select models {modelIds.length > 0 ? `(${modelIds.length} selected)` : `(${Object.keys(templateModels).length} available)`}
                </button>
                {#if expandedModels}
                    <div class="mt-2 border border-border rounded-md bg-background overflow-hidden">
                        <div class="flex items-center gap-1 p-2 border-b border-border/50 bg-muted/30">
                            <Search class="size-3 text-muted-foreground shrink-0" />
                            <input type="text" placeholder="Filter..." class="flex-1 bg-transparent text-[10px] placeholder:text-muted-foreground focus-visible:outline-none" bind:value={searchQuery} />
                            <Button variant="ghost" size="sm" class="text-[9px]" onclick={() => modelIds = []}>Clear</Button>
                            <Button variant="ghost" size="sm" class="text-[9px]" onclick={() => modelIds = [...filteredModelKeys]}>All</Button>
                        </div>
                        <div class="max-h-48 overflow-y-auto">
                            {#each filteredModelKeys as mid}
                                {@const m = templateModels[mid] || {}}
                                <label class="flex items-center gap-1.5 px-2 py-1 hover:bg-accent/40 cursor-pointer text-[10px] border-b border-border/30">
                                    <input type="checkbox" class="accent-primary shrink-0" checked={modelIds.includes(mid)} onchange={() => toggleModel(mid)} />
                                    <span class="flex-1 truncate">{m.name || mid}</span>
                                    {#if m.limit?.context}<span class="text-muted-foreground font-mono text-[9px]">{fmt(m.limit.context)}</span>{/if}
                                    {#if m.limit?.output}<span class="text-muted-foreground font-mono text-[9px]">/{fmt(m.limit.output)}</span>{/if}
                                    {#if m.reasoning}<span class="text-[8px] bg-purple-500/10 text-purple-400 px-1 rounded">R</span>{/if}
                                    {#if m.tool_call}<span class="text-[8px] bg-green-500/10 text-green-400 px-1 rounded">TC</span>{/if}
                                </label>
                            {/each}
                            {#if filteredModelKeys.length === 0}
                                <div class="px-4 py-6 text-xs text-muted-foreground text-center">No models match.</div>
                            {/if}
                        </div>
                    </div>
                {/if}
            </div>
        {/if}

        <div class="flex items-center gap-2 pt-1">
            <Button variant="outline" size="sm" class="text-[10px]" onclick={onTest} disabled={testStatus === "testing" || !baseUrl}>
                {#if testStatus === "testing"}<Loader2 class="size-3 animate-spin mr-1" />{/if}
                <FlaskConical class="size-3 mr-1" /> Test
            </Button>
            {#if testStatus === "ok"}
                <span class="text-[10px] text-green-500 flex items-center gap-1"><Check class="size-3" /> {testMessage}</span>
            {:else if testStatus === "error"}
                <span class="text-[10px] text-red-500 flex items-center gap-1"><X class="size-3" /> {testMessage}</span>
            {/if}
        </div>
        <div class="flex items-center gap-2 pt-1">
            <Button size="sm" class="text-[10px]" onclick={onRegister} disabled={registering || !providerId || !baseUrl}>
                {#if registering}<Loader2 class="size-3 animate-spin mr-1" />{/if} Register
            </Button>
            <Button variant="ghost" size="sm" class="text-[10px]" onclick={() => { showForm = false; reset() }}>Cancel</Button>
        </div>
    </div>
{:else}
    <Button variant="outline" size="sm" class="text-[10px] mt-3" onclick={() => { showForm = true; reset() }}>
        <Plus class="size-3 mr-1" /> Add Provider
    </Button>
{/if}
