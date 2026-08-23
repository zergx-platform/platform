<script lang="ts">
import { onMount } from 'svelte'
import { highlightTokens, type ThemedToken } from '$lib/highlight'

let { code, filepath }: { code: string; filepath: string } = $props()

let appTheme = $state<string>(
  typeof document !== 'undefined'
    ? document.documentElement.getAttribute('data-theme') || 'dark'
    : 'dark',
)

onMount(() => {
  const observer = new MutationObserver(() => {
    appTheme = document.documentElement.getAttribute('data-theme') || 'dark'
  })
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme'],
  })
  return () => observer.disconnect()
})

const theme = $derived(
  appTheme === 'dark' ? ('dark-plus' as const) : ('github-light' as const),
)

const lines = $derived(code.split('\n'))

let tokenRows = $state<ThemedToken[][]>([])

function tokensToHtml(tokens: ThemedToken[]): string {
  return tokens
    .map(t => {
      const s: string[] = [`color:${t.color}`]
      if (t.fontStyle === 1) s.push('font-style:italic')
      if (t.fontStyle === 2) s.push('font-weight:bold')
      const esc = t.content
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
      return `<span style="${s.join(';')}">${esc}</span>`
    })
    .join('')
}

$effect(() => {
  const c = code
  const fp = filepath
  if (!c || !fp) {
    tokenRows = []
    return
  }
  let cancelled = false
  highlightTokens(c, fp, theme).then(tokens => {
    if (!cancelled) tokenRows = tokens
  })
  return () => {
    cancelled = true
  }
})
</script>

<div class="h-full overflow-auto">
    <table class="w-full font-mono text-xs leading-relaxed">
        <tbody>
            {#each lines as line, i}
                <tr>
                    <td class="text-right pr-2 select-none w-12 border-r border-border/50 align-top sticky left-0 bg-background text-muted-foreground/60 whitespace-pre-wrap break-words">{i + 1}</td>
                    <td class="pl-2 whitespace-pre-wrap break-words align-top">{@html tokensToHtml(tokenRows[i] ?? [])}</td>
                </tr>
            {/each}
        </tbody>
    </table>
</div>
