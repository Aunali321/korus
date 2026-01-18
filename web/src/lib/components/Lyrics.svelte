<script lang="ts">
    import { X, Mic } from "lucide-svelte";
    import { player } from "$lib/stores/player.svelte";
    import { api } from "$lib/api";

    let { isOpen, onClose }: { isOpen: boolean; onClose: () => void } = $props();

    interface LyricLine {
        time: number;
        text: string;
    }

    let lyrics = $state<string | null>(null);
    let syncedLyrics = $state<LyricLine[]>([]);
    let loading = $state(false);
    let lyricsContainer = $state<HTMLDivElement | null>(null);

    function parseLRC(lrc: string): LyricLine[] {
        const lines: LyricLine[] = [];
        const regex = /\[(\d{2}):(\d{2})(?:\.(\d{2,3}))?\](.*)/;

        for (const line of lrc.split('\n')) {
            const match = line.match(regex);
            if (match) {
                const mins = parseInt(match[1], 10);
                const secs = parseInt(match[2], 10);
                const ms = match[3] ? parseInt(match[3].padEnd(3, '0'), 10) : 0;
                const time = mins * 60 + secs + ms / 1000;
                const text = match[4].trim();
                if (text) {
                    lines.push({ time, text });
                }
            }
        }

        return lines.sort((a, b) => a.time - b.time);
    }

    const currentLineIndex = $derived.by(() => {
        if (syncedLyrics.length === 0) return -1;
        const progress = player.progress;
        
        for (let i = syncedLyrics.length - 1; i >= 0; i--) {
            if (syncedLyrics[i].time <= progress) {
                return i;
            }
        }
        return -1;
    });

    function isLRCFormat(text: string): boolean {
        return /\[\d{2}:\d{2}/.test(text);
    }

    $effect(() => {
        const song = player.currentSong;
        if (!song || !isOpen) {
            lyrics = null;
            syncedLyrics = [];
            return;
        }

        loading = true;
        api.getLyrics(song.id)
            .then((data) => {
                // Check synced field first, then check if lyrics field contains LRC
                const lrcSource = data.synced || (data.lyrics && isLRCFormat(data.lyrics) ? data.lyrics : null);
                
                if (lrcSource) {
                    syncedLyrics = parseLRC(lrcSource);
                    lyrics = null;
                } else if (data.lyrics) {
                    lyrics = data.lyrics;
                    syncedLyrics = [];
                } else {
                    lyrics = null;
                    syncedLyrics = [];
                }
            })
            .catch(() => {
                lyrics = null;
                syncedLyrics = [];
            })
            .finally(() => {
                loading = false;
            });
    });

    // Auto-scroll to current line
    $effect(() => {
        if (currentLineIndex >= 0 && lyricsContainer) {
            const lineElement = lyricsContainer.querySelector(`[data-line="${currentLineIndex}"]`);
            if (lineElement) {
                lineElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
            }
        }
    });

    function handleLineClick(time: number) {
        player.seek(time);
    }
</script>

<div
    class="fixed top-0 right-0 bottom-0 w-full md:w-96 bg-zinc-900 border-l border-zinc-800 transform transition-transform duration-300 z-30 flex flex-col {isOpen
        ? 'translate-x-0'
        : 'translate-x-full'}"
>
    <div class="flex items-center justify-between p-4 border-b border-zinc-800">
        <div class="flex items-center gap-2">
            <Mic size={20} class="text-emerald-400" />
            <h3 class="font-semibold">Lyrics</h3>
        </div>
        <button
            onclick={onClose}
            class="p-2 hover:bg-zinc-800 rounded-full transition-colors"
        >
            <X size={20} />
        </button>
    </div>

    <div
        bind:this={lyricsContainer}
        class="flex-1 overflow-y-auto p-4 pb-28 scrollbar-thin"
    >
        {#if loading}
            <div class="flex items-center justify-center h-32">
                <span class="text-zinc-500">Loading lyrics...</span>
            </div>
        {:else if syncedLyrics.length > 0}
            <div class="space-y-4 py-8">
                {#each syncedLyrics as line, i}
                    <button
                        data-line={i}
                        onclick={() => handleLineClick(line.time)}
                        class="block w-full text-left text-lg transition-all duration-300 {i === currentLineIndex
                            ? 'text-emerald-400 font-semibold scale-105 origin-left'
                            : i < currentLineIndex
                              ? 'text-zinc-600'
                              : 'text-zinc-400 hover:text-zinc-200'}"
                    >
                        {line.text}
                    </button>
                {/each}
            </div>
        {:else if lyrics}
            <div class="text-zinc-300 whitespace-pre-wrap leading-relaxed">
                {lyrics}
            </div>
        {:else if player.currentSong}
            <div class="flex flex-col items-center justify-center h-32 text-zinc-500">
                <Mic size={32} class="mb-2 opacity-50" />
                <span>No lyrics available</span>
            </div>
        {:else}
            <div class="flex items-center justify-center h-32 text-zinc-500">
                <span>No song playing</span>
            </div>
        {/if}
    </div>
</div>
