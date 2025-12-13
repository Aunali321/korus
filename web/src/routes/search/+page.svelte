<script lang="ts">
    import { onMount } from "svelte";
    import { Search as SearchIcon } from "lucide-svelte";
    import { api } from "$lib/api";
    import { player } from "$lib/stores/player.svelte";
    import type { SearchResults, Song } from "$lib/types";
    import Card from "$lib/components/Card.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let query = $state("");
    let results = $state<SearchResults | null>(null);
    let loading = $state(false);
    let activeTab = $state<
        "all" | "songs" | "albums" | "artists" | "playlists"
    >("all");
    let debounceTimer: ReturnType<typeof setTimeout>;

    function handleInput() {
        clearTimeout(debounceTimer);
        if (!query.trim()) {
            results = null;
            return;
        }
        debounceTimer = setTimeout(search, 300);
    }

    async function search() {
        if (!query.trim()) return;
        loading = true;
        try {
            results = await api.search(query);
        } catch (e) {
            console.error("Search failed:", e);
        } finally {
            loading = false;
        }
    }

    function playSong(song: Song, songs: Song[]) {
        player.playQueue(
            songs,
            songs.findIndex((s) => s.id === song.id),
        );
    }
</script>

<div class="p-6 space-y-6">
    <div>
        <h2 class="text-3xl font-bold mb-4">Search</h2>
        <div class="relative max-w-xl">
            <SearchIcon
                size={20}
                class="absolute left-4 top-1/2 -translate-y-1/2 text-zinc-400"
            />
            <input
                type="text"
                bind:value={query}
                oninput={handleInput}
                placeholder="Search songs, albums, artists..."
                class="w-full pl-12 pr-4 py-3 bg-zinc-900 border border-zinc-800 rounded-full focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
        </div>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Searching...</div>
        </div>
    {:else if results}
        <div class="flex gap-2 border-b border-zinc-800 pb-2">
            {#each ["all", "songs", "albums", "artists", "playlists"] as tab}
                <button
                    onclick={() => (activeTab = tab as typeof activeTab)}
                    class="px-4 py-2 rounded-full text-sm transition-colors {activeTab ===
                    tab
                        ? 'bg-emerald-500 text-black'
                        : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800'}"
                >
                    {tab.charAt(0).toUpperCase() + tab.slice(1)}
                </button>
            {/each}
        </div>

        {#if activeTab === "all" || activeTab === "songs"}
            {#if results.songs.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Songs</h3>
                    <div class="space-y-1">
                        {#each results.songs.slice(0, activeTab === "all" ? 5 : undefined) as song, i (song.id)}
                            <TrackRow {song} index={i} songs={results.songs} />
                        {/each}
                    </div>
                </section>
            {/if}
        {/if}

        {#if activeTab === "all" || activeTab === "albums"}
            {#if results.albums.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Albums</h3>
                    <div
                        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                    >
                        {#each results.albums.slice(0, activeTab === "all" ? 6 : undefined) as album (album.id)}
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
                </section>
            {/if}
        {/if}

        {#if activeTab === "all" || activeTab === "artists"}
            {#if results.artists.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Artists</h3>
                    <div
                        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                    >
                        {#each results.artists.slice(0, activeTab === "all" ? 6 : undefined) as artist (artist.id)}
                            <Card
                                title={artist.name}
                                subtitle="Artist"
                                image={artist.image_path}
                                href="/artists/{artist.id}"
                                rounded
                            />
                        {/each}
                    </div>
                </section>
            {/if}
        {/if}

        {#if activeTab === "all" || activeTab === "playlists"}
            {#if results.playlists.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Playlists</h3>
                    <div
                        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                    >
                        {#each results.playlists as playlist (playlist.id)}
                            <Card
                                title={playlist.name}
                                subtitle="{playlist.song_count || 0} songs"
                                href="/playlists/{playlist.id}"
                            />
                        {/each}
                    </div>
                </section>
            {/if}
        {/if}

        {#if results.songs.length === 0 && results.albums.length === 0 && results.artists.length === 0 && results.playlists.length === 0}
            <div class="text-center py-12 text-zinc-500">
                <p>No results found for "{query}"</p>
            </div>
        {/if}
    {:else if query}
        <div class="text-center py-12 text-zinc-500">
            <p>Type to search...</p>
        </div>
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>Search for songs, albums, artists, or playlists</p>
        </div>
    {/if}
</div>
