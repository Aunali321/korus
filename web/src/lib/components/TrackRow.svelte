<script lang="ts">
    import { Play, Heart, MoreHorizontal, ListPlus, ListMinus, ChevronRight, Plus } from "lucide-svelte";
    import type { Song, Playlist } from "$lib/types";
    import { player } from "$lib/stores/player.svelte";
    import { favorites } from "$lib/stores/favorites.svelte";
    import { api } from "$lib/api";

    let {
        song,
        index,
        songs,
        showAlbum = true,
        compact = false,
        playlistId,
        onRemoveFromPlaylist,
    }: {
        song: Song;
        index?: number;
        songs?: Song[];
        showAlbum?: boolean;
        compact?: boolean;
        playlistId?: number;
        onRemoveFromPlaylist?: () => void;
    } = $props();

    let showMenu = $state(false);
    let showPlaylistSubmenu = $state(false);
    let playlists = $state<Playlist[]>([]);
    let isFavorited = $state(false);
    let menuRef = $state<HTMLDivElement | null>(null);

    $effect(() => {
        favorites.load();
    });

    $effect(() => {
        isFavorited = favorites.isFavorite(song.id);
    });

    function handleClickOutside(e: MouseEvent) {
        if (showMenu && menuRef && !menuRef.contains(e.target as Node)) {
            closeMenu();
        }
    }

    function formatDuration(seconds: number): string {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, "0")}`;
    }

    function handlePlay() {
        if (songs && index !== undefined) {
            player.playQueue(songs, index);
        } else {
            player.play(song);
        }
    }

    async function handleFavorite(e: MouseEvent) {
        e.stopPropagation();
        isFavorited = await favorites.toggle(song.id);
    }

    function handleAddToQueue(e: MouseEvent) {
        e.stopPropagation();
        player.addToQueue(song);
        showMenu = false;
    }

    async function handleShowPlaylists(e: MouseEvent) {
        e.stopPropagation();
        if (playlists.length === 0) {
            try {
                playlists = await api.getPlaylists();
            } catch (err) {
                console.error('Failed to load playlists:', err);
            }
        }
        showPlaylistSubmenu = !showPlaylistSubmenu;
    }

    async function handleAddToPlaylist(e: MouseEvent, playlistId: number) {
        e.stopPropagation();
        try {
            await api.addSongToPlaylist(playlistId, song.id);
        } catch (err) {
            console.error('Failed to add to playlist:', err);
        }
        showMenu = false;
        showPlaylistSubmenu = false;
    }

    async function handleRemoveFromPlaylist(e: MouseEvent) {
        e.stopPropagation();
        if (!playlistId) return;
        try {
            await api.removeSongFromPlaylist(playlistId, song.id);
            onRemoveFromPlaylist?.();
        } catch (err) {
            console.error('Failed to remove from playlist:', err);
        }
        showMenu = false;
    }

    function toggleMenu(e: MouseEvent) {
        e.stopPropagation();
        showMenu = !showMenu;
        showPlaylistSubmenu = false;
    }

    function closeMenu() {
        showMenu = false;
        showPlaylistSubmenu = false;
    }

    const isPlaying = $derived(player.currentSong?.id === song.id);
</script>

<svelte:window onclick={handleClickOutside} />

<div
    class="flex items-center gap-4 p-3 rounded-lg hover:bg-zinc-900 group cursor-pointer {isPlaying
        ? 'bg-zinc-900'
        : ''}"
>
    {#if index !== undefined}
        <div class="w-6 text-center group-hover:hidden">
            <span
                class="text-sm text-zinc-500 {isPlaying
                    ? 'text-emerald-400'
                    : ''}">{index + 1}</span
            >
        </div>
        <button
            onclick={handlePlay}
            class="hidden group-hover:flex w-6 h-6 items-center justify-center"
        >
            <Play size={14} class="text-emerald-400" />
        </button>
    {/if}

    {#if !compact}
        <img
            src={api.getArtworkUrl(song.id)}
            alt={song.title}
            class="w-12 h-12 rounded object-cover bg-zinc-800"
        />
    {/if}

    <button onclick={handlePlay} class="flex-1 min-w-0 text-left">
        <h4 class="font-medium truncate {isPlaying ? 'text-emerald-400' : ''}">
            {song.title}
        </h4>
        <p class="text-sm text-zinc-400 truncate">
            {song.artist?.name || "Unknown"}
            {#if showAlbum && song.album}
                <span class="text-zinc-500"> • {song.album.title}</span>
            {/if}
        </p>
    </button>

    <div class="text-sm text-zinc-400">{formatDuration(song.duration)}</div>

    <button
        onclick={handleFavorite}
        class="opacity-0 group-hover:opacity-100 p-2 hover:bg-zinc-800 rounded-full transition-all {isFavorited ? 'opacity-100' : ''}"
    >
        <Heart size={18} class={isFavorited ? 'fill-red-500 text-red-500' : ''} />
    </button>

    {#if index === undefined}
        <button
            onclick={handlePlay}
            class="opacity-0 group-hover:opacity-100 w-10 h-10 bg-emerald-500 rounded-full flex items-center justify-center transition-all hover:scale-110"
        >
            <Play size={16} fill="currentColor" class="text-black ml-0.5" />
        </button>
    {/if}

    <div class="relative" bind:this={menuRef}>
        <button
            onclick={toggleMenu}
            class="opacity-0 group-hover:opacity-100 p-2 hover:bg-zinc-800 rounded-full transition-all"
        >
            <MoreHorizontal size={18} />
        </button>

        {#if showMenu}
            <div
                class="absolute right-0 top-full mt-1 bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl py-1 z-50 min-w-[180px]"
            >
                <button
                    onclick={handleAddToQueue}
                    class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors"
                >
                    <ListPlus size={16} />
                    Add to Queue
                </button>

                <div class="relative">
                    <button
                        onclick={handleShowPlaylists}
                        class="flex items-center justify-between gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors"
                    >
                        <span class="flex items-center gap-3">
                            <Plus size={16} />
                            Add to Playlist
                        </span>
                        <ChevronRight size={14} />
                    </button>

                    {#if showPlaylistSubmenu}
                        <div
                            class="absolute left-full top-0 -ml-1 pl-2 bg-transparent"
                        >
                            <div class="bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl py-1 min-w-[160px] max-h-[200px] overflow-y-auto">
                                {#if playlists.length === 0}
                                    <div class="px-4 py-2 text-sm text-zinc-500">No playlists</div>
                                {:else}
                                    {#each playlists as playlist (playlist.id)}
                                        <button
                                            onclick={(e) => handleAddToPlaylist(e, playlist.id)}
                                            class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors truncate"
                                        >
                                            {playlist.name}
                                        </button>
                                    {/each}
                                {/if}
                            </div>
                        </div>
                    {/if}
                </div>

                {#if playlistId}
                    <button
                        onclick={handleRemoveFromPlaylist}
                        class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors text-red-400"
                    >
                        <ListMinus size={16} />
                        Remove from Playlist
                    </button>
                {/if}
            </div>
        {/if}
    </div>
</div>
