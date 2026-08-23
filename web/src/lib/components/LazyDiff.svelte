<script lang="ts">
import { onMount } from 'svelte'
import * as api from '$lib/api'
import DiffView from './DiffView.svelte'

let {
  sessionOrg,
  sessionRepo,
  changeId,
  filePath,
}: {
  sessionOrg: string
  sessionRepo: string
  changeId: string
  filePath: string
} = $props()

let diffText = $state('')

onMount(async () => {
  const r = await api.repos.fileDiff(
    sessionOrg,
    sessionRepo,
    changeId,
    filePath,
  )
  if (r.isOk()) diffText = r.value
})
</script>

{#if diffText}
    <DiffView diffText={diffText} />
{:else}
    <p class="text-[10px] text-muted-foreground px-2 py-1 italic">no diff</p>
{/if}
