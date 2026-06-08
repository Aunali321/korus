<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { Send, Sparkles } from '@lucide/svelte';
	import { streamChat, runAction, type ChatEvent, type ChatAction } from '$lib/ai';
	import { ask } from '$lib/stores/ask.svelte';
	import { player } from '$lib/stores/player.svelte';
	import SpecRenderer from '$lib/components/SpecRenderer.svelte';

	const TOOL_LABELS: Record<string, string> = {
		search_songs: 'Searched the library',
		get_top_tracks: 'Checked your top tracks',
		get_recent_plays: 'Checked recent plays',
		get_favorites: 'Checked your favorites',
		get_artist_songs: 'Looked up an artist',
		get_now_playing: "Checked what's playing",
		get_queue: 'Checked the queue',
	};
	const toolLabel = (n: string) => TOOL_LABELS[n] ?? n.replace(/_/g, ' ');
	const isReadTool = (n: string) => n.startsWith('get_') || n === 'search_songs';

	let input = $state('');
	let scroller = $state<HTMLElement>();

	const suggestions = [
		'Build me an upbeat playlist for working out',
		'Play something mellow',
		"Add songs similar to what's playing",
		'What have I been listening to lately?',
	];

	async function scrollToBottom() {
		await tick();
		scroller?.scrollTo({ top: scroller.scrollHeight, behavior: 'smooth' });
	}

	onMount(() => {
		if (ask.messages.length > 0) scrollToBottom();
	});

	function actionLabel(ev: ChatAction): string {
		const n = ev.songs?.length ?? 0;
		switch (ev.action) {
			case 'play_now':
				return `▶ Playing ${n} song${n === 1 ? '' : 's'}`;
			case 'queue':
				return `＋ Queued ${n} song${n === 1 ? '' : 's'}${ev.position === 'next' ? ' next' : ''}`;
			case 'clear_queue':
				return '✕ Cleared the queue';
			case 'remove_from_queue':
				return '✕ Removed from queue';
			case 'reorder_queue':
				return '⇅ Reordered the queue';
			case 'playback_control':
				return `⏯ ${ev.control}`;
			case 'playlist_created':
				return '✓ Created a playlist';
			case 'playlist_updated':
				return '✓ Updated a playlist';
			default:
				return ev.action;
		}
	}

	async function send(text: string) {
		text = text.trim();
		if (!text || ask.busy) return;
		input = '';
		ask.busy = true;
		ask.messages.push({ role: 'user', parts: [{ kind: 'text', text }] });
		ask.messages.push({ role: 'assistant', parts: [] });
		const ai = ask.messages[ask.messages.length - 1];
		scrollToBottom();

		const pushText = (delta: string) => {
			const last = ai.parts[ai.parts.length - 1];
			if (last && last.kind === 'text') last.text += delta;
			else ai.parts.push({ kind: 'text', text: delta });
		};

		const playerCtx = {
			now_playing_id: player.currentSong?.id ?? 0,
			queue_ids: player.queue.map((s) => s.id),
		};

		try {
			await streamChat(
				text,
				ask.conversationId,
				(ev: ChatEvent) => {
					switch (ev.type) {
						case 'text':
							pushText(ev.delta);
							break;
						case 'tool':
							if (ev.phase === 'start' && isReadTool(ev.name)) {
								ai.parts.push({ kind: 'tool', label: toolLabel(ev.name) });
							}
							break;
						case 'ui':
							ai.parts.push({ kind: 'ui', spec: ev.spec });
							break;
						case 'action':
							runAction(ev);
							ai.parts.push({ kind: 'action', label: actionLabel(ev) });
							break;
						case 'done':
							ask.conversationId = ev.conversation_id;
							break;
						case 'error':
							pushText(`\n\n⚠️ ${ev.message}`);
							break;
					}
					scrollToBottom();
				},
				playerCtx,
			);
		} catch (e) {
			pushText(`\n\n⚠️ ${e instanceof Error ? e.message : 'Something went wrong.'}`);
		} finally {
			ask.busy = false;
		}
	}
</script>

