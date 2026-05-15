<script lang="ts">
    import Play from "@lucide/svelte/icons/play";
    import UserPlus from "@lucide/svelte/icons/user-plus";
    import UserCheck from "@lucide/svelte/icons/user-check";
    import { api } from "$lib/api";
    import { player } from "$lib/stores/player.svelte";
    import { favorites } from "$lib/stores/favorites.svelte";
    import Card from "$lib/components/Card.svelte";
    import TrackRow from "$lib/components/TrackRow.svelte";
    import type { PageData } from "./$types";

    let { data }: { data: PageData } = $props();

    let showAllSongs = $state(false);

    const artist = $derived(data.artist);
    const albums = $derived(artist.albums ?? []);
    const topSongs = $derived(artist.songs ?? []);

    $effect(() => {
        // Reset expand state when navigating between artists.
        artist.id;
        showAllSongs = false;
    });
</script>

<div class="p-6 space-y-8">
    <div class="flex gap-6 items-end">
        {#if artist.image_path}
            <img
                src={api.getArtistImageUrl(artist.id)}
                alt={artist.name}
                class="w-56 h-56 rounded-full object-cover bg-zinc-800 shadow-xl"
            />
        {:else}
            <div class="w-56 h-56 rounded-full bg-gradient-to-br from-zinc-700 to-zinc-800 flex items-center justify-center shadow-xl">
                <span class="text-6xl font-bold text-zinc-500">{artist.name.charAt(0)}</span>
            </div>
        {/if}
        <div class="pb-4">
            <p class="text-sm text-zinc-400 uppercase tracking-wider">Artist</p>
            <h1 class="text-5xl font-bold mt-2 mb-4">{artist.name}</h1>
            <div class="flex items-center gap-4">
                <button
                    onclick={() => topSongs.length && player.playQueue(topSongs, 0)}
                    class="px-6 py-3 bg-emerald-500 hover:bg-emerald-600 text-black font-semibold rounded-full flex items-center gap-2 hover:scale-105"
                >
                    <Play size={20} fill="currentColor" />
                    Play
                </button>
                <button
                    onclick={() => favorites.toggleArtist(artist.id)}
                    class="p-3 hover:bg-zinc-800 rounded-full"
                >
                    {#if favorites.isArtistFollowed(artist.id)}
                        <UserCheck size={20} class="text-emerald-500" />
                    {:else}
                        <UserPlus size={20} class="text-zinc-400" />
                    {/if}
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
            <div class="flex items-center justify-between mb-4">
                <h3 class="text-xl font-bold">Popular Tracks</h3>
                {#if topSongs.length > 5}
                    <button
                        onclick={() => (showAllSongs = !showAllSongs)}
                        class="text-sm text-zinc-400 hover:text-white"
                    >
                        {showAllSongs ? "Show less" : "See all"}
                    </button>
                {/if}
            </div>
            <div class="space-y-1">
                {#each (showAllSongs ? topSongs : topSongs.slice(0, 5)) as song, i (song.id)}
                    <TrackRow {song} index={i} songs={topSongs} />
                {/each}
            </div>
        </section>
    {/if}

    {#if albums.length > 0}
        <section>
            <h3 class="text-xl font-bold mb-4">Albums</h3>
            <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                {#each albums as album (album.id)}
                    <Card
                        title={album.title}
                        subtitle={album.year?.toString() || ""}
                        image={album.cover_path
                            ? api.getArtworkUrl(album.id, "album")
                            : undefined}
                        href="/albums/{album.id}"
                    />
                {/each}
            </div>
        </section>
    {/if}
</div>
