<script lang="ts">
    import { goto } from "$app/navigation";
    import BarChart3 from "@lucide/svelte/icons/bar-chart-3";
    import Clock from "@lucide/svelte/icons/clock";
    import Flame from "@lucide/svelte/icons/flame";
    import { api } from "$lib/api";
    import Card from "$lib/components/Card.svelte";
    import type { PageData } from "./$types";

    let { data }: { data: PageData } = $props();

    let activeTab = $state<"overview" | "history" | "streaks">("overview");

    const periods = [
        { value: "hour", label: "Hour" },
        { value: "today", label: "Today" },
        { value: "week", label: "Week" },
        { value: "month", label: "Month" },
        { value: "year", label: "Year" },
        { value: "all_time", label: "All Time" },
    ];

    const period = $derived(data.period);
    const stats = $derived(data.bundle.stats);
    const history = $derived(data.bundle.history);
    const insights = $derived(data.bundle.insights);

    function selectPeriod(value: string) {
        goto(`?period=${value}`, { replaceState: true, noScroll: true, keepFocus: true });
    }

    function formatDuration(seconds: number): string {
        if (!seconds || isNaN(seconds)) return "0m";
        const hours = Math.floor(seconds / 3600);
        const mins = Math.floor((seconds % 3600) / 60);
        if (hours > 0) return `${hours}h ${mins}m`;
        return `${mins}m`;
    }

    function formatPlayedAt(timestamp: string): string {
        const date = new Date(timestamp);
        const now = new Date();
        const diff = now.getTime() - date.getTime();
        const mins = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (mins < 60) return `${mins}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;
        return date.toLocaleDateString();
    }
</script>

<div class="p-4 md:p-6 space-y-6">
    <div class="space-y-4">
        <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div class="flex flex-col md:flex-row md:items-center gap-4 md:gap-6">
                <h2 class="text-2xl md:text-3xl font-bold">Stats</h2>
                <div class="flex gap-1 bg-zinc-900 rounded-lg p-1 overflow-x-auto">
                    <button
                        onclick={() => (activeTab = "overview")}
                        class="flex items-center gap-2 px-3 md:px-4 py-2 rounded-md text-sm whitespace-nowrap {activeTab === 'overview'
                            ? 'bg-zinc-800 text-white'
                            : 'text-zinc-400 hover:text-white'}"
                    >
                        <BarChart3 size={16} />
                        Overview
                    </button>
                    <button
                        onclick={() => (activeTab = "history")}
                        class="flex items-center gap-2 px-3 md:px-4 py-2 rounded-md text-sm whitespace-nowrap {activeTab === 'history'
                            ? 'bg-zinc-800 text-white'
                            : 'text-zinc-400 hover:text-white'}"
                    >
                        <Clock size={16} />
                        History
                    </button>
                    <button
                        onclick={() => (activeTab = "streaks")}
                        class="flex items-center gap-2 px-3 md:px-4 py-2 rounded-md text-sm whitespace-nowrap {activeTab === 'streaks'
                            ? 'bg-zinc-800 text-white'
                            : 'text-zinc-400 hover:text-white'}"
                    >
                        <Flame size={16} />
                        Streaks
                    </button>
                </div>
            </div>

            {#if activeTab === "overview"}
                <div class="flex flex-wrap gap-2">
                    {#each periods as p}
                        <button
                            onclick={() => selectPeriod(p.value)}
                            class="px-3 py-1.5 rounded-full text-sm {period === p.value
                                ? 'bg-emerald-500 text-black'
                                : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800'}"
                        >
                            {p.label}
                        </button>
                    {/each}
                </div>
            {/if}
        </div>
    </div>

    {#if activeTab === "overview"}
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-emerald-400">
                    {stats.total_plays || 0}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Total Plays</div>
            </div>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-cyan-400">
                    {formatDuration(stats.total_duration || 0)}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Listening Time</div>
            </div>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-purple-400">
                    {stats.unique_songs || 0}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Unique Songs</div>
            </div>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="text-3xl font-bold text-pink-400">
                    {stats.unique_artists || 0}
                </div>
                <div class="text-sm text-zinc-400 mt-1">Unique Artists</div>
            </div>
        </div>

        {#if stats.top_songs?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Top Songs</h3>
                <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                    {#each stats.top_songs.slice(0, 5) as item, i (item.song.id)}
                        <div class="relative">
                            <div class="absolute top-2 left-2 w-6 h-6 bg-black/70 rounded-full flex items-center justify-center text-xs font-bold z-10">
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
                <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
                    {#each stats.top_artists.slice(0, 6) as item, i (item.artist.id)}
                        <div class="relative">
                            <div class="absolute top-2 left-2 w-6 h-6 bg-black/70 rounded-full flex items-center justify-center text-xs font-bold z-10">
                                {i + 1}
                            </div>
                            <Card
                                title={item.artist.name}
                                subtitle="{item.play_count} plays"
                                image={item.artist.image_path ? api.getArtistImageUrl(item.artist.id) : undefined}
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
                <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                    {#each stats.top_albums.slice(0, 5) as item, i (item.album.id)}
                        <div class="relative">
                            <div class="absolute top-2 left-2 w-6 h-6 bg-black/70 rounded-full flex items-center justify-center text-xs font-bold z-10">
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

        {#if !stats.top_songs?.length && !stats.top_artists?.length && !stats.top_albums?.length}
            <div class="text-center py-12 text-zinc-500">
                <p>No listening data yet</p>
                <p class="text-sm mt-1">
                    Start playing music to see your stats!
                </p>
            </div>
        {/if}
    {:else if activeTab === "history"}
        {#if history.length > 0}
            <div class="space-y-2">
                {#each history as item (item.id)}
                    <div class="flex items-center gap-4 bg-zinc-900 border border-zinc-800 rounded-lg p-4 hover:bg-zinc-800/50">
                        <div class="flex-1">
                            <div class="font-medium">
                                {item.song?.title || "Unknown"}
                            </div>
                            <div class="text-sm text-zinc-400">
                                {formatPlayedAt(item.played_at)}
                            </div>
                        </div>
                        {#if item.source}
                            <div class="text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                                {item.source}
                            </div>
                        {/if}
                        <div class="text-sm text-zinc-500">
                            {Math.round((item.completion_rate || 0) * 100)}% played
                        </div>
                    </div>
                {/each}
            </div>
        {:else}
            <div class="text-center py-12 text-zinc-500">
                <p>No listening history yet</p>
                <p class="text-sm mt-1">
                    Start playing music to see your history!
                </p>
            </div>
        {/if}
    {:else if activeTab === "streaks"}
        <div class="grid md:grid-cols-2 gap-6">
            <div class="bg-gradient-to-br from-orange-900/50 to-red-900/50 border border-orange-500/30 rounded-2xl p-8">
                <div class="flex items-center gap-3 mb-4">
                    <Flame class="text-orange-400" size={28} />
                    <span class="text-zinc-400">Current Streak</span>
                </div>
                <div class="text-5xl font-bold text-orange-400">
                    {insights.current_streak || 0}
                </div>
                <p class="text-zinc-400 mt-2">days in a row</p>
            </div>

            <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
                <div class="flex items-center gap-3 mb-4">
                    <Flame class="text-red-400" size={28} />
                    <span class="text-zinc-400">Longest Streak</span>
                </div>
                <div class="text-5xl font-bold text-red-400">
                    {insights.longest_streak || 0}
                </div>
                <p class="text-zinc-400 mt-2">days</p>
            </div>
        </div>
    {/if}
</div>
