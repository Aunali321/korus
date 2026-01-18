<script lang="ts">
	import "../app.css";
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { page } from "$app/stores";
	import { auth } from "$lib/stores/auth.svelte";
	import { player } from "$lib/stores/player.svelte";
	import { settings } from "$lib/stores/settings.svelte";
	import Sidebar from "$lib/components/Sidebar.svelte";
	import Player from "$lib/components/Player.svelte";
	import Queue from "$lib/components/Queue.svelte";
	import Lyrics from "$lib/components/Lyrics.svelte";
	import Toast from "$lib/components/Toast.svelte";
	import Onboarding from "$lib/components/Onboarding.svelte";
	import ContextMenu from "$lib/components/ContextMenu.svelte";
	import CommandPalette from "$lib/components/CommandPalette.svelte";
	import { Menu } from "lucide-svelte";

	let { children } = $props();
	let showQueue = $state(false);
	let showLyrics = $state(false);
	let showOnboarding = $state(false);
	let showSidebar = $state(false);

	const publicRoutes = ["/login", "/register", "/setup"];

	const pageTitle = $derived(
		player.currentSong
			? `${player.currentSong.title} - ${player.currentSong.artists?.map(a => a.name).join(', ') || "Unknown"} | Korus`
			: "Korus"
	);

	onMount(async () => {
		await auth.init();
		if (auth.isAuthenticated) {
			await settings.load();
			await player.loadState();
		}
	});

	$effect(() => {
		if (auth.isAuthenticated && auth.user && !auth.user.onboarded) {
			showOnboarding = true;
		}
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

	function handleOnboardingComplete() {
		showOnboarding = false;
		if (auth.user) {
			auth.user.onboarded = true;
		}
	}

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
		class="h-screen h-[100dvh] bg-black text-zinc-100 flex flex-col overflow-hidden font-sans"
	>
		<div class="flex flex-1 min-h-0 overflow-hidden">
			<Sidebar isOpen={showSidebar} onClose={() => (showSidebar = false)} />
			<main
				class="flex-1 overflow-y-auto scrollbar-thin bg-gradient-to-b from-zinc-900 to-black pb-[100px] md:pb-0"
			>
				<!-- Mobile header with hamburger -->
				<div class="sticky top-0 z-30 bg-zinc-900/95 backdrop-blur border-b border-zinc-800 px-4 py-3 flex items-center gap-3 md:hidden">
					<button
						onclick={() => (showSidebar = true)}
						class="p-2 -ml-2 hover:bg-zinc-800 rounded-lg transition-colors"
						aria-label="Open menu"
					>
						<Menu size={24} />
					</button>
					<h1 class="text-lg font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent">
						Korus
					</h1>
				</div>
				{@render children()}
			</main>
			<Queue isOpen={showQueue} onClose={() => (showQueue = false)} />
			<Lyrics isOpen={showLyrics} onClose={() => (showLyrics = false)} />
		</div>
		<Player onToggleQueue={() => (showQueue = !showQueue)} onToggleLyrics={() => (showLyrics = !showLyrics)} />
		<Toast />
		<ContextMenu />
		<CommandPalette onToggleQueue={() => (showQueue = !showQueue)} onToggleLyrics={() => (showLyrics = !showLyrics)} />
		{#if showOnboarding}
			<Onboarding onComplete={handleOnboardingComplete} />
		{/if}
	</div>
{:else}
	<div class="min-h-screen bg-black text-zinc-100 font-sans">
		{@render children()}
		<Toast />
	</div>
{/if}
