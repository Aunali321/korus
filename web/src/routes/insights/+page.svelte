<script lang="ts">
    import { Flame, TrendingUp, Lightbulb, Sparkles } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { Insights } from "$lib/types";

    let insights = $state<Insights | null>(null);
    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadInsights();
        }
    });

    async function loadInsights() {
        loaded = true;
        try {
            insights = await api.getInsights();
        } catch (e) {
            console.error("Failed to load insights:", e);
        } finally {
            loading = false;
        }
    }
</script>

<div class="p-6 space-y-8">
    <div class="flex items-center gap-3">
        <Lightbulb class="text-yellow-400" size={32} />
        <h2 class="text-3xl font-bold">Insights</h2>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if insights}
        <div class="grid md:grid-cols-2 gap-6">
            <div
                class="bg-gradient-to-br from-orange-900/50 to-red-900/50 border border-orange-500/30 rounded-2xl p-8"
            >
                <div class="flex items-center gap-3 mb-4">
                    <Flame class="text-orange-400" size={28} />
                    <span class="text-zinc-400">Current Streak</span>
                </div>
                <div class="text-5xl font-bold text-orange-400">
                    {insights.current_streak}
                </div>
                <p class="text-zinc-400 mt-2">days in a row</p>
            </div>

            <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
                <div class="flex items-center gap-3 mb-4">
                    <Flame class="text-red-400" size={28} />
                    <span class="text-zinc-400">Longest Streak</span>
                </div>
                <div class="text-5xl font-bold text-red-400">
                    {insights.longest_streak}
                </div>
                <p class="text-zinc-400 mt-2">days</p>
            </div>
        </div>

        {#if insights.trends?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                    <TrendingUp size={20} class="text-emerald-400" />
                    Trends
                </h3>
                <div class="grid md:grid-cols-3 gap-4">
                    {#each insights.trends as trend}
                        <div
                            class="bg-zinc-900 border border-zinc-800 rounded-xl p-4"
                        >
                            <div class="text-2xl font-bold">{trend.value}</div>
                            <div class="text-sm text-zinc-400">
                                {trend.label}
                            </div>
                            {#if trend.change !== 0}
                                <div
                                    class="text-xs mt-1 {trend.change > 0
                                        ? 'text-emerald-400'
                                        : 'text-red-400'}"
                                >
                                    {trend.change > 0 ? "+" : ""}{trend.change}%
                                </div>
                            {/if}
                        </div>
                    {/each}
                </div>
            </section>
        {/if}

        {#if insights.fun_facts?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                    <Sparkles size={20} class="text-yellow-400" />
                    Fun Facts
                </h3>
                <div class="space-y-2">
                    {#each insights.fun_facts as fact}
                        <div
                            class="bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3"
                        >
                            {fact}
                        </div>
                    {/each}
                </div>
            </section>
        {/if}

        {#if insights.predictions?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4">Predictions</h3>
                <div class="space-y-2">
                    {#each insights.predictions as prediction}
                        <div
                            class="bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3 text-zinc-400"
                        >
                            {prediction}
                        </div>
                    {/each}
                </div>
            </section>
        {/if}
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No insights available yet</p>
        </div>
    {/if}
</div>
