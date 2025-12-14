<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { Album } from "$lib/types";
    import Card from "$lib/components/Card.svelte";

    let albums = $state<Album[]>([]);
    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadAlbums();
        }
    });

    async function loadAlbums() {
        loaded = true;
        try {
            const data = await api.getLibrary();
            albums = data.albums || [];
        } catch (e) {
            console.error("Failed to load albums:", e);
        } finally {
            loading = false;
        }
    }
</script>

<div class="p-6 space-y-6">
    <h2 class="text-3xl font-bold">Albums</h2>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if albums.length > 0}
        <div
            class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 xl:grid-cols-6 gap-4"
        >
            {#each albums as album (album.id)}
                <Card
                    title={album.title}
                    subtitle="{album.artist?.name || 'Unknown'} • {album.year ||
                        ''}"
                    image={album.cover_path
                        ? api.getArtworkUrl(album.id, "album")
                        : undefined}
                    href="/albums/{album.id}"
                />
            {/each}
        </div>
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No albums found</p>
        </div>
    {/if}
</div>
