<script lang="ts">
import { Pencil, Trash2 } from '@lucide/svelte'
import type { ProviderInfo, ProviderModel } from '@rucoder/schema'
import { Button } from '$lib/components/ui/button'

let {
  providerId,
  provider,
  onEditModel,
  onDelete,
}: {
  providerId: string
  provider: ProviderInfo
  onEditModel: (pid: string, idx: number, m: ProviderModel) => void
  onDelete: (id: string) => void
} = $props()

function fmt(n?: number): string {
  if (!n) return ''
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`
  return String(n)
}
</script>

<div class="rounded-md border border-border bg-card/50 overflow-hidden">
    <div class="flex items-center gap-2 px-3 py-2">
        <span class="text-sm font-medium flex-1">{providerId}</span>
        <span class="text-[9px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded font-mono">{provider.api_type}</span>
        <span class="text-[10px] text-muted-foreground break-all max-w-[200px] text-right">{provider.base_url}</span>
        <span class="text-[10px] text-muted-foreground">{provider.models?.length || 0} models</span>
        {#if provider.headers && Object.keys(provider.headers).length > 0}
            <span class="text-[9px] bg-blue-500/10 text-blue-400 px-1 rounded">headers</span>
        {/if}
        <Button variant="ghost" size="icon" class="size-6" title="Delete" onclick={() => onDelete(providerId)}>
            <Trash2 class="size-3.5" />
        </Button>
    </div>
    {#if provider.models && provider.models.length > 0}
        <div class="border-t border-border/50 px-3 py-1.5 bg-muted/20">
            <div class="grid grid-cols-[1fr_60px_60px_auto] gap-2 text-[10px] text-muted-foreground mb-1">
                <span>Model</span><span>Ctx</span><span>Out</span><span></span>
            </div>
            {#each provider.models as m, i}
                <div class="grid grid-cols-[1fr_60px_60px_auto] gap-2 text-xs items-center py-0.5">
                    <span class="truncate">{m.name}<span class="text-[9px] text-muted-foreground ml-1">({m.id})</span></span>
                    <span class="text-[10px] font-mono">{fmt(m.context_limit)}</span>
                    <span class="text-[10px] font-mono">{fmt(m.output_limit)}</span>
                    <Button variant="ghost" size="icon" class="size-5" title="Edit" onclick={() => onEditModel(providerId, i, m)}>
                        <Pencil class="size-3" />
                    </Button>
                </div>
            {/each}
        </div>
    {/if}
</div>