<div class="flex h-full flex-col">
	{#if ask.messages.length > 0}
		<div class="flex items-center justify-between border-b border-zinc-800 px-4 py-2.5">
			<span class="text-sm font-medium text-zinc-300">Ask</span>
			<button
				onclick={() => ask.reset()}
				class="rounded-lg px-3 py-1 text-xs text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-zinc-200"
			>
				New chat
			</button>
		</div>
	{/if}

	<div bind:this={scroller} class="min-h-0 flex-1 overflow-y-auto px-4 py-6">
		<div class="mx-auto max-w-2xl space-y-6">
			{#if ask.messages.length === 0}
				<div
					class="flex flex-col items-center gap-6 pt-16 text-center"
					in:fly={{ y: 20, duration: 400, easing: cubicOut }}
				>
					<div class="grid h-14 w-14 place-items-center rounded-2xl bg-emerald-500/15 text-emerald-400">
						<Sparkles class="h-7 w-7" />
					</div>
					<div>
						<h1 class="text-2xl font-bold text-white">Ask your library</h1>
						<p class="mt-1 text-zinc-400">Build playlists, play music, explore your stats — just ask.</p>
					</div>
					<div class="grid w-full gap-2 sm:grid-cols-2">
						{#each suggestions as s (s)}
							<button
								onclick={() => send(s)}
								class="rounded-xl border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-left text-sm text-zinc-300 transition-colors hover:border-zinc-700 hover:bg-zinc-800/60"
							>
								{s}
							</button>
						{/each}
					</div>
				</div>
			{/if}

			{#each ask.messages as m, mi (mi)}
				<div
					in:fly={{ y: 12, duration: 300, easing: cubicOut }}
					class="flex {m.role === 'user' ? 'justify-end' : 'justify-start'}"
				>
					<div class={m.role === 'user' ? 'max-w-[80%] rounded-2xl bg-emerald-600 px-4 py-2 text-white' : 'w-full space-y-2'}>
						{#each m.parts as part, pi (pi)}
							{#if part.kind === 'text'}
								{#if part.text.trim()}
									<div class={m.role === 'user' ? '' : 'whitespace-pre-wrap leading-relaxed text-zinc-200'}>
										{part.text}
									</div>
								{/if}
							{:else if part.kind === 'tool'}
								<div class="inline-flex items-center gap-2 text-xs text-zinc-500">
									<span class="h-1 w-1 rounded-full bg-zinc-600"></span>
									{part.label}
								</div>
							{:else if part.kind === 'ui'}
								<div class="rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
									<SpecRenderer node={part.spec} />
								</div>
							{:else if part.kind === 'action'}
								<div class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-3 py-1 text-xs font-medium text-emerald-300">
									{part.label}
								</div>
							{/if}
						{/each}

						{#if m.role === 'assistant' && mi === ask.messages.length - 1 && ask.busy}
							<div class="flex items-center gap-1 pt-1">
								<span class="h-1.5 w-1.5 animate-bounce rounded-full bg-zinc-500"></span>
								<span class="h-1.5 w-1.5 animate-bounce rounded-full bg-zinc-500 [animation-delay:120ms]"></span>
								<span class="h-1.5 w-1.5 animate-bounce rounded-full bg-zinc-500 [animation-delay:240ms]"></span>
							</div>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="border-t border-zinc-800 bg-zinc-950/80 px-4 py-3 backdrop-blur">
		<form
			class="mx-auto flex max-w-2xl items-end gap-2"
			onsubmit={(e) => {
				e.preventDefault();
				send(input);
			}}
		>
			<textarea
				bind:value={input}
				rows="1"
				placeholder="Ask anything about your music…"
				onkeydown={(e) => {
					e.stopPropagation();
					if (e.key === 'Enter' && !e.shiftKey) {
						e.preventDefault();
						send(input);
					}
				}}
				class="max-h-32 flex-1 resize-none rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-2.5 text-sm text-white placeholder-zinc-500 focus:border-emerald-600 focus:outline-none"
			></textarea>
			<button
				type="submit"
				disabled={ask.busy || !input.trim()}
				class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-emerald-600 text-white transition-colors hover:bg-emerald-500 disabled:opacity-40"
				aria-label="Send"
			>
				<Send class="h-5 w-5" />
			</button>
		</form>
	</div>
</div>
