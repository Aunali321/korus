<script lang="ts">
	import "../app.css";
	import { goto, onNavigate } from "$app/navigation";
	import { page } from "$app/stores";
	import { auth } from "$lib/stores/auth.svelte";
	import { player } from "$lib/stores/player.svelte";
	import Sidebar from "$lib/components/Sidebar.svelte";
	import Player from "$lib/components/Player.svelte";
	import Queue from "$lib/components/Queue.svelte";
	import Lyrics from "$lib/components/Lyrics.svelte";
	import Toast from "$lib/components/Toast.svelte";
	import Onboarding from "$lib/components/Onboarding.svelte";
	import ContextMenu from "$lib/components/ContextMenu.svelte";
	import CommandPalette from "$lib/components/CommandPalette.svelte";
	import Menu from "@lucide/svelte/icons/menu";

	let { children } = $props();

	let showQueue = $state(false);
	let showLyrics = $state(false);
	let showOnboarding = $state(false);
	let showSidebar = $state(false);

	const publicRoutes = ["/login", "/register", "/setup"];
	const isPublicRoute = $derived(
		publicRoutes.some((r) => $page.url.pathname.startsWith(r))
	);

	const pageTitle = $derived(
		player.currentSong
			? `${player.currentSong.title} - ${player.currentSong.artists?.map(a => a.name).join(', ') || "Unknown"} | Korus`
			: "Korus"
	);

	// Native View Transitions API — cross-fade routes without remounting.
	onNavigate((navigation) => {
		const doc = document as Document & {
			startViewTransition?: (cb: () => void | Promise<void>) => unknown;
		};
		if (!doc.startViewTransition) return;

		return new Promise<void>((resolve) => {
			doc.startViewTransition!(async () => {
				resolve();
				await navigation.complete;
			});
		});
	});

	$effect(() => {
		if (auth.isAuthenticated && auth.user && !auth.user.onboarded) {
			showOnboarding = true;
		}
	});

	// Post-logout: logout() clears auth state but doesn't navigate; route the
	// user back to /login once they're unauthenticated and not on a public route.
	$effect(() => {
		if (!auth.isAuthenticated && !isPublicRoute) {
			goto("/login");
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

{#if auth.isAuthenticated && !isPublicRoute}
	<div
		class="h-screen h-[100dvh] bg-black text-zinc-100 flex flex-col overflow-hidden font-sans"
	>
		<div class="flex flex-1 min-h-0 overflow-hidden">
			<Sidebar isOpen={showSidebar} onClose={() => (showSidebar = false)} />
			<main
				class="flex-1 overflow-y-auto scrollbar-thin bg-gradient-to-b from-zinc-900 to-black pb-[100px] md:pb-0"
			>
				<div class="sticky top-0 z-30 bg-zinc-900/95 backdrop-blur border-b border-zinc-800 px-4 py-3 flex items-center gap-3 md:hidden">
					<button
						onclick={() => (showSidebar = true)}
						class="p-2 -ml-2 hover:bg-zinc-800 rounded-lg"
						aria-label="Open menu"
					>
						<Menu size={24} />
					</button>
					<img src="/logo.svg" alt="Korus" class="w-6 h-6" />
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
