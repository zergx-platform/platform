<script lang="ts">
import {
  Braces,
  FileCode,
  FileSearch,
  FileText,
  Globe,
  ListTodo,
  Pencil,
  ScanSearch,
  SquareTerminal,
  Trash2,
  Wrench,
} from '@lucide/svelte'

let { name }: { name: string } = $props()

const n = $derived(name.toLowerCase())
</script>

{#if n.includes('sandbox-run') || n.includes('shell') || n.includes('exec')}
    <SquareTerminal class="size-3.5 text-amber-500 shrink-0" />
{:else if n.includes('read')}
    <FileText class="size-3.5 text-sky-500 shrink-0" />
{:else if n.includes('grep')}
    <ScanSearch class="size-3.5 text-sky-500 shrink-0" />
{:else if n.includes('glob') || n.includes('ls') || n.includes('explore')}
    <FileSearch class="size-3.5 text-sky-500 shrink-0" />
{:else if n.includes('write') || n.includes('edit')}
    <Pencil class="size-3.5 text-emerald-500 shrink-0" />
{:else if n.includes('delete')}
    <Trash2 class="size-3.5 text-rose-500 shrink-0" />
{:else if n.includes('web') || n.includes('fetch')}
    <Globe class="size-3.5 text-violet-500 shrink-0" />
{:else if n.includes('todo') || n.includes('job') || n.includes('task')}
    <ListTodo class="size-3.5 text-rose-500 shrink-0" />
{:else if n.includes('git') || n.includes('diff') || n.includes('commit')}
    <FileCode class="size-3.5 text-slate-400 shrink-0" />
{:else}
    <Wrench class="size-3.5 text-slate-400 shrink-0" />
{/if}
