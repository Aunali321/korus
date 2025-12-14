<script lang="ts">
    import { Sparkles, Music, User, Disc, Lock } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { WrappedData } from "$lib/types";

    let wrapped = $state<WrappedData | null>(null);
    let loading = $state(true);
    let loaded = $state(false);
    let period = $state("year");

    // Wrapped is available only during last week of month or in December
    function isWrappedSeason(): boolean {
        const now = new Date();
        const month = now.getMonth(); // 0-11
        const date = now.getDate();
        const lastDay = new Date(now.getFullYear(), month + 1, 0).getDate();

        return month === 11 || lastDay - date < 7;
    }

    $effect(() => {
        if (auth.isAuthenticated && isWrappedSeason()) {
            loadWrapped();
        } else {
            loading = false;
        }
    });

    async function loadWrapped() {
        loaded = true;
        loading = true;
        try {
            wrapped = await api.getWrapped(period);
        } catch (e) {
            console.error("Failed to load wrapped:", e);
        } finally {
            loading = false;
        }
    }

    function formatMinutes(mins: number): string {
        if (!mins || isNaN(mins)) return "0 minutes";
        const hours = Math.floor(mins / 60);
        if (hours > 0) return `${hours.toLocaleString()} hours`;
        return `${mins} minutes`;
    }
</script>

<div class="p-6 space-y-8">
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
            <Sparkles class="text-emerald-400" size={32} />
            <h2 class="text-3xl font-bold">Your Wrapped</h2>
        </div>
        <div class="flex gap-2">
            <button
                onclick={() => {
                    period = "year";
                    loadWrapped();
                }}
                class="px-4 py-2 rounded-full text-sm {period === 'year'
                    ? 'bg-emerald-500 text-black'
                    : 'bg-zinc-800 text-zinc-400'}"
            >
                This Year
            </button>
            <button
                onclick={() => {
                    period = "all_time";
                    loadWrapped();
                }}
                class="px-4 py-2 rounded-full text-sm {period === 'all_time'
                    ? 'bg-emerald-500 text-black'
                    : 'bg-zinc-800 text-zinc-400'}"
            >
                All Time
            </button>
        </div>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if !isWrappedSeason()}
        <div
            class="flex flex-col items-center justify-center py-20 text-center"
        >
            <Lock class="text-zinc-600 mb-4" size={64} />
            <h3 class="text-2xl font-bold text-zinc-400 mb-2">
                Wrapped Not Available
            </h3>
            <p class="text-zinc-500 max-w-md">
                Your Wrapped summary is only available during the last week of
                each month and throughout December. Check back soon!
            </p>
        </div>
    {:else if wrapped}
        <div class="grid md:grid-cols-2 gap-6">
            <div
                class="bg-gradient-to-br from-emerald-900/50 to-cyan-900/50 border border-emerald-500/30 rounded-2xl p-8"
            >
                <div
                    class="text-6xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent"
                >
                    {formatMinutes(wrapped.total_minutes)}
                </div>
                <p class="text-zinc-400 mt-2">of music listened</p>
            </div>

            <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
                <div class="flex items-center gap-4 mb-4">
                    <Music class="text-emerald-400" size={24} />
                    <span class="text-zinc-400">Top Song</span>
                </div>
                {#if wrapped.top_song}
                    <div class="text-2xl font-bold">
                        {wrapped.top_song.title}
                    </div>
                    <p class="text-zinc-400">
                        {wrapped.top_song.artist?.name || "Unknown"}
                    </p>
                {:else}
                    <p class="text-zinc-500">No data yet</p>
                {/if}
            </div>

            <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
                <div class="flex items-center gap-4 mb-4">
                    <User class="text-cyan-400" size={24} />
                    <span class="text-zinc-400">Top Artist</span>
                </div>
                {#if wrapped.top_artist}
                    <div class="text-2xl font-bold">
                        {wrapped.top_artist.name}
                    </div>
                {:else}
                    <p class="text-zinc-500">No data yet</p>
                {/if}
            </div>

            <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
                <div class="flex items-center gap-4 mb-4">
                    <Disc class="text-purple-400" size={24} />
                    <span class="text-zinc-400">Top Album</span>
                </div>
                {#if wrapped.top_album}
                    <div class="text-2xl font-bold">
                        {wrapped.top_album.title}
                    </div>
                    <p class="text-zinc-400">
                        {wrapped.top_album.artist?.name || "Unknown"}
                    </p>
                {:else}
                    <p class="text-zinc-500">No data yet</p>
                {/if}
            </div>
        </div>

        <div class="grid md:grid-cols-3 gap-4">
            <div
                class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 text-center"
            >
                <div class="text-3xl font-bold text-emerald-400">
                    {wrapped.unique_songs}
                </div>
                <p class="text-zinc-400 mt-1">Unique Songs</p>
            </div>
            <div
                class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 text-center"
            >
                <div class="text-3xl font-bold text-cyan-400">
                    {wrapped.unique_artists}
                </div>
                <p class="text-zinc-400 mt-1">Unique Artists</p>
            </div>
            {#if wrapped.personality}
                <div
                    class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 text-center"
                >
                    <div class="text-xl font-bold text-purple-400">
                        {wrapped.personality}
                    </div>
                    <p class="text-zinc-400 mt-1">Your Personality</p>
                </div>
            {/if}
        </div>

        {#if wrapped.milestones?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Milestones</h3>
                <div class="space-y-2">
                    {#each wrapped.milestones as milestone}
                        <div
                            class="bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3 flex items-center gap-3"
                        >
                            <Sparkles class="text-yellow-400" size={16} />
                            <span>{milestone}</span>
                        </div>
                    {/each}
                </div>
            </section>
        {/if}
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No wrapped data available yet</p>
            <p class="text-sm mt-1">Keep listening to build your story!</p>
        </div>
    {/if}
</div>
