<script lang="ts">
import {
  ArrowLeft,
  Box,
  ChevronRight,
  Cpu,
  Image as ImageIcon,
  Layers,
  Palette,
  Server,
  SlidersHorizontal,
  Sparkles,
  Wrench,
} from '@lucide/svelte'
import type { ModelInfo, ProviderInfo } from '@zergx/schema'
import { onMount } from 'svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import { VERSION } from '$lib/version'
import AppearanceSection from './AppearanceSection.svelte'
import InstallSection from './InstallSection.svelte'
import PresetSection from './PresetSection.svelte'
import ProviderSection from './ProviderSection.svelte'
import ToolsSection from './ToolsSection.svelte'

type DetailId =
  | 'providers'
  | 'presets'
  | 'appearance'
  | 'tools'
  | 'model'
  | 'container'
  | 'base-image'
  | 'advanced'

type Detail = { id: DetailId; title: string; icon: typeof Server }

const DETAILS: Detail[] = [
  { id: 'providers', title: 'LLM Providers', icon: Server },
  { id: 'presets', title: 'Presets', icon: Sparkles },
  { id: 'appearance', title: 'Appearance', icon: Palette },
  { id: 'tools', title: 'Tools', icon: Wrench },
  { id: 'model', title: 'Default Model', icon: Cpu },
  { id: 'container', title: 'Container Backend', icon: Box },
  { id: 'base-image', title: 'Worker Base Image', icon: ImageIcon },
  { id: 'advanced', title: 'Advanced', icon: SlidersHorizontal },
]

let values = $state<Record<string, string>>({})
let loading = $state(true)
let saving = $state(false)
let saved = $state(false)
let providers = $state<Record<string, ProviderInfo>>({})
let models = $state<ModelInfo[]>([])
let backend = $derived(values.container_backend || 'kubernetes')
let k8sInfo = $state<{ namespace: string; worker_image: string } | null>(null)

let stack = $state<DetailId[]>([])
let activeId = $derived(stack.length > 0 ? stack[stack.length - 1] : null)
let activeDetail = $derived(DETAILS.find(d => d.id === activeId) ?? null)

onMount(async () => {
  const [cr, pr, mr, kr] = await Promise.all([
    api.config.get(),
    api.providers.list(),
    api.models.list(),
    api.infra.k8sConfig(),
  ])
  values = cr.isOk() ? cr.value : {}
  providers = pr.isOk() ? pr.value : {}
  models = mr.isOk() ? mr.value : []
  k8sInfo = kr.isOk() ? kr.value : null
  loading = false
})

function openDetail(id: DetailId) {
  stack = [...stack, id]
}

function back() {
  stack = stack.slice(0, -1)
}

async function save() {
  saving = true
  saved = false
  await api.config.set(values)
  saving = false
  saved = true
  setTimeout(() => (saved = false), 2000)
}

async function refreshProviders() {
  const pr = await api.providers.list()
  if (pr.isOk()) providers = pr.value
  const mr = await api.models.list()
  if (mr.isOk()) models = mr.value
}
</script>

