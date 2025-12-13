<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import type { PlayHistory } from "$lib/types";

    let history = $state<PlayHistory[]>([]);
    let loading = $state(true);
    let offset = $state(0);
    let hasMore = $state(true);
    let loaded = $state(false);
    const limit = 50;

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadMore();
        }
    });

    async function loadMore() {
        if (!hasMore && loaded) return;
        loaded = true;
        loading = true;
        try {
            const data = await api.getHistory(limit, offset);
            history = [...history, ...data];
            offset += limit;
            hasMore = data.length === limit;
        } catch (e) {
            console.error("Failed to load history:", e);
        } finally {
            loading = false;
        }
    }

    function formatTime(timestamp: string): string {
        const date = new Date(timestamp);
        const now = new Date();
        const diff = now.getTime() - date.getTime();
        const mins = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (mins < 60) return `${mins}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;
        return date.toLocaleDateString();
    }

    function formatDuration(seconds: number): string {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, "0")}`;
    }
</script>

<div class="p-6 space-y-6">
    <h2 class="text-3xl font-bold">History</h2>

    {#if history.length > 0}
        <div class="space-y-2">
            {#each history as item (item.id)}
                <button
                    onclick={() => player.play(item.song)}
                    class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-zinc-900 transition-colors text-left"
                >
                    <img
                        src={api.getArtworkUrl(item.song.id)}
                        alt={item.song.title}
                        class="w-12 h-12 rounded object-cover bg-zinc-800"
                    />
                    <div class="flex-1 min-w-0">
                        <h4 class="font-medium truncate">{item.song.title}</h4>
                        <p class="text-sm text-zinc-400 truncate">
                            {item.song.artist?.name || "Unknown"}
                        </p>
                    </div>
                    <div class="text-right text-sm text-zinc-500">
                        <div>{formatTime(item.timestamp)}</div>
                        <div class="text-xs">
                            {Math.round(item.completion_rate * 100)}% played
                        </div>
                    </div>
                </button>
            {/each}
        </div>

        {#if hasMore}
            <div class="flex justify-center py-4">
                <button
                    onclick={loadMore}
                    disabled={loading}
                    class="px-6 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-full text-sm transition-colors disabled:opacity-50"
                >
                    {loading ? "Loading..." : "Load More"}
                </button>
            </div>
        {/if}
    {:else if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No listening history yet</p>
        </div>
    {/if}
</div>
