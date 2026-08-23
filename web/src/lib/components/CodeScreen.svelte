<script lang="ts">
import { ArrowLeft } from '@lucide/svelte'
import { onMount } from 'svelte'
import * as api from '$lib/api'
import { Button } from '$lib/components/ui/button'

let {
  path,
  sessionOrg,
  sessionRepo,
  onclose,
}: {
  path: string
  sessionOrg?: string
  sessionRepo?: string
  onclose: () => void
} = $props()

let content = $state('')

onMount(async () => {
  if (!sessionOrg || !sessionRepo) return
  const r = await api.repos.readFile(sessionOrg, sessionRepo, path)
  content = r.isOk() ? r.value : ''
})
</script>

<div class="flex flex-col h-full">
    <div class="flex items-center gap-2 border-b border-border px-3 py-2 shrink-0">
        <Button variant="ghost" size="icon" onclick={onclose}><ArrowLeft class="size-4" /></Button>
        <span class="text-sm font-medium">{path.split("/").pop() || path}</span>
    </div>
    <div class="flex-1 overflow-auto">
        <pre class="p-3 text-xs font-mono whitespace-pre">{content}</pre>
    </div>
</div>
