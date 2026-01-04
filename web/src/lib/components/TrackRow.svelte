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
    class="flex items-center gap-2 md:gap-4 p-2 md:p-3 rounded-lg hover:bg-zinc-900 group cursor-pointer {isPlaying
        ? 'bg-zinc-900'
        : ''}"
    oncontextmenu={handleContextMenu}
    role="button"
    tabindex="-1"
>
    {#if index !== undefined}
        <div class="w-5 md:w-6 text-center group-hover:hidden shrink-0">
            <span
                class="text-xs md:text-sm text-zinc-500 {isPlaying
                    ? 'text-emerald-400'
                    : ''}">{index + 1}</span
            >
        </div>
        <button
            onclick={handlePlay}
            class="hidden group-hover:flex w-5 md:w-6 h-5 md:h-6 items-center justify-center shrink-0"
        >
            <Play size={14} class="text-emerald-400" />
        </button>
    {/if}

    {#if !compact}
        <img
            src={api.getArtworkUrl(song.id)}
            alt={song.title}
            class="w-10 h-10 md:w-12 md:h-12 rounded object-cover bg-zinc-800 shrink-0"
        />
    {/if}

    <button onclick={handlePlay} class="flex-1 min-w-0 text-left">
        <h4 class="font-medium truncate text-sm md:text-base {isPlaying ? 'text-emerald-400' : ''}">
            {song.title}
        </h4>
        <p class="text-xs md:text-sm text-zinc-400 truncate">
            {#if song.artists && song.artists.length > 0}
                {#each song.artists as artist, i}
                    <a
                        href="/artists/{artist.id}"
                        onclick={(e) => e.stopPropagation()}
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
    </button>

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
            class="hidden md:flex opacity-0 group-hover:opacity-100 w-8 h-8 md:w-10 md:h-10 bg-emerald-500 rounded-full items-center justify-center transition-all hover:scale-110"
        >
            <Play size={14} fill="currentColor" class="text-black ml-0.5 md:w-4 md:h-4" />
        </button>
    {/if}
</div>
