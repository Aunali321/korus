<script lang="ts">
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { onMount } from "svelte";
    import { Play, Trash2, Edit2, Clock, Upload } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { player } from "$lib/stores/player.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import type { Playlist } from "$lib/types";
    import TrackRow from "$lib/components/TrackRow.svelte";

    let playlist = $state<Playlist | null>(null);
    let loading = $state(true);
    let loadedId = $state<number | null>(null);
    let editing = $state(false);
    let editName = $state("");
    let coverFileInput = $state<HTMLInputElement | null>(null);
    let uploadingCover = $state(false);

    $effect(() => {
        const idParam = $page.params.id;
        if (auth.isAuthenticated && idParam) {
            const id = parseInt(idParam);
            if (id && id !== loadedId) {
                loadPlaylist(id);
            }
        }
    });

    async function loadPlaylist(id: number) {
        loadedId = id;
        loading = true;
        try {
            playlist = await api.getPlaylist(id);
            editName = playlist.name;
        } catch (e) {
            console.error("Failed to load playlist:", e);
        } finally {
            loading = false;
        }
    }

    async function saveEdit() {
        if (!playlist || !editName.trim()) return;
        try {
            await api.updatePlaylist(
                playlist.id,
                editName,
                playlist.description,
                playlist.public,
            );
            playlist.name = editName;
            editing = false;
            toast.success("Playlist updated");
        } catch {
            toast.error("Failed to update playlist");
        }
    }

    async function handleCoverUpload(e: Event) {
        const input = e.target as HTMLInputElement;
        const file = input.files?.[0];
        if (!file || !playlist) return;

        uploadingCover = true;
        try {
            const result = await api.uploadPlaylistCover(playlist.id, file);
            playlist.cover_path = result.cover_path;
            playlist = playlist;
            toast.success("Cover updated");
        } catch {
            toast.error("Failed to upload cover");
        } finally {
            uploadingCover = false;
            input.value = "";
        }
    }

    async function deletePlaylist() {
        if (!playlist || !confirm("Delete this playlist?")) return;
        try {
            await api.deletePlaylist(playlist.id);
            toast.success("Playlist deleted");
            goto("/playlists");
        } catch {
            toast.error("Failed to delete playlist");
        }
    }

    async function removeSong(songId: number) {
        if (!playlist) return;
        playlist.songs = playlist.songs?.filter((s) => s.id !== songId);
        toast.success("Song removed");
    }

    function handlePlaylistUpdate(e: CustomEvent<number>) {
        if (playlist && e.detail === playlist.id) {
            loadPlaylist(playlist.id);
        }
    }

    onMount(() => {
        window.addEventListener('playlist-updated', handlePlaylistUpdate as EventListener);
        return () => {
            window.removeEventListener('playlist-updated', handlePlaylistUpdate as EventListener);
        };
    });

    const songs = $derived(playlist?.songs || []);
    
    function getCoverUrl(): string | null {
        if (!playlist) return null;
        if (playlist.cover_path) {
            return api.getPlaylistCoverUrl(playlist.id);
        }
        if (playlist.first_song_id) {
            return api.getArtworkUrl(playlist.first_song_id);
        }
        if (songs.length > 0) {
            return api.getArtworkUrl(songs[0].id);
        }
        return null;
    }
    
    const coverUrl = $derived(getCoverUrl());
</script>

