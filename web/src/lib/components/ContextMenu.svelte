<script lang="ts">
    import { ListPlus, Radio, Plus, Minus, ChevronRight } from "lucide-svelte";
    import type { Playlist } from "$lib/types";
    import { contextMenu } from "$lib/stores/contextMenu.svelte";
    import { settings } from "$lib/stores/settings.svelte";
    import { player } from "$lib/stores/player.svelte";
    import { api } from "$lib/api";

    let showPlaylistSubmenu = $state(false);
    let playlists = $state<Playlist[]>([]);
    let loadingPlaylists = $state(false);

    const submenuPosition = $derived(
        contextMenu.position.x > window.innerWidth - 400 ? 'left' : 'right'
    );

    async function loadPlaylists() {
        if (playlists.length > 0 || loadingPlaylists) return;
        loadingPlaylists = true;
        try {
            playlists = await api.getPlaylists();
        } catch (err) {
            console.error('Failed to load playlists:', err);
        } finally {
            loadingPlaylists = false;
        }
    }

    async function handleSelect(id: string) {
        const song = contextMenu.song;
        const playlistId = contextMenu.playlistId;
        contextMenu.close();
        showPlaylistSubmenu = false;

        if (!song) return;

        switch (id) {
            case "add-to-queue":
                player.addToQueue(song);
                break;
            case "start-radio":
                player.startRadio(song);
                break;
            case "remove-from-playlist":
                if (playlistId) {
                    try {
                        await api.removeSongFromPlaylist(playlistId, song.id);
                        window.dispatchEvent(new CustomEvent('playlist-updated', { detail: playlistId }));
                    } catch (err) {
                        console.error('Failed to remove from playlist:', err);
                    }
                }
                break;
            default:
                if (id.startsWith("playlist:")) {
                    const plId = parseInt(id.split(":")[1]);
                    try {
                        await api.addSongToPlaylist(plId, song.id);
                    } catch (err) {
                        console.error('Failed to add to playlist:', err);
                    }
                }
        }
    }

    function handlePlaylistHover() {
        showPlaylistSubmenu = true;
        loadPlaylists();
    }

    function handleMenuClick(e: MouseEvent) {
        e.stopPropagation();
    }

    function handleWindowClick() {
        if (contextMenu.isOpen) {
            contextMenu.close();
            showPlaylistSubmenu = false;
        }
    }

    $effect(() => {
        if (!contextMenu.isOpen) {
            showPlaylistSubmenu = false;
        }
    });
</script>

<svelte:window onclick={handleWindowClick} />

{#if contextMenu.isOpen && contextMenu.song}
    <div
        class="fixed bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl py-1 min-w-[200px]"
        style="left: {contextMenu.position.x}px; top: {contextMenu.position.y}px; z-index: 99999;"
        onclick={handleMenuClick}
        oncontextmenu={(e) => { e.preventDefault(); e.stopPropagation(); }}
        role="menu"
    >
        <button
            onclick={() => handleSelect("add-to-queue")}
            class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors"
            role="menuitem"
        >
            <ListPlus size={16} />
            Add to Queue
        </button>

        {#if settings.radioEnabled}
            <button
                onclick={() => handleSelect("start-radio")}
                class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors"
                role="menuitem"
            >
                <Radio size={16} />
                Start Radio
            </button>
        {/if}

        <div
            class="relative"
            onmouseenter={handlePlaylistHover}
            onmouseleave={() => showPlaylistSubmenu = false}
            role="none"
        >
            <button
                class="flex items-center justify-between gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors"
                role="menuitem"
            >
                <span class="flex items-center gap-3">
                    <Plus size={16} />
                    Add to Playlist
                </span>
                <ChevronRight size={14} class="text-zinc-400" />
            </button>

            {#if showPlaylistSubmenu}
                <div
                    class="absolute top-0 bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl py-1 min-w-[180px] max-h-64 overflow-y-auto"
                    class:left-full={submenuPosition === 'right'}
                    class:right-full={submenuPosition === 'left'}
                    style="margin-left: {submenuPosition === 'right' ? '4px' : '0'}; margin-right: {submenuPosition === 'left' ? '4px' : '0'};"
                    role="menu"
                >
                    {#if loadingPlaylists}
                        <div class="px-4 py-2 text-sm text-zinc-400">Loading...</div>
                    {:else if playlists.length === 0}
                        <div class="px-4 py-2 text-sm text-zinc-400">No playlists</div>
                    {:else}
                        {#each playlists as playlist (playlist.id)}
                            <button
                                onclick={() => handleSelect(`playlist:${playlist.id}`)}
                                class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors truncate"
                                role="menuitem"
                            >
                                {playlist.name}
                            </button>
                        {/each}
                    {/if}
                </div>
            {/if}
        </div>

        {#if contextMenu.playlistId}
            <div class="border-t border-zinc-700 my-1"></div>
            <button
                onclick={() => handleSelect("remove-from-playlist")}
                class="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-zinc-700 transition-colors text-red-400"
                role="menuitem"
            >
                <Minus size={16} />
                Remove from Playlist
            </button>
        {/if}
    </div>
{/if}
