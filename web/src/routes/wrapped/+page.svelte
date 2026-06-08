<script lang="ts">
	import { onMount } from 'svelte';
	import { RefreshCw } from '@lucide/svelte';
	import { getWrapped } from '$lib/ai';
	import WrappedRenderer from '$lib/components/WrappedRenderer.svelte';

	let period = $state<'month' | 'year'>('month');
	let html = $state<string>('');
	let loading = $state(true);
	let regenerating = $state(false);
	let error = $state('');

	async function load(refresh = false) {
		if (refresh) regenerating = true;
		else loading = true;
		error = '';
		try {
			const res = await getWrapped(period, refresh);
			html = res.html;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load your Wrapped.';
		} finally {
			loading = false;
			regenerating = false;
		}
	}

	function setPeriod(p: 'month' | 'year') {
		if (p !== period && !regenerating) {
			period = p;
			html = '';
			load();
		}
	}

	onMount(() => load());
</script>

<div class="relative min-h-screen bg-[#060608]">
	<div class="absolute top-4 right-4 z-20 flex items-center gap-2">
		<button
			onclick={() => load(true)}
			disabled={loading || regenerating}
			title="Regenerate this Wrapped with a fresh design"
			class="flex items-center gap-1.5 rounded-full bg-black/40 px-3 py-1.5 text-sm text-white/70 backdrop-blur transition-colors hover:text-white disabled:opacity-50"
		>
			<RefreshCw class="h-4 w-4 {regenerating ? 'animate-spin' : ''}" />
			<span class="hidden sm:inline">{regenerating ? 'Regenerating…' : 'Regenerate'}</span>
		</button>
		<div class="flex gap-1 rounded-full bg-black/40 p-1 backdrop-blur">
			<button
				onclick={() => setPeriod('month')}
				class="rounded-full px-4 py-1.5 text-sm transition-colors {period === 'month'
					? 'bg-white/15 text-white'
					: 'text-white/60 hover:text-white'}">Month</button
			>
			<button
				onclick={() => setPeriod('year')}
				class="rounded-full px-4 py-1.5 text-sm transition-colors {period === 'year'
					? 'bg-white/15 text-white'
					: 'text-white/60 hover:text-white'}">Year</button
			>
		</div>
	</div>

	{#if loading}
		<div class="mx-auto max-w-2xl space-y-6 px-6 py-24">
			{#each Array.from({ length: 5 }) as _, i (i)}
				<div class="h-24 animate-pulse rounded-2xl bg-white/5"></div>
			{/each}
		</div>
	{:else if error}
		<div class="px-6 py-24 text-center text-white/60">{error}</div>
	{:else if html}
		<WrappedRenderer {html} />
	{/if}
</div>
