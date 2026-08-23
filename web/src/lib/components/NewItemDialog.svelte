<script lang="ts">
import { Button } from '$lib/components/ui/button'

let {
  open = $bindable(false),
  title,
  label,
  placeholder,
  confirmText = 'Create',
  onSubmit,
}: {
  open?: boolean
  title: string
  label?: string
  placeholder?: string
  confirmText?: string
  onSubmit: (value: string) => Promise<void> | void
} = $props()

let value = $state('')
let busy = $state(false)
let error = $state('')

async function submit() {
  const v = value.trim()
  if (!v) {
    error = 'Name required'
    return
  }
  error = ''
  busy = true
  await onSubmit(v)
  busy = false
  value = ''
  open = false
}

function reset() {
  value = ''
  error = ''
  busy = false
}
</script>

{#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="fixed inset-0 z-50" onclick={() => { open = false; reset() }} role="presentation"></div>
    <div class="fixed inset-0 z-50 flex items-center justify-center pointer-events-none">
        <div class="pointer-events-auto relative w-full max-w-sm rounded-lg border bg-card p-5 shadow-lg mx-4" role="dialog" aria-label={title}>
            <h2 class="text-base font-semibold mb-3">{title}</h2>
            {#if label}
                <label class="text-[11px] font-semibold text-muted-foreground block mb-1" for="nid-input">{label}</label>
            {/if}
            <!-- svelte-ignore a11y_autofocus -->
            <input
                id="nid-input"
                class="w-full rounded-md border {error ? 'border-destructive' : 'border-input'} bg-background px-3 py-2 text-sm"
                {placeholder}
                bind:value
                oninput={() => (error = '')}
                onkeydown={(e) => { if (e.key === 'Enter') submit() }}
                onclose={() => reset()}
                autofocus
            />
            {#if error}
                <p class="text-xs text-destructive mt-1.5">{error}</p>
            {/if}
            <div class="flex gap-2 justify-end mt-4">
                <Button variant="outline" size="sm" onclick={() => { open = false; reset() }}>Cancel</Button>
                <Button size="sm" onclick={submit} disabled={busy || !value.trim()}>
                    {busy ? 'Working...' : confirmText}
                </Button>
            </div>
        </div>
    </div>
{/if}
