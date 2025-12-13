<script lang="ts">
    import { Play } from "lucide-svelte";
    import type { Snippet } from "svelte";

    let {
        title,
        subtitle,
        image,
        href,
        rounded = false,
        onPlay,
    }: {
        title: string;
        subtitle?: string;
        image?: string;
        href?: string;
        rounded?: boolean;
        onPlay?: () => void;
    } = $props();
</script>

{#if href}
    <a
        {href}
        class="block bg-zinc-900/50 border border-zinc-800 hover:bg-zinc-800 transition-all cursor-pointer group rounded-lg p-4"
    >
        <div class="relative mb-3">
            {#if image}
                <img
                    src={image}
                    alt={title}
                    class="w-full aspect-square object-cover {rounded
                        ? 'rounded-full'
                        : 'rounded-lg'} bg-zinc-800"
                />
            {:else}
                <div
                    class="w-full aspect-square {rounded
                        ? 'rounded-full'
                        : 'rounded-lg'} bg-zinc-800"
                ></div>
            {/if}
            {#if onPlay}
                <button
                    onclick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        onPlay?.();
                    }}
                    class="absolute bottom-2 right-2 w-12 h-12 bg-emerald-500 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transform translate-y-2 group-hover:translate-y-0 transition-all shadow-lg hover:scale-110"
                >
                    <Play
                        size={20}
                        fill="currentColor"
                        class="text-black ml-0.5"
                    />
                </button>
            {/if}
        </div>
        <h4
            class="font-semibold text-sm mb-1 truncate {rounded
                ? 'text-center'
                : ''}"
        >
            {title}
        </h4>
        {#if subtitle}
            <p
                class="text-xs text-zinc-400 truncate {rounded
                    ? 'text-center'
                    : ''}"
            >
                {subtitle}
            </p>
        {/if}
    </a>
{:else}
    <div
        class="bg-zinc-900/50 border border-zinc-800 hover:bg-zinc-800 transition-all cursor-pointer group rounded-lg p-4"
    >
        <div class="relative mb-3">
            {#if image}
                <img
                    src={image}
                    alt={title}
                    class="w-full aspect-square object-cover {rounded
                        ? 'rounded-full'
                        : 'rounded-lg'} bg-zinc-800"
                />
            {:else}
                <div
                    class="w-full aspect-square {rounded
                        ? 'rounded-full'
                        : 'rounded-lg'} bg-zinc-800"
                ></div>
            {/if}
            {#if onPlay}
                <button
                    onclick={() => onPlay?.()}
                    class="absolute bottom-2 right-2 w-12 h-12 bg-emerald-500 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transform translate-y-2 group-hover:translate-y-0 transition-all shadow-lg hover:scale-110"
                >
                    <Play
                        size={20}
                        fill="currentColor"
                        class="text-black ml-0.5"
                    />
                </button>
            {/if}
        </div>
        <h4
            class="font-semibold text-sm mb-1 truncate {rounded
                ? 'text-center'
                : ''}"
        >
            {title}
        </h4>
        {#if subtitle}
            <p
                class="text-xs text-zinc-400 truncate {rounded
                    ? 'text-center'
                    : ''}"
            >
                {subtitle}
            </p>
        {/if}
    </div>
{/if}
