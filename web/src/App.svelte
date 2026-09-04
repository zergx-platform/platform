<script lang="ts">
import { onMount } from 'svelte'
import { navigate, router, switchTab } from '$lib/router.svelte'
import { createPwaStore } from '$lib/stores/pwa.svelte'
import { initTheme } from '$lib/stores/theme.svelte'
import { createStore } from '$lib/stores.svelte'

createPwaStore()

const store = createStore(
  typeof localStorage !== 'undefined'
    ? localStorage.getItem('oc-appearance') || 'dark'
    : 'dark',
)

import {
  Box,
  FolderGit2,
  MessageSquare,
  Package,
  Settings,
  Wrench,
} from '@lucide/svelte'
import ChatPage from '$lib/components/ChatPage.svelte'
import ChatSidebar from '$lib/components/ChatSidebar.svelte'
import CodePage from '$lib/components/CodePage.svelte'
import ConfigPage from '$lib/components/ConfigPage.svelte'
import ForkDialog from '$lib/components/ForkDialog.svelte'
import ProviderManager from '$lib/components/ProviderManager.svelte'
import { Button } from '$lib/components/ui/button'
import ContainersPage from '$lib/pages/ContainersPage.svelte'
import PackagesPage from '$lib/pages/PackagesPage.svelte'
import { VERSION } from '$lib/version'

let showProviders = $state(false)

const NAV = [
  { route: 'chat' as const, icon: MessageSquare, label: 'Chat' },
  { route: 'code' as const, icon: FolderGit2, label: 'Code' },
  { route: 'containers' as const, icon: Box, label: 'Containers' },
  { route: 'packages' as const, icon: Package, label: 'Packages' },
  { route: 'config' as const, icon: Wrench, label: 'Config' },
]

onMount(() => {
  initTheme()
  store.refreshSessions()
  store.refreshRepos()
})

let syncingUrl = false

const syncFromHash = () => {
  const loc = router.chat
  if (loc.sessionId) {
    if (store.activeSessionId !== loc.sessionId) {
      store.activeSessionId = loc.sessionId
    }
    if ((store.sessionOverlay ?? null) !== loc.overlay) {
      store.sessionOverlay = loc.overlay
    }
    if ((store.diffChangeId ?? null) !== loc.changeId) {
      store.diffChangeId = loc.changeId
    }
    // Files overlay drill-in is url-driven like timeline.
    if (loc.overlay === 'files') {
      const wanted = loc.filePath
      if (wanted && store.selectedFilePath !== wanted) {
        void store.openFileFromUrl(wanted, loc.fileChangeId ?? undefined)
      } else if (store.selectedFilePath && !wanted) {
        store.selectedFilePath = null
        store.fileContent = ''
        store.activeDiffChangeId = null
        store.showFileHistory = false
      }
      if (wanted && (store.activeDiffChangeId ?? null) !== loc.fileChangeId) {
        if (loc.fileChangeId) {
          store.openFileChange(wanted, loc.fileChangeId)
        } else {
          store.activeDiffChangeId = null
        }
      }
    }
  } else if (store.activeSessionId) {
    store.activeSessionId = null
    store.sessionOverlay = null
    store.diffChangeId = null
  }
}

$effect(() => {
  void router.current
  void router.chat.sessionId
  void router.chat.overlay
  void router.chat.changeId
  void router.chat.filePath
  void router.chat.fileChangeId
  syncFromHash()
})

const showTabBar = $derived(router.current !== 'chat' || !store.activeSession)
</script>

<div
    class="flex h-dvh w-full max-w-[100vw] flex-col overflow-hidden bg-background text-foreground"
    style="padding-top: env(safe-area-inset-top); padding-left: env(safe-area-inset-left); padding-right: env(safe-area-inset-right);"
>
    <!-- Desktop: compact icon activity bar + main outlet -->
    <div class="flex-1 min-h-0 flex">
        <aside
            class="hidden lg:flex shrink-0 w-12 flex-col items-center border-r border-border bg-card py-2 gap-1"
            aria-label="Primary navigation"
        >
            {#each NAV as n (n.route)}
                <button
                    type="button"
                    class="flex items-center justify-center size-9 rounded-md transition-colors {router.current === n.route ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'}"
                    onclick={() => switchTab(n.route)}
                    aria-current={router.current === n.route ? 'page' : undefined}
                    title={n.label}
                >
                    <n.icon class="size-5" />
                </button>
            {/each}
        </aside>

        <!-- Page outlet -->
        <div class="flex-1 min-h-0 page-enter" data-route={router.current}>
            {#if router.current === "containers"}
                <ContainersPage />
            {:else if router.current === "packages"}
                <PackagesPage />
            {:else if router.current === "config"}
                <div class="h-full overflow-auto">
                    <ConfigPage />
                </div>
            {:else if router.current === "chat"}
                {#if store.activeSession}
                    <ChatPage />
                {:else}
                    <div class="h-full flex flex-col">
                        <div class="px-4 pt-4 pb-2 shrink-0 flex items-center gap-2">
                            <h1 class="text-lg font-semibold">Sessions</h1>
                            <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-muted text-muted-foreground lg:hidden" title="deployed version">{VERSION}</span>
                            <div class="flex-1"></div>
                            <Button variant="ghost" size="icon" class="size-8" onclick={() => (showProviders = true)} title="Providers">
                                <Settings class="size-4" />
                            </Button>
                        </div>
                        <div class="text-xs text-muted-foreground px-4 pb-2 shrink-0">Pick a repo &amp; bookmark to start chatting</div>
                        <div class="flex-1 overflow-y-auto">
                            <ChatSidebar />
                        </div>
                    </div>
                {/if}
            {:else if router.current === "code"}
                <CodePage />
            {/if}
        </div>
    </div>

    <!-- Bottom tab bar (mobile only; hidden inside a chat conversation) -->
    {#if showTabBar}
        <nav
            class="shrink-0 flex items-stretch border-t border-border bg-card lg:hidden"
            style="padding-bottom: env(safe-area-inset-bottom);"
            aria-label="Primary"
        >
            {#each NAV as n (n.route)}
                <button
                    type="button"
                    class="flex flex-col items-center justify-center gap-0.5 flex-1 py-1.5 transition-colors {router.current === n.route ? 'text-primary' : 'text-muted-foreground hover:text-foreground'}"
                    onclick={() => switchTab(n.route)}
                    aria-current={router.current === n.route ? 'page' : undefined}
                >
                    <n.icon class="size-5" />
                    <span class="text-[10px] leading-none">{n.label}</span>
                </button>
            {/each}
        </nav>
    {/if}

    {#if store.showFork}
        <ForkDialog />
    {/if}
    {#if showProviders}
        <ProviderManager onclose={() => showProviders = false} />
    {/if}
</div>
