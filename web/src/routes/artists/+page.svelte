<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { Artist } from "$lib/types";
    import Card from "$lib/components/Card.svelte";

    let artists = $state<Artist[]>([]);
    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadArtists();
        }
    });

    async function loadArtists() {
        loaded = true;
        try {
            const data = await api.getLibrary();
            artists = data.artists || [];
        } catch (e) {
            console.error("Failed to load artists:", e);
        } finally {
            loading = false;
        }
    }
</script>

<div class="p-6 space-y-6">
    <h2 class="text-3xl font-bold">Artists</h2>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if artists.length > 0}
        <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
            {#each artists as artist (artist.id)}
                <Card
                    title={artist.name}
                    subtitle="Artist"
                    image={artist.image_path}
                    href="/artists/{artist.id}"
                    rounded
                />
            {/each}
        </div>
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No artists found</p>
        </div>
    {/if}
</div>
