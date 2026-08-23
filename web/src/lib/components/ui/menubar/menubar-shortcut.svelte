<script lang="ts">
import type { HTMLAttributes } from 'svelte/elements'
import { cn, type WithElementRef } from '$lib/utils.js'

let {
  ref = $bindable(null),
  class: className,
  children,
  ...restProps
}: WithElementRef<HTMLAttributes<HTMLSpanElement>> = $props()
</script>

<span
	bind:this={ref}
	data-slot="menubar-shortcut"
	class={cn("text-xs tracking-widest text-muted-foreground group-focus/menubar-item:text-accent-foreground ml-auto", className)}
	{...restProps}
>
	{@render children?.()}
</span>
