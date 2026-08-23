<script lang="ts">
import { Moon, Palette, Sun } from '@lucide/svelte'
import {
  ALL_THEMES,
  applyTheme,
  getSavedMode,
  getSavedTheme,
  isDarkOnlyTheme,
} from '$lib/stores/theme.svelte'
import { themeData } from '$lib/stores/themes-data'

let currentTheme = $state(
  typeof localStorage !== 'undefined' ? getSavedTheme() : 'opencode',
)
let mode = $state<'dark' | 'light'>(
  typeof localStorage !== 'undefined' ? getSavedMode() : 'dark',
)
let themeQuery = $state('')

const themeLabels: Record<string, string> = {
  opencode: 'OpenCode',
  tokyonight: 'Tokyo Night',
  everforest: 'Everforest',
  catppuccin: 'Catppuccin',
  'catppuccin-frappe': 'Catppuccin Frappé',
  'catppuccin-macchiato': 'Catppuccin Macchiato',
  ayu: 'Ayu',
  aura: 'Aura',
  nord: 'Nord',
  gruvbox: 'Gruvbox',
  kanagawa: 'Kanagawa',
  matrix: 'Matrix',
  'one-dark': 'One Dark',
  carbonfox: 'Carbonfox',
  cobalt2: 'Cobalt2',
  cursor: 'Cursor',
  dracula: 'Dracula',
  flexoki: 'Flexoki',
  github: 'GitHub',
  'lucent-orng': 'Lucent Orange',
  material: 'Material',
  mercury: 'Mercury',
  monokai: 'Monokai',
  nightowl: 'Night Owl',
  orng: 'Orange',
  'osaka-jade': 'Osaka Jade',
  palenight: 'Palenight',
  rosepine: 'Rose Pine',
  solarized: 'Solarized',
  synthwave84: 'Synthwave 84',
  vercel: 'Vercel',
  vesper: 'Vesper',
  zenburn: 'Zenburn',
}

const isDarkOnly = $derived(isDarkOnlyTheme(currentTheme))
const filteredThemes = $derived(
  ALL_THEMES.filter(
    t =>
      !themeQuery ||
      t.toLowerCase().includes(themeQuery.toLowerCase()) ||
      (themeLabels[t] || t).toLowerCase().includes(themeQuery.toLowerCase()),
  ),
)

function pickTheme(tid: string) {
  currentTheme = tid
  applyTheme(tid, mode)
}

function toggleMode() {
  mode = mode === 'dark' ? 'light' : 'dark'
  applyTheme(currentTheme, mode)
}
</script>

<section>
    <h3 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">Appearance</h3>
    <div class="flex items-center gap-3 mb-3">
        <Palette class="size-4 text-muted-foreground shrink-0" />
        <select
            class="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            value={currentTheme}
            onchange={(e) => pickTheme(e.currentTarget.value)}
        >
            {#each ALL_THEMES as tid (tid)}
                <option value={tid}>{themeLabels[tid] || tid}</option>
            {/each}
        </select>
        <button
            type="button"
            class="flex items-center gap-1.5 rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-50"
            disabled={isDarkOnly}
            onclick={toggleMode}
            title={isDarkOnly ? 'Theme is dark-only' : 'Toggle light/dark'}
        >
            {#if mode === 'dark'}
                <Sun class="size-4" />
                Light
            {:else}
                <Moon class="size-4" />
                Dark
            {/if}
        </button>
    </div>
    <input
        type="text"
        class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        placeholder="Search themes..."
        bind:value={themeQuery}
    />
    {#if themeQuery}
        <div class="mt-2 grid grid-cols-2 sm:grid-cols-3 gap-1.5 max-h-48 overflow-y-auto">
            {#each filteredThemes as tid (tid)}
                <button
                    type="button"
                    class="flex items-center gap-1.5 rounded-md border border-border px-2 py-1.5 text-xs text-left hover:bg-accent transition-colors {tid === currentTheme ? 'border-primary' : ''}"
                    onclick={() => pickTheme(tid)}
                >
                    <span class="w-3 h-3 rounded-full border border-border shrink-0" style="background:{themeData[tid]?.dark?.primary || '#888'}"></span>
                    <span class="truncate">{themeLabels[tid] || tid}</span>
                    {#if tid === currentTheme}<span class="ml-auto font-medium">✓</span>{/if}
                </button>
            {/each}
        </div>
    {/if}
</section>
