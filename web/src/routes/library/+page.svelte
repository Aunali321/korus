<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { Song, LibraryStats } from "$lib/types";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let songs = $state<Song[]>([]);
    let stats = $state<LibraryStats | null>(null);
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
            const data = await api.getLibrary(limit, offset);
            songs = [...songs, ...data.songs];
            stats = data.stats;
            offset += limit;
            hasMore = data.songs.length === limit;
        } catch (e) {
            console.error("Failed to load library:", e);
        } finally {
            loading = false;
        }
    }
</script>

<div class="p-6 space-y-6">
    <div>
        <h2 class="text-3xl font-bold mb-2">Your Library</h2>
        {#if stats}
            <p class="text-zinc-400">
                {stats.total_songs} songs • {stats.total_albums} albums • {stats.total_artists}
                artists
            </p>
        {/if}
    </div>

    {#if songs.length > 0}
        <div class="space-y-1">
            {#each songs as song, i (song.id)}
                <TrackRow {song} index={i} {songs} />
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
            <p>No songs in library</p>
        </div>
    {/if}
</div>
