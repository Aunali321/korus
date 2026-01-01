<script lang="ts">
    import { Play, Heart, MoreHorizontal } from "lucide-svelte";
    import type { Song } from "$lib/types";
    import { player } from "$lib/stores/player.svelte";
    import { favorites } from "$lib/stores/favorites.svelte";
    import { settings } from "$lib/stores/settings.svelte";
    import { contextMenu } from "$lib/stores/contextMenu.svelte";
    import { api } from "$lib/api";

    let {
        song,
        index,
        songs,
        showAlbum = true,
        compact = false,
        playlistId,
    }: {
        song: Song;
        index?: number;
        songs?: Song[];
        showAlbum?: boolean;
        compact?: boolean;
        playlistId?: number;
    } = $props();

    let isFavorited = $state(false);

    $effect(() => {
        favorites.load();
    });

    $effect(() => {
        isFavorited = favorites.isFavorite(song.id);
    });

    function formatDuration(seconds: number): string {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, "0")}`;
    }

    function handlePlay() {
        if (songs && index !== undefined) {
            player.playQueue(songs, index);
        } else if (settings.radioEnabled) {
            player.startRadio(song);
        } else {
            player.play(song);
        }
    }

    async function handleFavorite(e: MouseEvent) {
        e.stopPropagation();
        isFavorited = await favorites.toggle(song.id);
    }

    function handleMenuClick(e: MouseEvent) {
        e.stopPropagation();
        const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
        contextMenu.open(song, rect.left, rect.bottom + 4, playlistId);
    }

    function handleContextMenu(e: MouseEvent) {
        e.preventDefault();
        e.stopPropagation();
        contextMenu.open(song, e.clientX, e.clientY, playlistId);
    }

    const isPlaying = $derived(player.currentSong?.id === song.id);
</script>

<div
    class="flex items-center gap-4 p-3 rounded-lg hover:bg-zinc-900 group cursor-pointer {isPlaying
        ? 'bg-zinc-900'
        : ''}"
    oncontextmenu={handleContextMenu}
    role="button"
    tabindex="-1"
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
                <span class="text-zinc-500"> - {song.album.title}</span>
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

    <button
        onclick={handleMenuClick}
        class="opacity-0 group-hover:opacity-100 p-2 hover:bg-zinc-800 rounded-full transition-all"
    >
        <MoreHorizontal size={18} />
    </button>

    {#if index === undefined}
        <button
            onclick={handlePlay}
            class="opacity-0 group-hover:opacity-100 w-10 h-10 bg-emerald-500 rounded-full flex items-center justify-center transition-all hover:scale-110"
        >
            <Play size={16} fill="currentColor" class="text-black ml-0.5" />
        </button>
    {/if}
</div>
