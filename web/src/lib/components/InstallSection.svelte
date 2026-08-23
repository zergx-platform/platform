<script lang="ts">
import { Check, Download, Smartphone } from '@lucide/svelte'
import { Button } from '$lib/components/ui/button'
import { getPwaStore } from '$lib/stores/pwa.svelte'

const pwa = getPwaStore()
</script>

<div class="flex items-start gap-3">
    <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary shrink-0">
        <Smartphone class="size-5" />
    </div>
    <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
            <span class="text-sm font-medium">Install to home screen</span>
            {#if pwa.installed}
                <span class="text-[10px] text-green-500 flex items-center gap-0.5"><Check class="size-3" /> Installed</span>
            {/if}
        </div>
        <p class="text-xs text-muted-foreground mt-0.5">
            {#if pwa.installed}
                Running in standalone mode. The shortcut is on your home screen.
            {:else if pwa.canInstall}
                Installing adds ReCoder to your home screen for a full-screen, app-like experience.
            {:else}
                Install ReCoder as a standalone app for a full-screen, app-like experience.
            {/if}
        </p>
        {#if pwa.installed}
            <span class="text-xs text-green-500 flex items-center gap-1 mt-2"><Check class="size-3.5" /> Installed — running as a standalone app</span>
        {:else if pwa.canInstall}
            <Button size="sm" class="mt-2" onclick={() => pwa.promptInstall()}>
                <Download class="size-3.5 mr-1" /> Install
            </Button>
        {:else}
            <p class="text-[10px] text-muted-foreground mt-2">
                Use your browser menu
                <span class="font-mono">&ldquo;Add to Home screen&rdquo;</span> to install.
                {#if pwa.isAndroid}
                    On Android: tap the ⋮ menu → <span class="font-medium">&ldquo;Add to Home screen&rdquo;</span> (or <span class="font-medium">&ldquo;Install app&rdquo;</span>).
                {/if}
            </p>
        {/if}
    </div>
</div>
