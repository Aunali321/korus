<script lang="ts">
    import { goto } from "$app/navigation";
    import Lock from "@lucide/svelte/icons/lock";
    import type { WrappedData } from "$lib/types";
    import type { Component } from "svelte";
    import type { PageData } from "./$types";

    let { data }: { data: PageData } = $props();

    let ThemeComponent = $state<Component<{ wrapped: WrappedData }> | null>(null);

    const period = $derived(data.period);
    const wrapped = $derived(data.wrapped);
    const inSeason = $derived(data.inSeason);

    function getThemeKey(): string {
        const now = new Date();
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, "0");
        return `${year}-${month}`;
    }

    $effect(() => {
        if (!inSeason) return;
        const key = getThemeKey();
        const modules = import.meta.glob<{ default: Component<{ wrapped: WrappedData }> }>(
            "./themes/*.svelte"
        );
        const path = `./themes/${key}.svelte`;
        const target = modules[path] ?? modules[Object.keys(modules).sort().reverse()[0]];
        target?.().then((mod) => { ThemeComponent = mod.default; });
    });

    function selectPeriod(p: "year" | "month") {
        goto(`?period=${p}`, { replaceState: true, noScroll: true, keepFocus: true });
    }
</script>

{#if !inSeason}
    <div class="flex flex-col items-center justify-center min-h-screen bg-black text-center p-6">
        <Lock class="text-zinc-600 mb-4" size={64} />
        <h3 class="text-2xl font-bold text-zinc-400 mb-2">Wrapped Not Available</h3>
        <p class="text-zinc-500 max-w-md">
            Your Wrapped summary is only available during the last week of each month
            and throughout December. Check back soon!
        </p>
    </div>
{:else if wrapped && ThemeComponent}
    <div class="wrapped-container">
        <div class="period-toggle">
            <button onclick={() => selectPeriod("year")} class:active={period === "year"}>
                This Year
            </button>
            <button onclick={() => selectPeriod("month")} class:active={period === "month"}>
                This Month
            </button>
        </div>
        <ThemeComponent {wrapped} />
    </div>
{:else}
    <div class="flex flex-col items-center justify-center min-h-screen bg-black text-center">
        <p class="text-zinc-500">No wrapped data available yet</p>
        <p class="text-sm text-zinc-600 mt-1">Keep listening to build your story!</p>
    </div>
{/if}

<style>
    .wrapped-container {
        position: relative;
        min-height: 100vh;
    }

    .period-toggle {
        position: fixed;
        top: 1rem;
        right: 1rem;
        z-index: 100;
        display: flex;
        gap: 0.5rem;
    }

    .period-toggle button {
        padding: 0.5rem 1rem;
        border-radius: 9999px;
        font-size: 0.875rem;
        background: rgba(39, 39, 42, 0.8);
        backdrop-filter: blur(8px);
        color: rgb(161, 161, 170);
        border: 1px solid rgba(255, 255, 255, 0.1);
        cursor: pointer;
        transition: all 0.2s;
    }

    .period-toggle button:hover {
        background: rgba(63, 63, 70, 0.8);
    }

    .period-toggle button.active {
        background: rgb(16, 185, 129);
        color: black;
        border-color: transparent;
    }
</style>
