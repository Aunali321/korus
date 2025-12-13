<script lang="ts">
    import { Users, Trophy, Heart } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import type { SocialStats } from "$lib/types";

    let social = $state<SocialStats | null>(null);
    let loading = $state(true);
    let loaded = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadSocial();
        }
    });

    async function loadSocial() {
        loaded = true;
        try {
            social = await api.getSocial();
        } catch (e) {
            console.error("Failed to load social:", e);
        } finally {
            loading = false;
        }
    }
</script>

<div class="p-6 space-y-8">
    <div class="flex items-center gap-3">
        <Users class="text-blue-400" size={32} />
        <h2 class="text-3xl font-bold">Social</h2>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else if social}
        {#if social.your_rank > 0}
            <div
                class="bg-gradient-to-r from-yellow-900/50 to-orange-900/50 border border-yellow-500/30 rounded-2xl p-8 flex items-center gap-6"
            >
                <Trophy class="text-yellow-400" size={48} />
                <div>
                    <div class="text-4xl font-bold">#{social.your_rank}</div>
                    <p class="text-zinc-400">Your Rank</p>
                </div>
            </div>
        {/if}

        {#if social.leaderboard?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                    <Trophy size={20} class="text-yellow-400" />
                    Leaderboard
                </h3>
                <div
                    class="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden"
                >
                    {#each social.leaderboard as entry, i}
                        <div
                            class="flex items-center gap-4 px-4 py-3 {i !==
                            social.leaderboard.length - 1
                                ? 'border-b border-zinc-800'
                                : ''}"
                        >
                            <div
                                class="w-8 text-center font-bold {entry.rank <=
                                3
                                    ? 'text-yellow-400'
                                    : 'text-zinc-500'}"
                            >
                                {entry.rank}
                            </div>
                            <div
                                class="w-10 h-10 rounded-full bg-gradient-to-br from-zinc-700 to-zinc-800 flex items-center justify-center"
                            >
                                <span class="font-bold text-sm"
                                    >{entry.user.username
                                        .charAt(0)
                                        .toUpperCase()}</span
                                >
                            </div>
                            <div class="flex-1">
                                <div class="font-medium">
                                    {entry.user.username}
                                </div>
                            </div>
                            <div class="text-zinc-400">
                                {entry.play_count} plays
                            </div>
                        </div>
                    {/each}
                </div>
            </section>
        {/if}

        {#if social.taste_matches?.length > 0}
            <section>
                <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                    <Heart size={20} class="text-pink-400" />
                    Taste Matches
                </h3>
                <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {#each social.taste_matches as match}
                        <div
                            class="bg-zinc-900 border border-zinc-800 rounded-xl p-4 flex items-center gap-4"
                        >
                            <div
                                class="w-12 h-12 rounded-full bg-gradient-to-br from-pink-700 to-purple-800 flex items-center justify-center"
                            >
                                <span class="font-bold"
                                    >{match.user.username
                                        .charAt(0)
                                        .toUpperCase()}</span
                                >
                            </div>
                            <div class="flex-1">
                                <div class="font-medium">
                                    {match.user.username}
                                </div>
                                <div class="text-sm text-pink-400">
                                    {Math.round(match.similarity * 100)}% match
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            </section>
        {/if}
    {:else}
        <div class="text-center py-12 text-zinc-500">
            <p>No social data available yet</p>
        </div>
    {/if}
</div>
