<script lang="ts">
    import { VList } from "virtua/svelte";
    import { X, Play, Loader2 } from "lucide-svelte";
    import { player } from "$lib/stores/player.svelte";
    import { api } from "$lib/api";

    let { isOpen, onClose }: { isOpen: boolean; onClose: () => void } =
        $props();

    function formatTime(seconds: number): string {
        if (!seconds) return "0:00";
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, "0")}`;
    }

    const upNextTracks = $derived(player.queue.slice(player.queueIndex + 1));
</script>

{#if isOpen}
    <button
        class="fixed inset-0 bg-black/50 z-40 backdrop-blur-sm"
        onclick={onClose}
        aria-label="Close queue"
    ></button>
{/if}

<div
    class="fixed right-0 top-0 h-full w-96 bg-zinc-950 border-l border-zinc-800 z-50 transform transition-transform duration-300 {isOpen
        ? 'translate-x-0'
        : 'translate-x-full'}"
>
    <div
        class="h-16 border-b border-zinc-800 flex items-center justify-between px-6"
    >
        <h2 class="text-lg font-semibold">Now Playing</h2>
        <button
            onclick={onClose}
            class="p-2 hover:bg-zinc-800 rounded-lg transition-colors"
        >
            <X size={20} />
        </button>
    </div>

    {#if isOpen}
        {#if player.currentSong}
            <div class="p-6 border-b border-zinc-800">
                <img
                    src={api.getArtworkUrl(player.currentSong.id)}
                    alt={player.currentSong.title}
                    class="w-full aspect-square object-cover rounded-lg mb-4 bg-zinc-800"
                />
                <h3 class="font-bold text-xl mb-1">{player.currentSong.title}</h3>
                <p class="text-zinc-400">
                    {player.currentSong.artist?.name || "Unknown"}
                </p>
            </div>
        {/if}

        <div class="flex-1 flex flex-col">
            <div class="px-6 py-4 border-b border-zinc-800">
                <h3
                    class="text-sm font-semibold text-zinc-400 uppercase tracking-wider"
                >
                    Up Next ({upNextTracks.length})
                </h3>
            </div>
            {#if player.radioLoading}
                <div class="flex items-center justify-center gap-2 py-8 text-zinc-400">
                    <Loader2 size={16} class="animate-spin" />
                    <span class="text-sm">Loading radio...</span>
                </div>
            {:else if upNextTracks.length > 0}
                <VList data={upNextTracks} style="height: calc(100vh - 28rem);" getKey={(track) => track.id}>
                    {#snippet children(track, index)}
                        <button
                            onclick={() => player.playQueue(player.queue, player.queueIndex + 1 + index)}
                            class="w-full flex items-center gap-3 p-3 mx-3 rounded-lg hover:bg-zinc-900 group cursor-pointer text-left"
                        >
                            <div
                                class="text-sm text-zinc-500 w-6 text-center group-hover:hidden"
                            >
                                {index + 1}
                            </div>
                            <div
                                class="hidden group-hover:flex w-6 h-6 items-center justify-center"
                            >
                                <Play size={14} class="text-emerald-400" />
                            </div>
                            <img
                                src={api.getArtworkUrl(track.id)}
                                alt={track.title}
                                class="w-12 h-12 rounded object-cover bg-zinc-800"
                            />
                            <div class="flex-1 min-w-0">
                                <h4 class="text-sm font-medium truncate">
                                    {track.title}
                                </h4>
                                <p class="text-xs text-zinc-400 truncate">
                                    {track.artist?.name || "Unknown"}
                                </p>
                            </div>
                            <div class="text-xs text-zinc-500">
                                {formatTime(track.duration)}
                            </div>
                        </button>
                    {/snippet}
                </VList>
            {:else}
                <p class="text-center text-zinc-500 py-8 text-sm">
                    Queue is empty
                </p>
            {/if}
        </div>
    {/if}
</div>
