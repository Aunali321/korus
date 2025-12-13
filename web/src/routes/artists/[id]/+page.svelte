<script lang="ts">
    import { page } from "$app/stores";
    import { Play, UserPlus } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import type { Artist, Album, Song } from "$lib/types";
    import Card from "$lib/components/Card.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let artist = $state<Artist | null>(null);
    let albums = $state<Album[]>([]);
    let topSongs = $state<Song[]>([]);
    let loading = $state(true);
    let loadedId = $state<number | null>(null);

    $effect(() => {
        const idParam = $page.params.id;
        if (auth.isAuthenticated && idParam) {
            const id = parseInt(idParam);
            if (id && id !== loadedId) {
                loadArtist(id);
            }
        }
    });

    async function loadArtist(id: number) {
        loadedId = id;
        loading = true;
        try {
            const data = await api.getArtist(id);
            artist = data.artist;
            albums = data.albums || [];
            topSongs = data.top_songs || [];
        } catch (e) {
            console.error("Failed to load artist:", e);
        } finally {
            loading = false;
        }
    }
</script>

{#if loading}
    <div class="flex justify-center py-12">
        <div class="text-zinc-500">Loading...</div>
    </div>
{:else if artist}
    <div class="p-6 space-y-8">
        <div class="flex gap-6 items-end">
            {#if artist.image_path}
                <img
                    src={artist.image_path}
                    alt={artist.name}
                    class="w-56 h-56 rounded-full object-cover bg-zinc-800 shadow-xl"
                />
            {:else}
                <div
                    class="w-56 h-56 rounded-full bg-gradient-to-br from-zinc-700 to-zinc-800 flex items-center justify-center shadow-xl"
                >
                    <span class="text-6xl font-bold text-zinc-500"
                        >{artist.name.charAt(0)}</span
                    >
                </div>
            {/if}
            <div class="pb-4">
                <p class="text-sm text-zinc-400 uppercase tracking-wider">
                    Artist
                </p>
                <h1 class="text-5xl font-bold mt-2 mb-4">{artist.name}</h1>
                <div class="flex items-center gap-4">
                    <button
                        onclick={() =>
                            topSongs.length && player.playQueue(topSongs, 0)}
                        class="px-6 py-3 bg-emerald-500 hover:bg-emerald-600 text-black font-semibold rounded-full flex items-center gap-2 transition-all hover:scale-105"
                    >
                        <Play size={20} fill="currentColor" />
                        Play
                    </button>
                    <button
                        class="p-3 hover:bg-zinc-800 rounded-full transition-colors"
                    >
                        <UserPlus size={20} class="text-zinc-400" />
                    </button>
                </div>
            </div>
        </div>

        {#if artist.bio}
            <section>
                <h3 class="text-xl font-bold mb-3">About</h3>
                <p class="text-zinc-400 max-w-3xl">{artist.bio}</p>
            </section>
        {/if}

        {#if topSongs.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Popular Tracks</h3>
                <div class="space-y-1">
                    {#each topSongs.slice(0, 5) as song, i (song.id)}
                        <TrackRow {song} index={i} songs={topSongs} />
                    {/each}
                </div>
            </section>
        {/if}

        {#if albums.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Albums</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4"
                >
                    {#each albums as album (album.id)}
                        <Card
                            title={album.title}
                            subtitle={album.year?.toString() || ""}
                            image={album.cover_path
                                ? api.getArtworkUrl(album.id)
                                : undefined}
                            href="/albums/{album.id}"
                        />
                    {/each}
                </div>
            </section>
        {/if}
    </div>
{:else}
    <div class="text-center py-12 text-zinc-500">
        <p>Artist not found</p>
    </div>
{/if}
