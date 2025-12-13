<script lang="ts">
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { Stats } from "$lib/types";
    import Card from "$lib/components/Card.svelte";

    let stats = $state<Stats | null>(null);
    let loading = $state(true);
    let loaded = $state(false);
    let period = $state("all_time");

    const periods = [
        { value: "hour", label: "Hour" },
        { value: "today", label: "Today" },
        { value: "week", label: "Week" },
        { value: "month", label: "Month" },
        { value: "year", label: "Year" },
        { value: "all_time", label: "All Time" },
    ];

    $effect(() => {
        if (auth.isAuthenticated) {
            loadStats();
        }
    });

    async function loadStats() {
        loaded = true;
        loading = true;
        try {
            stats = await api.getStats(period);
        } catch (e) {
            console.error("Failed to load stats:", e);
        } finally {
            loading = false;
        }
    }

    function formatDuration(seconds: number): string {
        const hours = Math.floor(seconds / 3600);
        const mins = Math.floor((seconds % 3600) / 60);
        if (hours > 0) return `${hours}h ${mins}m`;
        return `${mins}m`;
    }
</script>

<div class="p-6 space-y-6">
    <div class="flex items-center justify-between">
        <h2 class="text-3xl font-bold">Stats</h2>
        <div class="flex gap-2">
            {#each periods as p}
                <button
                    onclick={() => {
                        period = p.value;
                        loadStats();
                    }}
                    class="px-3 py-1.5 rounded-full text-sm transition-colors {period ===
                    p.value
                        ? 'bg-emerald-500 text-black'
                        : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800'}"
                >
                    {p.label}
                </button>
            {/each}
        </div>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if stats}
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-emerald-400">
                    {stats.total_plays}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Total Plays</div>
            </div>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-cyan-400">
                    {formatDuration(stats.total_duration)}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Listening Time</div>
            </div>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-purple-400">
                    {stats.unique_songs}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Unique Songs</div>
            </div>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-pink-400">
                    {stats.unique_artists}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Unique Artists</div>
            </div>
        </div>

        {#if stats.top_songs?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Top Songs</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4"
                >
                    {#each stats.top_songs.slice(0, 5) as item, i (item.song.id)}
                        <div class="relative">
                            <div
                                class="absolute top-2 left-2 w-6 h-6 bg-black/70 rounded-full flex items-center justify-center text-xs font-bold"
                            >
                                {i + 1}
                            </div>
                            <Card
                                title={item.song.title}
                                subtitle="{item.play_count} plays"
                                image={api.getArtworkUrl(item.song.id)}
                            />
                        </div>
                    {/each}
                </div>
            </section>
        {/if}

        {#if stats.top_artists?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Top Artists</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4"
                >
                    {#each stats.top_artists.slice(0, 6) as item, i (item.artist.id)}
                        <div class="relative">
                            <div
                                class="absolute top-2 left-2 w-6 h-6 bg-black/70 rounded-full flex items-center justify-center text-xs font-bold z-10"
                            >
                                {i + 1}
                            </div>
                            <Card
                                title={item.artist.name}
                                subtitle="{item.play_count} plays"
                                image={item.artist.image_path}
                                href="/artists/{item.artist.id}"
                                rounded
                            />
                        </div>
                    {/each}
                </div>
            </section>
        {/if}

        {#if stats.top_albums?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Top Albums</h3>
                <div
                    class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4"
                >
                    {#each stats.top_albums.slice(0, 5) as item, i (item.album.id)}
                        <div class="relative">
                            <div
                                class="absolute top-2 left-2 w-6 h-6 bg-black/70 rounded-full flex items-center justify-center text-xs font-bold z-10"
                            >
                                {i + 1}
                            </div>
                            <Card
                                title={item.album.title}
                                subtitle="{item.play_count} plays"
                                image={api.getArtworkUrl(item.album.id)}
                                href="/albums/{item.album.id}"
                            />
                        </div>
                    {/each}
                </div>
            </section>
        {/if}
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No stats available</p>
        </div>
    {/if}
</div>