{#if loading}
    <div class="flex justify-center py-12">
        <div class="text-zinc-500">Loading...</div>
    </div>
{:else if playlist}
    <div class="p-6">
        <div class="flex gap-6 mb-8">
            <div class="relative group">
                {#if coverUrl}
                    <img
                        src={coverUrl}
                        alt={playlist.name}
                        class="w-56 h-56 rounded-lg object-cover shadow-xl bg-zinc-800"
                    />
                {:else}
                    <div
                        class="w-56 h-56 rounded-lg bg-gradient-to-br from-zinc-700 to-zinc-800 flex items-center justify-center shadow-xl"
                    >
                        <span class="text-6xl">🎵</span>
                    </div>
                {/if}
                {#if editing}
                    <button
                        onclick={() => coverFileInput?.click()}
                        disabled={uploadingCover}
                        class="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center rounded-lg"
                    >
                        <div class="text-center">
                            {#if uploadingCover}
                                <div class="text-sm">Uploading...</div>
                            {:else}
                                <Upload size={32} class="mx-auto mb-2" />
                                <span class="text-sm">Upload Cover</span>
                            {/if}
                        </div>
                    </button>
                    <input
                        bind:this={coverFileInput}
                        type="file"
                        accept="image/*"
                        class="hidden"
                        onchange={handleCoverUpload}
                    />
                {/if}
            </div>
            <div class="flex flex-col justify-end">
                <p class="text-sm text-zinc-400 uppercase tracking-wider">
                    Playlist
                </p>
                {#if editing}
                    <form
                        onsubmit={(e) => {
                            e.preventDefault();
                            saveEdit();
                        }}
                        class="flex gap-2 mt-2"
                    >
                        <input
                            type="text"
                            bind:value={editName}
                            class="text-3xl font-bold bg-transparent border-b border-zinc-600 focus:outline-none focus:border-emerald-500"
                        />
                        <button type="submit" class="text-emerald-400 text-sm"
                            >Save</button
                        >
                        <button
                            type="button"
                            onclick={() => (editing = false)}
                            class="text-zinc-400 text-sm">Cancel</button
                        >
                    </form>
                {:else}
                    <h1 class="text-5xl font-bold mt-2 mb-4">
                        {playlist.name}
                    </h1>
                {/if}
                {#if playlist.description}
                    <p class="text-zinc-400 mb-2">{playlist.description}</p>
                {/if}
                <div class="text-sm text-zinc-400">
                    {songs.length} songs
                    {#if playlist.public}
                        <span
                            class="ml-2 px-2 py-0.5 bg-emerald-500/20 text-emerald-400 rounded text-xs"
                            >Public</span
                        >
                    {/if}
                </div>
            </div>
        </div>

        <div class="flex items-center gap-4 mb-6">
            <button
                onclick={() => songs.length && player.playQueue(songs, 0)}
                disabled={songs.length === 0}
                class="w-14 h-14 bg-emerald-500 rounded-full flex items-center justify-center hover:scale-110 transition-all shadow-lg disabled:opacity-50"
            >
                <Play size={24} fill="currentColor" class="text-black ml-1" />
            </button>
            <button
                onclick={() => (editing = true)}
                class="p-3 hover:bg-zinc-800 rounded-full transition-colors"
            >
                <Edit2 size={20} class="text-zinc-400" />
            </button>
            <button
                onclick={deletePlaylist}
                class="p-3 hover:bg-zinc-800 rounded-full transition-colors"
            >
                <Trash2 size={20} class="text-zinc-400 hover:text-red-400" />
            </button>
        </div>

        {#if songs.length > 0}
            <div
                class="border-b border-zinc-800 pb-2 mb-2 flex items-center text-xs text-zinc-500 uppercase tracking-wider px-3"
            >
                <span class="w-10">#</span>
                <span class="flex-1">Title</span>
                <Clock size={14} class="mr-3" />
            </div>
            <div class="space-y-1">
                {#each songs as song, i (song.id)}
                    <TrackRow 
                        {song} 
                        index={i} 
                        {songs} 
                        playlistId={playlist.id}
                    />
                {/each}
            </div>
        {:else}
            <div class="text-center py-12 text-zinc-500">
                <p>This playlist is empty</p>
                <p class="text-sm mt-1">Add songs from the library or search</p>
            </div>
        {/if}
    </div>
{:else}
    <div class="text-center py-12 text-zinc-500">
        <p>Playlist not found</p>
    </div>
{/if}
