<script lang="ts">
import {
  Box,
  ChevronDown,
  ChevronRight,
  Copy,
  Download,
  ExternalLink,
  FileIcon,
  Loader2,
  Package,
  RefreshCw,
  Search,
  Server,
  Trash2,
  Upload,
  X,
} from '@lucide/svelte'
import type { PackageTypeEntry, ZergxConfig } from '@zergx/schema'
import { onMount } from 'svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'
import { openTaskStream, type TaskStreamLine } from '$lib/taskStream'
import * as Card from '$lib/components/ui/card'

// ── State ──────────────────────────────────────────────
let tab = $state<'registries' | 'packages'>('registries')
let types = $state<PackageTypeEntry[]>([])
let zergxCfg = $state<ZergxConfig | null>(null)
let repositories = $state<string[]>([])
let loading = $state(false)
let error = $state('')
let copied = $state<string | null>(null)
let query = $state('')

// Packages tab state
let pkgs = $state<
  Array<{
    name: string
    type: string
    latest_version: string | null
    versions: number
  }>
>([])
let pkgLoading = $state(false)
let pkgError = $state('')
let pkgQuery = $state('')
let typeFilter = $state<string>('')
let expandedKey = $state<string | null>(null)
let versionDetail = $state<
  Array<{
    version: string
    download_count: number
    created_unix: number
    files: Array<{ name: string; size: number; sha256: string }>
  }>
>([])
let versionLoading = $state(false)
let deletingPkg = $state<string | null>(null)
let confirmDelete = $state<{
  type: string
  name: string
  version?: string
} | null>(null)
let pkgTotal = $state(0)
let pkgOffset = $state(0)
const PAGE_SIZE = 50

const TYPE_LABELS: Record<string, string> = {
  cargo: 'Cargo (Rust)',
  composer: 'Composer (PHP)',
  conan: 'Conan (C/C++)',
  generic: 'Generic',
  go: 'Go',
  helm: 'Helm Charts',
  hex: 'Hex (Elixir)',
  maven: 'Maven (Java)',
  npm: 'npm (Node)',
  nuget: 'NuGet (.NET)',
  oci: 'OCI (Containers)',
  pub: 'Pub (Dart)',
  pypi: 'PyPI (Python)',
  rubygems: 'RubyGems',
  swift: 'Swift',
}

onMount(() => {
  loadAll()
})

// Publish form state
let showPublish = $state(false)
let publishSpecs = $state<Array<{ protocol: string; args: string[] | null; required: string[] | null }>>([])
let publishProtocol = $state('')
let publishName = $state('')
let publishVersion = $state('')
let publishOrg = $state('')
let publishRepo = $state('')
let publishBookmark = $state('')
let publishFile = $state('')
let publishing = $state(false)
let publishLogs = $state<TaskStreamLine[]>([])
let publishState = $state('')
let publishError = $state('')

async function openPublish() {
  publishError = ''
  const r = await api.containers.publishSpecs()
  publishSpecs = r.isOk() ? r.value : []
  publishProtocol = ''
  publishLogs = []
  publishState = ''
  publishError = ''
  showPublish = true
}

async function onPublish() {
  if (!publishProtocol || !publishOrg || !publishRepo || !publishBookmark) {
    publishError = 'Protocol, org, repo and bookmark required'
    return
  }
  publishing = true
  publishError = ''
  const r = await api.containers.publishPackage({
    protocol: publishProtocol,
    org: publishOrg,
    repo: publishRepo,
    bookmark: publishBookmark,
    session: '',
    name: publishName,
    version: publishVersion,
    file: publishFile,
    dockerfile_path: '',
  })
  if (r.isErr()) {
    publishError = r.error
    publishing = false
    return
  }
  // Stream the publish task's log live (same build-task SSE channel).
  openTaskStream(r.value.build_id, {
    onLog: lines => (publishLogs = [...publishLogs, ...lines]),
    onState: st => (publishState = st),
    onDone: done => {
      publishing = false
      publishState = done.state
      if (done.state !== 'done' && done.error) publishError = done.error
      if (done.state === 'done') setTimeout(() => { showPublish = false; void loadAll() }, 800)
    },
    onError: msg => {
      publishing = false
      publishError = msg
    },
  })
}

