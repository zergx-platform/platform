<script lang="ts">
import { onMount } from 'svelte'
import { highlightTokens, type ThemedToken } from '$lib/highlight'

let { output, filePath }: { output: string; filePath?: string } = $props()

interface ReadLine {
  lineNum: number | null
  content: string
  isFooter: boolean
}

const readLines = $derived.by(() => {
  const lines = output.split('\n')
  const result: ReadLine[] = []
  for (const line of lines) {
    const m = line.match(/^(\d+):\s?(.*)$/)
    if (m) {
      result.push({
        lineNum: parseInt(m[1], 10),
        content: m[2],
        isFooter: false,
      })
    } else {
      result.push({ lineNum: null, content: line, isFooter: true })
    }
  }
  return result
})

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

const codeRows = $derived.by(() =>
  readLines.filter(l => !l.isFooter).map(l => l.content),
)

$effect(() => {
  const code = codeRows
  highlightTokens(code.join('\n'), filePath ?? '', 'dark-plus').then(tokens => {
    tokenRows =
      tokens.length === code.length
        ? tokens
        : tokens.slice(0, code.length) || tokens
  })
})
</script>

<div class="bg-muted rounded overflow-hidden text-[11px] max-h-64 overflow-y-auto">
    {#if readLines.length > 0}
        <table class="w-full font-mono leading-relaxed">
            <tbody>
                {#each readLines as l, i}
                    <tr class={l.isFooter ? 'text-muted-foreground/60 italic' : ''}>
                        <td class="text-right pr-2 select-none w-12 border-r border-border/50 align-top sticky left-0 {l.isFooter ? 'text-muted-foreground/40' : 'text-muted-foreground/60'}">
                            {l.lineNum != null ? l.lineNum : ''}
                        </td>
                        {#if l.isFooter}
                            <td class="pl-2 whitespace-pre-wrap break-all">{l.content}</td>
                        {:else}
                            <td class="pl-2 whitespace-pre-wrap break-all">{@html tokensToHtml(tokenRows[i] ?? [])}</td>
                        {/if}
                    </tr>
                {/each}
            </tbody>
        </table>
    {:else}
        <pre class="p-1.5 whitespace-pre-wrap break-all font-mono">{output}</pre>
    {/if}
</div>
