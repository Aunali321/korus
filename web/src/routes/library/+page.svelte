<script lang="ts">
    import { auth } from "$lib/stores/auth.svelte";
    import { library } from "$lib/stores/library.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loaded = true;
            library.load().finally(() => loading = false);
        }
    });
</script>

<div class="p-6 space-y-6">
    <div>
        <h2 class="text-3xl font-bold mb-2">Your Library</h2>
        {#if library.songs.length > 0}
            <p class="text-zinc-400">
                {library.songs.length} songs
            </p>
        {/if}
    </div>

    {#if library.songs.length > 0}
        <div class="space-y-1">
            {#each library.songs as song, i (song.id)}
                <TrackRow {song} index={i} songs={library.songs} />
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
