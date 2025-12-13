<script lang="ts">
    import { Plus } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import type { Playlist } from "$lib/types";
    import Card from "$lib/components/Card.svelte";

    let playlists = $state<Playlist[]>([]);
    let loading = $state(true);
    let loaded = $state(false);
    let showCreate = $state(false);
    let newName = $state("");
    let creating = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadPlaylists();
        }
    });

    async function loadPlaylists() {
        loaded = true;
        try {
            playlists = await api.getPlaylists();
        } catch (e) {
            console.error("Failed to load playlists:", e);
        } finally {
            loading = false;
        }
    }

    async function createPlaylist() {
        if (!newName.trim()) return;
        creating = true;
        try {
            const playlist = await api.createPlaylist(newName);
            playlists = [playlist, ...playlists];
            newName = "";
            showCreate = false;
            toast.success("Playlist created");
        } catch (e) {
            toast.error("Failed to create playlist");
        } finally {
            creating = false;
        }
    }
</script>

<div class="p-6 space-y-6">
    <div class="flex items-center justify-between">
        <h2 class="text-3xl font-bold">Playlists</h2>
        <button
            onclick={() => (showCreate = !showCreate)}
            class="p-2 hover:bg-zinc-800 rounded-full transition-colors"
        >
            <Plus size={24} class="text-zinc-400 hover:text-zinc-100" />
        </button>
    </div>

    {#if showCreate}
        <form
            onsubmit={(e) => {
                e.preventDefault();
                createPlaylist();
            }}
            class="flex gap-2 max-w-md"
        >
            <input
                type="text"
                bind:value={newName}
                placeholder="Playlist name"
                class="flex-1 px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
            <button
                type="submit"
                disabled={creating || !newName.trim()}
                class="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-black font-medium rounded-lg"
            >
                {creating ? "Creating..." : "Create"}
            </button>
        </form>
    {/if}

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if playlists.length > 0}
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
            {#each playlists as playlist (playlist.id)}
                <Card
                    title={playlist.name}
                    subtitle="{playlist.song_count || 0} songs"
                    href="/playlists/{playlist.id}"
                />
            {/each}
        </div>
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No playlists yet</p>
            <p class="text-sm mt-1">Create one to get started</p>
        </div>
    {/if}
</div>
