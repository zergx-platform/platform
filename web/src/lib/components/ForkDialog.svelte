<script lang="ts">
import { getStore } from '$lib/stores.svelte'

const store = getStore()

import { Button } from '$lib/components/ui/button'

let bookmark = $state('')
let forking = $state(false)
let error = $state('')

async function submit() {
  const b = bookmark.trim()
  if (!b) {
    error = 'Bookmark name required'
    return
  }
  if (store.existingBookmarks.includes(b)) {
    error = 'Bookmark already exists'
    return
  }
  error = ''
  forking = true
  const ok = await store.forkSession(b)
  forking = false
  if (ok) store.closeFork()
}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center">
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="absolute inset-0 bg-black/60" onclick={store.closeFork} role="presentation"></div>
    <div class="relative z-10 w-full max-w-sm rounded-lg border bg-card p-6 shadow-lg">
        <h2 class="text-lg font-semibold mb-4">Fork Session</h2>
        <div class="space-y-4">
            <!-- svelte-ignore a11y_autofocus -->
            <input
                class="w-full rounded-md border bg-background px-3 py-2 text-sm {error ? 'border-destructive' : 'border-input'}"
                placeholder="Bookmark name"
                bind:value={bookmark}
                oninput={() => error = ""}
                onkeydown={(e) => { if (e.key === "Enter") submit() }}
                autofocus
            />
            {#if error}
                <p class="text-xs text-destructive">{error}</p>
            {/if}
            <div class="flex gap-2 justify-end">
                <Button variant="outline" onclick={store.closeFork}>Cancel</Button>
                <Button onclick={submit} disabled={forking || !bookmark.trim()}>
                    {forking ? "Forking..." : "Fork"}
                </Button>
            </div>
        </div>
    </div>
</div>
