<script lang="ts">
import { Circle, CircleCheck, CircleDot, X } from "@lucide/svelte";
import type { ContainerInfo, Todo } from "$lib/api";
import { Button } from "$lib/components/ui/button";
import ContainerWorkspace from "./ContainerWorkspace.svelte";
import DiffScreen from "./DiffScreen.svelte";
import FilesPage from "./FilesPage.svelte";
import MailboxPage from "./MailboxPage.svelte";
import TimelinePage from "./TimelinePage.svelte";

let {
	overlay,
	diffChangeId,
	sessionOrg,
	sessionRepo,
	containerRow,
	containerLoading,
	sessionWorkerId,
	todos,
	onclose,
	onselectFile,
	onselectDiff,
	oncreateContainer,
}: {
	overlay: "timeline" | "files" | "mailbox" | "container" | "todos" | null;
	diffChangeId: string | null;
	sessionOrg?: string;
	sessionRepo?: string;
	containerRow: ContainerInfo | null;
	containerLoading: boolean;
	sessionWorkerId: string | null;
	todos: Todo[];
	onclose: () => void;
	onselectFile: (path: string) => void;
	onselectDiff: (id: string) => void;
	oncreateContainer: () => void;
} = $props();
</script>

{#if overlay === 'timeline'}
    <div class="absolute inset-0">
        {#if diffChangeId}
            <DiffScreen
                changeId={diffChangeId}
                {sessionOrg}
                {sessionRepo}
                onclose={onclose}
                onselectFile={onselectFile}
            />
        {:else}
            <TimelinePage onSelectDiff={onselectDiff} />
        {/if}
    </div>
{:else if overlay === 'files'}
    <div class="absolute inset-0">
        <FilesPage />
    </div>
{:else if overlay === 'mailbox'}
    <div class="absolute inset-0">
        <MailboxPage />
    </div>
{:else if overlay === 'container'}
    <div class="absolute inset-0 flex flex-col">
        {#if containerLoading}
            <p class="text-xs text-muted-foreground p-3">Loading...</p>
        {:else if containerRow}
            <ContainerWorkspace
                containerId={containerRow.id}
                containerName={containerRow.name}
                {onclose}
            />
        {:else if sessionWorkerId}
            <ContainerWorkspace
                containerId={sessionWorkerId}
                containerName="session-worker"
                {onclose}
            />
        {:else}
            <div class="flex-1 flex flex-col items-center justify-center gap-2 text-xs text-muted-foreground px-4 text-center">
                <p>No worker container yet — it starts automatically when the agent runs bash or other tools.</p>
                <Button size="sm" variant="outline" onclick={oncreateContainer} disabled={containerLoading}>
                    Create container now
                </Button>
            </div>
        {/if}
    </div>
{:else if overlay === 'todos'}
    <div class="overflow-y-auto p-3 space-y-1">
        {#if todos.length === 0}
            <p class="text-xs text-muted-foreground text-center py-6">No todos yet — the agent tracks its plan here via <span class="font-mono">todowrite</span>.</p>
        {/if}
        {#each todos as t (t.id)}
            <div class="flex items-start gap-2 text-xs px-1 py-1.5 rounded hover:bg-accent/40">
                {#if t.status === 'completed'}
                    <CircleCheck class="size-3.5 text-green-500 mt-0.5 shrink-0" />
                {:else if t.status === 'in_progress'}
                    <CircleDot class="size-3.5 text-yellow-500 mt-0.5 shrink-0" />
                {:else if t.status === 'cancelled'}
                    <X class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                {:else}
                    <Circle class="size-3.5 text-muted-foreground mt-0.5 shrink-0" />
                {/if}
                <span class="flex-1 {t.status === 'completed' || t.status === 'cancelled' ? 'line-through text-muted-foreground' : ''}">{t.content}</span>
            </div>
        {/each}
    </div>
{/if}