async function loadAll() {
  loading = true
  error = ''
  const [typesR, cfgR, catalogR] = await Promise.all([
    api.packages.listTypes(),
    api.packages.zergxConfig(),
    api.packages.ociCatalog(),
  ])
  if (typesR.isOk()) types = typesR.value
  else error = typesR.error
  if (cfgR.isOk()) zergxCfg = cfgR.value as ZergxConfig
  if (catalogR.isOk()) repositories = catalogR.value.repositories
  loading = false
}

async function loadPackages() {
  pkgLoading = true
  pkgError = ''
  const r = await api.packages.listAll({
    type: typeFilter || undefined,
    q: pkgQuery || undefined,
    limit: PAGE_SIZE,
    offset: pkgOffset,
  })
  if (r.isOk()) {
    pkgs = r.value.packages
    pkgTotal = r.value.total
  } else {
    pkgError = r.error
  }
  pkgLoading = false
}

function switchTab(t: 'registries' | 'packages') {
  tab = t
  if (t === 'packages' && pkgs.length === 0 && !pkgLoading) loadPackages()
}

function endpointFor(type: string): string {
  if (type === 'oci') return '/v2/'
  return `/api/v1/packages/${type}/`
}

async function copyEndpoint(type: string) {
  const url = `${window.location.origin}${endpointFor(type)}`
  try {
    await navigator.clipboard.writeText(url)
    copied = type
    setTimeout(() => (copied = null), 1500)
  } catch {
    /* clipboard blocked */
  }
}

function pkgKey(p: { name: string; type: string }): string {
  return `${p.type}/${p.name}`
}

async function toggleExpand(p: { name: string; type: string }) {
  const key = pkgKey(p)
  if (expandedKey === key) {
    expandedKey = null
    versionDetail = []
    return
  }
  expandedKey = key
  versionDetail = []
  versionLoading = true
  const r = await api.packages.versions(p.type, p.name)
  if (r.isOk()) versionDetail = r.value.versions
  versionLoading = false
}

async function doDelete() {
  if (!confirmDelete) return
  const { type, name, version } = confirmDelete
  const key = pkgKey({ type, name })
  deletingPkg = version ? `${key}/${version}` : key
  const r = version
    ? await api.packages.deleteVersion(type, name, version)
    : await api.packages.deletePackage(type, name)
  deletingPkg = null
  confirmDelete = null
  if (r.isOk()) {
    if (version) {
      versionDetail = versionDetail.filter(v => v.version !== version)
      if (versionDetail.length === 0) expandedKey = null
    } else {
      expandedKey = null
      versionDetail = []
    }
    await loadPackages()
  }
}

function fmtSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function fmtTime(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const availableTypes = $derived([...new Set(pkgs.map(p => p.type))].sort())

const filtered = $derived(
  query.trim()
    ? types.filter(
        t =>
          t.type.toLowerCase().includes(query.toLowerCase()) ||
          (TYPE_LABELS[t.type] || '')
            .toLowerCase()
            .includes(query.toLowerCase()),
      )
    : types,
)
</script>

<div class="flex flex-col h-full">
	<!-- Header -->
	<div class="shrink-0 border-b border-border bg-card px-4 sm:px-6 py-4">
		<div class="flex items-center justify-between gap-4 flex-wrap">
			<div class="flex items-center gap-2.5">
				<div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
					<Package class="size-5" />
				</div>
				<div>
					<h1 class="text-lg font-semibold leading-tight">Packages</h1>
					<p class="text-xs text-muted-foreground">
						Proxy registries for {types.length} ecosystems
						{#if zergxCfg?.self_base}
							· self_base: <span class="font-mono">{zergxCfg.self_base}</span>
						{/if}
					</p>
				</div>
			</div>
			<div class="flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={openPublish}
				>
					<Upload class="size-3.5" />
					<span class="hidden sm:inline">Publish</span>
				</Button>
				<Button
					variant="outline"
					size="sm"
					onclick={() => (tab === 'packages' ? loadPackages() : loadAll())}
					disabled={loading || pkgLoading}
				>
					<RefreshCw class="size-3.5 {loading || pkgLoading ? 'animate-spin' : ''}" />
					<span class="hidden sm:inline">Refresh</span>
				</Button>
			</div>
		</div>

		<!-- Tab switcher -->
		<div class="flex items-center gap-1 mt-3">
			<button
				class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors {tab ===
				'registries'
					? 'bg-secondary text-secondary-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => switchTab('registries')}
			>
				<Package class="size-3.5 inline-block mr-1 -mt-0.5" />
				Registries
			</button>
			<button
				class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors {tab ===
				'packages'
					? 'bg-secondary text-secondary-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => switchTab('packages')}
			>
				<Box class="size-3.5 inline-block mr-1 -mt-0.5" />
				Packages
				{#if pkgs.length > 0}
					<span
						class="ml-1 px-1.5 py-0.5 rounded-full text-[10px] {tab === 'packages' ? 'bg-background/40 text-secondary-foreground' : 'bg-muted text-muted-foreground'}"
					>
						{pkgs.length}
					</span>
				{/if}
			</button>
		</div>
	</div>

	<div
		class="flex-1 min-h-0 overflow-auto px-4 sm:px-6 py-5"
		style="padding-bottom: max(1.25rem, env(safe-area-inset-bottom));"
	>
		<div class="mx-auto max-w-6xl space-y-5">
			{#if tab === 'registries'}
				<!-- ════════ TAB 1: REGISTRIES ════════ -->
				{#if error}
					<div class="text-sm text-destructive bg-destructive/10 rounded-md px-3 py-2">
						{error}
					</div>
				{/if}

				{#if loading && types.length === 0}
					<div
						class="flex items-center justify-center py-12 text-sm text-muted-foreground"
					>
						<Loader2 class="size-4 animate-spin mr-2" /> Loading...
					</div>
				{:else}
					<!-- Search -->
					<div class="relative max-w-sm">
						<Search
							class="size-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
						/>
						<input
							type="text"
							placeholder="Filter ecosystems..."
							bind:value={query}
							class="w-full rounded-md border border-input bg-background pl-9 pr-3 py-1.5 text-xs"
						/>
					</div>

					<!-- Registry type grid -->
					<section>
						<div class="flex items-center gap-2 mb-3">
							<Package class="size-4 text-muted-foreground" />
							<h3 class="text-sm font-semibold">Proxy Registries</h3>
							<span class="text-xs text-muted-foreground">({filtered.length})</span>
						</div>
						<div class="grid gap-2.5 sm:grid-cols-2 lg:grid-cols-3">
							{#each filtered as t (t.type)}
								<Card.Root class="group">
									<Card.Header class="pb-2">
										<div class="flex items-start justify-between gap-2">
											<Card.Title class="text-sm font-mono">{t.type}</Card.Title>
											<span
												class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground shrink-0 uppercase tracking-wide"
											>
												{t.type === 'oci' ? 'OCI' : 'pkg'}
											</span>
										</div>
										<Card.Description class="text-xs">
											{TYPE_LABELS[t.type] || t.type}
										</Card.Description>
									</Card.Header>
									<Card.Content class="text-xs space-y-1.5 pt-1">
										{#if t.upstream}											<div class="flex items-center gap-1.5">
												<ExternalLink class="size-3 shrink-0 text-muted-foreground" />
												<span class="truncate font-mono text-muted-foreground"
													>{t.upstream}</span
												>
											</div>
										{:else}
											<div class="text-muted-foreground/60 italic">
												no upstream (local only)
											</div>
										{/if}
										<div class="flex items-center gap-1.5">
											<Server class="size-3 shrink-0 text-muted-foreground" />
											<span class="truncate font-mono">{endpointFor(t.type)}</span>
											<button
												class="ml-auto shrink-0 p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground"
												title="Copy endpoint URL"
												onclick={() => copyEndpoint(t.type)}
											>
												{#if copied === t.type}
													<span class="text-[10px] text-green-500">copied</span>
												{:else}
													<Copy class="size-3" />
												{/if}
											</button>
										</div>
									</Card.Content>
								</Card.Root>
							{/each}
						</div>
					</section>

					<!-- OCI images -->
					<section>
						<div class="flex items-center gap-2 mb-3">
							<Box class="size-4 text-muted-foreground" />
							<h3 class="text-sm font-semibold">OCI Image Catalog</h3>
							<span class="text-xs text-muted-foreground">({repositories.length})</span>
						</div>
						{#if repositories.length === 0}
							<div class="rounded-md border border-dashed px-4 py-5 text-center">
								<p class="text-sm text-muted-foreground">No images stored.</p>
								<p class="text-xs text-muted-foreground/70 mt-1">
									Push images via <span class="font-mono"
										>docker push &lt;registry&gt;/repo:tag</span
									>
								</p>
							</div>
						{:else}
							<div class="space-y-1">
								{#each repositories as repo (repo)}
									<div
										class="flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm bg-card/30"
									>
										<Box class="size-3.5 text-muted-foreground shrink-0" />
										<span class="font-mono truncate">{repo}</span>
									</div>
								{/each}
							</div>
						{/if}
					</section>

					<!-- zergx config (read-only) -->
					{#if zergxCfg}
						<section>
							<div class="flex items-center gap-2 mb-3">
								<Server class="size-4 text-muted-foreground" />
								<h3 class="text-sm font-semibold">Registry Backend Config</h3>
								<span class="text-[10px] text-muted-foreground/60 italic"
									>read-only</span
								>
							</div>
							<Card.Root>
								<Card.Content class="text-xs space-y-1.5 pt-3">
									<div class="flex items-center gap-2">
										<span class="text-muted-foreground w-24 shrink-0">self_base</span>
										<span class="font-mono truncate">{zergxCfg.self_base || '—'}</span>
									</div>
									<div class="flex items-center gap-2">
										<span class="text-muted-foreground w-24 shrink-0">http_proxy</span>
										<span class="font-mono truncate">{zergxCfg.http_proxy || '—'}</span>
									</div>
								</Card.Content>
							</Card.Root>
						</section>
					{/if}
				{/if}
			{:else}
				<!-- ════════ TAB 2: PACKAGES ════════ -->
				{#if pkgError}
					<div class="text-sm text-destructive bg-destructive/10 rounded-md px-3 py-2">
						{pkgError}
					</div>
				{/if}

				<!-- Filters -->
				<div class="flex items-center gap-2 flex-wrap">
					<div class="relative flex-1 min-w-[180px] max-w-sm">
						<Search
							class="size-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
						/>
						<input
							type="text"
							placeholder="Search packages..."
							bind:value={pkgQuery}
							oninput={() => {}}
							onkeydown={(e) => e.key === 'Enter' && loadPackages()}
							class="w-full rounded-md border border-input bg-background pl-9 pr-3 py-1.5 text-xs"
						/>
					</div>
					<select
						bind:value={typeFilter}
						onchange={loadPackages}
						class="rounded-md border border-input bg-background px-3 py-1.5 text-xs"
					>
						<option value="">All Types</option>
						{#each availableTypes as t}
							<option value={t}>{t}</option>
						{/each}
						{#each types as t}
							{#if !availableTypes.includes(t.type)}
								<option value={t.type}>{t.type}</option>
							{/if}
						{/each}
					</select>
					<Button variant="outline" size="sm" onclick={() => { pkgOffset = 0; loadPackages() }} disabled={pkgLoading}>
						<Search class="size-3.5" />
						<span class="hidden sm:inline">Search</span>
					</Button>
				</div>

				<!-- Package list -->
				{#if pkgLoading && pkgs.length === 0}
					<div
						class="flex items-center justify-center py-12 text-sm text-muted-foreground"
					>
						<Loader2 class="size-4 animate-spin mr-2" /> Loading packages...
					</div>
				{:else if pkgs.length === 0}
					<div
						class="flex flex-col items-center justify-center py-12 text-center border border-dashed rounded-lg"
					>
						<Package class="size-6 text-muted-foreground/40 mb-2" />
						<p class="text-sm text-muted-foreground">No packages registered yet.</p>
						<p class="text-xs text-muted-foreground/70 mt-1">
							Publish packages via any of the {types.length} registry endpoints.
						</p>
					</div>
				{:else}
					<div class="rounded-lg border border-border overflow-hidden">
						{#each pkgs as p (pkgKey(p))}
							<div
								class="group relative border-b border-border last:border-b-0 transition-colors {expandedKey ===
								pkgKey(p)
									? 'bg-accent/30'
									: 'hover:bg-accent/20'}"
							>
								<!-- Row -->
							<button
								type="button"
								class="flex items-center gap-2 px-3 py-2.5 cursor-pointer text-sm w-full text-left transition-colors hover:bg-accent/20 pr-10 {expandedKey ===
								pkgKey(p)
									? 'bg-accent/30'
									: ''}"
								onclick={() => toggleExpand(p)}
							>
									{#if expandedKey === pkgKey(p)}
										<ChevronDown class="size-3.5 text-muted-foreground shrink-0" />
									{:else}
										<ChevronRight class="size-3.5 text-muted-foreground shrink-0" />
									{/if}
									<span class="font-mono font-medium truncate flex-1 min-w-0"
										>{p.name}</span
									>
									<span
										class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground shrink-0 font-mono"
									>
										{p.type}
									</span>
									{#if p.latest_version}
										<span
											class="text-xs font-mono text-muted-foreground shrink-0 hidden sm:inline"
										>
											v{p.latest_version}
										</span>
									{/if}
									<span
										class="text-[10px] text-muted-foreground/70 shrink-0 hidden md:inline mr-6"
									>
										{p.versions} version{p.versions !== 1 ? 's' : ''}
									</span>
								</button>
								<button
									type="button"
									class="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
									title="Delete package"
									onclick={(e) => { e.stopPropagation(); confirmDelete = { type: p.type, name: p.name } }}
								>
									<Trash2 class="size-3.5" />
								</button>

								<!-- Expanded detail -->
								{#if expandedKey === pkgKey(p)}
									<div class="px-3 pb-3 pt-1 space-y-2 bg-background/50">
										{#if versionLoading}
											<div
												class="flex items-center gap-2 text-xs text-muted-foreground py-2"
											>
												<Loader2 class="size-3 animate-spin" /> Loading versions...
											</div>
										{:else if versionDetail.length === 0}
											<p class="text-xs text-muted-foreground py-2">No versions found.</p>
										{:else}
											{#each versionDetail as v (v.version)}
												<div
													class="rounded-md border border-border bg-card/50 overflow-hidden"
												>
													<div
														class="flex items-center gap-2 px-3 py-2 text-xs border-b border-border"
													>
														<span class="font-mono font-medium">{v.version}</span>
														<span class="text-muted-foreground">
															· <Download class="size-3 inline-block" />
															{v.download_count} downloads
														</span>
													<span class="text-muted-foreground/70 ml-auto">
														{fmtTime(v.created_unix)}
													</span>
													<button
														type="button"
														class="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive shrink-0"
														title="Delete version"
														onclick={() => (confirmDelete = { type: p.type, name: p.name, version: v.version })}
													>
														<Trash2 class="size-3" />
													</button>
												</div>
													{#if v.files.length > 0}
														<div class="divide-y divide-border">
															{#each v.files as f (f.name)}
																<a
																	href="/api/v1/package-files/{f.sha256}"
																	class="flex items-center gap-2 px-3 py-1.5 text-xs hover:bg-accent/30 transition-colors"
																	target="_blank"
																	rel="noopener"
																>
																	<FileIcon class="size-3 text-muted-foreground shrink-0" />
																	<span class="font-mono truncate flex-1">{f.name}</span>
																	<span class="text-muted-foreground/70 shrink-0"
																		>{fmtSize(f.size)}</span
																	>
																	<Download class="size-3 text-muted-foreground shrink-0" />
																</a>
															{/each}
														</div>
													{/if}
												</div>
											{/each}
										{/if}
									</div>
								{/if}
							</div>
						{/each}
					</div>
				<div class="flex items-center justify-between pt-2">
						<p class="text-xs text-muted-foreground/60">
							Showing {pkgOffset + 1}–{Math.min(pkgOffset + pkgs.length, pkgTotal)} of {pkgTotal}
						</p>
						<div class="flex items-center gap-1">
							<Button
								variant="outline"
								size="sm"
								disabled={pkgOffset === 0 || pkgLoading}
								onclick={() => { pkgOffset = Math.max(0, pkgOffset - PAGE_SIZE); loadPackages() }}
							>
								Prev
							</Button>
							<Button
								variant="outline"
								size="sm"
								disabled={pkgOffset + PAGE_SIZE >= pkgTotal || pkgLoading}
								onclick={() => { pkgOffset += PAGE_SIZE; loadPackages() }}
							>
								Next
							</Button>
						</div>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</div>

<!-- Delete confirmation modal -->
{#if confirmDelete}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		role="presentation"
		class="fixed inset-0 z-50 flex items-start justify-center pt-[20vh] bg-black/40"
		onclick={() => (confirmDelete = null)}
	>
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div
			role="presentation"
			class="bg-card border border-border rounded-lg shadow-xl w-full max-w-sm mx-4 space-y-3 p-4"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-semibold">Confirm Delete</h3>
				<Button
					variant="ghost"
					size="icon"
					class="size-6"
					onclick={() => (confirmDelete = null)}
				>
					<X class="size-3.5" />
				</Button>
			</div>
			<p class="text-xs text-muted-foreground">
				Delete
				{#if confirmDelete.version}
					version <span class="font-mono font-medium text-foreground">{confirmDelete.version}</span>
					of
				{/if}
				package <span class="font-mono font-medium text-foreground">{confirmDelete.name}</span>
				<span class="text-[10px] px-1.5 py-0.5 rounded bg-muted font-mono">{confirmDelete.type}</span>?
				{#if !confirmDelete.version}
					<br />This will remove <strong>all versions</strong>.
				{/if}
			</p>
			<div class="flex items-center gap-2 pt-1">
				<Button
					size="sm"
					variant="destructive"
					onclick={doDelete}
					disabled={deletingPkg !== null}
				>
					{#if deletingPkg}
						<Loader2 class="size-3.5 animate-spin mr-1" />
					{:else}
						<Trash2 class="size-3.5 mr-1" />
					{/if}
					Delete
				</Button>
				<Button variant="ghost" size="sm" onclick={() => (confirmDelete = null)}>
					Cancel
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Publish Modal -->
{#if showPublish}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] bg-black/40" role="presentation" onclick={() => showPublish = false}>
		<div class="bg-card border border-border rounded-lg shadow-xl w-full max-w-lg mx-4 space-y-3 p-4" role="dialog" tabindex="-1" aria-label="Publish package" onclick={(e) => e.stopPropagation()}>
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-semibold">Publish Package</h3>
				<Button variant="ghost" size="icon" class="size-6" onclick={() => showPublish = false}><X class="size-3.5" /></Button>
			</div>

			<div>
				<label class="text-xs font-medium text-muted-foreground" for="pub-protocol">Protocol</label>
				<select id="pub-protocol" class="mt-1 w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs" bind:value={publishProtocol}>
					<option value="">Select protocol</option>
					{#each publishSpecs as s (s.protocol)}
						<option value={s.protocol}>{s.protocol}</option>
					{/each}
				</select>
			</div>
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class="text-xs font-medium text-muted-foreground" for="pub-org">Org</label>
					<input id="pub-org" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs font-mono" bind:value={publishOrg} />
				</div>
				<div>
					<label class="text-xs font-medium text-muted-foreground" for="pub-repo">Repo</label>
					<input id="pub-repo" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs font-mono" bind:value={publishRepo} />
				</div>
			</div>
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class="text-xs font-medium text-muted-foreground" for="pub-bookmark">Bookmark</label>
					<input id="pub-bookmark" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs font-mono" bind:value={publishBookmark} />
				</div>
				<div>
					<label class="text-xs font-medium text-muted-foreground" for="pub-version">Version</label>
					<input id="pub-version" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs font-mono" bind:value={publishVersion} />
				</div>
			</div>
			<div>
				<label class="text-xs font-medium text-muted-foreground" for="pub-name">Package name (optional)</label>
				<input id="pub-name" type="text" class="mt-1 w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs font-mono" bind:value={publishName} />
			</div>

			{#if publishError}
				<p class="text-xs text-destructive">{publishError}</p>
			{/if}

			{#if publishing || publishLogs.length > 0}
				<div class="rounded border border-border bg-muted/30 p-2 max-h-48 overflow-y-auto" aria-label="Publish log">
					{#if publishState}<p class="text-[10px] text-muted-foreground mb-1 font-mono">state: {publishState}</p>{/if}
					{#each publishLogs as ln, i (i)}
						<pre class="text-[10px] leading-relaxed font-mono whitespace-pre-wrap {ln.stream === 'stderr' ? 'text-red-500' : ''}">{ln.line}</pre>
					{/each}
					{#if publishing}<div class="flex items-center gap-1.5 text-[10px] text-muted-foreground mt-1"><Loader2 class="size-3 animate-spin" /> streaming publish output…</div>{/if}
				</div>
			{/if}

			<div class="flex items-center gap-2 pt-1">
				<Button size="sm" onclick={onPublish} disabled={publishing || !publishProtocol}>
					{#if publishing}<Loader2 class="size-3.5 animate-spin mr-1" />{/if}Publish
				</Button>
				<Button variant="ghost" size="sm" onclick={() => showPublish = false}>Cancel</Button>
			</div>
		</div>
	</div>
{/if}
