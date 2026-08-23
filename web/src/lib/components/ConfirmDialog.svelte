<script lang="ts">
import { Loader2, Trash2, X } from '@lucide/svelte'
import { Button } from '$lib/components/ui/button'

let {
  open = false,
  title = 'Confirm Delete',
  description = '',
  confirmText = 'Delete',
  busy = false,
  onConfirm,
  onClose,
}: {
  open?: boolean
  title?: string
  description?: string
  confirmText?: string
  busy?: boolean
  onConfirm: () => Promise<void> | void
  onClose?: () => void
} = $props()

function close() {
  if (busy) return
  onClose?.()
}
</script>

{#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
        role="presentation"
        class="fixed inset-0 z-50 flex items-start justify-center pt-[20vh] bg-black/40"
        onclick={close}
    >
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
            role="presentation"
            class="bg-card border border-border rounded-lg shadow-xl w-full max-w-sm mx-4 space-y-3 p-4"
            onclick={(e) => e.stopPropagation()}
        >
            <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold">{title}</h3>
                <Button variant="ghost" size="icon" class="size-6" onclick={close}>
                    <X class="size-3.5" />
                </Button>
            </div>
            <p class="text-xs text-muted-foreground">{@html description}</p>
            <div class="flex items-center gap-2 pt-1">
                <Button
                    size="sm"
                    variant="destructive"
                    onclick={async () => { await onConfirm(); onClose?.() }}
                    disabled={busy}
                >
                    {#if busy}<Loader2 class="size-3.5 animate-spin mr-1" />{:else}<Trash2 class="size-3.5 mr-1" />{/if}
                    {confirmText}
                </Button>
                <Button variant="ghost" size="sm" onclick={close}>Cancel</Button>
            </div>
        </div>
    </div>
{/if}
