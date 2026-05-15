<script lang="ts">
    import Play from "@lucide/svelte/icons/play";
    import Heart from "@lucide/svelte/icons/heart";
    import MoreHorizontal from "@lucide/svelte/icons/more-horizontal";
    import type { Song } from "$lib/types";
    import { player } from "$lib/stores/player.svelte";
    import { favorites } from "$lib/stores/favorites.svelte";
    import { settings } from "$lib/stores/settings.svelte";
    import { contextMenu } from "$lib/stores/contextMenu.svelte";
    import { api } from "$lib/api";
    import { goto } from "$app/navigation";

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
    class="track-row flex items-center gap-2 md:gap-4 p-2 md:p-3 rounded-lg group cursor-pointer {isPlaying
        ? 'is-playing'
        : ''}"
    oncontextmenu={handleContextMenu}
    role="button"
    tabindex="-1"
>
    {#if index !== undefined}
        <div class="w-5 md:w-6 text-center shrink-0 relative h-5 md:h-6 flex items-center justify-center">
            <span class="num text-xs md:text-sm text-zinc-500 {isPlaying ? 'text-emerald-400' : ''}">{index + 1}</span>
            <button
                onclick={handlePlay}
                aria-label="Play"
                class="play-icon absolute inset-0 flex items-center justify-center"
            >
                <Play size={14} class="text-emerald-400" />
            </button>
        </div>
    {/if}

    {#if !compact}
        <img
            src={api.getArtworkUrl(song.id)}
            alt={song.title}
            class="w-10 h-10 md:w-12 md:h-12 rounded object-cover bg-zinc-800 shrink-0"
        />
    {/if}

    <div onclick={handlePlay} class="flex-1 min-w-0 text-left cursor-pointer" role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && handlePlay()}>
        <h4 class="font-medium truncate text-sm md:text-base {isPlaying ? 'text-emerald-400' : ''}">
            {song.title}
        </h4>
        <p class="text-xs md:text-sm text-zinc-400 truncate">
            {#if song.artists && song.artists.length > 0}
                {#each song.artists as artist, i}
                    <a
                        href="/artists/{artist.id}"
                        onclick={(e) => { e.stopPropagation(); e.preventDefault(); goto(`/artists/${artist.id}`); }}
                        class="hover:underline hover:text-zinc-300"
                    >{artist.name}</a>{#if i < song.artists.length - 1}, {/if}
                {/each}
            {:else}
                Unknown Artist
            {/if}
            {#if showAlbum && song.album}
                <span class="hidden md:inline text-zinc-500"> - {song.album.title}</span>
            {/if}
        </p>
    </div>

    <div class="text-xs md:text-sm text-zinc-500 shrink-0">{formatDuration(song.duration)}</div>

    <button
        onclick={handleFavorite}
        class="hidden md:block p-1.5 md:p-2 hover:bg-zinc-800 rounded-full transition-all {isFavorited ? '!block' : 'md:opacity-0 md:group-hover:opacity-100'}"
    >
        <Heart size={16} class="md:w-[18px] md:h-[18px] {isFavorited ? 'fill-red-500 text-red-500' : ''}" />
    </button>

    <button
        onclick={handleMenuClick}
        class="hidden md:block p-1.5 md:p-2 hover:bg-zinc-800 rounded-full transition-all md:opacity-0 md:group-hover:opacity-100"
    >
        <MoreHorizontal size={16} class="md:w-[18px] md:h-[18px]" />
    </button>

    {#if index === undefined}
        <button
            onclick={handlePlay}
            aria-label="Play"
            class="row-fab hidden md:flex w-8 h-8 md:w-10 md:h-10 bg-emerald-500 rounded-full items-center justify-center text-black shadow-[0_8px_20px_-8px_rgba(16,185,129,0.6)]"
        >
            <Play size={14} fill="currentColor" class="ml-0.5 md:w-4 md:h-4" />
        </button>
    {/if}
</div>

<style>
    .track-row {
        transition: background-color var(--dur-base) var(--ease-soft);
    }
    .track-row:hover {
        background-color: rgb(24 24 27);
    }
    .track-row.is-playing {
        background-color: rgba(24, 24, 27, 0.7);
    }
    .track-row .num,
    .track-row .play-icon {
        transition: opacity 140ms var(--ease-soft);
    }
    .track-row .play-icon {
        opacity: 0;
    }
    .track-row:hover .num {
        opacity: 0;
    }
    .track-row:hover .play-icon {
        opacity: 1;
    }
    .row-fab {
        opacity: 0;
        transition: opacity 200ms var(--ease-out-expo);
    }
    .track-row:hover .row-fab {
        opacity: 1;
    }
</style>
