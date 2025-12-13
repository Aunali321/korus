<script lang="ts">
    import { page } from "$app/stores";
    import { Play, Heart, Clock } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import type { Album, Song } from "$lib/types";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let album = $state<Album | null>(null);
    let songs = $state<Song[]>([]);
    let loading = $state(true);
    let loadedId = $state<number | null>(null);

    $effect(() => {
        const idParam = $page.params.id;
        if (auth.isAuthenticated && idParam) {
            const id = parseInt(idParam);
            if (id && id !== loadedId) {
                loadAlbum(id);
            }
        }
    });

    async function loadAlbum(id: number) {
        loadedId = id;
        loading = true;
        try {
            const data = await api.getAlbum(id);
            album = data.album;
            songs = data.songs;
        } catch (e) {
            console.error("Failed to load album:", e);
        } finally {
            loading = false;
        }
    }

    function formatDuration(seconds: number): string {
        const mins = Math.floor(seconds / 60);
        return `${mins} min`;
    }

    const totalDuration = $derived(
        songs.reduce((acc, s) => acc + s.duration, 0),
    );
</script>

{#if loading}
    <div class="flex justify-center py-12">
        <div class="text-zinc-500">Loading...</div>
    </div>
{:else if album}
    <div class="p-6">
        <div class="flex gap-6 mb-8">
            <img
                src={api.getArtworkUrl(album.id)}
                alt={album.title}
                class="w-56 h-56 rounded-lg object-cover bg-zinc-800 shadow-xl"
            />
            <div class="flex flex-col justify-end">
                <p class="text-sm text-zinc-400 uppercase tracking-wider">
                    Album
                </p>
                <h1 class="text-5xl font-bold mt-2 mb-4">{album.title}</h1>
                <div class="flex items-center gap-2 text-sm text-zinc-400">
                    <a
                        href="/artists/{album.artist?.id}"
                        class="hover:text-zinc-100 hover:underline"
                    >
                        {album.artist?.name || "Unknown"}
                    </a>
                    <span>•</span>
                    <span>{album.year || "Unknown year"}</span>
                    <span>•</span>
                    <span>{songs.length} songs</span>
                    <span>•</span>
                    <span>{formatDuration(totalDuration)}</span>
                </div>
            </div>
        </div>

        <div class="flex items-center gap-4 mb-6">
            <button
                onclick={() => player.playQueue(songs, 0)}
                class="w-14 h-14 bg-emerald-500 rounded-full flex items-center justify-center hover:scale-110 transition-all shadow-lg"
            >
                <Play size={24} fill="currentColor" class="text-black ml-1" />
            </button>
            <button
                class="p-3 hover:bg-zinc-800 rounded-full transition-colors"
            >
                <Heart size={24} class="text-zinc-400 hover:text-red-400" />
            </button>
        </div>

        <div
            class="border-b border-zinc-800 pb-2 mb-2 flex items-center text-xs text-zinc-500 uppercase tracking-wider px-3"
        >
            <span class="w-10">#</span>
            <span class="flex-1">Title</span>
            <Clock size={14} class="mr-3" />
        </div>

        <div class="space-y-1">
            {#each songs as song, i (song.id)}
                <TrackRow {song} index={i} {songs} showAlbum={false} />
            {/each}
        </div>
    </div>
{:else}
    <div class="text-center py-12 text-zinc-500">
        <p>Album not found</p>
    </div>
{/if}
