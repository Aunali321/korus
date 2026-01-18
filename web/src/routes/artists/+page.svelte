<script lang="ts">
    import { VList } from "virtua/svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { library } from "$lib/stores/library.svelte";
    import Card from "$lib/components/Card.svelte";

    let loading = $state(true);
    let loaded = $state(false);
    let innerWidth = $state(0);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loaded = true;
            library.load().finally(() => loading = false);
        }
    });

    // Match Tailwind breakpoints: grid-cols-2 md:grid-cols-4 lg:grid-cols-6
    const columnCount = $derived(
        innerWidth >= 1024 ? 6 :
        innerWidth >= 768 ? 4 : 2
    );

    const rows = $derived.by(() => {
        const result = [];
        for (let i = 0; i < library.artists.length; i += columnCount) {
            result.push(library.artists.slice(i, i + columnCount));
        }
        return result;
    });
</script>

<svelte:window bind:innerWidth />

<div class="p-6 space-y-6">
    <h2 class="text-3xl font-bold">Artists</h2>

    {#if loading && library.artists.length === 0}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if library.artists.length > 0}
        {#if innerWidth > 0}
            <VList data={rows} style="height: calc(100vh - 220px);" getKey={(_, i) => i}>
                {#snippet children(row, _rowIndex)}
                    <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4 pb-4">
                        {#each row as artist (artist.id)}
                            <Card
                                title={artist.name}
                                subtitle="Artist"
                                image={artist.image_path ? api.getArtistImageUrl(artist.id) : undefined}
                                href="/artists/{artist.id}"
                                rounded
                            />
                        {/each}
                    </div>
                {/snippet}
            </VList>
        {/if}
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No artists found</p>
        </div>
    {/if}
</div>
