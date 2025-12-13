<script lang="ts">
    import {
        Play,
        Pause,
        SkipBack,
        SkipForward,
        Shuffle,
        Repeat,
        Repeat1,
        Volume2,
        VolumeX,
        Heart,
        ListMusic,
    } from "lucide-svelte";
    import { player } from "$lib/stores/player.svelte";
    import { api } from "$lib/api";

    let { onToggleQueue }: { onToggleQueue: () => void } = $props();

    function formatTime(seconds: number): string {
        if (!seconds || !isFinite(seconds)) return "0:00";
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, "0")}`;
    }

    function handleSeek(e: Event) {
        const target = e.target as HTMLInputElement;
        player.seek(parseFloat(target.value));
    }

    function handleVolume(e: Event) {
        const target = e.target as HTMLInputElement;
        player.setVolume(parseFloat(target.value));
    }
</script>

<div
    class="h-24 bg-zinc-950 border-t border-zinc-800 flex items-center px-4 gap-4"
>
    {#if player.currentSong}
        <div class="flex items-center gap-3 w-80">
            <img
                src={api.getArtworkUrl(player.currentSong.id)}
                alt={player.currentSong.title}
                class="w-14 h-14 rounded-lg object-cover bg-zinc-800"
            />
            <div class="flex-1 min-w-0">
                <h4 class="font-semibold text-sm truncate">
                    {player.currentSong.title}
                </h4>
                <p class="text-xs text-zinc-400 truncate">
                    {player.currentSong.artist?.name || "Unknown"}
                </p>
            </div>
            <button
                class="p-2 hover:bg-zinc-800 rounded-full transition-colors"
            >
                <Heart size={18} class="text-zinc-400 hover:text-red-400" />
            </button>
        </div>

        <div class="flex-1 flex flex-col items-center gap-2">
            <div class="flex items-center gap-4">
                <button
                    onclick={() => player.toggleShuffle()}
                    class="p-2 transition-colors {player.shuffle
                        ? 'text-emerald-400'
                        : 'text-zinc-400 hover:text-zinc-100'}"
                >
                    <Shuffle size={18} />
                </button>
                <button
                    onclick={() => player.prev()}
                    class="p-2 text-zinc-400 hover:text-zinc-100 transition-colors"
                >
                    <SkipBack size={20} />
                </button>
                <button
                    onclick={() => player.toggle()}
                    class="w-10 h-10 bg-white hover:bg-zinc-200 text-black rounded-full flex items-center justify-center transition-all hover:scale-105"
                >
                    {#if player.isPlaying}
                        <Pause size={20} fill="currentColor" />
                    {:else}
                        <Play size={20} fill="currentColor" class="ml-0.5" />
                    {/if}
                </button>
                <button
                    onclick={() => player.next()}
                    class="p-2 text-zinc-400 hover:text-zinc-100 transition-colors"
                >
                    <SkipForward size={20} />
                </button>
                <button
                    onclick={() => player.toggleRepeat()}
                    class="p-2 transition-colors {player.repeat !== 'off'
                        ? 'text-emerald-400'
                        : 'text-zinc-400 hover:text-zinc-100'}"
                >
                    {#if player.repeat === "one"}
                        <Repeat1 size={18} />
                    {:else}
                        <Repeat size={18} />
                    {/if}
                </button>
            </div>

            <div class="flex items-center gap-2 w-full max-w-2xl">
                <span class="text-xs text-zinc-400 w-10 text-right"
                    >{formatTime(player.progress)}</span
                >
                <input
                    type="range"
                    min="0"
                    max={player.duration || 100}
                    value={player.progress}
                    oninput={handleSeek}
                    class="flex-1"
                />
                <span class="text-xs text-zinc-400 w-10"
                    >{formatTime(player.duration)}</span
                >
            </div>
        </div>

        <div class="flex items-center gap-3 w-80 justify-end">
            <button
                onclick={onToggleQueue}
                class="p-2 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 rounded-lg transition-all"
            >
                <ListMusic size={20} />
            </button>
            <div class="flex items-center gap-2">
                <button
                    onclick={() =>
                        player.setVolume(player.volume > 0 ? 0 : 0.7)}
                >
                    {#if player.volume === 0}
                        <VolumeX size={20} class="text-zinc-400" />
                    {:else}
                        <Volume2 size={20} class="text-zinc-400" />
                    {/if}
                </button>
                <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.01"
                    value={player.volume}
                    oninput={handleVolume}
                    class="w-24"
                />
            </div>
        </div>
    {:else}
        <div
            class="flex-1 flex items-center justify-center text-zinc-500 text-sm"
        >
            Select a song to play
        </div>
    {/if}
</div>
