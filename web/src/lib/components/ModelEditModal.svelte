<script lang="ts">
import type { ProviderModel } from '@recoder-neo/schema'
import { Button } from '$lib/components/ui/button'

let {
  editModel,
  onSave,
  onCancel,
}: {
  editModel: ProviderModel
  onSave: () => void
  onCancel: () => void
} = $props()
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" role="presentation" onclick={onCancel}>
    <div class="bg-card border border-border rounded-lg shadow-xl w-full max-w-sm p-4 space-y-3" role="dialog" tabindex="-1" aria-label="Edit model" onclick={(e) => e.stopPropagation()}>
        <h4 class="text-sm font-semibold">Edit {editModel.name || editModel.id}</h4>
        <div class="grid grid-cols-2 gap-2">
            <div>
                <label class="text-[10px] font-medium" for="em-context">Context Limit</label>
                <input id="em-context" type="number" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={editModel.context_limit} />
            </div>
            <div>
                <label class="text-[10px] font-medium" for="em-output">Output Limit</label>
                <input id="em-output" type="number" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={editModel.output_limit} />
            </div>
            <div>
                <label class="text-[10px] font-medium" for="em-max-tokens">Max Tokens</label>
                <input id="em-max-tokens" type="number" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={editModel.max_tokens} />
            </div>
            <div>
                <label class="text-[10px] font-medium" for="em-temp">Temperature (0-100)</label>
                <input id="em-temp" type="number" min="0" max="100" class="mt-0.5 w-full rounded border border-input bg-background px-2 py-1 text-xs" bind:value={editModel.temperature} placeholder="70" />
            </div>
        </div>
        <div class="flex items-center gap-2 text-[10px]">
            <label class="flex items-center gap-1"><input type="checkbox" bind:checked={editModel.reasoning} class="accent-primary" /> Reasoning</label>
            <label class="flex items-center gap-1"><input type="checkbox" bind:checked={editModel.tool_call} class="accent-primary" /> Tool Call</label>
        </div>
        <div class="flex items-center gap-2 pt-1">
            <Button size="sm" class="text-[10px]" onclick={onSave}>Save</Button>
            <Button variant="ghost" size="sm" class="text-[10px]" onclick={onCancel}>Cancel</Button>
        </div>
    </div>
</div>
