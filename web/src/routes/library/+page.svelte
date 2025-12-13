<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { Song } from "$lib/types";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let songs = $state<Song[]>([]);
    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadLibrary();
        }
    });

    async function loadLibrary() {
        loaded = true;
        loading = true;
        try {
            const data = await api.getLibrary();
            songs = data.songs;
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
        {#if songs.length > 0}
            <p class="text-zinc-400">
                {songs.length} songs
            </p>
        {/if}
    </div>

    {#if songs.length > 0}
        <div class="space-y-1">
            {#each songs as song, i (song.id)}
                <TrackRow {song} index={i} {songs} />
            {/each}
        </div>
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
