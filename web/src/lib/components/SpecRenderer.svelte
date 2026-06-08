<script lang="ts">
	import { api } from '$lib/api';
	import { goto } from '$app/navigation';
	import { playSongIds, enqueueSongIds, type UINode } from '$lib/ai';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { Play } from '@lucide/svelte';

	let { node }: { node: UINode } = $props();

	function textClass(p: Record<string, any>): string {
		const size =
			{ sm: 'text-sm', base: 'text-base', lg: 'text-lg', xl: 'text-xl' }[p.size as string] ??
			'text-base';
		const weight = p.weight === 'bold' ? 'font-semibold' : 'font-normal';
		const color = p.muted ? 'text-zinc-500' : 'text-zinc-200';
		return `${size} ${weight} ${color} leading-relaxed`;
	}

	function barMax(data: any[]): number {
		return Math.max(1, ...((data ?? []).map((d: any) => Number(d.value) || 0)));
	}

	async function runSpecAction(a: any) {
		if (!a) return;
		switch (a.type) {
			case 'play_now':
				await playSongIds(a.song_ids ?? []);
				break;
			case 'queue':
				await enqueueSongIds(a.song_ids ?? [], a.position === 'next' ? 'next' : 'end');
				break;
			case 'open_album':
				goto(`/albums/${a.album_id}`);
				break;
			case 'open_artist':
				goto(`/artists/${a.artist_id}`);
				break;
			case 'open_playlist':
				goto(`/playlists/${a.playlist_id}`);
				break;
		}
	}
</script>

{#snippet songCard(s: any)}
	{#if s}
		<div class="group flex items-center gap-3 rounded-lg bg-zinc-900/60 p-2 transition-colors hover:bg-zinc-800/70">
			<div class="relative h-12 w-12 shrink-0 overflow-hidden rounded">
				<img src={api.getArtworkUrl(s.id)} alt="" class="h-full w-full object-cover" loading="lazy" />
				<button
					onclick={() => playSongIds([s.id])}
					class="absolute inset-0 grid place-items-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
					aria-label="Play"
				>
					<Play class="h-5 w-5 text-white" fill="white" />
				</button>
			</div>
			<div class="min-w-0">
				<div class="truncate text-sm font-medium text-white">{s.title}</div>
				<div class="truncate text-xs text-zinc-400">{s.artist}</div>
			</div>
		</div>
	{/if}
{/snippet}

{#snippet renderNode(n: UINode, index: number)}
	{@const type = n?.type ?? ''}
	{@const props = n?.props ?? {}}
	<div in:fly={{ y: 14, duration: 360, delay: Math.min(index * 50, 400), easing: cubicOut }}>
		{#if type === 'section'}
			<div class="space-y-3">
				{#if props.title}
					<h3 class="text-sm font-semibold tracking-wide text-zinc-400 uppercase">{props.title}</h3>
				{/if}
				<div
					class="flex {props.direction === 'horizontal' ? 'flex-row flex-wrap' : 'flex-col'}"
					style="gap: {props.gap ?? 12}px"
				>
					{#each n.children ?? [] as child, i (i)}
						{@render renderNode(child, i)}
					{/each}
				</div>
			</div>
		{:else if type === 'heading'}
			<h2 class="text-2xl font-bold text-white">{props.value}</h2>
		{:else if type === 'text'}
			<p class={textClass(props)}>{props.value}</p>
		{:else if type === 'stat'}
			<div class="rounded-xl bg-zinc-900/60 px-4 py-3">
				<div class="text-3xl font-bold text-emerald-400">{props.value}</div>
				<div class="mt-0.5 text-sm text-zinc-400">{props.label}</div>
			</div>
		{:else if type === 'badge'}
			<span class="inline-flex rounded-full bg-emerald-500/15 px-3 py-1 text-xs font-medium text-emerald-300"
				>{props.value}</span
			>
		{:else if type === 'song_card'}
			{@render songCard(props.song)}
		{:else if type === 'song_list'}
			<div class="flex w-full flex-col gap-2">
				{#each props.songs ?? [] as s (s.id)}
					{@render songCard(s)}
				{/each}
			</div>
		{:else if type === 'bar_chart'}
			{@const max = barMax(props.data)}
			<div class="flex w-full flex-col gap-2">
				{#each props.data ?? [] as d (d.label)}
					<div class="flex items-center gap-3">
						<span class="w-28 shrink-0 truncate text-sm text-zinc-300">{d.label}</span>
						<div class="h-2 flex-1 overflow-hidden rounded-full bg-zinc-800">
							<div
								class="h-full rounded-full bg-emerald-500 transition-[width] duration-700 ease-out"
								style="width: {((Number(d.value) || 0) / max) * 100}%"
							></div>
						</div>
						<span class="w-10 shrink-0 text-right text-xs text-zinc-500">{d.value}</span>
					</div>
				{/each}
			</div>
		{:else if type === 'button'}
			<button
				onclick={() => runSpecAction(props.action)}
				class="inline-flex items-center gap-2 rounded-full bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-emerald-500"
			>
				{props.label}
			</button>
		{:else if type === 'divider'}
			<hr class="border-zinc-800" />
		{:else if type === 'image'}
			<img src={props.url} alt={props.alt ?? ''} class="rounded-lg" loading="lazy" />
		{/if}
	</div>
{/snippet}

{@render renderNode(node, 0)}
