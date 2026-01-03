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
        Mic,
    } from "lucide-svelte";
    import { player } from "$lib/stores/player.svelte";
    import { favorites } from "$lib/stores/favorites.svelte";
    import { api } from "$lib/api";

    let { onToggleQueue, onToggleLyrics }: { onToggleQueue: () => void; onToggleLyrics: () => void } = $props();

    let isFavorited = $state(false);
    let isSeeking = $state(false);
    let seekValue = $state(0);

    $effect(() => {
        favorites.load();
    });

    $effect(() => {
        if (player.currentSong) {
            isFavorited = favorites.isFavorite(player.currentSong.id);
        }
    });

    async function handleFavorite() {
        if (!player.currentSong) return;
        isFavorited = await favorites.toggle(player.currentSong.id);
    }

    function formatTime(seconds: number): string {
        if (!seconds || !isFinite(seconds)) return "0:00";
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, "0")}`;
    }

    function handleSeekStart() {
        isSeeking = true;
        seekValue = player.progress;
    }

    function handleSeekInput(e: Event) {
        const target = e.target as HTMLInputElement;
        seekValue = parseFloat(target.value);
    }

    function handleSeekEnd(e: Event) {
        const target = e.target as HTMLInputElement;
        player.seek(parseFloat(target.value));
        isSeeking = false;
    }

    function handleVolume(e: Event) {
        const target = e.target as HTMLInputElement;
        player.setVolume(parseFloat(target.value));
    }

    const displayProgress = $derived(isSeeking ? seekValue : player.progress);
    const progressPercent = $derived(player.duration ? (displayProgress / player.duration) * 100 : 0);
    const volumePercent = $derived(player.volume * 100);
</script>

<div
    class="h-24 bg-zinc-950 border-t border-zinc-800 flex items-center px-4 gap-4 relative z-40"
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
                    {#if player.currentSong.artists && player.currentSong.artists.length > 0}
                        {#each player.currentSong.artists as artist, i}
                            <a
                                href="/artists/{artist.id}"
                                class="hover:text-zinc-100 hover:underline"
                            >{artist.name}</a>{#if i < player.currentSong.artists.length - 1}, {/if}
                        {/each}
                    {:else}
                        Unknown
                    {/if}
                </p>
            </div>
            <button
                onclick={handleFavorite}
                class="p-2 hover:bg-zinc-800 rounded-full transition-colors"
            >
                <Heart size={18} class={isFavorited ? 'fill-red-500 text-red-500' : 'text-zinc-400 hover:text-red-400'} />
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
                    >{formatTime(displayProgress)}</span
                >
                <input
                    type="range"
                    min="0"
                    max={player.duration || 100}
                    value={displayProgress}
                    onmousedown={handleSeekStart}
                    ontouchstart={handleSeekStart}
                    oninput={handleSeekInput}
                    onchange={handleSeekEnd}
                    class="flex-1 range-progress"
                    style="--progress: {progressPercent}%"
                />
                <span class="text-xs text-zinc-400 w-10"
                    >{formatTime(player.duration)}</span
                >
            </div>
        </div>

        <div class="flex items-center gap-3 w-80 justify-end">
            <button
                onclick={onToggleLyrics}
                class="p-2 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 rounded-lg transition-all"
            >
                <Mic size={20} />
            </button>
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
                    class="w-24 range-progress"
                    style="--progress: {volumePercent}%"
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
