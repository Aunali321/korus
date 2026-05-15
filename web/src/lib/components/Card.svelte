<script lang="ts">
    import Play from "@lucide/svelte/icons/play";

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

{#snippet body()}
    <div class="relative mb-3">
        {#if image}
            <img
                src={image}
                alt={title}
                class="w-full aspect-square object-cover {rounded
                    ? 'rounded-full'
                    : 'rounded-lg'} bg-zinc-800"
                loading="lazy"
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
                aria-label="Play"
                class="play-fab absolute bottom-2 right-2 w-12 h-12 bg-emerald-500 rounded-full flex items-center justify-center text-black shadow-[0_10px_22px_-10px_rgba(16,185,129,0.7)]"
            >
                <Play size={20} fill="currentColor" class="ml-0.5" />
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
{/snippet}

{#if href}
    <a {href} class="card group rounded-lg p-4 block">
        {@render body()}
    </a>
{:else}
    <div class="card group rounded-lg p-4">
        {@render body()}
    </div>
{/if}

<style>
    .card {
        background-color: rgba(24, 24, 27, 0.5);
        border: 1px solid rgb(39 39 42);
        cursor: pointer;
        transition:
            background-color var(--dur-base) var(--ease-soft),
            border-color var(--dur-base) var(--ease-soft);
    }
    .card:hover {
        background-color: rgb(39 39 42);
        border-color: rgb(63 63 70);
    }
    .play-fab {
        opacity: 0;
        transform: translateY(6px);
        transition:
            opacity 220ms var(--ease-out-expo),
            transform 220ms var(--ease-out-expo);
        will-change: transform, opacity;
    }
    .card:hover .play-fab,
    .card:focus-visible .play-fab {
        opacity: 1;
        transform: translateY(0);
    }
    .play-fab:hover {
        background-color: #34d399;
    }
</style>
