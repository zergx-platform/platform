<script lang="ts">
import Terminal from './ContainerTerminal.svelte'
import Jobs from './JobPanel.svelte'

let {
  containerId = '',
  containerName = '',
  onclose = () => {},
  crumbs = [],
}: {
  containerId: string
  containerName: string
  onclose: () => void
  crumbs?: { label: string; onclick: () => void }[]
} = $props()

let tab = $state<'terminal' | 'jobs'>('terminal')
</script>

<div class="h-full flex flex-col">
    <div class="flex items-center gap-1 border-b border-border px-3 shrink-0">
        <button
            class="px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors {tab === 'terminal' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
            onclick={() => (tab = 'terminal')}
        >
            Terminal
        </button>
        <button
            class="px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors {tab === 'jobs' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
            onclick={() => (tab = 'jobs')}
        >
            Jobs
        </button>
    </div>
    <div class="flex-1 min-h-0">
        {#if tab === 'terminal'}
            <Terminal {containerId} {containerName} {onclose} {crumbs} />
        {:else}
            <Jobs {containerId} {containerName} {onclose} {crumbs} />
        {/if}
    </div>
</div>
