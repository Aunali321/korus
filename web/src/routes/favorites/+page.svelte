<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import type { Song, Album, Artist } from "$lib/types";
    import Card from "$lib/components/Card.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let songs = $state<Song[]>([]);
    let albums = $state<Album[]>([]);
    let artists = $state<Artist[]>([]);
    let loading = $state(true);
    let loaded = $state(false);
    let activeTab = $state<"songs" | "albums" | "artists">("songs");

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadFavorites();
        }
    });

    async function loadFavorites() {
        loaded = true;
        try {
            const data = await api.getFavorites();
            songs = data.songs || [];
            albums = data.albums || [];
            artists = data.artists || [];
        } catch (e) {
            console.error("Failed to load favorites:", e);
        } finally {
            loading = false;
        }
    }
</script>

<div class="p-6 space-y-6">
    <h2 class="text-3xl font-bold">Favorites</h2>

    <div class="flex gap-2 border-b border-zinc-800 pb-2">
        {#each [["songs", `Songs (${songs.length})`], ["albums", `Albums (${albums.length})`], ["artists", `Artists (${artists.length})`]] as [tab, label]}
            <button
                onclick={() => (activeTab = tab as typeof activeTab)}
                class="px-4 py-2 rounded-full text-sm transition-colors {activeTab ===
                tab
                    ? 'bg-emerald-500 text-black'
                    : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800'}"
            >
                {label}
            </button>
        {/each}
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if activeTab === "songs"}
        {#if songs.length > 0}
            <div class="space-y-1">
                {#each songs as song, i (song.id)}
                    <TrackRow {song} index={i} {songs} />
                {/each}
            </div>
        {:else}
            <div class="text-center py-12 text-zinc-500">
                <p>No favorite songs yet</p>
            </div>
        {/if}
    {:else if activeTab === "albums"}
        {#if albums.length > 0}
            <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                {#each albums as album (album.id)}
                    <Card
                        title={album.title}
                        subtitle={album.artist?.name || "Unknown"}
                        image={album.cover_path
                            ? api.getArtworkUrl(album.id)
                            : undefined}
                        href="/albums/{album.id}"
                    />
                {/each}
            </div>
        {:else}
            <div class="text-center py-12 text-zinc-500">
                <p>No favorite albums yet</p>
            </div>
        {/if}
    {:else if activeTab === "artists"}
        {#if artists.length > 0}
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
                <p>No followed artists yet</p>
            </div>
        {/if}
    {/if}
</div>