<div class="flex flex-col h-full">
    {#if activeDetail}
        <div class="flex items-center gap-2 border-b border-border px-3 py-2.5 shrink-0">
            <Button variant="ghost" size="icon" class="size-7" onclick={back} title="Back">
                <ArrowLeft class="size-4" />
            </Button>
            <activeDetail.icon class="size-4 text-muted-foreground" />
            <span class="text-sm font-semibold truncate">{activeDetail.title}</span>
        </div>
        <div class="flex-1 overflow-y-auto p-4">
            {#if activeId === 'providers'}
                <ProviderSection />
            {:else if activeId === 'presets'}
                <PresetSection />
            {:else if activeId === 'appearance'}
                <AppearanceSection />
            {:else if activeId === 'tools'}
                <ToolsSection {providers} />
            {:else if activeId === 'model'}
                {#if models.length > 0}
                    <select class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        value={values["llm_model"] || ""} onchange={(e) => values = { ...values, llm_model: e.currentTarget.value }}>
                        <option value="">Select...</option>
                        {#each models as m (m.id)}
                            <option value={m.id}>{m.provider_id}: {m.name}</option>
                        {/each}
                    </select>
                {:else}
                    <input type="text" class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        placeholder="e.g. deepseek-v4-pro" value={values["llm_model"] || ""} oninput={(e) => values = { ...values, llm_model: e.currentTarget.value }} />
                    <p class="text-[10px] text-muted-foreground mt-1">Add a provider above to populate the model list.</p>
                {/if}
                <div class="flex items-center gap-2 pt-4">
                    <Button onclick={save} disabled={saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </div>
            {:else if activeId === 'container'}
                <div class="flex items-center gap-6 mb-4">
                    <label class="flex items-center gap-2 text-sm cursor-pointer">
                        <input type="radio" name="backend" value="kubernetes" checked={backend === "kubernetes"} onchange={() => values = { ...values, container_backend: "kubernetes" }} class="accent-primary" /> Kubernetes
                    </label>
                    <label class="flex items-center gap-2 text-sm cursor-pointer">
                        <input type="radio" name="backend" value="docker" checked={backend === "docker"} onchange={() => values = { ...values, container_backend: "docker" }} class="accent-primary" /> Docker
                    </label>
                </div>
                {#if backend === "kubernetes"}
                    <div class="space-y-3">
                        {#if k8sInfo}
                            <div class="rounded-md border border-border bg-muted/40 p-3 text-xs space-y-1">
                                <div class="flex justify-between gap-2"><span class="text-muted-foreground">Namespace</span><span class="font-mono">{k8sInfo.namespace}</span></div>
                                <div class="flex justify-between gap-2"><span class="text-muted-foreground">Worker Image</span><span class="font-mono truncate max-w-[240px]">{k8sInfo.worker_image}</span></div>
                            </div>
                        {/if}
                        <div><label class="text-sm font-medium" for="k8s_img">Worker Image</label>
                            <input id="k8s_img" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="artifact.zergx.svc.cluster.local/zergx-worker:latest" value={values["worker_base_image"] || ""} oninput={(e) => values = { ...values, worker_base_image: e.currentTarget.value }} /></div>
                        <div><label class="text-sm font-medium" for="k8s_ns">Namespace</label>
                            <input id="k8s_ns" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="temp" value={values["k8s_namespace"] || ""} oninput={(e) => values = { ...values, k8s_namespace: e.currentTarget.value }} /></div>
                        <div><label class="text-sm font-medium" for="kube_cfg">Kubeconfig</label>
                            <textarea id="kube_cfg" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-xs font-mono min-h-[100px] resize-y" placeholder="Paste kubeconfig.yaml..." value={values["kubeconfig"] || ""} oninput={(e) => values = { ...values, kubeconfig: e.currentTarget.value }}></textarea></div>
                    </div>
                {:else}
                    <div class="space-y-3">
                        <div><label class="text-sm font-medium" for="docker_img">Worker Image</label>
                            <input id="docker_img" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="artifact.zergx.svc.cluster.local/zergx-worker:latest" value={values["worker_image"] || ""} oninput={(e) => values = { ...values, worker_image: e.currentTarget.value }} /></div>
                        <div><label class="text-sm font-medium" for="docker_url">Docker API URL</label>
                            <input id="docker_url" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="http://podman.podman.svc.cluster.local:8080" value={values["docker_api_url"] || ""} oninput={(e) => values = { ...values, docker_api_url: e.currentTarget.value }} /></div>
                    </div>
                {/if}
                <div class="flex items-center gap-2 pt-4">
                    <Button onclick={save} disabled={saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </div>
            {:else if activeId === 'base-image'}
                <div>
                    <label class="text-sm font-medium" for="base_img">Default Sandbox Base</label>
                    <input id="base_img" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="debian:trixie-slim" value={values["worker_base_image"] || ""} oninput={(e) => values = { ...values, worker_base_image: e.currentTarget.value }} />
                    <p class="text-[10px] text-muted-foreground mt-1">Default base image for worker sandbox containers. Sessions can override this via their own settings.</p>
                </div>
                <div class="flex items-center gap-2 pt-4">
                    <Button onclick={save} disabled={saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </div>
            {:else if activeId === 'advanced'}
                <div class="grid grid-cols-1 gap-3">
                    <div>
                        <label class="text-sm font-medium" for="repos_root">Repos Root</label>
                        <input id="repos_root" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="/home/user/.zergx/repos" value={values["repos_root"] || ""} oninput={(e) => values = { ...values, repos_root: e.currentTarget.value }} />
                    </div>
                    <div>
                        <label class="text-sm font-medium" for="server_url">Server URL</label>
                        <input id="server_url" type="text" class="mt-1 flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="http://0.0.0.0:8601" value={values["server_url"] || ""} oninput={(e) => values = { ...values, server_url: e.currentTarget.value }} />
                    </div>
                </div>
                <div class="flex items-center gap-2 pt-4">
                    <Button onclick={save} disabled={saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </div>
            {/if}
        </div>
    {:else}
        <div class="flex items-center gap-2 border-b border-border px-4 py-2.5 shrink-0">
            <Server class="size-4 text-muted-foreground" />
            <span class="text-sm font-semibold">Settings</span>
        </div>
        <div class="flex-1 overflow-y-auto p-4">
            {#if loading}
                <div class="flex items-center gap-2 text-sm text-muted-foreground"><span class="animate-pulse">Loading...</span></div>
            {:else}
                <section class="mb-6">
                    <div class="rounded-lg border border-border bg-card divide-y divide-border">
                        <InstallSection />
                    </div>
                </section>

                <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2 px-1">App</div>
                <div class="rounded-lg border border-border bg-card divide-y divide-border mb-6">
                    {#each DETAILS.filter(d => d.id === 'appearance' || d.id === 'providers' || d.id === 'presets') as d (d.id)}
                        <button type="button" class="w-full flex items-center gap-3 px-3 py-3 text-left hover:bg-accent/40 transition-colors" onclick={() => openDetail(d.id)}>
                            <d.icon class="size-4 text-muted-foreground shrink-0" />
                            <span class="text-sm flex-1 truncate">{d.title}</span>
                            <ChevronRight class="size-4 text-muted-foreground/60 shrink-0" />
                        </button>
                    {/each}
                </div>

                <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2 px-1">Workspace</div>
                <div class="rounded-lg border border-border bg-card divide-y divide-border">
                    {#each DETAILS.filter(d => d.id !== 'appearance' && d.id !== 'providers' && d.id !== 'presets') as d (d.id)}
                        <button type="button" class="w-full flex items-center gap-3 px-3 py-3 text-left hover:bg-accent/40 transition-colors" onclick={() => openDetail(d.id)}>
                            <d.icon class="size-4 text-muted-foreground shrink-0" />
                            <span class="text-sm flex-1 truncate">{d.title}</span>
                            <ChevronRight class="size-4 text-muted-foreground/60 shrink-0" />
                        </button>
                    {/each}
                </div>

                <div class="text-xs text-muted-foreground mt-6 px-1 flex items-center gap-1">
                    <Layers class="size-3" />
                    ReCoder · {VERSION}
                </div>
            {/if}
        </div>
    {/if}
</div>
