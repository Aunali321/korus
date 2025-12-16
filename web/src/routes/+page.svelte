<script lang="ts">
    import { Play, Shuffle } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import { library } from "$lib/stores/library.svelte";
    import type { Song, Album } from "$lib/types";
    import Card from "$lib/components/Card.svelte";

    let recentPlays = $state<Song[]>([]);
    let recommendations = $state<Song[]>([]);
    let newAdditions = $state<Album[]>([]);
    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadHome();
        }
    });

    async function loadHome() {
        loaded = true;
        try {
            const data = await api.getHome();
            recentPlays = data.recent_plays || [];
            recommendations = data.recommendations || [];
            newAdditions = data.new_additions || [];
        } catch (e) {
            console.error("Failed to load home data:", e);
        } finally {
            loading = false;
        }
    }

    function playTrack(song: Song, songs: Song[]) {
        player.playQueue(
            songs,
            songs.findIndex((s) => s.id === song.id),
        );
    }

    async function shuffleLibrary() {
        await library.load();
        if (library.songs.length === 0) return;
        player.playShuffled(library.songs);
    }
</script>

<div class="p-6 space-y-8">
    <div
        class="relative h-64 rounded-2xl overflow-hidden bg-gradient-to-r from-emerald-600/20 to-cyan-600/20 border border-emerald-500/20"
    >
        <div
            class="absolute inset-0 bg-gradient-to-br from-emerald-900/50 to-cyan-900/50"
        ></div>
        <div class="relative h-full flex items-end p-8">
            <div>
                <p class="text-sm text-emerald-400 font-medium mb-2">
                    Welcome Back
                </p>
                <h2 class="text-5xl font-bold mb-3">Your Music</h2>
                <p class="text-zinc-400 mb-4">{recentPlays.length > 0 ? "Pick up where you left off" : "Discover something new"}</p>
                {#if recentPlays.length > 0}
                    <button
                        onclick={() => player.playQueue(recentPlays, 0)}
                        class="px-6 py-3 bg-emerald-500 hover:bg-emerald-600 text-black font-semibold rounded-full flex items-center gap-2 transition-all hover:scale-105"
                    >
                        <Play size={20} fill="currentColor" />
                        Play Recent
                    </button>
                {:else if !loading}
                    <button
                        onclick={shuffleLibrary}
                        class="px-6 py-3 bg-emerald-500 hover:bg-emerald-600 text-black font-semibold rounded-full flex items-center gap-2 transition-all hover:scale-105"
                    >
                        <Shuffle size={20} />
                        Shuffle Library
                    </button>
                {/if}
            </div>
        </div>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else}
        {#if recentPlays.length > 0}
            <section>
                <h3 class="text-2xl font-bold mb-4">Recently Played</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4"
                >
                    {#each recentPlays.slice(0, 6) as song (song.id)}
                        <Card
                            title={song.title}
                            subtitle={song.artist?.name || "Unknown"}
                            image={api.getArtworkUrl(song.id)}
                            onPlay={() => playTrack(song, recentPlays)}
                        />
                    {/each}
                </div>
            </section>
        {/if}

        {#if recommendations.length > 0}
            <section>
                <h3 class="text-2xl font-bold mb-4">Recommended For You</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4"
                >
                    {#each recommendations.slice(0, 6) as song (song.id)}
                        <Card
                            title={song.title}
                            subtitle={song.artist?.name || "Unknown"}
                            image={api.getArtworkUrl(song.id)}
                            onPlay={() => playTrack(song, recommendations)}
                        />
                    {/each}
                </div>
            </section>
        {/if}

        {#if newAdditions.length > 0}
            <section>
                <h3 class="text-2xl font-bold mb-4">New Additions</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4"
                >
                    {#each newAdditions.slice(0, 5) as album (album.id)}
                        <Card
                            title={album.title}
                            subtitle="{album.artist?.name ||
                                'Unknown'} • {album.year || ''}"
                            image={album.cover_path
                                ? api.getArtworkUrl(album.id, "album")
                                : undefined}
                            href="/albums/{album.id}"
                        />
                    {/each}
                </div>
            </section>
        {/if}

        {#if recentPlays.length === 0 && recommendations.length === 0 && newAdditions.length === 0}
            <div class="text-center py-12 text-zinc-500">
                <p class="text-lg mb-2">No music yet</p>
                <p class="text-sm">
                    Start by scanning your library in the Admin panel
                </p>
            </div>
        {/if}
    {/if}
</div>
