<script lang="ts">
    import { Play, Shuffle, Radio } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import { library } from "$lib/stores/library.svelte";
    import { settings } from "$lib/stores/settings.svelte";
    import type { Song, Album } from "$lib/types";
    import Card from "$lib/components/Card.svelte";

    let recentPlays = $state<Song[]>([]);
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
            await settings.load();
            const data = await api.getHome();
            recentPlays = data.recent_plays || [];
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

    function startRadio(song: Song) {
        player.startRadio(song);
    }

    async function shuffleLibrary() {
        await library.load();
        if (library.songs.length === 0) return;
        player.playShuffled(library.songs);
    }
</script>

<div class="p-6 space-y-8">
    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else}
        {#if settings.radioEnabled && recentPlays.length > 0}
            <section>
                <h3 class="text-2xl font-bold mb-4">Quick Picks</h3>
                <div
                    class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3"
                >
                    {#each recentPlays.slice(0, 9) as song (song.id)}
                        <button
                            onclick={() => startRadio(song)}
                            class="flex items-center gap-3 p-3 rounded-lg bg-zinc-900/50 hover:bg-zinc-800 transition-all group text-left"
                        >
                            <img
                                src={api.getArtworkUrl(song.id)}
                                alt={song.title}
                                class="w-12 h-12 rounded object-cover bg-zinc-800 shrink-0"
                            />
                            <div class="min-w-0 flex-1">
                                <p class="font-medium truncate text-sm">
                                    {song.title}
                                </p>
                                <p class="text-xs text-zinc-400 truncate">
                                    {song.artists?.map(a => a.name).join(', ') || "Unknown"}
                                </p>
                            </div>
                            <div
                                class="opacity-0 group-hover:opacity-100 transition-opacity"
                            >
                                <Radio size={16} class="text-emerald-400" />
                            </div>
                        </button>
                    {/each}
                </div>
            </section>
        {/if}

        {#if recentPlays.length > 0}
            <section>
                <h3 class="text-2xl font-bold mb-4">Recently Played</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4"
                >
                    {#each recentPlays.slice(0, 6) as song (song.id)}
                        <Card
                            title={song.title}
                            subtitle={song.artists?.map(a => a.name).join(', ') || "Unknown"}
                            image={api.getArtworkUrl(song.id)}
                            onPlay={() => playTrack(song, recentPlays)}
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

        {#if recentPlays.length === 0 && newAdditions.length === 0}
            <div class="text-center py-12 text-zinc-500">
                <p class="text-lg mb-2">No music yet</p>
                <p class="text-sm">
                    Start by scanning your library in the Admin panel
                </p>
            </div>
        {/if}
    {/if}
</div>
