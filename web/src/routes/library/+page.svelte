<script lang="ts">
    import { VList } from "virtua/svelte";
    import { Shuffle } from "lucide-svelte";
    import { auth } from "$lib/stores/auth.svelte";
    import { library } from "$lib/stores/library.svelte";
    import { player } from "$lib/stores/player.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loaded = true;
            library.load().finally(() => loading = false);
        }
    });

    function shuffleLibrary() {
        if (library.songs.length === 0) return;
        player.playShuffled(library.songs);
    }
</script>

<div class="p-6 space-y-6">
    <div class="flex items-center justify-between">
        <div>
            <h2 class="text-3xl font-bold mb-2">Your Library</h2>
            {#if library.songs.length > 0}
                <p class="text-zinc-400">
                    {library.songs.length} songs
                </p>
            {/if}
        </div>
        {#if library.songs.length > 0}
            <button
                onclick={shuffleLibrary}
                class="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-black font-semibold rounded-full flex items-center gap-2 transition-all hover:scale-105"
            >
                <Shuffle size={18} />
                Shuffle
            </button>
        {/if}
    </div>

    {#if library.songs.length > 0}
        <VList data={library.songs} style="height: calc(100vh - 220px);" getKey={(song) => song.id}>
            {#snippet children(song, index)}
                <TrackRow {song} {index} songs={library.songs} />
            {/snippet}
        </VList>
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
