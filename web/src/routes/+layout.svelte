<script lang="ts">
	import "../app.css";
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { page } from "$app/stores";
	import { auth } from "$lib/stores/auth.svelte";
	import { player } from "$lib/stores/player.svelte";
	import Sidebar from "$lib/components/Sidebar.svelte";
	import Player from "$lib/components/Player.svelte";
	import Queue from "$lib/components/Queue.svelte";
	import Lyrics from "$lib/components/Lyrics.svelte";
	import Toast from "$lib/components/Toast.svelte";

	let { children } = $props();
	let showQueue = $state(false);
	let showLyrics = $state(false);

	const publicRoutes = ["/login", "/register", "/setup"];

	const pageTitle = $derived(
		player.currentSong
			? `${player.currentSong.title} - ${player.currentSong.artist?.name || "Unknown"} | Korus`
			: "Korus"
	);

	onMount(async () => {
		await auth.init();
	});

	$effect(() => {
		if (!auth.isLoading && !auth.isAuthenticated) {
			const isPublic = publicRoutes.some((r) =>
				$page.url.pathname.startsWith(r),
			);
			if (!isPublic) {
				goto("/login");
			}
		}
	});
</script>

<svelte:head>
	<title>{pageTitle}</title>
	<meta name="description" content="Self-hosted music streaming" />
</svelte:head>

{#if auth.isLoading}
	<div class="h-screen bg-black flex items-center justify-center">
		<div class="text-zinc-400">Loading...</div>
	</div>
{:else if auth.isAuthenticated}
	<div
		class="h-screen bg-black text-zinc-100 flex flex-col overflow-hidden font-sans"
	>
		<div class="flex flex-1 overflow-hidden">
			<Sidebar />
			<main
				class="flex-1 overflow-y-auto scrollbar-thin bg-gradient-to-b from-zinc-900 to-black"
			>
				{@render children()}
			</main>
			<Queue isOpen={showQueue} onClose={() => (showQueue = false)} />
			<Lyrics isOpen={showLyrics} onClose={() => (showLyrics = false)} />
		</div>
		<Player onToggleQueue={() => (showQueue = !showQueue)} onToggleLyrics={() => (showLyrics = !showLyrics)} />
		<Toast />
	</div>
{:else}
	<div class="min-h-screen bg-black text-zinc-100 font-sans">
		{@render children()}
		<Toast />
	</div>
{/if}
