<script lang="ts">
    import { Search as SearchIcon } from "lucide-svelte";
    import { api } from "$lib/api";
    import { player } from "$lib/stores/player.svelte";
    import { settings } from "$lib/stores/settings.svelte";
    import { search } from "$lib/stores/search.svelte";
    import type { Song } from "$lib/types";
    import Card from "$lib/components/Card.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let loading = $state(false);
    let debounceTimer: ReturnType<typeof setTimeout>;

    function handleInput(e: Event) {
        const target = e.target as HTMLInputElement;
        search.setQuery(target.value);
        clearTimeout(debounceTimer);
        if (!search.query.trim()) {
            search.setResults(null);
            return;
        }
        debounceTimer = setTimeout(doSearch, 300);
    }

    async function doSearch() {
        if (!search.query.trim()) return;
        loading = true;
        try {
            const results = await api.search(search.query);
            search.setResults(results);
        } catch (e) {
            console.error("Search failed:", e);
        } finally {
            loading = false;
        }
    }

    function playSong(song: Song) {
        if (settings.radioEnabled) {
            player.startRadio(song);
        } else {
            player.play(song);
        }
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
                value={search.query}
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
    {:else if search.results}
        <div class="flex gap-2 border-b border-zinc-800 pb-2">
            {#each ["all", "songs", "albums", "artists", "playlists"] as tab}
                <button
                    onclick={() => search.setActiveTab(tab as "all" | "songs" | "albums" | "artists" | "playlists")}
                    class="px-4 py-2 rounded-full text-sm transition-colors {search.activeTab ===
                    tab
                        ? 'bg-emerald-500 text-black'
                        : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800'}"
                >
                    {tab.charAt(0).toUpperCase() + tab.slice(1)}
                </button>
            {/each}
        </div>

        {#if search.activeTab === "all" || search.activeTab === "songs"}
            {#if search.results.songs.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Songs</h3>
                    <div class="space-y-1">
                        {#each search.results.songs.slice(0, search.activeTab === "all" ? 5 : undefined) as song (song.id)}
                            <TrackRow {song} />
                        {/each}
                    </div>
                </section>
            {/if}
        {/if}

        {#if search.activeTab === "all" || search.activeTab === "albums"}
            {#if search.results.albums.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Albums</h3>
                    <div
                        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                    >
                        {#each search.results.albums.slice(0, search.activeTab === "all" ? 6 : undefined) as album (album.id)}
                            <Card
                                title={album.title}
                                subtitle={album.artist?.name || "Unknown"}
                                image={album.cover_path
                                    ? api.getArtworkUrl(album.id, "album")
                                    : undefined}
                                href="/albums/{album.id}"
                            />
                        {/each}
                    </div>
                </section>
            {/if}
        {/if}

        {#if search.activeTab === "all" || search.activeTab === "artists"}
            {#if search.results.artists.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Artists</h3>
                    <div
                        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                    >
                        {#each search.results.artists.slice(0, search.activeTab === "all" ? 6 : undefined) as artist (artist.id)}
                            <Card
                                title={artist.name}
                                subtitle="Artist"
                                image={artist.image_path ? api.getArtistImageUrl(artist.id) : undefined}
                                href="/artists/{artist.id}"
                                rounded
                            />
                        {/each}
                    </div>
                </section>
            {/if}
        {/if}

        {#if search.activeTab === "all" || search.activeTab === "playlists"}
            {#if search.results.playlists.length > 0}
                <section>
                    <h3 class="text-xl font-bold mb-4">Playlists</h3>
                    <div
                        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                    >
                        {#each search.results.playlists as playlist (playlist.id)}
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

        {#if search.results.songs.length === 0 && search.results.albums.length === 0 && search.results.artists.length === 0 && search.results.playlists.length === 0}
            <div class="text-center py-12 text-zinc-500">
                <p>No results found for "{search.query}"</p>
            </div>
        {/if}
    {:else if search.query}
        <div class="text-center py-12 text-zinc-500">
            <p>Type to search...</p>
        </div>
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>Search for songs, albums, artists, or playlists</p>
        </div>
    {/if}
</div>
